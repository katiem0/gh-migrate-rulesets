package utils

import (
	"fmt"
	"net/http"
	"testing"
)

func TestGetAuthToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		hostname string
		want     string
	}{
		{
			name:     "provided token",
			token:    "ghp_testtoken123",
			hostname: "github.com",
			want:     "ghp_testtoken123",
		},
		{
			name:     "empty token",
			token:    "",
			hostname: "github.com",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetAuthToken(tt.token, tt.hostname)
			if tt.token != "" && got != tt.want {
				t.Errorf("GetAuthToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetNextPageURL(t *testing.T) {
	tests := []struct {
		name       string
		linkHeader string
		want       string
	}{
		{
			name:       "valid next link",
			linkHeader: `<https://api.github.com/orgs/test/repos?page=2>; rel="next", <https://api.github.com/orgs/test/repos?page=5>; rel="last"`,
			want:       "orgs/test/repos?page=2",
		},
		{
			name:       "ghes strips api v3 prefix",
			linkHeader: `<https://github.example.com/api/v3/orgs/test/repos?page=2>; rel="next", <https://github.example.com/api/v3/orgs/test/repos?page=5>; rel="last"`,
			want:       "orgs/test/repos?page=2",
		},
		{
			name:       "ghe.com data residency",
			linkHeader: `<https://api.tenant.ghe.com/orgs/test/repos?page=2>; rel="next"`,
			want:       "orgs/test/repos?page=2",
		},
		{
			name:       "next link without query",
			linkHeader: `<https://api.github.com/orgs/test/repos>; rel="next"`,
			want:       "orgs/test/repos",
		},
		{
			name:       "no next link",
			linkHeader: `<https://api.github.com/orgs/test/repos?page=1>; rel="first", <https://api.github.com/orgs/test/repos?page=5>; rel="last"`,
			want:       "",
		},
		{
			name:       "empty link header",
			linkHeader: "",
			want:       "",
		},
		{
			name:       "malformed link header",
			linkHeader: "not a valid link header",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getNextPageURL(tt.linkHeader)
			if got != tt.want {
				t.Errorf("getNextPageURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRepoExists(t *testing.T) {
	t.Run("existing repo returns true", func(t *testing.T) {
		g := newTestAPIGetter(t, func(_ *http.Request) (*http.Response, error) {
			return jsonResponse(200, `{"id":1}`), nil
		})
		if !g.RepoExists("testorg/repo") {
			t.Error("RepoExists() = false, want true")
		}
	})

	t.Run("missing repo returns false", func(t *testing.T) {
		g := newTestAPIGetter(t, func(_ *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("not found")
		})
		if g.RepoExists("testorg/missing") {
			t.Error("RepoExists() = true, want false")
		}
	})
}
