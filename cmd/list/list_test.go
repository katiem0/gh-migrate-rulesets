package list

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
)

// MockAPIGetter implements the necessary methods for testing
type MockAPIGetter struct {
	OrgID        int
	OrgRulesets  []data.Rulesets
	Repos        []data.RepoInfo
	RepoRulesets []data.RepoNameRule
	ShouldError  bool
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
	if m.ShouldError {
		return nil, errors.New("mock error fetching org rulesets")
	}
	return m.OrgRulesets, nil
}

func (m *MockAPIGetter) GatherRepositories(owner string, repos []string) ([]data.RepoInfo, error) {
	if m.ShouldError {
		return nil, errors.New("mock error gathering repositories")
	}
	return m.Repos, nil
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

// TestMain runs before all tests and can be used for setup/teardown
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Cleanup any CSV files created during tests
	cleanupTestCSVFiles()

	os.Exit(code)
}

func cleanupTestCSVFiles() {
	// Get current directory
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	// Find all CSV files matching the test pattern
	pattern := filepath.Join(dir, "ruleset-*.csv")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	// Remove each matched file
	for _, file := range matches {
		if err := os.Remove(file); err != nil {
			// Silently ignore errors during cleanup
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
			errContains: "mock error fetching org ID",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary buffer for output
			var buf bytes.Buffer

			// Create temporary cmdFlags
			cmdFlags := &cmdFlags{
				ruleType: "all",
			}

			// Run the command
			err := runCmdList(tt.owner, tt.repos, cmdFlags, tt.mockGetter, &buf)

			// Check error expectations
			if (err != nil) != tt.wantErr {
				t.Errorf("runCmdList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("runCmdList() error = %v, should contain %v", err, tt.errContains)
			}

			// If no error expected, check that output was written
			if !tt.wantErr && buf.Len() == 0 {
				t.Error("runCmdList() produced no output")
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
