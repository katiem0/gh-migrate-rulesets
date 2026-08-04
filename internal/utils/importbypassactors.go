package utils

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"go.uber.org/zap"
)

func (g *APIGetter) ParseBypassActorsForImport(owner string, bypassActorsStr string, actorMapping map[string]int) []data.BypassActor {
	bypassActors := strings.Split(bypassActorsStr, "|")
	actors := make([]data.BypassActor, 0, len(bypassActors))
	var actorID *int

	for _, actor := range bypassActors {
		actorData := strings.Split(actor, ";")
		if len(actorData) < 4 {
			zap.S().Debug("No Bypass Actor data found")
			continue
		}
		// Explicit mapping wins: covers predefined repository roles (no name lookup API) and renamed actors.
		if sourceID, err := strconv.Atoi(actorData[0]); err == nil {
			if targetID, ok := resolveMappedActorID(actorMapping, actorData[1], sourceID); ok {
				zap.S().Debugf("Applying actor mapping for %s %d -> %d", actorData[1], sourceID, targetID)
				mappedID := targetID
				actors = append(actors, data.BypassActor{
					ActorID:    &mappedID,
					ActorType:  actorData[1],
					BypassMode: actorData[3],
				})
				continue
			}
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
					// Default to the source ID up front; a later non-matching role must not
					// overwrite a match found earlier in the list.
					idData, _ := strconv.Atoi(actorData[0])
					actorID = &idData
					for _, CustomRole := range roleData.CustomRoles {
						if CustomRole.Name == actorData[2] {
							actorID = &CustomRole.ID
							break
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

func (g *APIGetter) UpdateBypassActorID(owner string, sourceOrg string, sourceOrgID int, ruleset data.RepoRuleset, s Getter, actorMapping map[string]int) (data.RepoRuleset, error) {
	zap.S().Debugf("Updating Bypass Actor ID for new org %s", owner)

	var errs error
	for i, actor := range ruleset.BypassActors {
		if actor.ActorType == "DeployKey" {
			zap.S().Debugf("Keeping for DeployKey in ruleset %s", ruleset.Name)
			continue
		}
		if actor.ActorID == nil {
			zap.S().Warnf("Skipping bypass actor with nil ActorID (type %s) in ruleset %s", actor.ActorType, ruleset.Name)
			continue
		}
		// Explicit mapping wins: covers predefined repository roles (no name lookup API) and renamed actors.
		if targetID, ok := resolveMappedActorID(actorMapping, actor.ActorType, *actor.ActorID); ok {
			zap.S().Debugf("Applying actor mapping for %s %d -> %d", actor.ActorType, *actor.ActorID, targetID)
			mappedID := targetID
			ruleset.BypassActors[i].ActorID = &mappedID
			continue
		}
		if _, ok := data.RolesMap[strconv.Itoa(*actor.ActorID)]; !ok {
			if actor.ActorType == "RepositoryRole" {
				zap.S().Debugf("Processing bypass actor custom repository role")
				sourceRole, err := s.GetCustomRoles(sourceOrg, *actor.ActorID)
				if err != nil {
					zap.S().Errorf("Failed to get custom role data for actor ID %d: %v", *actor.ActorID, err)
					errs = errors.Join(errs, fmt.Errorf("bypass actor: looking up source custom role for actor ID %d in %s: %w", *actor.ActorID, sourceOrg, err))
					continue
				}
				roleData, err := g.GetRepoCustomRoles(owner)
				if err != nil || len(roleData.CustomRoles) == 0 {
					zap.S().Infof("Failed to get new custom role data for Role ID %d", *actor.ActorID)
					if err != nil {
						errs = errors.Join(errs, fmt.Errorf("bypass actor: looking up target repository roles in %s for role %q: %w", owner, sourceRole.Name, err))
					} else {
						errs = errors.Join(errs, fmt.Errorf("bypass actor: no repository roles found in target org %s for role %q", owner, sourceRole.Name))
					}
					continue
				}
				matched := false
				for _, CustomRole := range roleData.CustomRoles {
					if CustomRole.Name == sourceRole.Name {
						ruleset.BypassActors[i].ActorID = &CustomRole.ID
						matched = true
					}
				}
				if !matched {
					errs = errors.Join(errs, fmt.Errorf("bypass actor: no matching repository role %q in target org %s", sourceRole.Name, owner))
				}
			} else if actor.ActorType == "Integration" {
				zap.S().Debugf("Processing bypass actor integration from %s", sourceOrg)
				sourceAppIntegration, err := s.GetAppInstallations(sourceOrg)
				if err != nil {
					zap.S().Errorf("Failed to get integration app data for actor ID %d: %v", *actor.ActorID, err)
					errs = errors.Join(errs, fmt.Errorf("bypass actor: looking up integrations in source org %s for actor ID %d: %w", sourceOrg, *actor.ActorID, err))
					continue
				}
				matched := false
				for _, app := range sourceAppIntegration.Installations {
					zap.S().Debugf("Processing bypass actor integration %s", app.AppSlug)
					if *actor.ActorID == app.AppID {
						matched = true
						appIntegrationInfo, err := g.GetAnApp(app.AppSlug)
						if err != nil {
							zap.S().Errorf("Failed to get new integration app data for actor ID %d: %v", *actor.ActorID, err)
							errs = errors.Join(errs, fmt.Errorf("bypass actor: looking up integration %q in target org %s: %w", app.AppSlug, owner, err))
						} else {
							ruleset.BypassActors[i].ActorID = &appIntegrationInfo.AppID
						}
						break
					}
				}
				if !matched {
					errs = errors.Join(errs, fmt.Errorf("bypass actor: integration with app ID %d not found in source org %s", *actor.ActorID, sourceOrg))
				}
			} else if actor.ActorType == "Team" {
				zap.S().Debugf("Processing bypass actor team")
				sourceTeamData, err := s.GetTeamData(sourceOrgID, *actor.ActorID)
				if err != nil {
					zap.S().Infof("Failed to get team data for team id %d", *actor.ActorID)
					errs = errors.Join(errs, fmt.Errorf("bypass actor: looking up team ID %d in source org %s: %w", *actor.ActorID, sourceOrg, err))
					continue
				}
				teamData, err := g.GetTeamByName(owner, sourceTeamData.Name)
				if err != nil {
					zap.S().Infof("Failed to get team data for team name %s", sourceTeamData.Name)
					errs = errors.Join(errs, fmt.Errorf("bypass actor: looking up team %q in target org %s: %w", sourceTeamData.Name, owner, err))
					continue
				}
				ruleset.BypassActors[i].ActorID = &teamData.ID
			}
		}
	}
	return ruleset, errs
}
