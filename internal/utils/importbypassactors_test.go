package utils

import (
	"net/http"
	"strings"
	"testing"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
)

func TestParseBypassActorsForImport(t *testing.T) {
	g := &APIGetter{}

	tests := []struct {
		name            string
		owner           string
		bypassActorsStr string
		actorMapping    map[string]int
		wantCount       int
		wantFirstID     *int
	}{
		{
			name:            "empty string",
			owner:           "testorg",
			bypassActorsStr: "",
			wantCount:       0,
		},
		{
			name:            "single actor with known role",
			owner:           "testorg",
			bypassActorsStr: "1;OrgAdmin;OrgAdmin;always",
			wantCount:       1,
		},
		{
			name:            "multiple actors",
			owner:           "testorg",
			bypassActorsStr: "1;OrgAdmin;OrgAdmin;always|5;Admin;Admin;pull_request",
			wantCount:       2,
		},
		{
			name:            "deploy key actor",
			owner:           "testorg",
			bypassActorsStr: "0;DeployKey;AllDeployKeys;always",
			wantCount:       1,
		},
		{
			name:            "invalid format",
			owner:           "testorg",
			bypassActorsStr: "invalid",
			wantCount:       0,
		},
		{
			name:            "predefined role remapped via actor mapping",
			owner:           "testorg",
			bypassActorsStr: "5;RepositoryRole;Admin;always",
			actorMapping:    map[string]int{actorMappingKey("RepositoryRole", 5): 42},
			wantCount:       1,
			wantFirstID:     intPtr(42),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.ParseBypassActorsForImport(tt.owner, tt.bypassActorsStr, tt.actorMapping)
			if len(got) != tt.wantCount {
				t.Errorf("ParseBypassActorsForImport() returned %d actors, want %d", len(got), tt.wantCount)
			}
			if tt.wantFirstID != nil {
				if len(got) == 0 || got[0].ActorID == nil || *got[0].ActorID != *tt.wantFirstID {
					t.Errorf("ParseBypassActorsForImport() first actor ID = %v, want %d", got, *tt.wantFirstID)
				}
			}
		})
	}
}

func TestUpdateBypassActorID(t *testing.T) {
	g := &APIGetter{}
	s := &APIGetter{}

	tests := []struct {
		name         string
		owner        string
		sourceOrg    string
		sourceOrgID  int
		ruleset      data.RepoRuleset
		actorMapping map[string]int
		wantActors   int
		wantActorID  *int
	}{
		{
			name:        "empty bypass actors",
			owner:       "neworg",
			sourceOrg:   "oldorg",
			sourceOrgID: 12345,
			ruleset: data.RepoRuleset{
				Name:         "test-ruleset",
				BypassActors: []data.BypassActor{},
			},
			wantActors: 0,
		},
		{
			name:        "deploy key actor",
			owner:       "neworg",
			sourceOrg:   "oldorg",
			sourceOrgID: 12345,
			ruleset: data.RepoRuleset{
				Name: "test-ruleset",
				BypassActors: []data.BypassActor{
					{
						ActorID:    nil,
						ActorType:  "DeployKey",
						BypassMode: "always",
					},
				},
			},
			wantActors: 1,
		},
		{
			name:        "known role actors",
			owner:       "neworg",
			sourceOrg:   "oldorg",
			sourceOrgID: 12345,
			ruleset: data.RepoRuleset{
				Name: "test-ruleset",
				BypassActors: []data.BypassActor{
					{
						ActorID:    intPtr(1),
						ActorType:  "OrgAdmin",
						BypassMode: "always",
					},
					{
						ActorID:    intPtr(5),
						ActorType:  "Admin",
						BypassMode: "pull_request",
					},
				},
			},
			wantActors: 2,
		},
		{
			name:        "predefined role overridden by actor mapping",
			owner:       "neworg",
			sourceOrg:   "oldorg",
			sourceOrgID: 12345,
			ruleset: data.RepoRuleset{
				Name: "test-ruleset",
				BypassActors: []data.BypassActor{
					{
						ActorID:    intPtr(5),
						ActorType:  "RepositoryRole",
						BypassMode: "always",
					},
				},
			},
			actorMapping: map[string]int{actorMappingKey("RepositoryRole", 5): 7},
			wantActors:   1,
			wantActorID:  intPtr(7),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := g.UpdateBypassActorID(tt.owner, tt.sourceOrg, tt.sourceOrgID, tt.ruleset, s, tt.actorMapping)
			if len(got.BypassActors) != tt.wantActors {
				t.Errorf("UpdateBypassActorID() returned %d actors, want %d", len(got.BypassActors), tt.wantActors)
			}
			if tt.wantActorID != nil {
				if got.BypassActors[0].ActorID == nil || *got.BypassActors[0].ActorID != *tt.wantActorID {
					t.Errorf("UpdateBypassActorID() actor ID = %v, want %d", got.BypassActors[0].ActorID, *tt.wantActorID)
				}
			}
		})
	}
}

// bypassRuleset builds a ruleset with a single bypass actor.
func bypassRuleset(actor data.BypassActor) data.RepoRuleset {
	return data.RepoRuleset{Name: "test-ruleset", BypassActors: []data.BypassActor{actor}}
}

func firstBypassActorID(t *testing.T, ruleset data.RepoRuleset) *int {
	t.Helper()
	if len(ruleset.BypassActors) == 0 {
		t.Fatalf("ruleset has no bypass actors")
	}
	return ruleset.BypassActors[0].ActorID
}

func assertActorID(t *testing.T, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Fatalf("ActorID = nil, want %d", want)
	}
	if *got != want {
		t.Fatalf("ActorID = %d, want %d", *got, want)
	}
}

// TestUpdateBypassActorID_RepositoryRole exercises the production custom repository
// role branch, including the synthesised no-match error that has no underlying API error.
func TestUpdateBypassActorID_RepositoryRole(t *testing.T) {
	// ActorID 100 is not in RolesMap (0-5), so the custom-role lookup path runs.
	actor := data.BypassActor{ActorID: intPtr(100), ActorType: "RepositoryRole", BypassMode: "always"}

	t.Run("happy path translates role id", func(t *testing.T) {
		g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.Path, "custom-repository-roles") {
				t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			}
			return jsonResponse(200, `{"total_count":1,"custom_roles":[{"name":"platform-admin","id":4242}]}`), nil
		})
		s := &MockWorkflowGetter{CustomRole: &data.CustomRole{Name: "platform-admin", ID: 100}}

		got, err := g.UpdateBypassActorID("neworg", "oldorg", 123, bypassRuleset(actor), s, nil)

		if err != nil {
			t.Fatalf("expected no errors, got %v", err)
		}
		assertActorID(t, firstBypassActorID(t, got), 4242)
	})

	t.Run("no matching role in target reports error and leaves id unchanged", func(t *testing.T) {
		g := newTestAPIGetter(t, func(_ *http.Request) (*http.Response, error) {
			return jsonResponse(200, `{"total_count":1,"custom_roles":[{"name":"different-role","id":4242}]}`), nil
		})
		s := &MockWorkflowGetter{CustomRole: &data.CustomRole{Name: "platform-admin", ID: 100}}

		got, err := g.UpdateBypassActorID("neworg", "oldorg", 123, bypassRuleset(actor), s, nil)

		if err == nil {
			t.Fatalf("expected an error for unresolved role, got none")
		}
		if !strings.Contains(err.Error(), "platform-admin") {
			t.Errorf("error should name the unresolved role, got %v", err)
		}
		assertActorID(t, firstBypassActorID(t, got), 100)
	})

	t.Run("empty role list in target reports error", func(t *testing.T) {
		g := newTestAPIGetter(t, func(_ *http.Request) (*http.Response, error) {
			return jsonResponse(200, `{"total_count":0,"custom_roles":[]}`), nil
		})
		s := &MockWorkflowGetter{CustomRole: &data.CustomRole{Name: "platform-admin", ID: 100}}

		got, err := g.UpdateBypassActorID("neworg", "oldorg", 123, bypassRuleset(actor), s, nil)

		if err == nil {
			t.Fatalf("expected an error for empty role list, got none")
		}
		assertActorID(t, firstBypassActorID(t, got), 100)
	})

	t.Run("source role lookup error reports error and leaves id unchanged", func(t *testing.T) {
		g := &APIGetter{} // target never reached when source lookup fails
		s := &MockWorkflowGetter{CustomRolesErr: true}

		got, err := g.UpdateBypassActorID("neworg", "oldorg", 123, bypassRuleset(actor), s, nil)

		if err == nil {
			t.Fatalf("expected an error for source lookup failure, got none")
		}
		assertActorID(t, firstBypassActorID(t, got), 100)
	})
}

// TestUpdateBypassActorID_Integration exercises the integration branch, including the
// synthesised no-match error when the actor ID is absent from the source installations.
func TestUpdateBypassActorID_Integration(t *testing.T) {
	actor := data.BypassActor{ActorID: intPtr(42), ActorType: "Integration", BypassMode: "always"}

	t.Run("happy path translates integration id", func(t *testing.T) {
		g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.Path, "apps/ci-app") {
				t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			}
			return jsonResponse(200, `{"id":777,"slug":"ci-app"}`), nil
		})
		s := &MockWorkflowGetter{AppInstallations: &data.AppIntegrations{
			Installations: []data.AppInstallation{{AppID: 42, AppSlug: "ci-app"}},
		}}

		got, err := g.UpdateBypassActorID("neworg", "oldorg", 123, bypassRuleset(actor), s, nil)

		if err != nil {
			t.Fatalf("expected no errors, got %v", err)
		}
		assertActorID(t, firstBypassActorID(t, got), 777)
	})

	t.Run("actor absent from source installations reports error and leaves id unchanged", func(t *testing.T) {
		g := &APIGetter{} // target GetAnApp never called when there is no source match
		s := &MockWorkflowGetter{AppInstallations: &data.AppIntegrations{
			Installations: []data.AppInstallation{{AppID: 99, AppSlug: "other-app"}},
		}}

		got, err := g.UpdateBypassActorID("neworg", "oldorg", 123, bypassRuleset(actor), s, nil)

		if err == nil {
			t.Fatalf("expected an error for unmatched integration, got none")
		}
		assertActorID(t, firstBypassActorID(t, got), 42)
	})

	t.Run("source installations lookup error reports error", func(t *testing.T) {
		g := &APIGetter{}
		s := &MockWorkflowGetter{AppInstallationsErr: true}

		got, err := g.UpdateBypassActorID("neworg", "oldorg", 123, bypassRuleset(actor), s, nil)

		if err == nil {
			t.Fatalf("expected an error for source installations failure, got none")
		}
		assertActorID(t, firstBypassActorID(t, got), 42)
	})
}

// TestUpdateBypassActorID_MappingPathNoError verifies that when actorMapping resolves an
// actor, the ID is translated and no error is produced for that actor.
func TestUpdateBypassActorID_MappingPathNoError(t *testing.T) {
	actor := data.BypassActor{ActorID: intPtr(100), ActorType: "RepositoryRole", BypassMode: "always"}
	g := &APIGetter{} // no API calls: the mapping short-circuits lookups
	s := &MockWorkflowGetter{}
	mapping := map[string]int{actorMappingKey("RepositoryRole", 100): 55}

	got, err := g.UpdateBypassActorID("neworg", "oldorg", 123, bypassRuleset(actor), s, mapping)

	if err != nil {
		t.Fatalf("mapping path must not report errors, got %v", err)
	}
	assertActorID(t, firstBypassActorID(t, got), 55)
}

// The match sits in the MIDDLE of the list deliberately: the old loop reset the resolved
// ID back to the source on every non-matching iteration, so a trailing miss discarded it.
func TestParseBypassActorsForImport_CustomRoleOrdering(t *testing.T) {
	t.Run("match in the middle resolves to target role id", func(t *testing.T) {
		g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.Path, "custom-repository-roles") {
				t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			}
			// Match ("platform-admin") is NOT last; "after-role" follows it.
			return jsonResponse(200, `{"total_count":3,"custom_roles":[{"name":"before-role","id":11},{"name":"platform-admin","id":4242},{"name":"after-role","id":99}]}`), nil
		})

		// Source ID 100 is not in RolesMap, so the custom-role lookup path runs.
		got := g.ParseBypassActorsForImport("neworg", "100;RepositoryRole;platform-admin;always", nil)

		if len(got) != 1 {
			t.Fatalf("ParseBypassActorsForImport() returned %d actors, want 1", len(got))
		}
		assertActorID(t, got[0].ActorID, 4242)
	})

	t.Run("no match falls back to source id", func(t *testing.T) {
		g := newTestAPIGetter(t, func(_ *http.Request) (*http.Response, error) {
			return jsonResponse(200, `{"total_count":2,"custom_roles":[{"name":"before-role","id":11},{"name":"after-role","id":99}]}`), nil
		})

		got := g.ParseBypassActorsForImport("neworg", "100;RepositoryRole;missing-role;always", nil)

		if len(got) != 1 {
			t.Fatalf("ParseBypassActorsForImport() returned %d actors, want 1", len(got))
		}
		// From-file path has no error rollup: an unresolved role keeps the source ID.
		assertActorID(t, got[0].ActorID, 100)
	})
}
