package list

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
)

type MockAPIGetter struct {
	OrgID              int
	OrgRulesets        []data.Rulesets
	Repos              []data.RepoInfo
	RepoRulesets       []data.RepoNameRule
	ShouldError        bool
	ShouldErrorRuleset bool
}

func (m *MockAPIGetter) FetchOrgId(owner string) (*data.OrgIdQuery, error) {
	if m.ShouldError {
		return nil, errors.New("mock error fetching org ID")
	}
	return &data.OrgIdQuery{
		Organization: struct {
			DatabaseID int `json:"databaseId"`
		}{
			DatabaseID: m.OrgID,
		},
	}, nil
}

func (m *MockAPIGetter) FetchOrgRulesets(owner string) ([]data.Rulesets, error) {
	if m.ShouldErrorRuleset {
		return nil, errors.New("mock error fetching org rulesets")
	}
	return m.OrgRulesets, nil
}

func (m *MockAPIGetter) GatherRepositories(owner string, repos []string) []data.RepoInfo {
	if m.ShouldError {
		return []data.RepoInfo{}
	}
	return m.Repos
}

func (m *MockAPIGetter) FetchRepoRulesets(owner string, repos []data.RepoInfo) ([]data.RepoNameRule, error) {
	if m.ShouldError {
		return nil, errors.New("mock error fetching repo rulesets")
	}
	return m.RepoRulesets, nil
}

func (m *MockAPIGetter) GetOrgLevelRuleset(owner string, rulesetId int) ([]byte, error) {
	if m.ShouldError {
		return nil, errors.New("mock error getting org level ruleset")
	}
	return []byte(`{
        "id": 123,
        "name": "test-ruleset",
        "target": "branch",
        "source_type": "Organization",
        "enforcement": "active",
        "bypass_actors": [],
        "conditions": null,
        "rules": [],
        "created_at": "2023-01-01T00:00:00Z",
        "updated_at": "2023-01-01T00:00:00Z"
    }`), nil
}

func (m *MockAPIGetter) GetRepoLevelRuleset(owner string, repo string, rulesetId int) ([]byte, error) {
	if m.ShouldError {
		return nil, errors.New("mock error getting repo level ruleset")
	}
	return []byte(`{
        "id": 456,
        "name": "repo-ruleset",
        "target": "branch",
        "source_type": "Repository",
        "enforcement": "active",
        "bypass_actors": [],
        "conditions": null,
        "rules": [],
        "created_at": "2023-01-01T00:00:00Z",
        "updated_at": "2023-01-01T00:00:00Z"
    }`), nil
}

func (m *MockAPIGetter) ProcessActorsForExport(actors []data.BypassActor, owner string, orgID int, ruleID string) []string {
	return []string{}
}

func (m *MockAPIGetter) ProcessRules(rules []data.Rules) map[string]string {
	return map[string]string{}
}

func TestMain(m *testing.M) {
	code := m.Run()

	cleanupTestCSVFiles()

	os.Exit(code)
}

func cleanupTestCSVFiles() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	pattern := filepath.Join(dir, "ruleset-*.csv")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	for _, file := range matches {
		if err := os.Remove(file); err != nil {
			continue
		}
	}
}

func TestNewCmdList(t *testing.T) {
	cmd := NewCmdList()

	if cmd.Use != "list [flags] <organization> [repo ...]" {
		t.Errorf("NewCmdList() Use = %v, want %v", cmd.Use, "list [flags] <organization> [repo ...]")
	}

	if !strings.Contains(cmd.Short, "Generate a report of rulesets") {
		t.Errorf("NewCmdList() Short description doesn't mention generating report")
	}
}

func TestRunCmdList(t *testing.T) {
	tests := []struct {
		name        string
		owner       string
		repos       []string
		mockGetter  *MockAPIGetter
		wantErr     bool
		errContains string
	}{
		{
			name:  "successful org rulesets list",
			owner: "test-org",
			repos: []string{},
			mockGetter: &MockAPIGetter{
				OrgID: 12345,
				OrgRulesets: []data.Rulesets{
					{
						ID:         "R_123",
						DatabaseID: 123,
						Name:       "test-ruleset",
					},
				},
				Repos:        []data.RepoInfo{},
				RepoRulesets: []data.RepoNameRule{},
				ShouldError:  false,
			},
			wantErr: false,
		},
		{
			name:  "error fetching org ID",
			owner: "test-org",
			repos: []string{},
			mockGetter: &MockAPIGetter{
				ShouldError: true,
			},
			wantErr:     true,
			errContains: "error fetching organization ID",
		},
		{
			name:  "successful repo rulesets list",
			owner: "test-org",
			repos: []string{"repo1"},
			mockGetter: &MockAPIGetter{
				OrgID:       12345,
				OrgRulesets: []data.Rulesets{},
				Repos: []data.RepoInfo{
					{
						DatabaseId: 456,
						Name:       "repo1",
					},
				},
				RepoRulesets: []data.RepoNameRule{
					{
						RepoName: "repo1",
						Rule: data.Rulesets{
							ID:         "R_456",
							DatabaseID: 456,
							Name:       "repo-ruleset",
						},
					},
				},
				ShouldError: false,
			},
			wantErr: false,
		},
		{
			name:  "no rulesets found",
			owner: "test-org",
			repos: []string{},
			mockGetter: &MockAPIGetter{
				OrgID:        12345,
				OrgRulesets:  []data.Rulesets{},
				Repos:        []data.RepoInfo{},
				RepoRulesets: []data.RepoNameRule{},
				ShouldError:  false,
			},
			wantErr:     true,
			errContains: "no rulesets found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "test-output.csv")

			cmdFlags := &cmdFlags{
				ruleType: "all",
			}

			err := runCmdList(tt.owner, tt.repos, cmdFlags, tt.mockGetter, tmpFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("runCmdList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("runCmdList() error = %v, should contain %v", err, tt.errContains)
			}

			if !tt.wantErr {
				if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
					t.Error("runCmdList() did not create output file when data was processed")
				} else {
					content, err := os.ReadFile(tmpFile)
					if err != nil {
						t.Errorf("Failed to read output file: %v", err)
					}
					if len(content) == 0 {
						t.Error("Output file is empty")
					}
					contentStr := string(content)
					if !strings.Contains(contentStr, "RulesetLevel") {
						t.Error("Output file missing expected CSV headers")
					}
				}
			}

			if tt.wantErr && strings.Contains(tt.errContains, "no rulesets") {
				if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
					t.Error("runCmdList() created output file when no rulesets were found")
				}
			}
		})
	}
}

func TestCmdFlags_Validation(t *testing.T) {
	tests := []struct {
		name     string
		ruleType string
		wantErr  bool
	}{
		{
			name:     "valid ruleType all",
			ruleType: "all",
			wantErr:  false,
		},
		{
			name:     "valid ruleType repoOnly",
			ruleType: "repoOnly",
			wantErr:  false,
		},
		{
			name:     "valid ruleType orgOnly",
			ruleType: "orgOnly",
			wantErr:  false,
		},
		{
			name:     "invalid ruleType",
			ruleType: "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validRuleTypes := map[string]struct{}{
				"all":      {},
				"repoOnly": {},
				"orgOnly":  {},
			}

			_, isValid := validRuleTypes[tt.ruleType]
			gotErr := !isValid

			if gotErr != tt.wantErr {
				t.Errorf("validation for ruleType %s: got error = %v, want error = %v", tt.ruleType, gotErr, tt.wantErr)
			}
		})
	}
}

func TestRunCmdList_WithAPIErrors(t *testing.T) {
	tests := []struct {
		name       string
		mockGetter *MockAPIGetter
		wantErr    bool
	}{
		{
			name: "handles nil team data gracefully",
			mockGetter: &MockAPIGetter{
				OrgID:       12345,
				OrgRulesets: []data.Rulesets{}, // Empty org rulesets
				Repos: []data.RepoInfo{
					{
						DatabaseId: 789,
						Name:       "test-repo",
					},
				},
				RepoRulesets: []data.RepoNameRule{
					{
						RepoName: "test-repo",
						Rule: data.Rulesets{
							ID:         "R_123",
							DatabaseID: 123,
							Name:       "test-ruleset",
						},
					},
				},
				ShouldError: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "test-output.csv")
			err := runCmdList("test-org", []string{}, &cmdFlags{ruleType: "all"}, tt.mockGetter, tmpFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("runCmdList() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
					t.Error("runCmdList() did not create output file")
				}
			}
		})
	}
}

func TestRunCmdList_OrgOnlyMode(t *testing.T) {
	tests := []struct {
		name        string
		mockGetter  *MockAPIGetter
		wantErr     bool
		errContains string
	}{
		{
			name: "orgOnly mode success",
			mockGetter: &MockAPIGetter{
				OrgID: 12345,
				OrgRulesets: []data.Rulesets{
					{
						ID:         "R_123",
						DatabaseID: 123,
						Name:       "test-ruleset",
					},
				},
				ShouldError: false,
			},
			wantErr: false,
		},
		{
			name: "orgOnly mode with ruleset fetch error",
			mockGetter: &MockAPIGetter{
				OrgID:              12345,
				ShouldErrorRuleset: true,
			},
			wantErr:     true,
			errContains: "orgOnly mode",
		},
		{
			name: "orgOnly mode with org ID fetch error",
			mockGetter: &MockAPIGetter{
				ShouldError: true,
			},
			wantErr:     true,
			errContains: "error fetching organization ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "test-output.csv")
			err := runCmdList("test-org", []string{}, &cmdFlags{ruleType: "orgOnly"}, tt.mockGetter, tmpFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("runCmdList() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("runCmdList() error = %v, should contain %v", err, tt.errContains)
			}

			if tt.wantErr {
				if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
					t.Error("runCmdList() created output file when there was an error")
				}
			}
		})
	}
}

func TestRunCmdList_FileNotCreatedOnError(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test-output.csv")

	mockGetter := &MockAPIGetter{
		ShouldError: true,
	}

	err := runCmdList("test-org", []string{}, &cmdFlags{ruleType: "all"}, mockGetter, tmpFile)

	if err == nil {
		t.Error("runCmdList() expected error but got none")
	}

	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("runCmdList() should not create file when error occurs before data processing")
	}
}
