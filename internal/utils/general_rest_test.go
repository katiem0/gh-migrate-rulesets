package utils

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

// mockRoundTripper lets tests intercept REST requests without touching the network.
type mockRoundTripper struct {
	fn func(*http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.fn(req)
}

// newTestAPIGetter builds an APIGetter whose REST client is backed by the given
// round-trip function so REST methods can be exercised in-process.
func newTestAPIGetter(t *testing.T, fn func(*http.Request) (*http.Response, error)) *APIGetter {
	t.Helper()
	rest, err := api.NewRESTClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "test-token",
		Transport: &mockRoundTripper{fn: fn},
	})
	if err != nil {
		t.Fatalf("failed to build REST client: %v", err)
	}
	return &APIGetter{restClient: rest}
}

// newTestAPIGetterWithGraphQL is like newTestAPIGetter but also backs the GraphQL
// client so REST and GraphQL methods can be exercised together. The round-trip
// function should dispatch on req.URL.Path; GraphQL requests hit the "/graphql" path.
func newTestAPIGetterWithGraphQL(t *testing.T, fn func(*http.Request) (*http.Response, error)) *APIGetter {
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

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestCreateOrgLevelRuleset(t *testing.T) {
	g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != "POST" || !strings.Contains(req.URL.Path, "orgs/testorg/rulesets") {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		return jsonResponse(201, `{"id":1}`), nil
	})
	if err := g.CreateOrgLevelRuleset("testorg", strings.NewReader(`{}`)); err != nil {
		t.Errorf("CreateOrgLevelRuleset() error = %v, want nil", err)
	}
}

func TestCreateRepoLevelRuleset(t *testing.T) {
	g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != "POST" || !strings.Contains(req.URL.Path, "repos/testorg/repo/rulesets") {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		return jsonResponse(201, `{"id":1}`), nil
	})
	if err := g.CreateRepoLevelRuleset("testorg/repo", strings.NewReader(`{}`)); err != nil {
		t.Errorf("CreateRepoLevelRuleset() error = %v, want nil", err)
	}
}

func TestGetOrgLevelRuleset(t *testing.T) {
	g := newTestAPIGetter(t, func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"id":42,"name":"rs"}`), nil
	})
	got, err := g.GetOrgLevelRuleset("testorg", 42)
	if err != nil {
		t.Fatalf("GetOrgLevelRuleset() error = %v", err)
	}
	if !strings.Contains(string(got), `"id":42`) {
		t.Errorf("GetOrgLevelRuleset() = %s, want body containing id 42", got)
	}
}

func TestGetRepoLevelRuleset(t *testing.T) {
	g := newTestAPIGetter(t, func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"id":7,"name":"rs"}`), nil
	})
	got, err := g.GetRepoLevelRuleset("testorg", "repo", 7)
	if err != nil {
		t.Fatalf("GetRepoLevelRuleset() error = %v", err)
	}
	if !strings.Contains(string(got), `"id":7`) {
		t.Errorf("GetRepoLevelRuleset() = %s, want body containing id 7", got)
	}
}

func TestGetAnApp(t *testing.T) {
	g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Path, "apps/my-app") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return jsonResponse(200, `{"id":123,"slug":"my-app"}`), nil
	})
	got, err := g.GetAnApp("my-app")
	if err != nil {
		t.Fatalf("GetAnApp() error = %v", err)
	}
	if got.AppID != 123 || got.AppSlug != "my-app" {
		t.Errorf("GetAnApp() = %+v, want AppID 123 / slug my-app", got)
	}
}

func TestGetAppInstallations(t *testing.T) {
	g := newTestAPIGetter(t, func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"total_count":1,"installations":[{"id":1,"app_id":2,"app_slug":"x"}]}`), nil
	})
	got, err := g.GetAppInstallations("testorg")
	if err != nil {
		t.Fatalf("GetAppInstallations() error = %v", err)
	}
	if got.TotalCount != 1 || len(got.Installations) != 1 {
		t.Errorf("GetAppInstallations() = %+v, want 1 installation", got)
	}
}

func TestGetCustomRoles(t *testing.T) {
	g := newTestAPIGetter(t, func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"name":"custom","id":5,"base_role":"write"}`), nil
	})
	got, err := g.GetCustomRoles("testorg", 5)
	if err != nil {
		t.Fatalf("GetCustomRoles() error = %v", err)
	}
	if got.ID != 5 || got.Name != "custom" {
		t.Errorf("GetCustomRoles() = %+v, want ID 5 / name custom", got)
	}
}

func TestGetRepoCustomRoles(t *testing.T) {
	g := newTestAPIGetter(t, func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"total_count":1,"custom_roles":[{"name":"r","id":9,"base_role":"read"}]}`), nil
	})
	got, err := g.GetRepoCustomRoles("testorg")
	if err != nil {
		t.Fatalf("GetRepoCustomRoles() error = %v", err)
	}
	if got.TotalCount != 1 || len(got.CustomRoles) != 1 || got.CustomRoles[0].ID != 9 {
		t.Errorf("GetRepoCustomRoles() = %+v, want one role with ID 9", got)
	}
}

func TestGetRepoByID(t *testing.T) {
	g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Path, "repositories/7") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return jsonResponse(200, `{"databaseId":7,"name":"repo","visibility":"public"}`), nil
	})
	got, err := g.GetRepoByID(7)
	if err != nil {
		t.Fatalf("GetRepoByID() error = %v", err)
	}
	if got.Name != "repo" || got.Visibility != "public" {
		t.Errorf("GetRepoByID() = %+v, want name repo / visibility public", got)
	}
}

func TestGetTeamData(t *testing.T) {
	g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Path, "organizations/10/team/3") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return jsonResponse(200, `{"name":"team","id":3,"slug":"team"}`), nil
	})
	got, err := g.GetTeamData(10, 3)
	if err != nil {
		t.Fatalf("GetTeamData() error = %v", err)
	}
	if got.ID != 3 || got.Slug != "team" {
		t.Errorf("GetTeamData() = %+v, want ID 3 / slug team", got)
	}
}

func TestGetTeamByName(t *testing.T) {
	g := newTestAPIGetter(t, func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Path, "orgs/testorg/teams/team-slug") {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		return jsonResponse(200, `{"name":"team","id":4,"slug":"team-slug"}`), nil
	})
	got, err := g.GetTeamByName("testorg", "team-slug")
	if err != nil {
		t.Fatalf("GetTeamByName() error = %v", err)
	}
	if got.ID != 4 {
		t.Errorf("GetTeamByName() = %+v, want ID 4", got)
	}
}
