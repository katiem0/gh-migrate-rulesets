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
