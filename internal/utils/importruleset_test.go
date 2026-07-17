package utils

import (
	"encoding/json"
	"reflect"
	"strings"
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

// TestProcessRulesetsCodeCoverageSendsThresholds guards the fix for the bug where
// minimum_coverage and max_coverage_drop were silently dropped. Because both fields
// are registered as NonOmitEmpty, the create payload must include them even when the
// value is zero (a valid threshold).
func TestProcessRulesetsCodeCoverageSendsThresholds(t *testing.T) {
	ruleset := data.RepoRuleset{
		Name:        "coverage-ruleset",
		Target:      "branch",
		Enforcement: "active",
		Rules: []data.Rules{
			{
				Type: "code_coverage",
				Parameters: &data.Parameters{
					MinimumCoverage: 80,
					MaxCoverageDrop: 0, // zero must still be sent
				},
			},
		},
	}

	got, err := ProcessRulesets(ruleset)
	if err != nil {
		t.Fatalf("ProcessRulesets() error = %v", err)
	}
	if len(got.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(got.Rules))
	}

	payload, err := json.Marshal(got.Rules[0].Parameters)
	if err != nil {
		t.Fatalf("failed to marshal parameters: %v", err)
	}
	js := string(payload)
	if !strings.Contains(js, `"minimum_coverage":80`) {
		t.Errorf("expected minimum_coverage:80 to be sent, got: %s", js)
	}
	if !strings.Contains(js, `"max_coverage_drop":0`) {
		t.Errorf("expected max_coverage_drop:0 to be sent (omitempty must be stripped), got: %s", js)
	}
}

// TestProcessRulesetsParameterlessRule verifies that a rule type with no parameters
// (license_compliance_scanning) is processed without attaching a parameters object.
func TestProcessRulesetsParameterlessRule(t *testing.T) {
	ruleset := data.RepoRuleset{
		Name:        "license-ruleset",
		Target:      "branch",
		Enforcement: "active",
		Rules: []data.Rules{
			{
				Type:       "license_compliance_scanning",
				Parameters: nil,
			},
		},
	}

	got, err := ProcessRulesets(ruleset)
	if err != nil {
		t.Fatalf("ProcessRulesets() error = %v", err)
	}
	if len(got.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(got.Rules))
	}
	if got.Rules[0].Type != "license_compliance_scanning" {
		t.Errorf("expected type license_compliance_scanning, got %q", got.Rules[0].Type)
	}
	if got.Rules[0].Parameters != nil {
		t.Errorf("expected nil parameters for parameterless rule, got %+v", got.Rules[0].Parameters)
	}
}

func TestParsePropertyPatterns(t *testing.T) {
	t.Run("parses multiple patterns with property values", func(t *testing.T) {
		got := parsePropertyPatterns("env;custom;{prod|staging}|tier;custom;{gold}")
		if len(got) != 2 {
			t.Fatalf("parsePropertyPatterns() returned %d patterns, want 2", len(got))
		}
		if got[0].Name != "env" || got[0].Source != "custom" {
			t.Errorf("parsePropertyPatterns()[0] = %+v, want Name env / Source custom", got[0])
		}
		if !reflect.DeepEqual(got[0].PropertyValues, []string{"prod", "staging"}) {
			t.Errorf("parsePropertyPatterns()[0].PropertyValues = %v, want [prod staging]", got[0].PropertyValues)
		}
		if !reflect.DeepEqual(got[1].PropertyValues, []string{"gold"}) {
			t.Errorf("parsePropertyPatterns()[1].PropertyValues = %v, want [gold]", got[1].PropertyValues)
		}
	})

	t.Run("empty string returns no patterns", func(t *testing.T) {
		if got := parsePropertyPatterns(""); len(got) != 0 {
			t.Errorf("parsePropertyPatterns(\"\") = %+v, want empty", got)
		}
	})
}

func TestParseConditions(t *testing.T) {
	conditions := []string{
		"main;develop",       // RefName include
		"release/*",          // RefName exclude
		"repo1;repo2",        // RepositoryName include
		"archived",           // RepositoryName exclude
		"true",               // RepositoryName protected
		"env;custom;{prod}",  // RepositoryProperty include
		"tier;custom;{gold}", // RepositoryProperty exclude
	}

	got := parseConditions(conditions)
	if got == nil {
		t.Fatal("parseConditions() = nil, want conditions")
		return
	}
	if !reflect.DeepEqual(got.RefName.Include, []string{"main", "develop"}) {
		t.Errorf("RefName.Include = %v, want [main develop]", got.RefName.Include)
	}
	if !reflect.DeepEqual(got.RefName.Exclude, []string{"release/*"}) {
		t.Errorf("RefName.Exclude = %v, want [release/*]", got.RefName.Exclude)
	}
	if !got.RepositoryName.Protected {
		t.Error("RepositoryName.Protected = false, want true")
	}
	if len(got.RepositoryProperty.Include) != 1 || got.RepositoryProperty.Include[0].Name != "env" {
		t.Errorf("RepositoryProperty.Include = %+v, want one entry named env", got.RepositoryProperty.Include)
	}
}

func TestParseRules(t *testing.T) {
	g := &APIGetter{}
	headers := []string{"RulesPullRequest", "RulesCreation", "RulesDeletion"}
	values := []string{"DismissStaleReviewsOnPush:true|RequiredApprovingReviewCount:2", "true", ""}

	got := g.parseRules("testorg", headers, values)
	if len(got) != 2 {
		t.Fatalf("parseRules() returned %d rules, want 2 (empty value skipped)", len(got))
	}
	if got[0].Type != "pull_request" {
		t.Errorf("parseRules()[0].Type = %q, want pull_request", got[0].Type)
	}
	if got[0].Parameters == nil || !got[0].Parameters.DismissStaleReviewsOnPush || got[0].Parameters.RequiredApprovingReviewCount != 2 {
		t.Errorf("parseRules()[0].Parameters = %+v, want DismissStaleReviewsOnPush true / count 2", got[0].Parameters)
	}
	if got[1].Type != "creation" {
		t.Errorf("parseRules()[1].Type = %q, want creation", got[1].Type)
	}
	if got[1].Parameters != nil {
		t.Errorf("parseRules()[1].Parameters = %+v, want nil for parameterless rule", got[1].Parameters)
	}
}

func TestCreateRepoRulesetsData(t *testing.T) {
	g := &APIGetter{}
	header := []string{
		"RulesetLevel", "SourceRepositoryName", "TargetRepositoryName", "RuleID", "RulesetName", "Target", "Enforcement", "BypassActors",
		"ConditionsRefNameInclude", "ConditionsRefNameExclude", "ConditionsRepoNameInclude",
		"ConditionsRepoNameExclude", "ConditionsRepoNameProtected", "ConditionRepoPropertyInclude",
		"ConditionRepoPropertyExclude", "RulesPullRequest", "CreatedAt", "UpdatedAt",
	}
	row := []string{
		"Organization", "N/A", "N/A", "1", "test-ruleset", "branch", "active", "",
		"main", "", "", "", "false", "", "",
		"DismissStaleReviewsOnPush:true|RequiredApprovingReviewCount:2",
		"2023-01-01T00:00:00Z", "2023-01-02T00:00:00Z",
	}
	fileData := [][]string{header, row}

	got := g.CreateRepoRulesetsData("testorg", fileData, nil)
	if len(got) != 1 {
		t.Fatalf("CreateRepoRulesetsData() returned %d rulesets, want 1", len(got))
	}
	rs := got[0]
	if rs.Name != "test-ruleset" || rs.Target != "branch" || rs.SourceType != "Organization" {
		t.Errorf("CreateRepoRulesetsData() = %+v, want name test-ruleset / target branch / level Organization", rs)
	}
	if rs.Source != "testorg" {
		t.Errorf("CreateRepoRulesetsData() Source = %q, want testorg", rs.Source)
	}
	if len(rs.Rules) != 1 || rs.Rules[0].Type != "pull_request" {
		t.Errorf("CreateRepoRulesetsData() Rules = %+v, want one pull_request rule", rs.Rules)
	}
}

func TestCreateRepoRulesetsData_TargetRepositoryRename(t *testing.T) {
	g := &APIGetter{}
	header := []string{
		"RulesetLevel", "SourceRepositoryName", "TargetRepositoryName", "RuleID", "RulesetName", "Target", "Enforcement", "BypassActors",
		"ConditionsRefNameInclude", "ConditionsRefNameExclude", "ConditionsRepoNameInclude",
		"ConditionsRepoNameExclude", "ConditionsRepoNameProtected", "ConditionRepoPropertyInclude",
		"ConditionRepoPropertyExclude", "RulesPullRequest", "CreatedAt", "UpdatedAt",
	}
	row := []string{
		"Repository", "old-repo", "new-repo", "2", "repo-ruleset", "branch", "active", "",
		"main", "", "", "", "false", "", "",
		"",
		"2023-01-01T00:00:00Z", "2023-01-02T00:00:00Z",
	}
	fileData := [][]string{header, row}

	got := g.CreateRepoRulesetsData("testorg", fileData, nil)
	if len(got) != 1 {
		t.Fatalf("CreateRepoRulesetsData() returned %d rulesets, want 1", len(got))
	}
	rs := got[0]
	if rs.Source != "testorg/old-repo" {
		t.Errorf("CreateRepoRulesetsData() Source = %q, want testorg/old-repo", rs.Source)
	}
	if rs.TargetSource != "testorg/new-repo" {
		t.Errorf("CreateRepoRulesetsData() TargetSource = %q, want testorg/new-repo", rs.TargetSource)
	}
}

func TestCreateRepoRulesetsData_LegacyRepositoryNameHeader(t *testing.T) {
	g := &APIGetter{}
	header := []string{
		"RulesetLevel", "RepositoryName", "RuleID", "RulesetName", "Target", "Enforcement", "BypassActors",
		"ConditionsRefNameInclude", "ConditionsRefNameExclude", "ConditionsRepoNameInclude",
		"ConditionsRepoNameExclude", "ConditionsRepoNameProtected", "ConditionRepoPropertyInclude",
		"ConditionRepoPropertyExclude", "RulesPullRequest", "CreatedAt", "UpdatedAt",
	}
	row := []string{
		"Repository", "legacy-repo", "3", "repo-ruleset", "branch", "active", "",
		"main", "", "", "", "false", "", "",
		"",
		"2023-01-01T00:00:00Z", "2023-01-02T00:00:00Z",
	}
	fileData := [][]string{header, row}

	got := g.CreateRepoRulesetsData("testorg", fileData, nil)
	if len(got) != 1 {
		t.Fatalf("CreateRepoRulesetsData() returned %d rulesets, want 1", len(got))
	}
	rs := got[0]
	if rs.Source != "testorg/legacy-repo" {
		t.Errorf("CreateRepoRulesetsData() Source = %q, want testorg/legacy-repo", rs.Source)
	}
	if rs.TargetSource != "testorg/legacy-repo" {
		t.Errorf("CreateRepoRulesetsData() TargetSource = %q, want testorg/legacy-repo (defaults to source)", rs.TargetSource)
	}
}
