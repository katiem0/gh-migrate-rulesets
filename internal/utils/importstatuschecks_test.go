package utils

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
)

func statusCheckRuleset(checks []data.StatusChecks) data.RepoRuleset {
	return data.RepoRuleset{
		Rules: []data.Rules{
			{
				Type: "required_status_checks",
				Parameters: &data.Parameters{
					RequiredStatusChecks: checks,
				},
			},
		},
	}
}

func TestUpdateStatusCheckIntegrationID(t *testing.T) {
	g := &APIGetter{}

	tests := []struct {
		name    string
		ruleset data.RepoRuleset
		source  *MockWorkflowGetter
		wantID  *int
	}{
		{
			name: "nil integration id is left unchanged",
			ruleset: statusCheckRuleset([]data.StatusChecks{
				{Context: "build", IntegrationID: nil},
			}),
			source: &MockWorkflowGetter{
				AppInstallations: &data.AppIntegrations{Installations: []data.AppInstallation{}},
			},
			wantID: nil,
		},
		{
			name: "zero integration id is left unchanged",
			ruleset: statusCheckRuleset([]data.StatusChecks{
				{Context: "build", IntegrationID: intPtr(0)},
			}),
			source: &MockWorkflowGetter{
				AppInstallations: &data.AppIntegrations{Installations: []data.AppInstallation{}},
			},
			wantID: intPtr(0),
		},
		{
			name: "installations lookup error leaves id unchanged",
			ruleset: statusCheckRuleset([]data.StatusChecks{
				{Context: "build", IntegrationID: intPtr(42)},
			}),
			source: &MockWorkflowGetter{
				AppInstallationsErr: true,
			},
			wantID: intPtr(42),
		},
		{
			name: "no matching installation leaves id unchanged",
			ruleset: statusCheckRuleset([]data.StatusChecks{
				{Context: "build", IntegrationID: intPtr(42)},
			}),
			source: &MockWorkflowGetter{
				AppInstallations: &data.AppIntegrations{
					Installations: []data.AppInstallation{
						{AppID: 99, AppSlug: "other-app"},
					},
				},
			},
			wantID: intPtr(42),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := g.UpdateStatusCheckIntegrationID("sourceorg", tt.ruleset, tt.source)
			gotID := got.Rules[0].Parameters.RequiredStatusChecks[0].IntegrationID
			switch {
			case tt.wantID == nil && gotID != nil:
				t.Errorf("IntegrationID = %d, want nil", *gotID)
			case tt.wantID != nil && gotID == nil:
				t.Errorf("IntegrationID = nil, want %d", *tt.wantID)
			case tt.wantID != nil && gotID != nil && *gotID != *tt.wantID:
				t.Errorf("IntegrationID = %d, want %d", *gotID, *tt.wantID)
			}
		})
	}
}

func TestUpdateStatusCheckIntegrationID_NonStatusCheckRuleUntouched(t *testing.T) {
	g := &APIGetter{}
	ruleset := data.RepoRuleset{
		Rules: []data.Rules{
			{Type: "creation", Parameters: nil},
		},
	}

	got, _ := g.UpdateStatusCheckIntegrationID("sourceorg", ruleset, &MockWorkflowGetter{})
	if got.Rules[0].Type != "creation" || got.Rules[0].Parameters != nil {
		t.Errorf("non required_status_checks rule was modified: %+v", got.Rules[0])
	}
}

func TestUpdateStatusCheckIntegrationID_TargetAppRewrite(t *testing.T) {
	source := func() *MockWorkflowGetter {
		return &MockWorkflowGetter{
			AppInstallations: &data.AppIntegrations{
				Installations: []data.AppInstallation{
					{AppID: 42, AppSlug: "ci-app"},
				},
			},
		}
	}

	t.Run("happy path rewrites matching integration id", func(t *testing.T) {
		g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
			if req.Method != "GET" || !strings.Contains(req.URL.Path, "apps/ci-app") {
				t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			}
			return jsonResponse(200, `{"id":777,"slug":"ci-app"}`), nil
		})
		ruleset := statusCheckRuleset([]data.StatusChecks{
			{Context: "build", IntegrationID: intPtr(42)},
		})

		got, _ := g.UpdateStatusCheckIntegrationID("sourceorg", ruleset, source())

		assertFirstRuleCheckIntegrationID(t, got, 0, 777)
	})

	t.Run("target app lookup error leaves id unchanged", func(t *testing.T) {
		g := newTestAPIGetter(t, func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("target app lookup failed")
		})
		ruleset := statusCheckRuleset([]data.StatusChecks{
			{Context: "build", IntegrationID: intPtr(42)},
		})

		got, _ := g.UpdateStatusCheckIntegrationID("sourceorg", ruleset, source())

		assertFirstRuleCheckIntegrationID(t, got, 0, 42)
	})

	t.Run("mixed checks rewrites matching id only", func(t *testing.T) {
		g := newTestAPIGetter(t, func(_ *http.Request) (*http.Response, error) {
			return jsonResponse(200, `{"id":777,"slug":"ci-app"}`), nil
		})
		ruleset := statusCheckRuleset([]data.StatusChecks{
			{Context: "build", IntegrationID: intPtr(42)},
			{Context: "lint", IntegrationID: intPtr(99)},
		})

		got, _ := g.UpdateStatusCheckIntegrationID("sourceorg", ruleset, source())

		assertFirstRuleCheckIntegrationID(t, got, 0, 777)
		assertFirstRuleCheckIntegrationID(t, got, 1, 99)
	})
}

func assertFirstRuleCheckIntegrationID(t *testing.T, ruleset data.RepoRuleset, index int, want int) {
	t.Helper()

	gotID := ruleset.Rules[0].Parameters.RequiredStatusChecks[index].IntegrationID
	if gotID == nil {
		t.Fatalf("IntegrationID = nil, want %d", want)
	}
	if *gotID != want {
		t.Fatalf("IntegrationID = %d, want %d", *gotID, want)
	}
}

// TestUpdateStatusCheckIntegrationID_ErrorCases asserts on the returned error slice
// (the existing table test discards it), including the synthesised no-match branch.
func TestUpdateStatusCheckIntegrationID_ErrorCases(t *testing.T) {
	t.Run("installations lookup error reports error and leaves id unchanged", func(t *testing.T) {
		g := &APIGetter{}
		ruleset := statusCheckRuleset([]data.StatusChecks{{Context: "build", IntegrationID: intPtr(42)}})

		got, err := g.UpdateStatusCheckIntegrationID("sourceorg", ruleset, &MockWorkflowGetter{AppInstallationsErr: true})

		if err == nil {
			t.Fatalf("expected an error for installations lookup failure, got none")
		}
		assertFirstRuleCheckIntegrationID(t, got, 0, 42)
	})

	t.Run("integration absent from source reports error and leaves id unchanged", func(t *testing.T) {
		g := &APIGetter{} // target GetAnApp never called when there is no source match
		ruleset := statusCheckRuleset([]data.StatusChecks{{Context: "build", IntegrationID: intPtr(42)}})
		source := &MockWorkflowGetter{AppInstallations: &data.AppIntegrations{
			Installations: []data.AppInstallation{{AppID: 99, AppSlug: "other-app"}},
		}}

		got, err := g.UpdateStatusCheckIntegrationID("sourceorg", ruleset, source)

		if err == nil {
			t.Fatalf("expected an error for unmatched integration, got none")
		}
		if !strings.Contains(err.Error(), "build") {
			t.Errorf("error should name the status check context, got %v", err)
		}
		assertFirstRuleCheckIntegrationID(t, got, 0, 42)
	})

	t.Run("happy path reports no error", func(t *testing.T) {
		g := newTestAPIGetter(t, func(_ *http.Request) (*http.Response, error) {
			return jsonResponse(200, `{"id":777,"slug":"ci-app"}`), nil
		})
		ruleset := statusCheckRuleset([]data.StatusChecks{{Context: "build", IntegrationID: intPtr(42)}})
		source := &MockWorkflowGetter{AppInstallations: &data.AppIntegrations{
			Installations: []data.AppInstallation{{AppID: 42, AppSlug: "ci-app"}},
		}}

		got, err := g.UpdateStatusCheckIntegrationID("sourceorg", ruleset, source)

		if err != nil {
			t.Fatalf("expected no errors, got %v", err)
		}
		assertFirstRuleCheckIntegrationID(t, got, 0, 777)
	})

	t.Run("target app lookup error reports error and leaves id unchanged", func(t *testing.T) {
		g := newTestAPIGetter(t, func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("target app lookup failed")
		})
		ruleset := statusCheckRuleset([]data.StatusChecks{{Context: "build", IntegrationID: intPtr(42)}})
		source := &MockWorkflowGetter{AppInstallations: &data.AppIntegrations{
			Installations: []data.AppInstallation{{AppID: 42, AppSlug: "ci-app"}},
		}}

		got, err := g.UpdateStatusCheckIntegrationID("sourceorg", ruleset, source)

		if err == nil {
			t.Fatalf("expected an error for target app lookup failure, got none")
		}
		assertFirstRuleCheckIntegrationID(t, got, 0, 42)
	})

	// A ruleset whose checks are all context-only (no integration ID) has nothing to
	// translate, so a failing installations lookup must not run and must not skip the
	// ruleset. Guards against re-hoisting the fetch ahead of the per-check need guard.
	t.Run("context-only checks skip the lookup even when it would fail", func(t *testing.T) {
		g := &APIGetter{}
		ruleset := statusCheckRuleset([]data.StatusChecks{
			{Context: "build", IntegrationID: nil},
			{Context: "lint", IntegrationID: intPtr(0)},
		})

		got, err := g.UpdateStatusCheckIntegrationID("sourceorg", ruleset, &MockWorkflowGetter{AppInstallationsErr: true})

		if err != nil {
			t.Fatalf("context-only checks should not trigger the lookup, got error %v", err)
		}
		if got.Rules[0].Parameters.RequiredStatusChecks[0].IntegrationID != nil {
			t.Errorf("nil IntegrationID was modified")
		}
		if id := got.Rules[0].Parameters.RequiredStatusChecks[1].IntegrationID; id == nil || *id != 0 {
			t.Errorf("zero IntegrationID was modified: %v", id)
		}
	})
}
