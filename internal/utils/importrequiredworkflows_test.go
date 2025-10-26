package utils

import (
	"errors"
	"io"
	"testing"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
)

type MockWorkflowGetter struct {
	ShouldError      bool
	RepoExistsResult bool
	RepoInfo         *data.RepoSingleQuery
	RepoByIDInfo     *data.RepoInfo
}

func (m *MockWorkflowGetter) GetRepo(owner string, name string) (*data.RepoSingleQuery, error) {
	if m.ShouldError {
		return nil, errors.New("mock error getting repo")
	}
	if m.RepoInfo != nil {
		return m.RepoInfo, nil
	}
	return &data.RepoSingleQuery{
		Repository: data.RepoInfo{
			DatabaseId: 456,
			Name:       name,
		},
	}, nil
}

func (m *MockWorkflowGetter) GetRepoByID(repoID int) (*data.RepoInfo, error) {
	if m.ShouldError {
		return nil, errors.New("mock error getting repo by ID")
	}
	if m.RepoByIDInfo != nil {
		return m.RepoByIDInfo, nil
	}
	return &data.RepoInfo{
		DatabaseId: repoID,
		Name:       "source-repo",
	}, nil
}

func (m *MockWorkflowGetter) RepoExists(ownerRepo string) bool {
	return m.RepoExistsResult
}

func (m *MockWorkflowGetter) CreateOrgLevelRuleset(owner string, data io.Reader) error {
	return nil
}

func (m *MockWorkflowGetter) CreateRepoLevelRuleset(ownerRepo string, data io.Reader) error {
	return nil
}

func (m *MockWorkflowGetter) CreateRepoRulesetsData(owner string, fileData [][]string) []data.RepoRuleset {
	return nil
}

func (m *MockWorkflowGetter) FetchOrgId(owner string) (*data.OrgIdQuery, error) {
	return nil, nil
}

func (m *MockWorkflowGetter) FetchOrgRulesets(owner string) ([]data.Rulesets, error) {
	return nil, nil
}

func (m *MockWorkflowGetter) FetchRepoRulesets(owner string, repos []data.RepoInfo) ([]data.RepoNameRule, error) {
	return nil, nil
}

func (m *MockWorkflowGetter) GatherRepositories(owner string, repos []string) []data.RepoInfo {
	return nil
}

func (m *MockWorkflowGetter) GetAnApp(appSlug string) (*data.AppInfo, error) {
	return &data.AppInfo{
		AppID:   123,
		AppSlug: appSlug,
	}, nil
}

func (m *MockWorkflowGetter) GetOrgLevelRuleset(owner string, rulesetId int) ([]byte, error) {
	return nil, nil
}

func (m *MockWorkflowGetter) GetRepoLevelRuleset(owner string, repo string, rulesetId int) ([]byte, error) {
	return nil, nil
}

func (m *MockWorkflowGetter) GetRepoCustomRoles(owner string) (*data.CustomRepoRoles, error) {
	return &data.CustomRepoRoles{}, nil
}

func (m *MockWorkflowGetter) UpdateBypassActorID(owner string, sourceOrg string, sourceOrgID int, ruleset data.RepoRuleset, s Getter) data.RepoRuleset {
	return ruleset
}

func (m *MockWorkflowGetter) UpdateRequiredWorkflowRepoID(owner string, ruleset data.RepoRuleset, s Getter) data.RepoRuleset {
	for i, rule := range ruleset.Rules {
		if rule.Type == "workflows" && rule.Parameters != nil {
			for j, workflow := range rule.Parameters.Workflows {
				sourceWorkflowRepoQuery, err := s.GetRepoByID(workflow.RepositoryID)
				if err != nil {
					continue
				}
				workflowRepo, err := m.GetRepo(owner, sourceWorkflowRepoQuery.Name)
				if err != nil {
					continue
				}
				ruleset.Rules[i].Parameters.Workflows[j].RepositoryID = workflowRepo.Repository.DatabaseId
			}
		}
	}
	return ruleset
}

func (m *MockWorkflowGetter) GetAppInstallations(owner string) (*data.AppIntegrations, error) {
	return nil, nil
}

func (m *MockWorkflowGetter) GetCustomRoles(owner string, roleID int) (*data.CustomRole, error) {
	return nil, nil
}

func (m *MockWorkflowGetter) GetOrgRulesetsList(owner string, endCursor *string) (*data.OrgRulesetsQuery, error) {
	return nil, nil
}

func (m *MockWorkflowGetter) GetRepoRulesetsList(owner string, repo string, endCursor *string) (*data.RepoRulesetsQuery, error) {
	return nil, nil
}

func (m *MockWorkflowGetter) GetReposList(owner string, endCursor *string) (*data.ReposQuery, error) {
	return nil, nil
}

func (m *MockWorkflowGetter) GetTeamData(ownerID int, teamID int) (*data.TeamInfo, error) {
	return nil, nil
}

func (m *MockWorkflowGetter) GetTeamByName(owner string, teamSlug string) (*data.TeamInfo, error) {
	return nil, nil
}

func (m *MockWorkflowGetter) ParseBypassActorsForImport(owner string, bypassActorsStr string) []data.BypassActor {
	return nil
}

func (m *MockWorkflowGetter) ParseRequiredWorkflowsForImport(owner string, value interface{}) []data.Workflows {
	if workflowMaps, ok := value.([]map[string]string); ok {
		var workflows []data.Workflows
		for _, wfMap := range workflowMaps {
			workflows = append(workflows, data.Workflows{
				Path:         wfMap["Path"],
				Ref:          wfMap["Ref"],
				RepositoryID: 456,
				SHA:          wfMap["SHA"],
			})
		}
		return workflows
	}
	return nil
}

func (m *MockWorkflowGetter) ParametersToMap(params data.Parameters, ruleType string) map[string]string {
	return nil
}

func (m *MockWorkflowGetter) MapToParameters(owner string, paramsMap map[string]interface{}, ruleType string) *data.Parameters {
	return nil
}

func (m *MockWorkflowGetter) ProcessActorsForExport(actors []data.BypassActor, owner string, orgID int, ruleID string) []string {
	return nil
}

func (m *MockWorkflowGetter) ProcessRules(rules []data.Rules) map[string]string {
	return nil
}

func TestParseRequiredWorkflowsForImport(t *testing.T) {
	mockGetter := &MockWorkflowGetter{
		RepoInfo: &data.RepoSingleQuery{
			Repository: data.RepoInfo{
				DatabaseId: 456,
				Name:       "test-repo",
			},
		},
	}

	tests := []struct {
		name  string
		owner string
		value interface{}
		want  int
	}{
		{
			name:  "invalid type",
			owner: "testorg",
			value: "invalid",
			want:  0,
		},
		{
			name:  "empty workflows",
			owner: "testorg",
			value: []map[string]string{},
			want:  0,
		},
		{
			name:  "valid workflow map",
			owner: "testorg",
			value: []map[string]string{
				{
					"Path":           ".github/workflows/test.yml",
					"Ref":            "main",
					"RepositoryName": "test-repo",
					"SHA":            "abc123",
				},
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mockGetter.ParseRequiredWorkflowsForImport(tt.owner, tt.value)
			if len(got) != tt.want {
				t.Errorf("ParseRequiredWorkflowsForImport() returned %d workflows, want %d", len(got), tt.want)
			}
		})
	}
}

func TestUpdateRequiredWorkflowRepoID(t *testing.T) {
	mockGetter := &MockWorkflowGetter{
		RepoInfo: &data.RepoSingleQuery{
			Repository: data.RepoInfo{
				DatabaseId: 789,
				Name:       "target-repo",
			},
		},
		RepoByIDInfo: &data.RepoInfo{
			DatabaseId: 123,
			Name:       "source-repo",
		},
	}

	mockSource := &MockWorkflowGetter{
		RepoByIDInfo: &data.RepoInfo{
			DatabaseId: 123,
			Name:       "source-repo",
		},
	}

	tests := []struct {
		name    string
		owner   string
		ruleset data.RepoRuleset
		want    int
		wantID  int
	}{
		{
			name:  "no workflow rules",
			owner: "testorg",
			ruleset: data.RepoRuleset{
				Name: "test-ruleset",
				Rules: []data.Rules{
					{
						Type:       "creation",
						Parameters: nil,
					},
				},
			},
			want:   1,
			wantID: 0,
		},
		{
			name:  "workflow rule with no workflows",
			owner: "testorg",
			ruleset: data.RepoRuleset{
				Name: "test-ruleset",
				Rules: []data.Rules{
					{
						Type: "workflows",
						Parameters: &data.Parameters{
							Workflows: []data.Workflows{},
						},
					},
				},
			},
			want:   1,
			wantID: 0,
		},
		{
			name:  "workflow rule with workflows",
			owner: "testorg",
			ruleset: data.RepoRuleset{
				Name: "test-ruleset",
				Rules: []data.Rules{
					{
						Type: "workflows",
						Parameters: &data.Parameters{
							Workflows: []data.Workflows{
								{
									Path:         ".github/workflows/test.yml",
									Ref:          "main",
									RepositoryID: 123,
									SHA:          "abc123",
								},
							},
						},
					},
				},
			},
			want:   1,
			wantID: 789,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mockGetter.UpdateRequiredWorkflowRepoID(tt.owner, tt.ruleset, mockSource)
			if len(got.Rules) != tt.want {
				t.Errorf("UpdateRequiredWorkflowRepoID() returned %d rules, want %d", len(got.Rules), tt.want)
			}

			if tt.wantID > 0 {
				for _, rule := range got.Rules {
					if rule.Type == "workflows" && rule.Parameters != nil && len(rule.Parameters.Workflows) > 0 {
						if rule.Parameters.Workflows[0].RepositoryID != tt.wantID {
							t.Errorf("UpdateRequiredWorkflowRepoID() repository ID = %d, want %d",
								rule.Parameters.Workflows[0].RepositoryID, tt.wantID)
						}
					}
				}
			}
		})
	}
}
