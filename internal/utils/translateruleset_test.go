package utils

import (
	"net/http"
	"strings"
	"testing"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
)

// translateFixtureRuleset builds a ruleset that touches all three translation stages:
// a bypass integration actor, a required-workflow rule, and a status-check rule.
func translateFixtureRuleset() data.RepoRuleset {
	return data.RepoRuleset{
		Name: "test-ruleset",
		BypassActors: []data.BypassActor{
			{ActorID: intPtr(42), ActorType: "Integration", BypassMode: "always"},
		},
		Rules: []data.Rules{
			{
				Type: "workflows",
				Parameters: &data.Parameters{
					Workflows: []data.Workflows{{Path: ".github/workflows/ci.yml", Ref: "main", RepositoryID: 123}},
				},
			},
			{
				Type: "required_status_checks",
				Parameters: &data.Parameters{
					RequiredStatusChecks: []data.StatusChecks{{Context: "build", IntegrationID: intPtr(42)}},
				},
			},
		},
	}
}

// Regression guard: every stage's error must survive, so a future refactor can't
// reassign the error and silently drop the earlier stages.
func TestTranslateRuleset_AccumulatesAllStageErrors(t *testing.T) {
	// Source fails every lookup: ShouldError breaks GetRepoByID (stage 2) and
	// AppInstallationsErr breaks GetAppInstallations (stages 1 and 3).
	s := &MockWorkflowGetter{ShouldError: true, AppInstallationsErr: true}
	g := &APIGetter{} // target never reached because every stage fails at the source

	_, err := TranslateRuleset(g, s, "neworg", "oldorg", 123, translateFixtureRuleset(), nil)

	if err == nil {
		t.Fatalf("expected errors from all three stages, got nil")
	}
	joined := err.Error()
	// Distinct per-stage substrings prove all three stages ran and reported (no short-circuit).
	for _, want := range []string{"bypass actor", "required workflow", "status check"} {
		if !strings.Contains(joined, want) {
			t.Errorf("accumulated errors missing %q stage, got:\n%s", want, joined)
		}
	}
}

// TestTranslateRuleset_HappyPath verifies that when every stage resolves, no errors are
// returned and all three ID categories are translated to their target values.
func TestTranslateRuleset_HappyPath(t *testing.T) {
	g := newTestAPIGetterWithGraphQL(t, func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "graphql") {
			return jsonResponse(200, `{"data":{"repository":{"databaseId":789,"name":"source-repo"}}}`), nil
		}
		if strings.Contains(req.URL.Path, "apps/ci-app") {
			return jsonResponse(200, `{"id":777,"slug":"ci-app"}`), nil
		}
		return jsonResponse(200, `{}`), nil
	})
	s := &MockWorkflowGetter{
		AppInstallations: &data.AppIntegrations{
			Installations: []data.AppInstallation{{AppID: 42, AppSlug: "ci-app"}},
		},
		RepoByIDInfo: &data.RepoInfo{DatabaseId: 123, Name: "source-repo"},
	}

	updated, err := TranslateRuleset(g, s, "neworg", "oldorg", 123, translateFixtureRuleset(), nil)

	if err != nil {
		t.Fatalf("expected no errors, got %v", err)
	}
	if got := updated.BypassActors[0].ActorID; got == nil || *got != 777 {
		t.Errorf("bypass actor ID = %v, want 777", got)
	}
	if got := updated.Rules[0].Parameters.Workflows[0].RepositoryID; got != 789 {
		t.Errorf("workflow repository ID = %d, want 789", got)
	}
	if got := updated.Rules[1].Parameters.RequiredStatusChecks[0].IntegrationID; got == nil || *got != 777 {
		t.Errorf("status check integration ID = %v, want 777", got)
	}
}
