package utils

import (
	"reflect"
	"strings"
	"testing"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
)

func TestProcessProperties(t *testing.T) {
	tests := []struct {
		name       string
		properties []data.PropertyPattern
		want       []string
	}{
		{
			name: "single property",
			properties: []data.PropertyPattern{
				{
					Name:           "test-property",
					Source:         "custom",
					PropertyValues: []string{"value1", "value2"},
				},
			},
			want: []string{"test-property;custom;{value1|value2}"},
		},
		{
			name:       "empty properties",
			properties: []data.PropertyPattern{},
			want:       nil,
		},
		{
			name: "multiple properties",
			properties: []data.PropertyPattern{
				{
					Name:           "prop1",
					Source:         "source1",
					PropertyValues: []string{"val1"},
				},
				{
					Name:           "prop2",
					Source:         "source2",
					PropertyValues: []string{"val2", "val3"},
				},
			},
			want: []string{"prop1;source1;{val1}", "prop2;source2;{val2|val3}"},
		},
		{
			name: "property with empty values",
			properties: []data.PropertyPattern{
				{
					Name:           "empty-prop",
					Source:         "source",
					PropertyValues: []string{},
				},
			},
			want: []string{"empty-prop;source;{}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProcessProperties(tt.properties)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProcessProperties() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessConditions(t *testing.T) {
	tests := []struct {
		name    string
		ruleset data.RepoRuleset
		want    data.ProcessedConditions
	}{
		{
			name: "full conditions",
			ruleset: data.RepoRuleset{
				Conditions: &data.Conditions{
					RefName: &data.RefPatterns{
						Include: []string{"main", "develop"},
						Exclude: []string{"feature/*"},
					},
					RepositoryName: &data.NamePatterns{
						Include:   []string{"repo1", "repo2"},
						Exclude:   []string{"temp-*"},
						Protected: true,
					},
					RepositoryProperty: &data.PropertyPatterns{
						Include: []data.PropertyPattern{
							{
								Name:           "visibility",
								Source:         "github",
								PropertyValues: []string{"public"},
							},
						},
						Exclude: []data.PropertyPattern{
							{
								Name:           "archived",
								Source:         "github",
								PropertyValues: []string{"true"},
							},
						},
					},
				},
			},
			want: data.ProcessedConditions{
				IncludeRefNames: "main;develop",
				ExcludeRefNames: "feature/*",
				IncludeNames:    "repo1;repo2",
				ExcludeNames:    "temp-*",
				BoolNames:       "true",
				PropertyInclude: []string{"visibility;github;{public}"},
				PropertyExclude: []string{"archived;github;{true}"},
			},
		},
		{
			name: "nil conditions",
			ruleset: data.RepoRuleset{
				Conditions: nil,
			},
			want: data.ProcessedConditions{
				IncludeRefNames: "",
				ExcludeRefNames: "",
				IncludeNames:    "",
				ExcludeNames:    "",
				BoolNames:       "",
				PropertyInclude: nil,
				PropertyExclude: nil,
			},
		},
		{
			name: "partial conditions",
			ruleset: data.RepoRuleset{
				Conditions: &data.Conditions{
					RefName: &data.RefPatterns{
						Include: []string{"main"},
						Exclude: []string{},
					},
					RepositoryName:     nil,
					RepositoryProperty: nil,
				},
			},
			want: data.ProcessedConditions{
				IncludeRefNames: "main",
				ExcludeRefNames: "",
				IncludeNames:    "",
				ExcludeNames:    "",
				BoolNames:       "",
				PropertyInclude: nil,
				PropertyExclude: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProcessConditions(tt.ruleset)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProcessConditions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessRules(t *testing.T) {
	g := &APIGetter{}

	tests := []struct {
		name  string
		rules []data.Rules
		want  map[string]string
	}{
		{
			name: "rules with parameters",
			rules: []data.Rules{
				{
					Type: "pull_request",
					Parameters: &data.Parameters{
						RequiredApprovingReviewCount: 2,
						DismissStaleReviewsOnPush:    true,
					},
				},
			},
			want: map[string]string{
				"pull_request": "",
			},
		},
		{
			name: "rules without parameters",
			rules: []data.Rules{
				{
					Type:       "creation",
					Parameters: nil,
				},
			},
			want: map[string]string{
				"creation": "true",
			},
		},
		{
			name:  "empty rules",
			rules: []data.Rules{},
			want:  map[string]string{},
		},
		{
			name: "multiple rules",
			rules: []data.Rules{
				{
					Type:       "deletion",
					Parameters: nil,
				},
				{
					Type:       "required_signatures",
					Parameters: nil,
				},
			},
			want: map[string]string{
				"deletion":            "true",
				"required_signatures": "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.ProcessRules(tt.rules)

			if tt.name == "rules with parameters" {
				value, exists := got["pull_request"]
				if !exists {
					t.Errorf("ProcessRules() missing key 'pull_request'")
				}
				if !strings.Contains(value, "RequiredApprovingReviewCount:2") {
					t.Errorf("ProcessRules() value doesn't contain RequiredApprovingReviewCount:2")
				}
				if !strings.Contains(value, "DismissStaleReviewsOnPush:true") {
					t.Errorf("ProcessRules() value doesn't contain DismissStaleReviewsOnPush:true")
				}
			} else {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("ProcessRules() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestProcessActorsForExport(t *testing.T) {
	g := &APIGetter{}

	tests := []struct {
		name   string
		actors []data.BypassActor
		owner  string
		orgID  int
		ruleID string
		want   int
	}{
		{
			name:   "empty actors",
			actors: []data.BypassActor{},
			owner:  "testorg",
			orgID:  12345,
			ruleID: "R_123",
			want:   0,
		},
		{
			name: "actors with known roles",
			actors: []data.BypassActor{
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
			owner:  "testorg",
			orgID:  12345,
			ruleID: "R_123",
			want:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.ProcessActorsForExport(tt.actors, tt.owner, tt.orgID, tt.ruleID)
			if len(got) != tt.want {
				t.Errorf("ProcessActorsForExport() returned %d actors, want %d", len(got), tt.want)
			}
		})
	}
}

// Helper function for creating int pointers
func intPtr(i int) *int {
	return &i
}
