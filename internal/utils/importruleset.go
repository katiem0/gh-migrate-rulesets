package utils

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"go.uber.org/zap"
)

func CreateRepoRulesetsData(owner string, fileData [][]string) []data.RepoRuleset {
	var importRepoRuleset []data.RepoRuleset
	var repoRuleset data.RepoRuleset

	// Create a map for header indices
	headerMap := make(map[string]int)
	for i, header := range fileData[0] {
		headerMap[header] = i
	}

	for _, each := range fileData[1:] {
		zap.S().Debugf("Gathering info for each ruleset")
		repoRuleset.ID, _ = strconv.Atoi(each[headerMap["RuleID"]])
		repoRuleset.Name = each[headerMap["RulesetName"]]
		repoRuleset.Target = each[headerMap["Target"]]
		repoRuleset.SourceType = each[headerMap["RulesetLevel"]]
		repoRuleset.Source = determineSource(owner, each[headerMap["RulesetLevel"]], each[headerMap["RepositoryName"]])
		repoRuleset.Enforcement = each[headerMap["Enforcement"]]
		repoRuleset.BypassActors = parseBypassActors(each[headerMap["BypassActors"]])
		repoRuleset.Conditions = parseConditions(each[headerMap["ConditionsRefNameInclude"] : headerMap["ConditionRepoPropertyExclude"]+1])
		ruleHeaders := fileData[0][14:35]
		ruleValues := each[14:35]
		repoRuleset.Rules = parseRules(ruleHeaders, ruleValues)
		repoRuleset.CreatedAt = each[headerMap["CreatedAt"]]
		repoRuleset.UpdatedAt = each[headerMap["UpdatedAt"]]
		importRepoRuleset = append(importRepoRuleset, repoRuleset)
	}
	return importRepoRuleset
}

func determineSource(owner, sourceType, repoName string) string {
	if sourceType == "Organization" {
		return owner
	} else if sourceType == "Repository" {
		return fmt.Sprintf("%s/%s", owner, repoName)
	}
	return ""
}

func parseBypassActors(bypassActorsStr string) []data.BypassActor {
	bypassActors := strings.Split(bypassActorsStr, "|")
	actors := make([]data.BypassActor, 0, len(bypassActors))

	for _, actor := range bypassActors {
		actorData := strings.Split(actor, ";")
		if len(actorData) < 2 {
			zap.S().Debug("No Bypass Actor data found")
			continue
		}
		actorID, err := strconv.Atoi(actorData[0])
		if err != nil {
			zap.S().Errorf("Invalid actor ID: %s", actorData[0])
			continue
		}
		actors = append(actors, data.BypassActor{
			ActorID:    actorID,
			ActorType:  actorData[1],
			BypassMode: actorData[2],
		})
	}

	return actors
}

func parseConditions(conditions []string) *data.Conditions {
	zap.S().Debugln("validating ruleset conditions")
	return &data.Conditions{
		RefName: &data.RefPatterns{
			Include: strings.Split(conditions[0], ";"),
			Exclude: strings.Split(conditions[1], ";"),
		},
		RepositoryName: &data.NamePatterns{
			Include:   strings.Split(conditions[2], ";"),
			Exclude:   strings.Split(conditions[3], ";"),
			Protected: conditions[4] == "true",
		},
		RepositoryProperty: &data.PropertyPatterns{
			Include: parsePropertyPatterns(conditions[5]),
			Exclude: parsePropertyPatterns(conditions[6]),
		},
	}
}

func parsePropertyPatterns(patternsStr string) []data.PropertyPattern {
	patterns := splitIgnoringBraces(patternsStr, "|")
	propertyPatterns := make([]data.PropertyPattern, 0, len(patterns))
	for _, pattern := range patterns {
		patternData := strings.Split(pattern, ";")
		if len(patternData) < 2 {
			continue
		}
		valueTrimmed := strings.Trim(patternData[2], "{}")
		propertyPatterns = append(propertyPatterns, data.PropertyPattern{
			Name:           patternData[0],
			Source:         patternData[1],
			PropertyValues: strings.Split(valueTrimmed, "|"),
		})
	}
	return propertyPatterns
}

func parseRules(headerMap []string, ruleValues []string) []data.Rules {
	rules := make([]data.Rules, 0, len(headerMap))

	for i := 0; i < len(headerMap) && i < len(ruleValues); i++ {
		if ruleValues[i] == "" {
			continue // Skip rules with empty values
		}
		header := data.HeaderMap[headerMap[i]]
		rule := data.Rules{
			Type: header,
		}
		if ruleValues[i] != "" {
			parameters := parseParameters(ruleValues[i])
			rule.Parameters = MapToParameters(parameters, header)
		} else {
			zap.S().Debugf("%s does not contain Parameters", header)
		}
		rules = append(rules, rule)
	}
	return rules
}

func parseParameters(paramStr string) map[string]interface{} {
	params := make(map[string]interface{})
	if paramStr == "" {
		return nil
	} else {
		paramPairs := splitIgnoringBraces(paramStr, "|")
		for _, pair := range paramPairs {
			kv := strings.Split(pair, ":")
			if len(kv) != 2 {
				continue
			}
			key := kv[0]
			value := kv[1]

			if strings.Contains(value, "{") {
				subGroups := strings.Split(value, ";")

				var subMap []map[string]string
				for _, subGroup := range subGroups {
					valueTrimmed := strings.Trim(subGroup, "{}")
					pairGroups := strings.Split(valueTrimmed, "|")
					subGroupMap := make(map[string]string)
					for _, pairGroup := range pairGroups {
						subKv := strings.Split(pairGroup, "=")
						if len(subKv) == 2 {
							subGroupMap[subKv[0]] = subKv[1]
						}
					}
					subMap = append(subMap, subGroupMap)
				}
				params[key] = subMap
			} else if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
				value = strings.Trim(value, "[]")
				params[key] = strings.Split(value, " ")
			} else {
				params[key] = value
			}
		}
		return params
	}
}

func splitIgnoringBraces(s, delimiter string) []string {
	var result []string
	var currentSegment strings.Builder
	inBraces := false

	for i := 0; i < len(s); i++ {
		char := s[i]

		if char == '{' {
			inBraces = true
		} else if char == '}' {
			inBraces = false
		}

		if !inBraces && strings.HasPrefix(s[i:], delimiter) {
			result = append(result, currentSegment.String())
			currentSegment.Reset()
			i += len(delimiter) - 1
		} else {
			currentSegment.WriteByte(char)
		}
	}
	result = append(result, currentSegment.String())
	return result
}

func CleanConditions(conditions *data.Conditions) *data.Conditions {
	conditions.RefName.Include = CleanSlice(conditions.RefName.Include)
	conditions.RefName.Exclude = CleanSlice(conditions.RefName.Exclude)
	conditions.RepositoryName.Include = CleanSlice(conditions.RepositoryName.Include)
	conditions.RepositoryName.Exclude = CleanSlice(conditions.RepositoryName.Exclude)

	if ShouldRemoveRepositoryName(conditions.RepositoryName) {
		conditions.RepositoryName = nil
	}
	if ShouldRemoveRefName(conditions.RefName) {
		conditions.RefName = nil
	}
	if ShouldRemoveProperty(conditions.RepositoryProperty) {
		conditions.RepositoryProperty = nil
	}
	if conditions.RefName != nil {
		if conditions.RefName.Include == nil {
			conditions.RefName.Include = []string{}
		}
		if conditions.RefName.Exclude == nil {
			conditions.RefName.Exclude = []string{}
		}
	}
	if conditions.RepositoryProperty != nil {
		if conditions.RepositoryProperty.Include == nil {
			conditions.RepositoryProperty.Include = []data.PropertyPattern{}
		} else {
			for i, property := range conditions.RepositoryProperty.Include {
				property.PropertyValues = CleanSlice(property.PropertyValues)
				conditions.RepositoryProperty.Include[i] = property
			}
		}
		if conditions.RepositoryProperty.Exclude == nil {
			conditions.RepositoryProperty.Exclude = []data.PropertyPattern{}
		}
	}
	return conditions
}

func CleanSlice(slice []string) []string {
	var cleaned []string
	for _, item := range slice {
		if item != "" {
			cleaned = append(cleaned, item)
		}
	}
	return cleaned
}

func ShouldRemoveRepositoryName(repoName *data.NamePatterns) bool {
	return repoName != nil && len(repoName.Include) == 0 && len(repoName.Exclude) == 0 && !repoName.Protected
}

func ShouldRemoveRefName(refName *data.RefPatterns) bool {
	return refName != nil && len(refName.Include) == 0 && len(refName.Exclude) == 0
}

func ShouldRemoveProperty(propName *data.PropertyPatterns) bool {
	return propName != nil && len(propName.Include) == 0 && len(propName.Exclude) == 0
}

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func UpdateTag(field reflect.StructField, key, value string) reflect.StructField {
	tag := field.Tag.Get(key)
	if tag == "" {
		field.Tag = reflect.StructTag(key + `:"` + value + `"`)
	} else {
		parts := strings.Split(tag, ",")
		newParts := []string{}
		for _, part := range parts {
			if part != "omitempty" {
				newParts = append(newParts, part)
			}
		}
		field.Tag = reflect.StructTag(key + `:"` + strings.Join(newParts, ",") + `"`)
	}
	return field
}
