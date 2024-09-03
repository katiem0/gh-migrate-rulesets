package utils

import (
	"strconv"
	"strings"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"go.uber.org/zap"
)

func (g *APIGetter) ParseBypassActorsForImport(owner string, bypassActorsStr string) []data.BypassActor {
	bypassActors := strings.Split(bypassActorsStr, "|")
	actors := make([]data.BypassActor, 0, len(bypassActors))
	var actorID *int

	for _, actor := range bypassActors {
		actorData := strings.Split(actor, ";")
		if len(actorData) < 2 {
			zap.S().Debug("No Bypass Actor data found")
			continue
		}
		if _, ok := data.RolesMap[actorData[0]]; !ok {
			zap.S().Debugf("Gathering appropriate IDs for Bypass Actor: %s", actorData[2])
			if actorData[1] == "RepositoryRole" {
				zap.S().Debugf("Processing bypass actor custom repository role")
				roleData, err := g.GetRepoCustomRoles(owner)
				if err != nil || len(roleData.CustomRoles) == 0 {
					zap.S().Infof("Failed to get custom role data for Role Name %s", actorData[2])
					id, _ := strconv.Atoi(actorData[0])
					actorID = &id
					continue
				} else {
					for _, CustomRole := range roleData.CustomRoles {
						if CustomRole.Name == actorData[2] {
							actorID = &CustomRole.ID
						} else {
							idData, _ := strconv.Atoi(actorData[0])
							actorID = &idData
						}
					}
				}
			} else if actorData[1] == "Integration" {
				zap.S().Debugf("Processing bypass actor integration")
				appIntegrationData, err := g.GetAnApp(actorData[2])
				if err != nil {
					zap.S().Infof("Failed to get integration app data for actor ID %s", actorData[2])
					id, _ := strconv.Atoi(actorData[0])
					actorID = &id
					continue
				} else {
					actorID = &appIntegrationData.AppID
				}
			} else if actorData[1] == "Team" {
				zap.S().Debugf("Processing bypass actor team")
				teamData, err := g.GetTeamByName(owner, actorData[2])
				if err != nil {
					zap.S().Infof("Failed to get team data for team name %s", actorData[2])
					id, _ := strconv.Atoi(actorData[0])
					actorID = &id
					continue
				} else {
					actorID = &teamData.ID
				}
			}
		} else {
			if actorData[1] == "DeployKey" {
				actorID = nil
			} else {
				id, _ := strconv.Atoi(actorData[0])
				actorID = &id
			}
		}
		actors = append(actors, data.BypassActor{
			ActorID:    actorID,
			ActorType:  actorData[1],
			BypassMode: actorData[3],
		})
	}
	return actors
}

func (g *APIGetter) UpdateBypassActorID(owner string, sourceOrg string, sourceOrgID int, ruleset data.RepoRuleset, s *APIGetter) data.RepoRuleset {
	zap.S().Debugf("Updating Bypass Actor ID for new org %s", owner)

	for i, actor := range ruleset.BypassActors {
		if actor.ActorType == "DeployKey" {
			zap.S().Debugf("Keeping for DeployKey in ruleset %s", ruleset.Name)
			continue
		} else {
			if _, ok := data.RolesMap[strconv.Itoa(*actor.ActorID)]; !ok {
				if actor.ActorType == "RepositoryRole" {
					zap.S().Debugf("Processing bypass actor custom repository role")
					sourceRole, err := s.GetCustomRoles(sourceOrg, *actor.ActorID)
					if err != nil {
						zap.S().Errorf("Failed to get custom role data for actor ID %d: %v", actor.ActorID, err)
						continue
					}
					roleData, err := g.GetRepoCustomRoles(owner)
					if err != nil || len(roleData.CustomRoles) == 0 {
						zap.S().Infof("Failed to get new custom role data for Role ID %d", actor.ActorID)
						continue
					} else {
						for _, CustomRole := range roleData.CustomRoles {
							if CustomRole.Name == sourceRole.Name {
								ruleset.BypassActors[i].ActorID = &CustomRole.ID
							}
						}
					}
				} else if actor.ActorType == "Integration" {
					zap.S().Debugf("Processing bypass actor integration from %s", sourceOrg)
					sourceAppIntegration, err := s.GetAppInstallations(sourceOrg)
					if err != nil {
						zap.S().Errorf("Failed to get integration app data for actor ID %d: %v", actor.ActorID, err)
						continue
					} else {
						for _, app := range sourceAppIntegration.Installations {
							zap.S().Debugf("Processing bypass actor integration %s", app.AppSlug)
							if *actor.ActorID == app.AppID {
								appIntegrationInfo, err := g.GetAnApp(app.AppSlug)
								if err != nil {
									zap.S().Errorf("Failed to get new integration app data for actor ID %d: %v", actor.ActorID, err)
									continue
								} else {
									ruleset.BypassActors[i].ActorID = &appIntegrationInfo.AppID
								}
							}
						}
					}
				} else if actor.ActorType == "Team" {
					zap.S().Debugf("Processing bypass actor team")
					sourceTeamData, err := s.GetTeamData(sourceOrgID, *actor.ActorID)
					if err != nil {
						zap.S().Infof("Failed to get team data for team id %d", actor.ActorID)
						continue
					} else {
						teamData, err := g.GetTeamByName(owner, sourceTeamData.Name)
						if err != nil {
							zap.S().Infof("Failed to get team data for team name %s", sourceTeamData.Name)
							continue
						} else {
							ruleset.BypassActors[i].ActorID = &teamData.ID
						}
					}
				}
			}
		}
	}
	return ruleset
}
