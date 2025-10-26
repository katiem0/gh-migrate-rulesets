package create

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"github.com/katiem0/gh-migrate-rulesets/internal/utils"
)

type MockAPIGetter struct {
	ShouldError      bool
	RepoExistsResult bool
	OrgRulesets      []data.Rulesets
	RepoRulesets     []data.RepoNameRule
	Repos            []data.RepoInfo
	OrgID            int
}

// Implement the utils.Getter interface for MockAPIGetter

func (m *MockAPIGetter) CreateOrgLevelRuleset(owner string, data io.Reader) error {
	if m.ShouldError {
		return errors.New("mock error creating org ruleset")
	}
	return nil
}

func (m *MockAPIGetter) CreateRepoLevelRuleset(ownerRepo string, data io.Reader) error {
	if m.ShouldError {
		return errors.New("mock error creating repo ruleset")
	}
	return nil
}

func (m *MockAPIGetter) RepoExists(ownerRepo string) bool {
	return m.RepoExistsResult
}

func (m *MockAPIGetter) CreateRepoRulesetsData(owner string, fileData [][]string) []data.RepoRuleset {
	// Return a more realistic dataset for testing
	return []data.RepoRuleset{
		{
			ID:           1,
			Name:         "org-level-ruleset",
			Target:       "branch",
			SourceType:   "Organization",
			Source:       owner,
			Enforcement:  "active",
			BypassActors: []data.BypassActor{},
			Conditions:   &data.Conditions{},
			Rules:        []data.Rules{{Type: "creation"}},
		},
		{
			ID:           2,
			Name:         "repo-level-ruleset",
			Target:       "branch",
			SourceType:   "Repository",
			Source:       owner + "/test-repo",
			Enforcement:  "active",
			BypassActors: []data.BypassActor{},
			Conditions:   &data.Conditions{},
			Rules:        []data.Rules{{Type: "deletion"}},
		},
	}
}

func (m *MockAPIGetter) FetchOrgId(owner string) (*data.OrgIdQuery, error) {
	if m.ShouldError {
		return nil, errors.New("mock error fetching org id")
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

func (m *MockAPIGetter) FetchRepoRulesets(owner string, repos []data.RepoInfo) ([]data.RepoNameRule, error) {
	if m.ShouldError {
		return nil, errors.New("mock error fetching repo rulesets")
	}
	return m.RepoRulesets, nil
}

func (m *MockAPIGetter) GatherRepositories(owner string, repos []string) ([]data.RepoInfo, error) {
	if m.ShouldError {
		return nil, errors.New("mock error gathering repositories")
	}
	return m.Repos, nil
}

func (m *MockAPIGetter) GetAnApp(appSlug string) (*data.AppInfo, error) {
	return &data.AppInfo{
		AppID:   123,
		AppSlug: appSlug,
	}, nil
}

func (m *MockAPIGetter) GetAppInstallations(owner string) (*data.AppIntegrations, error) {
	return &data.AppIntegrations{}, nil
}

func (m *MockAPIGetter) GetCustomRoles(owner string, roleID int) (*data.CustomRole, error) {
	return &data.CustomRole{}, nil
}

func (m *MockAPIGetter) GetOrgLevelRuleset(owner string, rulesetId int) ([]byte, error) {
	if m.ShouldError {
		return nil, errors.New("mock error getting org ruleset")
	}
	return []byte(`{"id": 1, "name": "org-ruleset", "source_type": "Organization", "target": "branch", "enforcement": "active", "bypass_actors": [], "conditions": null, "rules": [{"type": "creation"}], "created_at": "2023-01-01T00:00:00Z", "updated_at": "2023-01-01T00:00:00Z"}`), nil
}

func (m *MockAPIGetter) GetOrgRulesetsList(owner string, endCursor *string) (*data.OrgRulesetsQuery, error) {
	return nil, nil
}

func (m *MockAPIGetter) GetRepo(owner string, name string) (*data.RepoSingleQuery, error) {
	return &data.RepoSingleQuery{
		Repository: data.RepoInfo{
			DatabaseId: 456,
			Name:       name,
		},
	}, nil
}

func (m *MockAPIGetter) GetRepoByID(repoID int) (*data.RepoInfo, error) {
	return &data.RepoInfo{
		DatabaseId: repoID,
		Name:       "test-repo",
	}, nil
}

func (m *MockAPIGetter) GetRepoCustomRoles(owner string) (*data.CustomRepoRoles, error) {
	return &data.CustomRepoRoles{}, nil
}

func (m *MockAPIGetter) GetRepoLevelRuleset(owner string, repo string, rulesetId int) ([]byte, error) {
	if m.ShouldError {
		return nil, errors.New("mock error getting repo ruleset")
	}
	return []byte(`{"id": 2, "name": "repo-ruleset", "source_type": "Repository", "source": "` + owner + `/` + repo + `", "target": "branch", "enforcement": "active", "bypass_actors": [], "conditions": null, "rules": [{"type": "deletion"}], "created_at": "2023-01-01T00:00:00Z", "updated_at": "2023-01-01T00:00:00Z"}`), nil
}

func (m *MockAPIGetter) GetRepoRulesetsList(owner string, repo string, endCursor *string) (*data.RepoRulesetsQuery, error) {
	return nil, nil
}

func (m *MockAPIGetter) GetReposList(owner string, endCursor *string) (*data.ReposQuery, error) {
	return nil, nil
}

func (m *MockAPIGetter) GetTeamData(ownerID int, teamID int) (*data.TeamInfo, error) {
	return &data.TeamInfo{}, nil
}

func (m *MockAPIGetter) GetTeamByName(owner string, teamSlug string) (*data.TeamInfo, error) {
	return &data.TeamInfo{}, nil
}

func (m *MockAPIGetter) MapToParameters(owner string, paramsMap map[string]interface{}, ruleType string) *data.Parameters {
	return &data.Parameters{}
}

func (m *MockAPIGetter) ParametersToMap(params data.Parameters, ruleType string) map[string]string {
	return map[string]string{}
}

func (m *MockAPIGetter) ParseBypassActorsForImport(owner string, bypassActorsStr string) []data.BypassActor {
	return []data.BypassActor{}
}

func (m *MockAPIGetter) ParseRequiredWorkflowsForImport(owner string, value interface{}) []data.Workflows {
	return []data.Workflows{}
}

func (m *MockAPIGetter) ProcessActorsForExport(actors []data.BypassActor, owner string, orgID int, ruleID string) []string {
	return []string{}
}

func (m *MockAPIGetter) ProcessRules(rules []data.Rules) map[string]string {
	return map[string]string{}
}

func (m *MockAPIGetter) UpdateBypassActorID(owner string, sourceOrg string, sourceOrgID int, ruleset data.RepoRuleset, s utils.Getter) data.RepoRuleset {
	return ruleset // Passthrough for testing
}

func (m *MockAPIGetter) UpdateRequiredWorkflowRepoID(owner string, ruleset data.RepoRuleset, s utils.Getter) data.RepoRuleset {
	return ruleset // Passthrough for testing
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

	// Find all error CSV files matching the test pattern (*-ruleset-errors-*.csv)
	errorPattern := filepath.Join(dir, "*-ruleset-errors-*.csv")
	errorMatches, err := filepath.Glob(errorPattern)
	if err == nil {
		for _, file := range errorMatches {
			if err := os.Remove(file); err != nil {
				// Silently ignore errors during cleanup
				continue
			}
		}
	}

	// Also clean up in project root (go up two directories from cmd/create)
	projectRoot := filepath.Join(dir, "..", "..")
	rootErrorPattern := filepath.Join(projectRoot, "*-ruleset-errors-*.csv")
	rootErrorMatches, err := filepath.Glob(rootErrorPattern)
	if err == nil {
		for _, file := range rootErrorMatches {
			if err := os.Remove(file); err != nil {
				// Silently ignore errors during cleanup
				continue
			}
		}
	}
}

func TestNewCmdCreate(t *testing.T) {
	cmd := NewCmdCreate()

	if cmd.Use != "create [flags] <organization>" {
		t.Errorf("NewCmdCreate() Use = %v, want %v", cmd.Use, "create [flags] <organization>")
	}

	if !strings.Contains(cmd.Short, "Create repository rulesets") {
		t.Errorf("NewCmdCreate() Short description incorrect")
	}
}

func TestCmdCreate_PreRunE(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		errMessage string
	}{
		{
			name:       "no file or source org",
			args:       []string{},
			wantErr:    true,
			errMessage: "a file or source organization must be specified",
		},
		{
			name:       "both file and source org",
			args:       []string{"--from-file", "test.csv", "--source-org", "testorg"},
			wantErr:    true,
			errMessage: "specify only one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCmdCreate()
			cmd.SetArgs(append([]string{"org"}, tt.args...))

			err := cmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				} else if !strings.Contains(err.Error(), tt.errMessage) {
					t.Errorf("Expected error message containing %q, got %q", tt.errMessage, err.Error())
				}
			}
		})
	}
}

func TestRunCmdCreate_FromFile(t *testing.T) {
	// Create a properly formatted CSV with all 37 columns
	csvContent := `RulesetLevel,RepositoryName,RuleID,RulesetName,Target,Enforcement,BypassActors,ConditionsRefNameInclude,ConditionsRefNameExclude,ConditionsRepoNameInclude,ConditionsRepoNameExclude,ConditionsRepoNameProtected,ConditionRepoPropertyInclude,ConditionRepoPropertyExclude,RulesCreation,RulesUpdate,RulesDeletion,RulesRequiredLinearHistory,RulesMergeQueue,RulesRequiredDeployments,RulesRequiredSignatures,RulesPullRequest,RulesRequiredStatusChecks,RulesNonFastForward,RulesCommitMessagePattern,RulesCommitAuthorEmailPattern,RulesCommitterEmailPattern,RulesBranchNamePattern,RulesTagNamePattern,RulesFilePathRestriction,RulesFilePathLength,RulesFileExtensionRestriction,RulesMaxFileSize,RulesWorkflows,RulesCodeScanning,CreatedAt,UpdatedAt
Organization,N/A,1,org-level-ruleset,branch,active,,,,,,,,,true,,,,,,,,,,,,,,,,,,,,,2023-01-01T00:00:00Z,2023-01-01T00:00:00Z
Repository,test-repo,2,repo-level-ruleset,branch,active,,,,,,,,,,true,,,,,,,,,,,,,,,,,,,,2023-01-01T00:00:00Z,2023-01-01T00:00:00Z`

	tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Warning: failed to remove temp file: %v", err)
		}
	}()
	if _, err := tmpFile.Write([]byte(csvContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		owner      string
		cmdFlags   *cmdFlags
		mockGetter *MockAPIGetter
		wantErr    bool
	}{
		{
			name:  "successful creation from file",
			owner: "testorg",
			cmdFlags: &cmdFlags{
				fileName: tmpFile.Name(),
			},
			mockGetter: &MockAPIGetter{
				RepoExistsResult: true,
			},
			wantErr: false,
		},
		{
			name:  "file does not exist",
			owner: "testorg",
			cmdFlags: &cmdFlags{
				fileName: "nonexistent.csv",
			},
			mockGetter: &MockAPIGetter{},
			wantErr:    true,
		},
		{
			name:  "API error on create",
			owner: "testorg",
			cmdFlags: &cmdFlags{
				fileName: tmpFile.Name(),
			},
			mockGetter: &MockAPIGetter{
				RepoExistsResult: true,
				ShouldError:      true, // Simulate API failure
			},
			wantErr: false, // The function handles the error and logs it, doesn't return it
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runCmdCreate(tt.owner, tt.cmdFlags, tt.mockGetter, tt.mockGetter)

			if (err != nil) != tt.wantErr {
				t.Errorf("runCmdCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunCmdCreate_FromSourceOrg(t *testing.T) {
	tests := []struct {
		name       string
		owner      string
		cmdFlags   *cmdFlags
		mockGetter *MockAPIGetter
		mockSource *MockAPIGetter
		wantErr    bool
	}{
		{
			name:  "successful creation from source org",
			owner: "target-org",
			cmdFlags: &cmdFlags{
				sourceOrg: "source-org",
				ruleType:  "all",
			},
			mockGetter: &MockAPIGetter{
				RepoExistsResult: true,
			},
			mockSource: &MockAPIGetter{
				OrgID: 123,
				OrgRulesets: []data.Rulesets{
					{ID: "R_1", DatabaseID: 1, Name: "org-ruleset"},
				},
				Repos: []data.RepoInfo{
					{DatabaseId: 10, Name: "repo1"},
				},
				RepoRulesets: []data.RepoNameRule{
					{RepoName: "repo1", Rule: data.Rulesets{ID: "R_2", DatabaseID: 2, Name: "repo-ruleset"}},
				},
			},
			wantErr: false,
		},
		{
			name:  "source org fetch fails",
			owner: "target-org",
			cmdFlags: &cmdFlags{
				sourceOrg: "source-org",
			},
			mockGetter: &MockAPIGetter{},
			mockSource: &MockAPIGetter{
				ShouldError: true, // Source API fails
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runCmdCreate(tt.owner, tt.cmdFlags, tt.mockGetter, tt.mockSource)

			if (err != nil) != tt.wantErr {
				t.Errorf("runCmdCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
