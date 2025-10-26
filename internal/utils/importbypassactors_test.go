package utils

import (
	"testing"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
)

func TestParseBypassActorsForImport(t *testing.T) {
	g := &APIGetter{}

	tests := []struct {
		name            string
		owner           string
		bypassActorsStr string
		wantCount       int
	}{
		{
			name:            "empty string",
			owner:           "testorg",
			bypassActorsStr: "",
			wantCount:       0,
		},
		{
			name:            "single actor with known role",
			owner:           "testorg",
			bypassActorsStr: "1;OrgAdmin;OrgAdmin;always",
			wantCount:       1,
		},
		{
			name:            "multiple actors",
			owner:           "testorg",
			bypassActorsStr: "1;OrgAdmin;OrgAdmin;always|5;Admin;Admin;pull_request",
			wantCount:       2,
		},
		{
			name:            "deploy key actor",
			owner:           "testorg",
			bypassActorsStr: "0;DeployKey;AllDeployKeys;always",
			wantCount:       1,
		},
		{
			name:            "invalid format",
			owner:           "testorg",
			bypassActorsStr: "invalid",
			wantCount:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.ParseBypassActorsForImport(tt.owner, tt.bypassActorsStr)
			if len(got) != tt.wantCount {
				t.Errorf("ParseBypassActorsForImport() returned %d actors, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestUpdateBypassActorID(t *testing.T) {
	g := &APIGetter{}
	s := &APIGetter{}

	tests := []struct {
		name        string
		owner       string
		sourceOrg   string
		sourceOrgID int
		ruleset     data.RepoRuleset
		wantActors  int
	}{
		{
			name:        "empty bypass actors",
			owner:       "neworg",
			sourceOrg:   "oldorg",
			sourceOrgID: 12345,
			ruleset: data.RepoRuleset{
				Name:         "test-ruleset",
				BypassActors: []data.BypassActor{},
			},
			wantActors: 0,
		},
		{
			name:        "deploy key actor",
			owner:       "neworg",
			sourceOrg:   "oldorg",
			sourceOrgID: 12345,
			ruleset: data.RepoRuleset{
				Name: "test-ruleset",
				BypassActors: []data.BypassActor{
					{
						ActorID:    nil,
						ActorType:  "DeployKey",
						BypassMode: "always",
					},
				},
			},
			wantActors: 1,
		},
		{
			name:        "known role actors",
			owner:       "neworg",
			sourceOrg:   "oldorg",
			sourceOrgID: 12345,
			ruleset: data.RepoRuleset{
				Name: "test-ruleset",
				BypassActors: []data.BypassActor{
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
			},
			wantActors: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.UpdateBypassActorID(tt.owner, tt.sourceOrg, tt.sourceOrgID, tt.ruleset, s)
			if len(got.BypassActors) != tt.wantActors {
				t.Errorf("UpdateBypassActorID() returned %d actors, want %d", len(got.BypassActors), tt.wantActors)
			}
		})
	}
}
