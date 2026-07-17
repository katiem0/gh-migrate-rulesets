package utils

import (
	"net/http"
	"strings"
	"testing"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
)

func TestNewAPIGetter(t *testing.T) {
	g := NewAPIGetter(nil, nil)
	if g == nil {
		t.Fatal("NewAPIGetter() returned nil")
	}
}

func TestProcessActorsForExportWithREST(t *testing.T) {
	teamID := 100
	intID := 200
	roleID := 300
	badID := 400
	deployID := 0

	g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.Path, "team/100"):
			return jsonResponse(200, `{"name":"my-team","id":100,"slug":"my-team"}`), nil
		case strings.Contains(req.URL.Path, "installations"):
			return jsonResponse(200, `{"total_count":1,"installations":[{"id":1,"app_id":200,"app_slug":"my-app"}]}`), nil
		case strings.Contains(req.URL.Path, "custom-repository-roles/300"):
			return jsonResponse(200, `{"name":"my-role","id":300,"base_role":"write"}`), nil
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			return jsonResponse(404, `{}`), nil
		}
	})

	actors := []data.BypassActor{
		{ActorID: &deployID, ActorType: "DeployKey", BypassMode: "always"},
		{ActorID: &teamID, ActorType: "Team", BypassMode: "always"},
		{ActorID: &intID, ActorType: "Integration", BypassMode: "always"},
		{ActorID: &roleID, ActorType: "RepositoryRole", BypassMode: "always"},
		{ActorID: &badID, ActorType: "Nonsense", BypassMode: "always"},
		{ActorID: nil, ActorType: "OrganizationAdmin", BypassMode: "pull_request"},
	}

	result := g.ProcessActorsForExport(actors, "testorg", 42, "ruleset-1")

	if len(result) != len(actors) {
		t.Fatalf("ProcessActorsForExport() returned %d entries, want %d", len(result), len(actors))
	}

	joined := strings.Join(result, "\n")
	for _, want := range []string{"my-team", "my-app", "my-role", "Unknown-400"} {
		if !strings.Contains(joined, want) {
			t.Errorf("ProcessActorsForExport() result missing %q; got:\n%s", want, joined)
		}
	}
}

func TestParseRequiredWorkflowsForImportInvalidType(t *testing.T) {
	g := NewAPIGetter(nil, nil)
	workflows := g.ParseRequiredWorkflowsForImport("testorg", "not-a-slice")
	if workflows != nil {
		t.Errorf("expected nil workflows for invalid type, got %v", workflows)
	}
}
