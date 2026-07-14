package utils

import "testing"

func TestGetNextPageURLHostAgnostic(t *testing.T) {
	tests := []struct {
		name       string
		linkHeader string
		want       string
	}{
		{
			name:       "github dotcom",
			linkHeader: `<https://api.github.com/orgs/x/installations?page=2>; rel="next"`,
			want:       "orgs/x/installations?page=2",
		},
		{
			name:       "ghes strips api v3",
			linkHeader: `<https://ghe.example.com/api/v3/orgs/x/installations?page=2>; rel="next"`,
			want:       "orgs/x/installations?page=2",
		},
		{
			name:       "data residency",
			linkHeader: `<https://api.tenant.ghe.com/orgs/x/installations?page=2>; rel="next"`,
			want:       "orgs/x/installations?page=2",
		},
		{
			name:       "no next rel",
			linkHeader: `<https://api.github.com/orgs/x/installations?page=2>; rel="last"`,
			want:       "",
		},
		{
			name:       "empty input",
			linkHeader: "",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNextPageURL(tt.linkHeader); got != tt.want {
				t.Errorf("getNextPageURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
