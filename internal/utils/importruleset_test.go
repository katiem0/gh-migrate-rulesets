package utils

import (
	"reflect"
	"testing"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
)

func TestCleanSlice(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		want  []string
	}{
		{
			name:  "remove empty strings",
			slice: []string{"a", "", "b", "", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "no empty strings",
			slice: []string{"a", "b", "c"},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "all empty strings",
			slice: []string{"", "", ""},
			want:  []string{},
		},
		{
			name:  "nil slice",
			slice: nil,
			want:  nil,
		},
		{
			name:  "single element",
			slice: []string{"a"},
			want:  []string{"a"},
		},
		{
			name:  "single empty element",
			slice: []string{""},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanSlice(tt.slice)
			if tt.want == nil && got != nil {
				t.Errorf("CleanSlice() = %v, want %v", got, tt.want)
			} else if tt.want != nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CleanSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldRemoveRepositoryName(t *testing.T) {
	tests := []struct {
		name     string
		repoName *data.NamePatterns
		want     bool
	}{
		{
			name:     "nil pointer",
			repoName: nil,
			want:     false,
		},
		{
			name: "empty include/exclude, not protected",
			repoName: &data.NamePatterns{
				Include:   []string{},
				Exclude:   []string{},
				Protected: false,
			},
			want: true,
		},
		{
			name: "empty include/exclude, protected",
			repoName: &data.NamePatterns{
				Include:   []string{},
				Exclude:   []string{},
				Protected: true,
			},
			want: false,
		},
		{
			name: "has include",
			repoName: &data.NamePatterns{
				Include:   []string{"repo1"},
				Exclude:   []string{},
				Protected: false,
			},
			want: false,
		},
		{
			name: "has exclude",
			repoName: &data.NamePatterns{
				Include:   []string{},
				Exclude:   []string{"temp-*"},
				Protected: false,
			},
			want: false,
		},
		{
			name: "has both include and exclude",
			repoName: &data.NamePatterns{
				Include:   []string{"repo1"},
				Exclude:   []string{"temp-*"},
				Protected: false,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldRemoveRepositoryName(tt.repoName); got != tt.want {
				t.Errorf("ShouldRemoveRepositoryName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldRemoveRefName(t *testing.T) {
	tests := []struct {
		name    string
		refName *data.RefPatterns
		want    bool
	}{
		{
			name:    "nil pointer",
			refName: nil,
			want:    false,
		},
		{
			name: "empty include and exclude",
			refName: &data.RefPatterns{
				Include: []string{},
				Exclude: []string{},
			},
			want: true,
		},
		{
			name: "has include",
			refName: &data.RefPatterns{
				Include: []string{"main"},
				Exclude: []string{},
			},
			want: false,
		},
		{
			name: "has exclude",
			refName: &data.RefPatterns{
				Include: []string{},
				Exclude: []string{"dev"},
			},
			want: false,
		},
		{
			name: "has both include and exclude",
			refName: &data.RefPatterns{
				Include: []string{"main", "develop"},
				Exclude: []string{"feature/*"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldRemoveRefName(tt.refName); got != tt.want {
				t.Errorf("ShouldRemoveRefName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldRemoveProperty(t *testing.T) {
	tests := []struct {
		name     string
		propName *data.PropertyPatterns
		want     bool
	}{
		{
			name:     "nil pointer",
			propName: nil,
			want:     false,
		},
		{
			name: "empty include and exclude",
			propName: &data.PropertyPatterns{
				Include: []data.PropertyPattern{},
				Exclude: []data.PropertyPattern{},
			},
			want: true,
		},
		{
			name: "has include",
			propName: &data.PropertyPatterns{
				Include: []data.PropertyPattern{
					{Name: "visibility", Source: "github", PropertyValues: []string{"public"}},
				},
				Exclude: []data.PropertyPattern{},
			},
			want: false,
		},
		{
			name: "has exclude",
			propName: &data.PropertyPatterns{
				Include: []data.PropertyPattern{},
				Exclude: []data.PropertyPattern{
					{Name: "archived", Source: "github", PropertyValues: []string{"true"}},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldRemoveProperty(tt.propName); got != tt.want {
				t.Errorf("ShouldRemoveProperty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetermineSource(t *testing.T) {
	tests := []struct {
		name       string
		owner      string
		sourceType string
		repoName   string
		want       string
	}{
		{
			name:       "organization source",
			owner:      "testorg",
			sourceType: "Organization",
			repoName:   "testrepo",
			want:       "testorg",
		},
		{
			name:       "repository source",
			owner:      "testorg",
			sourceType: "Repository",
			repoName:   "testrepo",
			want:       "testorg/testrepo",
		},
		{
			name:       "unknown source type",
			owner:      "testorg",
			sourceType: "Unknown",
			repoName:   "testrepo",
			want:       "",
		},
		{
			name:       "empty values",
			owner:      "",
			sourceType: "",
			repoName:   "",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := determineSource(tt.owner, tt.sourceType, tt.repoName); got != tt.want {
				t.Errorf("determineSource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCleanConditions(t *testing.T) {
	tests := []struct {
		name       string
		conditions *data.Conditions
		want       *data.Conditions
	}{
		{
			name: "clean empty conditions",
			conditions: &data.Conditions{
				RefName: &data.RefPatterns{
					Include: []string{"", "main", ""},
					Exclude: []string{"", "dev"},
				},
				RepositoryName: &data.NamePatterns{
					Include:   []string{},
					Exclude:   []string{},
					Protected: false,
				},
				RepositoryProperty: &data.PropertyPatterns{
					Include: []data.PropertyPattern{},
					Exclude: []data.PropertyPattern{},
				},
			},
			want: &data.Conditions{
				RefName: &data.RefPatterns{
					Include: []string{"main"},
					Exclude: []string{"dev"},
				},
				RepositoryName:     nil,
				RepositoryProperty: nil,
			},
		},
		{
			name:       "nil conditions",
			conditions: nil,
			want:       nil,
		},
		{
			name: "keep protected repository name",
			conditions: &data.Conditions{
				RepositoryName: &data.NamePatterns{
					Include:   []string{},
					Exclude:   []string{},
					Protected: true,
				},
			},
			want: &data.Conditions{
				RepositoryName: &data.NamePatterns{
					Include:   []string{},
					Exclude:   []string{},
					Protected: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanConditions(tt.conditions)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CleanConditions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessRulesets(t *testing.T) {
	tests := []struct {
		name    string
		ruleset data.RepoRuleset
		wantErr bool
	}{
		{
			name: "basic ruleset",
			ruleset: data.RepoRuleset{
				Name:         "test-ruleset",
				Target:       "branch",
				Enforcement:  "active",
				BypassActors: []data.BypassActor{},
				Conditions:   nil,
				Rules: []data.Rules{
					{
						Type:       "creation",
						Parameters: nil,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "ruleset with parameters",
			ruleset: data.RepoRuleset{
				Name:        "test-ruleset",
				Target:      "branch",
				Enforcement: "active",
				Rules: []data.Rules{
					{
						Type: "pull_request",
						Parameters: &data.Parameters{
							RequiredApprovingReviewCount: 2,
							DismissStaleReviewsOnPush:    true,
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProcessRulesets(tt.ruleset)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessRulesets() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if got.Name != tt.ruleset.Name {
					t.Errorf("ProcessRulesets() name = %v, want %v", got.Name, tt.ruleset.Name)
				}
				if len(got.Rules) != len(tt.ruleset.Rules) {
					t.Errorf("ProcessRulesets() rules count = %v, want %v", len(got.Rules), len(tt.ruleset.Rules))
				}
			}
		})
	}
}
