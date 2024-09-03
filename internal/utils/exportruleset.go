package utils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/katiem0/gh-migrate-rulesets/internal/data" // Add this import
	"go.uber.org/zap"
)

func (g *APIGetter) ProcessActorsForExport(actors []data.BypassActor, owner string, orgID int, ruleID string) []string {
	zap.S().Debugf("Processing bypass actors")
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
					zap.S().Errorf("Failed to get custom role data for actor ID %d: %v", actor.ActorID, err)
					continue
				}
				actorName = roleName.Name
			} else if actor.ActorType == "Integration" {
				zap.S().Debugf("Processing bypass actor integration")
				appIntegrationData, err := g.GetAppInstallations(owner)
				if err != nil {
					zap.S().Errorf("Failed to get integration app data for actor ID %d: %v", actor.ActorID, err)
					continue
				}
				for _, appIntegration := range appIntegrationData.Installations {
					if appIntegration.AppID == *actor.ActorID {
						actorName = appIntegration.AppSlug
					}
				}
			} else if actor.ActorType == "Team" {
				zap.S().Debugf("Processing bypass actor team")
				teamData, err := g.GetTeamData(orgID, *actor.ActorID)
				if err != nil {
					zap.S().Errorf("Failed to get team data for actor ID %d: %v", actor.ActorID, err)
					continue
				}
				actorName = teamData.Name
			} else {
				zap.S().Infof("Invalid actor type: %s", actor.ActorType)
				actorName = ""
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
		includeRefNames = strings.Join(ruleset.Conditions.RefName.Include, ";")
		excludeRefNames = strings.Join(ruleset.Conditions.RefName.Exclude, ";")
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
			var formattedParams []string
			for key, value := range parametersMap {
				formattedParams = append(formattedParams, fmt.Sprintf("%s:%v", key, value))
			}
			if len(formattedParams) > 0 {
				rulesMap[rule.Type] = strings.Join(formattedParams, "|")
			}
		}
	}
	return rulesMap
}
