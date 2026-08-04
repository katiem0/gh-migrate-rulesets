package create

import (
	"encoding/csv"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"github.com/katiem0/gh-migrate-rulesets/internal/utils"
)

type MockAPIGetter struct {
	ShouldError            bool
	RepoExistsResult       bool
	OrgRulesets            []data.Rulesets
	RepoRulesets           []data.RepoNameRule
	Repos                  []data.RepoInfo
	OrgID                  int
	CreatedRepoSources     []string
	FetchOrgRulesetsError  bool
	FetchRepoRulesetsError bool
	RepoRulesetsFromFile   []data.RepoRuleset
	TranslateBypassErr     error
	CreatedOrgCount        int
}

func (m *MockAPIGetter) CreateOrgLevelRuleset(owner string, data io.Reader) error {
	m.CreatedOrgCount++
	if m.ShouldError {
		return errors.New("mock error creating org ruleset")
	}
	return nil
}

func (m *MockAPIGetter) CreateRepoLevelRuleset(ownerRepo string, data io.Reader) error {
	m.CreatedRepoSources = append(m.CreatedRepoSources, ownerRepo)
	if m.ShouldError {
		return errors.New("mock error creating repo ruleset")
	}
	return nil
}

func (m *MockAPIGetter) RepoExists(ownerRepo string) bool {
	return m.RepoExistsResult
}

func (m *MockAPIGetter) CreateRepoRulesetsData(owner string, fileData [][]string, actorMapping map[string]int) []data.RepoRuleset {
	if m.RepoRulesetsFromFile != nil {
		return m.RepoRulesetsFromFile
	}
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
	if m.ShouldError || m.FetchOrgRulesetsError {
		return nil, errors.New("mock error fetching org rulesets")
	}
	return m.OrgRulesets, nil
}

func (m *MockAPIGetter) FetchRepoRulesets(owner string, repos []data.RepoInfo) ([]data.RepoNameRule, error) {
	if m.ShouldError || m.FetchRepoRulesetsError {
		return nil, errors.New("mock error fetching repo rulesets")
	}
	return m.RepoRulesets, nil
}

func (m *MockAPIGetter) GatherRepositories(owner string, repos []string) []data.RepoInfo {
	if m.ShouldError {
		return []data.RepoInfo{}
	}
	return m.Repos
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

func (m *MockAPIGetter) ParseBypassActorsForImport(owner string, bypassActorsStr string, actorMapping map[string]int) []data.BypassActor {
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

func (m *MockAPIGetter) UpdateBypassActorID(owner string, sourceOrg string, sourceOrgID int, ruleset data.RepoRuleset, s utils.Getter, actorMapping map[string]int) (data.RepoRuleset, error) {
	return ruleset, m.TranslateBypassErr
}

func (m *MockAPIGetter) UpdateRequiredWorkflowRepoID(owner string, ruleset data.RepoRuleset, s utils.Getter) (data.RepoRuleset, error) {
	return ruleset, nil
}

func (m *MockAPIGetter) UpdateStatusCheckIntegrationID(sourceOrg string, ruleset data.RepoRuleset, s utils.Getter) (data.RepoRuleset, error) {
	return ruleset, nil
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

	errorPattern := filepath.Join(dir, "*-ruleset-errors-*.csv")
	errorMatches, err := filepath.Glob(errorPattern)
	if err == nil {
		for _, file := range errorMatches {
			if err := os.Remove(file); err != nil {
				continue
			}
		}
	}

	projectRoot := filepath.Join(dir, "..", "..")
	rootErrorPattern := filepath.Join(projectRoot, "*-ruleset-errors-*.csv")
	rootErrorMatches, err := filepath.Glob(rootErrorPattern)
	if err == nil {
		for _, file := range rootErrorMatches {
			if err := os.Remove(file); err != nil {
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

	for _, flagName := range []string{"actor-mapping", "dry-run"} {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("NewCmdCreate() missing flag %q", flagName)
		}
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
		{
			name:       "actor-mapping with from-file",
			args:       []string{"--from-file", "test.csv", "--actor-mapping", "mapping.csv"},
			wantErr:    true,
			errMessage: "`--actor-mapping` cannot be used with `--from-file`",
		},
		{
			name:       "missing actor mapping file",
			args:       []string{"--source-org", "testorg", "--actor-mapping", "does-not-exist.csv"},
			wantErr:    true,
			errMessage: "actor mapping file not found",
		},
		{
			name:       "target-repo with multiple repos",
			args:       []string{"--source-org", "testorg", "--repos", "repo1,repo2", "--target-repo", "renamed"},
			wantErr:    true,
			errMessage: "requires exactly one repository",
		},
		{
			name:       "target-repo with source-org but no repos",
			args:       []string{"--source-org", "testorg", "--target-repo", "renamed"},
			wantErr:    true,
			errMessage: "requires exactly one repository",
		},
		{
			name:       "target-repo with from-file",
			args:       []string{"--from-file", "test.csv", "--target-repo", "renamed"},
			wantErr:    true,
			errMessage: "cannot be used with `--from-file`",
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
	csvContent := `RulesetLevel,SourceRepositoryName,TargetRepositoryName,RuleID,RulesetName,Target,Enforcement,BypassActors,ConditionsRefNameInclude,ConditionsRefNameExclude,ConditionsRepoNameInclude,ConditionsRepoNameExclude,ConditionsRepoNameProtected,ConditionRepoPropertyInclude,ConditionRepoPropertyExclude,RulesCreation,RulesUpdate,RulesDeletion,RulesRequiredLinearHistory,RulesMergeQueue,RulesRequiredDeployments,RulesRequiredSignatures,RulesPullRequest,RulesRequiredStatusChecks,RulesNonFastForward,RulesCommitMessagePattern,RulesCommitAuthorEmailPattern,RulesCommitterEmailPattern,RulesBranchNamePattern,RulesTagNamePattern,RulesFilePathRestriction,RulesFilePathLength,RulesFileExtensionRestriction,RulesMaxFileSize,RulesWorkflows,RulesCodeScanning,CreatedAt,UpdatedAt
Organization,N/A,N/A,1,org-level-ruleset,branch,active,,,,,,,,,true,,,,,,,,,,,,,,,,,,,,,2023-01-01T00:00:00Z,2023-01-01T00:00:00Z
Repository,test-repo,test-repo,2,repo-level-ruleset,branch,active,,,,,,,,,,true,,,,,,,,,,,,,,,,,,,,2023-01-01T00:00:00Z,2023-01-01T00:00:00Z`

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
				ShouldError:      true,
			},
			wantErr: false,
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

func TestRunCmdCreate_FromFile_TargetRepositoryRename(t *testing.T) {
	csvContent := `RulesetLevel,SourceRepositoryName,TargetRepositoryName,RuleID,RulesetName,Target,Enforcement,RulesPullRequest,CreatedAt,UpdatedAt
Repository,old-repo,new-repo,2,repo-level-ruleset,branch,active,,2023-01-01T00:00:00Z,2023-01-01T00:00:00Z`

	tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpFile.Write([]byte(csvContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	mockGetter := &MockAPIGetter{
		RepoExistsResult: true,
		RepoRulesetsFromFile: []data.RepoRuleset{
			{
				ID:           2,
				Name:         "repo-level-ruleset",
				Target:       "branch",
				SourceType:   "Repository",
				Source:       "testorg/old-repo",
				TargetSource: "testorg/new-repo",
				Enforcement:  "active",
				BypassActors: []data.BypassActor{},
				Conditions:   &data.Conditions{},
				Rules:        []data.Rules{{Type: "deletion"}},
			},
		},
	}

	if err := runCmdCreate("testorg", &cmdFlags{fileName: tmpFile.Name()}, mockGetter, mockGetter); err != nil {
		t.Fatalf("runCmdCreate() error = %v", err)
	}

	found := false
	for _, src := range mockGetter.CreatedRepoSources {
		if src == "testorg/new-repo" {
			found = true
		}
		if src == "testorg/old-repo" {
			t.Errorf("ruleset was created under source repo testorg/old-repo, expected target testorg/new-repo")
		}
	}
	if !found {
		t.Errorf("ruleset was not created under target repo testorg/new-repo, got %v", mockGetter.CreatedRepoSources)
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

func TestRunCmdCreate_FromSourceOrg_TargetRepoRename(t *testing.T) {
	mockGetter := &MockAPIGetter{
		RepoExistsResult: true,
	}
	mockSource := &MockAPIGetter{
		OrgID: 123,
		Repos: []data.RepoInfo{{DatabaseId: 10, Name: "repo1"}},
		RepoRulesets: []data.RepoNameRule{
			{RepoName: "repo1", Rule: data.Rulesets{ID: "R_2", DatabaseID: 2, Name: "repo-ruleset"}},
		},
	}

	flags := &cmdFlags{
		sourceOrg:  "source-org",
		ruleType:   "repoOnly",
		repos:      []string{"repo1"},
		targetRepo: "renamed-repo",
	}

	if err := runCmdCreate("target-org", flags, mockGetter, mockSource); err != nil {
		t.Fatalf("runCmdCreate() unexpected error = %v", err)
	}

	want := "target-org/renamed-repo"
	found := false
	for _, source := range mockGetter.CreatedRepoSources {
		if source == want {
			found = true
		}
		if source == "target-org/repo1" {
			t.Errorf("repo-level ruleset created against original repo %q, expected rename to %q", source, want)
		}
	}
	if !found {
		t.Errorf("expected repo-level ruleset created against %q, got %v", want, mockGetter.CreatedRepoSources)
	}
}

// readErrorCSV finds and reads the most recent error CSV written for owner in cwd.
func readErrorCSV(t *testing.T, owner string) string {
	t.Helper()
	matches, err := filepath.Glob(owner + "-ruleset-errors-*.csv")
	if err != nil {
		t.Fatalf("glob error: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected an error CSV file for owner %q, none found", owner)
	}
	t.Cleanup(func() {
		for _, m := range matches {
			_ = os.Remove(m)
		}
	})
	newest := matches[0]
	var newestMod time.Time
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil {
			t.Fatalf("failed to stat error CSV %q: %v", m, err)
		}
		if info.ModTime().After(newestMod) {
			newest = m
			newestMod = info.ModTime()
		}
	}
	content, err := os.ReadFile(newest)
	if err != nil {
		t.Fatalf("failed to read error CSV: %v", err)
	}
	return string(content)
}

func TestRunCmdCreate_FetchOrgRulesetsError_WrittenToCSV(t *testing.T) {
	mockGetter := &MockAPIGetter{}
	mockSource := &MockAPIGetter{OrgID: 123, FetchOrgRulesetsError: true}

	flags := &cmdFlags{
		sourceOrg: "fetch-org-fail",
		ruleType:  "orgOnly",
	}

	if err := runCmdCreate("fetch-org-fail", flags, mockGetter, mockSource); err != nil {
		t.Fatalf("runCmdCreate() unexpected error = %v", err)
	}

	content := readErrorCSV(t, "fetch-org-fail")
	if !strings.Contains(content, "failed to fetch organization rulesets") {
		t.Errorf("error CSV missing org rulesets fetch failure, got:\n%s", content)
	}
}

func TestRunCmdCreate_FetchRepoRulesetsError_WrittenToCSV(t *testing.T) {
	mockGetter := &MockAPIGetter{}
	mockSource := &MockAPIGetter{
		OrgID:                  123,
		Repos:                  []data.RepoInfo{{DatabaseId: 10, Name: "repo1"}},
		FetchRepoRulesetsError: true,
	}

	flags := &cmdFlags{
		sourceOrg: "fetch-repo-fail",
		ruleType:  "repoOnly",
	}

	if err := runCmdCreate("fetch-repo-fail", flags, mockGetter, mockSource); err != nil {
		t.Fatalf("runCmdCreate() unexpected error = %v", err)
	}

	content := readErrorCSV(t, "fetch-repo-fail")
	if !strings.Contains(content, "failed to fetch repository rulesets") {
		t.Errorf("error CSV missing repo rulesets fetch failure, got:\n%s", content)
	}
}

func TestRunCmdCreate_TargetRepoRename_ErrorRecordsDestination(t *testing.T) {
	// Target repo does not exist, so creation fails and should be recorded with
	// the destination repo (org/repo format) where creation was attempted.
	mockGetter := &MockAPIGetter{
		RepoExistsResult: false,
	}
	mockSource := &MockAPIGetter{
		OrgID: 123,
		Repos: []data.RepoInfo{{DatabaseId: 10, Name: "repo1"}},
		RepoRulesets: []data.RepoNameRule{
			{RepoName: "repo1", Rule: data.Rulesets{ID: "R_2", DatabaseID: 2, Name: "repo-ruleset"}},
		},
	}

	flags := &cmdFlags{
		sourceOrg:  "rename-fail",
		ruleType:   "repoOnly",
		repos:      []string{"repo1"},
		targetRepo: "new-repo",
	}

	if err := runCmdCreate("rename-fail", flags, mockGetter, mockSource); err != nil {
		t.Fatalf("runCmdCreate() unexpected error = %v", err)
	}

	content := readErrorCSV(t, "rename-fail")
	if !strings.Contains(content, "Source,RulesetName,Error") {
		t.Errorf("error CSV missing expected headers, got:\n%s", content)
	}
	if !strings.Contains(content, "rename-fail/new-repo,") {
		t.Errorf("error CSV missing destination repo in Source column, got:\n%s", content)
	}
	if !strings.Contains(content, "Repository does not exist") {
		t.Errorf("error CSV missing failure reason, got:\n%s", content)
	}
}

// TestRunCmdCreate_TranslationFailure_SkipsAndContinues also guards that the loop
// continues to the next ruleset after a skip.
func TestRunCmdCreate_TranslationFailure_SkipsAndContinues(t *testing.T) {
	mockGetter := &MockAPIGetter{
		RepoExistsResult: true,
		// Two joined lookup errors for a single ruleset; recorded as one row.
		TranslateBypassErr: errors.Join(
			errors.New("bypass actor lookup failed"),
			errors.New("required workflow lookup failed"),
		),
	}
	mockSource := &MockAPIGetter{
		OrgID: 123,
		OrgRulesets: []data.Rulesets{
			{ID: "R_1", DatabaseID: 1, Name: "org-ruleset-one"},
			{ID: "R_2", DatabaseID: 2, Name: "org-ruleset-two"},
		},
	}

	flags := &cmdFlags{sourceOrg: "translate-fail", ruleType: "orgOnly"}
	if err := runCmdCreate("translate-fail", flags, mockGetter, mockSource); err != nil {
		t.Fatalf("runCmdCreate() unexpected error = %v", err)
	}

	// Skipped, not created: creation must never be attempted for a failed translation.
	if mockGetter.CreatedOrgCount != 0 {
		t.Errorf("expected no org rulesets created, got %d", mockGetter.CreatedOrgCount)
	}

	records, err := csv.NewReader(strings.NewReader(readErrorCSV(t, "translate-fail"))).ReadAll()
	if err != nil {
		t.Fatalf("failed to parse error CSV: %v", err)
	}

	// Header + exactly one row per failed ruleset (two rulesets, not one row per lookup).
	dataRows := records[1:]
	if len(dataRows) != 2 {
		t.Fatalf("expected 2 error rows (one per ruleset), got %d: %v", len(dataRows), dataRows)
	}

	names := map[string]bool{}
	for _, row := range dataRows {
		names[row[1]] = true
		// Both joined lookup errors must survive in the single row (errors.Join semantics).
		if !strings.Contains(row[2], "bypass actor lookup failed") || !strings.Contains(row[2], "required workflow lookup failed") {
			t.Errorf("error row missing a joined lookup message: %q", row[2])
		}
	}
	// Loop continued: both rulesets were processed and recorded.
	if !names["org-ruleset-one"] || !names["org-ruleset-two"] {
		t.Errorf("expected error rows for both rulesets, got %v", names)
	}
}

func TestExtractErrorMessage(t *testing.T) {
	// Mirrors go-gh's HTTPError.Error(): the status line, then one line per
	// validation error (it joins httpError.Message with newlines).
	multi := &api.HTTPError{
		StatusCode: 422,
		RequestURL: mustParseURL(t, "https://api.github.com/orgs/acme/rulesets"),
		Message:    "Validation Failed\nInvalid bypass actor: '{{actor_id: 4}}'\nrepository_id is invalid",
	}
	single := &api.HTTPError{
		StatusCode: 404,
		RequestURL: mustParseURL(t, "https://api.github.com/orgs/acme/rulesets"),
		Message:    "Not Found",
	}

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "keeps every validation reason, not just the first",
			err:  multi,
			want: "Invalid bypass actor: '{{actor_id: 4}}'; repository_id is invalid",
		},
		{
			// Nothing to trim, so the status is preserved rather than lost.
			name: "single-line error kept verbatim",
			err:  single,
			want: "HTTP 404: Not Found (https://api.github.com/orgs/acme/rulesets)",
		},
		{
			name: "plain error kept verbatim",
			err:  errors.New("connection refused"),
			want: "connection refused",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractErrorMessage(tt.err); got != tt.want {
				t.Errorf("extractErrorMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parsing %q: %v", raw, err)
	}
	return u
}

func TestValidateOrgArg(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "single org passes", args: []string{"myorg"}},
		{name: "no args", args: []string{}, wantErr: "organization argument is required"},
		{
			// An empty variable shifts a flag's value into an extra positional.
			name:    "swallowed flag leaves extra positionals",
			args:    []string{"myorg", "nodejs-api-demo", "expert-services.ghe.com"},
			wantErr: "expected a single organization argument but received 3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOrgArg(nil, tt.args)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got none", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}
