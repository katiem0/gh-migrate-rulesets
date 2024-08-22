package utils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"go.uber.org/zap"
)

func ProcessActors(actors []data.BypassActor) []string {
	zap.S().Debugf("Processing bypass actors")
	var actorStrings []string
	for _, actor := range actors {
		actorList := []string{
			strconv.Itoa(actor.ActorID),
			actor.ActorType,
			actor.BypassMode,
		}
		actorStrings = append(actorStrings, strings.Join(actorList, ";"))
	}
	return actorStrings
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

func ProcessRules(rules []data.Rules) map[string]string {
	zap.S().Debugf("Processing rules")
	rulesMap := make(map[string]string)
	for _, rule := range rules {
		if rule.Parameters == nil {
			rulesMap[rule.Type] = "true"
		} else {
			parametersMap := ParametersToMap(*rule.Parameters, rule.Type)
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
