package utils

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/katiem0/gh-migrate-rulesets/internal/data"
)

// newTestFullGetter builds an APIGetter whose REST and GraphQL clients are both
// backed by the given round-trip function, so functions that mix REST and
// GraphQL calls can be exercised in-process.
func newTestFullGetter(t *testing.T, fn func(*http.Request) (*http.Response, error)) *APIGetter {
	t.Helper()
	rest, err := api.NewRESTClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "test-token",
		Transport: &mockRoundTripper{fn: fn},
	})
	if err != nil {
		t.Fatalf("failed to build REST client: %v", err)
	}
	gql, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "test-token",
		Transport: &mockRoundTripper{fn: fn},
	})
	if err != nil {
		t.Fatalf("failed to build GraphQL client: %v", err)
	}
	return &APIGetter{restClient: rest, gqlClient: gql}
}

func TestUpdateBypassActorID_RepositoryRole(t *testing.T) {
	g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
		path := req.URL.Path
		switch {
		case strings.Contains(path, "custom-repository-roles/100"):
			return jsonResponse(200, `{"name":"role-x","id":100}`), nil
		case strings.Contains(path, "custom-repository-roles"):
			return jsonResponse(200, `{"total_count":1,"custom_roles":[{"name":"role-x","id":555}]}`), nil
		}
		t.Errorf("unexpected request: %s %s", req.Method, path)
		return jsonResponse(404, `{}`), nil
	})

	ruleset := data.RepoRuleset{
		Name: "rs",
		BypassActors: []data.BypassActor{
			{ActorID: intPtr(100), ActorType: "RepositoryRole", BypassMode: "always"},
		},
	}
	got := g.UpdateBypassActorID("neworg", "oldorg", 999, ruleset, g, nil)
	if got.BypassActors[0].ActorID == nil || *got.BypassActors[0].ActorID != 555 {
		t.Errorf("UpdateBypassActorID() RepositoryRole actor ID = %v, want 555", got.BypassActors[0].ActorID)
	}
}

func TestUpdateBypassActorID_Integration(t *testing.T) {
	g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
		path := req.URL.Path
		switch {
		case strings.Contains(path, "installations"):
			return jsonResponse(200, `{"total_count":1,"installations":[{"id":1,"app_id":100,"app_slug":"myapp"}]}`), nil
		case strings.Contains(path, "apps/myapp"):
			return jsonResponse(200, `{"id":777,"slug":"myapp"}`), nil
		}
		t.Errorf("unexpected request: %s %s", req.Method, path)
		return jsonResponse(404, `{}`), nil
	})

	ruleset := data.RepoRuleset{
		Name: "rs",
		BypassActors: []data.BypassActor{
			{ActorID: intPtr(100), ActorType: "Integration", BypassMode: "always"},
		},
	}
	got := g.UpdateBypassActorID("neworg", "oldorg", 999, ruleset, g, nil)
	if got.BypassActors[0].ActorID == nil || *got.BypassActors[0].ActorID != 777 {
		t.Errorf("UpdateBypassActorID() Integration actor ID = %v, want 777", got.BypassActors[0].ActorID)
	}
}

func TestUpdateBypassActorID_Team(t *testing.T) {
	g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
		path := req.URL.Path
		switch {
		case strings.Contains(path, "/team/100"):
			return jsonResponse(200, `{"name":"my-team","id":100,"slug":"my-team"}`), nil
		case strings.Contains(path, "teams/my-team"):
			return jsonResponse(200, `{"name":"my-team","id":888,"slug":"my-team"}`), nil
		}
		t.Errorf("unexpected request: %s %s", req.Method, path)
		return jsonResponse(404, `{}`), nil
	})

	ruleset := data.RepoRuleset{
		Name: "rs",
		BypassActors: []data.BypassActor{
			{ActorID: intPtr(100), ActorType: "Team", BypassMode: "always"},
		},
	}
	got := g.UpdateBypassActorID("neworg", "oldorg", 999, ruleset, g, nil)
	if got.BypassActors[0].ActorID == nil || *got.BypassActors[0].ActorID != 888 {
		t.Errorf("UpdateBypassActorID() Team actor ID = %v, want 888", got.BypassActors[0].ActorID)
	}
}

func TestParseBypassActorsForImport_RepositoryRole(t *testing.T) {
	g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "custom-repository-roles") {
			return jsonResponse(200, `{"total_count":1,"custom_roles":[{"name":"role-x","id":555}]}`), nil
		}
		t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		return jsonResponse(404, `{}`), nil
	})

	got := g.ParseBypassActorsForImport("neworg", "100;RepositoryRole;role-x;always", nil)
	if len(got) != 1 || got[0].ActorID == nil || *got[0].ActorID != 555 {
		t.Errorf("ParseBypassActorsForImport() RepositoryRole = %+v, want actor id 555", got)
	}
}

func TestParseBypassActorsForImport_Integration(t *testing.T) {
	g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "apps/myapp") {
			return jsonResponse(200, `{"id":777,"slug":"myapp"}`), nil
		}
		t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		return jsonResponse(404, `{}`), nil
	})

	got := g.ParseBypassActorsForImport("neworg", "100;Integration;myapp;always", nil)
	if len(got) != 1 || got[0].ActorID == nil || *got[0].ActorID != 777 {
		t.Errorf("ParseBypassActorsForImport() Integration = %+v, want actor id 777", got)
	}
}

func TestParseBypassActorsForImport_Team(t *testing.T) {
	g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "teams/my-team") {
			return jsonResponse(200, `{"name":"my-team","id":888,"slug":"my-team"}`), nil
		}
		t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		return jsonResponse(404, `{}`), nil
	})

	got := g.ParseBypassActorsForImport("neworg", "100;Team;my-team;always", nil)
	if len(got) != 1 || got[0].ActorID == nil || *got[0].ActorID != 888 {
		t.Errorf("ParseBypassActorsForImport() Team = %+v, want actor id 888", got)
	}
}

func TestUpdateRequiredWorkflowRepoID_Real(t *testing.T) {
	g := newTestFullGetter(t, func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "graphql") {
			return jsonResponse(200, `{"data":{"repository":{"databaseId":42,"name":"repo-a","visibility":"private"}}}`), nil
		}
		if strings.Contains(req.URL.Path, "repositories/10") {
			return jsonResponse(200, `{"databaseId":10,"name":"repo-a","visibility":"private"}`), nil
		}
		t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		return jsonResponse(404, `{}`), nil
	})

	ruleset := data.RepoRuleset{
		Rules: []data.Rules{
			{
				Type: "workflows",
				Parameters: &data.Parameters{
					Workflows: []data.Workflows{{Path: ".github/workflows/ci.yml", RepositoryID: 10}},
				},
			},
		},
	}
	got := g.UpdateRequiredWorkflowRepoID("neworg", ruleset, g)
	if got.Rules[0].Parameters.Workflows[0].RepositoryID != 42 {
		t.Errorf("UpdateRequiredWorkflowRepoID() RepositoryID = %d, want 42", got.Rules[0].Parameters.Workflows[0].RepositoryID)
	}
}

func TestParseRequiredWorkflowsForImport_Real(t *testing.T) {
	g := newTestGraphQLGetter(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"data":{"repository":{"databaseId":42,"name":"repo-a","visibility":"private"}}}`), nil
	})

	value := []map[string]string{
		{"RepositoryName": "repo-a", "Path": ".github/workflows/ci.yml", "Ref": "main", "SHA": "abc123"},
	}
	got := g.ParseRequiredWorkflowsForImport("neworg", value)
	if len(got) != 1 || got[0].RepositoryID != 42 {
		t.Errorf("ParseRequiredWorkflowsForImport() = %+v, want one workflow with repo id 42", got)
	}
}

func TestParseRequiredWorkflowsForImport_InvalidType(t *testing.T) {
	g := &APIGetter{}
	if got := g.ParseRequiredWorkflowsForImport("neworg", "not-a-slice"); got != nil {
		t.Errorf("ParseRequiredWorkflowsForImport() = %+v, want nil for invalid type", got)
	}
}

func TestGetAppInstallations_Pagination(t *testing.T) {
	calls := 0
	g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if strings.Contains(req.URL.RawQuery, "page=2") {
			return jsonResponse(200, `{"total_count":2,"installations":[{"id":2,"app_id":20,"app_slug":"b"}]}`), nil
		}
		resp := jsonResponse(200, `{"total_count":2,"installations":[{"id":1,"app_id":10,"app_slug":"a"}]}`)
		resp.Header.Set("Link", `<https://api.github.com/orgs/testorg/installations?page=2&per_page=100>; rel="next"`)
		return resp, nil
	})

	got, err := g.GetAppInstallations("testorg")
	if err != nil {
		t.Fatalf("GetAppInstallations() error = %v", err)
	}
	if len(got.Installations) != 2 {
		t.Errorf("GetAppInstallations() returned %d installations, want 2 (calls=%d)", len(got.Installations), calls)
	}
}

func TestGetRepoCustomRoles_Pagination(t *testing.T) {
	g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.RawQuery, "page=2") {
			return jsonResponse(200, `{"total_count":2,"custom_roles":[{"name":"role-2","id":2}]}`), nil
		}
		resp := jsonResponse(200, `{"total_count":2,"custom_roles":[{"name":"role-1","id":1}]}`)
		resp.Header.Set("Link", `<https://api.github.com/orgs/testorg/custom-repository-roles?page=2&per_page=100>; rel="next"`)
		return resp, nil
	})

	got, err := g.GetRepoCustomRoles("testorg")
	if err != nil {
		t.Fatalf("GetRepoCustomRoles() error = %v", err)
	}
	if len(got.CustomRoles) != 2 {
		t.Errorf("GetRepoCustomRoles() returned %d roles, want 2", len(got.CustomRoles))
	}
}

func TestCSVFieldValue(t *testing.T) {
	if got := csvFieldValue([]string{"  trimmed  ", "b"}, 0); got != "trimmed" {
		t.Errorf("csvFieldValue() = %q, want trimmed", got)
	}
	if got := csvFieldValue([]string{"a"}, 5); got != "" {
		t.Errorf("csvFieldValue() out of range = %q, want empty", got)
	}
}

func TestIsEmptyCSVRecord(t *testing.T) {
	if !isEmptyCSVRecord([]string{"", "   ", "\t"}) {
		t.Error("isEmptyCSVRecord() = false, want true for all-blank record")
	}
	if isEmptyCSVRecord([]string{"", "value"}) {
		t.Error("isEmptyCSVRecord() = true, want false when a field has content")
	}
}

func TestRulesetRequests_TransportError(t *testing.T) {
	g := newTestAPIGetter(t, func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	})

	if err := g.CreateOrgLevelRuleset("o", strings.NewReader("{}")); err == nil {
		t.Error("CreateOrgLevelRuleset() error = nil, want error")
	}
	if err := g.CreateRepoLevelRuleset("o/r", strings.NewReader("{}")); err == nil {
		t.Error("CreateRepoLevelRuleset() error = nil, want error")
	}
	if _, err := g.GetOrgLevelRuleset("o", 1); err == nil {
		t.Error("GetOrgLevelRuleset() error = nil, want error")
	}
	if _, err := g.GetRepoLevelRuleset("o", "r", 1); err == nil {
		t.Error("GetRepoLevelRuleset() error = nil, want error")
	}
	if _, err := g.GetAnApp("some-app"); err == nil {
		t.Error("GetAnApp() error = nil, want error")
	}
	if _, err := g.GetTeamData(1, 2); err == nil {
		t.Error("GetTeamData() error = nil, want error")
	}
}
