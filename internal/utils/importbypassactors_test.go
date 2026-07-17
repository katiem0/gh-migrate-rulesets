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
		actorMapping    map[string]int
		wantCount       int
		wantFirstID     *int
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
		{
			name:            "predefined role remapped via actor mapping",
			owner:           "testorg",
			bypassActorsStr: "5;RepositoryRole;Admin;always",
			actorMapping:    map[string]int{actorMappingKey("RepositoryRole", 5): 42},
			wantCount:       1,
			wantFirstID:     intPtr(42),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.ParseBypassActorsForImport(tt.owner, tt.bypassActorsStr, tt.actorMapping)
			if len(got) != tt.wantCount {
				t.Errorf("ParseBypassActorsForImport() returned %d actors, want %d", len(got), tt.wantCount)
			}
			if tt.wantFirstID != nil {
				if len(got) == 0 || got[0].ActorID == nil || *got[0].ActorID != *tt.wantFirstID {
					t.Errorf("ParseBypassActorsForImport() first actor ID = %v, want %d", got, *tt.wantFirstID)
				}
			}
		})
	}
}

func TestUpdateBypassActorID(t *testing.T) {
	g := &APIGetter{}
	s := &APIGetter{}

	tests := []struct {
		name         string
		owner        string
		sourceOrg    string
		sourceOrgID  int
		ruleset      data.RepoRuleset
		actorMapping map[string]int
		wantActors   int
		wantActorID  *int
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
		{
			name:        "predefined role overridden by actor mapping",
			owner:       "neworg",
			sourceOrg:   "oldorg",
			sourceOrgID: 12345,
			ruleset: data.RepoRuleset{
				Name: "test-ruleset",
				BypassActors: []data.BypassActor{
					{
						ActorID:    intPtr(5),
						ActorType:  "RepositoryRole",
						BypassMode: "always",
					},
				},
			},
			actorMapping: map[string]int{actorMappingKey("RepositoryRole", 5): 7},
			wantActors:   1,
			wantActorID:  intPtr(7),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.UpdateBypassActorID(tt.owner, tt.sourceOrg, tt.sourceOrgID, tt.ruleset, s, tt.actorMapping)
			if len(got.BypassActors) != tt.wantActors {
				t.Errorf("UpdateBypassActorID() returned %d actors, want %d", len(got.BypassActors), tt.wantActors)
			}
			if tt.wantActorID != nil {
				if got.BypassActors[0].ActorID == nil || *got.BypassActors[0].ActorID != *tt.wantActorID {
					t.Errorf("UpdateBypassActorID() actor ID = %v, want %d", got.BypassActors[0].ActorID, *tt.wantActorID)
				}
			}
		})
	}
}
