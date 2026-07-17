package utils

import (
	"net/http"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/katiem0/gh-migrate-rulesets/internal/data"
)

// newTestGraphQLGetter builds an APIGetter whose GraphQL client is backed by the
// given round-trip function so GraphQL methods can be exercised in-process.
func newTestGraphQLGetter(t *testing.T, fn func(*http.Request) (*http.Response, error)) *APIGetter {
	t.Helper()
	gql, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "test-token",
		Transport: &mockRoundTripper{fn: fn},
	})
	if err != nil {
		t.Fatalf("failed to build GraphQL client: %v", err)
	}
	return &APIGetter{gqlClient: gql}
}

func TestGetRepoGraphQL(t *testing.T) {
	g := newTestGraphQLGetter(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"data":{"repository":{"databaseId":5,"name":"myrepo","visibility":"public"}}}`), nil
	})
	repo, err := g.GetRepo("testorg", "myrepo")
	if err != nil {
		t.Fatalf("GetRepo() error = %v", err)
	}
	if repo.Repository.DatabaseId != 5 || repo.Repository.Name != "myrepo" {
		t.Errorf("GetRepo() = %+v, want databaseId 5 name myrepo", repo.Repository)
	}
}

func TestFetchOrgId(t *testing.T) {
	g := newTestGraphQLGetter(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"data":{"organization":{"databaseId":123}}}`), nil
	})
	org, err := g.FetchOrgId("testorg")
	if err != nil {
		t.Fatalf("FetchOrgId() error = %v", err)
	}
	if org.Organization.DatabaseID != 123 {
		t.Errorf("FetchOrgId() databaseId = %d, want 123", org.Organization.DatabaseID)
	}
}

func TestFetchOrgRulesets(t *testing.T) {
	g := newTestGraphQLGetter(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"data":{"organization":{"rulesets":{"nodes":[{"id":"R1","databaseId":1,"name":"ruleset-1"}],"pageInfo":{"endCursor":"","hasNextPage":false}}}}}`), nil
	})
	rulesets, err := g.FetchOrgRulesets("testorg")
	if err != nil {
		t.Fatalf("FetchOrgRulesets() error = %v", err)
	}
	if len(rulesets) != 1 || rulesets[0].Name != "ruleset-1" {
		t.Errorf("FetchOrgRulesets() = %+v, want one ruleset named ruleset-1", rulesets)
	}
}

func TestFetchRepoRulesets(t *testing.T) {
	g := newTestGraphQLGetter(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"data":{"repository":{"rulesets":{"nodes":[{"id":"R2","databaseId":2,"name":"repo-ruleset"}],"pageInfo":{"endCursor":"","hasNextPage":false}}}}}`), nil
	})
	repos := []data.RepoInfo{{DatabaseId: 1, Name: "myrepo"}}
	rules, err := g.FetchRepoRulesets("testorg", repos)
	if err != nil {
		t.Fatalf("FetchRepoRulesets() error = %v", err)
	}
	if len(rules) != 1 || rules[0].RepoName != "myrepo" || rules[0].Rule.Name != "repo-ruleset" {
		t.Errorf("FetchRepoRulesets() = %+v, want one rule for myrepo", rules)
	}
}

func TestGatherRepositories(t *testing.T) {
	g := newTestGraphQLGetter(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"data":{"organization":{"repositories":{"totalCount":1,"nodes":[{"databaseId":1,"name":"repo-a","visibility":"public"}],"pageInfo":{"endCursor":"","hasNextPage":false}}}}}`), nil
	})
	repos := g.GatherRepositories("testorg", []string{})
	if len(repos) != 1 || repos[0].Name != "repo-a" {
		t.Errorf("GatherRepositories() = %+v, want one repo named repo-a", repos)
	}
}

func TestGatherRepositoriesByName(t *testing.T) {
	g := newTestGraphQLGetter(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(200, `{"data":{"repository":{"databaseId":7,"name":"named-repo","visibility":"private"}}}`), nil
	})
	repos := g.GatherRepositories("testorg", []string{"named-repo"})
	if len(repos) != 1 || repos[0].Name != "named-repo" {
		t.Errorf("GatherRepositories() = %+v, want one repo named named-repo", repos)
	}
}

func TestGraphQLError(t *testing.T) {
	g := newTestGraphQLGetter(t, func(req *http.Request) (*http.Response, error) {
		return jsonResponse(500, `{"message":"boom"}`), nil
	})
	if _, err := g.GetRepo("testorg", "myrepo"); err == nil {
		t.Error("GetRepo() expected error on 500 response, got nil")
	}
}
