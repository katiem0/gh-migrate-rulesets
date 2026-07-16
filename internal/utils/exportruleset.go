package utils

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/katiem0/gh-migrate-rulesets/internal/data" // Add this import
	"go.uber.org/zap"
)

func (g *APIGetter) ProcessActorsForExport(actors []data.BypassActor, owner string, orgID int, ruleID string) []string {
	zap.S().Debugf("Processing bypass actors for ruleset %s", ruleID)
	var actorStrings []string
	var actorName string

	for _, actor := range actors {
		if actor.ActorID == nil {
			defaultID := 0
			actor.ActorID = &defaultID
		}

		if _, ok := data.RolesMap[strconv.Itoa(*actor.ActorID)]; ok {
			actorName = data.RolesMap[strconv.Itoa(*actor.ActorID)]
		} else {
			if actor.ActorType == "RepositoryRole" {
				zap.S().Debugf("Processing bypass actor custom repository role")
				roleName, err := g.GetCustomRoles(owner, *actor.ActorID)
				if err != nil {
					zap.S().Warnf("Failed to get custom role data for actor ID %d: %v", *actor.ActorID, err)
					actorName = fmt.Sprintf("UnknownRole-%d", *actor.ActorID)
				} else {
					actorName = roleName.Name
				}
			} else if actor.ActorType == "Integration" {
				zap.S().Debugf("Processing bypass actor integration")
				appIntegrationData, err := g.GetAppInstallations(owner)
				if err != nil {
					zap.S().Warnf("Failed to get integration app data for actor ID %d: %v", *actor.ActorID, err)
					actorName = fmt.Sprintf("UnknownIntegration-%d", *actor.ActorID)
				} else {
					found := false
					for _, appIntegration := range appIntegrationData.Installations {
						if appIntegration.AppID == *actor.ActorID {
							actorName = appIntegration.AppSlug
							found = true
							break
						}
					}
					if !found {
						actorName = fmt.Sprintf("UnknownIntegration-%d", *actor.ActorID)
					}
				}
			} else if actor.ActorType == "Team" {
				zap.S().Debugf("Processing bypass actor team for org ID %d, team ID %d", orgID, *actor.ActorID)
				teamData, err := g.GetTeamData(orgID, *actor.ActorID)
				if err != nil {
					zap.S().Warnf("Failed to get team data for actor ID %d in org %d: %v", *actor.ActorID, orgID, err)
					actorName = fmt.Sprintf("UnknownTeam-%d", *actor.ActorID)
				} else if teamData != nil {
					actorName = teamData.Name
				} else {
					actorName = fmt.Sprintf("UnknownTeam-%d", *actor.ActorID)
				}
			} else {
				zap.S().Infof("Invalid actor type: %s", actor.ActorType)
				actorName = fmt.Sprintf("Unknown-%d", *actor.ActorID)
			}
		}

		actorList := []string{
			strconv.Itoa(*actor.ActorID),
			actor.ActorType,
			actorName,
			actor.BypassMode,
		}
		actorStrings = append(actorStrings, strings.Join(actorList, ";"))
	}
	return actorStrings
}
func ProcessConditions(ruleset data.RepoRuleset) data.ProcessedConditions {
	var PropertyInclude, PropertyExclude []string
	var includeNames, excludeNames, boolNames, includeRefNames, excludeRefNames string
	if ruleset.Conditions != nil {
		if ruleset.Conditions.RepositoryName != nil {
			includeNames = strings.Join(ruleset.Conditions.RepositoryName.Include, ";")
			excludeNames = strings.Join(ruleset.Conditions.RepositoryName.Exclude, ";")
			boolNames = strconv.FormatBool(ruleset.Conditions.RepositoryName.Protected)
		}
		if ruleset.Conditions.RepositoryProperty != nil {
			PropertyInclude = ProcessProperties(ruleset.Conditions.RepositoryProperty.Include)
			PropertyExclude = ProcessProperties(ruleset.Conditions.RepositoryProperty.Exclude)
		}
		if ruleset.Conditions.RefName != nil {
			includeRefNames = strings.Join(ruleset.Conditions.RefName.Include, ";")
			excludeRefNames = strings.Join(ruleset.Conditions.RefName.Exclude, ";")
		}
	}
	return data.ProcessedConditions{
		IncludeNames:    includeNames,
		ExcludeNames:    excludeNames,
		BoolNames:       boolNames,
		PropertyInclude: PropertyInclude,
		PropertyExclude: PropertyExclude,
		IncludeRefNames: includeRefNames,
		ExcludeRefNames: excludeRefNames,
	}
}

func ProcessProperties(properties []data.PropertyPattern) []string {
	var propertyStrings []string
	for _, property := range properties {
		propertyList := []string{
			property.Name,
			property.Source,
			fmt.Sprintf("{%s}", strings.Join(property.PropertyValues, "|")),
		}
		propertyStrings = append(propertyStrings, strings.Join(propertyList, ";"))
	}
	return propertyStrings
}

func (g *APIGetter) ProcessRules(rules []data.Rules) map[string]string {
	zap.S().Debugf("Processing rules")
	rulesMap := make(map[string]string)
	for _, rule := range rules {
		if rule.Parameters == nil {
			rulesMap[rule.Type] = "true"
		} else {
			parametersMap := g.ParametersToMap(*rule.Parameters, rule.Type)
			keys := make([]string, 0, len(parametersMap))
			for key := range parametersMap {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			var formattedParams []string
			for _, key := range keys {
				formattedParams = append(formattedParams, fmt.Sprintf("%s:%v", key, parametersMap[key]))
			}
			if len(formattedParams) > 0 {
				rulesMap[rule.Type] = strings.Join(formattedParams, "|")
			}
		}
	}
	return rulesMap
}
