package utils

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"go.uber.org/zap"
)

func (g *APIGetter) CreateRepoRulesetsData(owner string, fileData [][]string) []data.RepoRuleset {
	var importRepoRuleset []data.RepoRuleset
	var repoRuleset data.RepoRuleset
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
		repoRuleset.BypassActors = g.ParseBypassActorsForImport(owner, each[headerMap["BypassActors"]])
		repoRuleset.Conditions = parseConditions(each[headerMap["ConditionsRefNameInclude"] : headerMap["ConditionRepoPropertyExclude"]+1])
		ruleHeaders := fileData[0][14:35]
		ruleValues := each[14:35]
		repoRuleset.Rules = g.parseRules(owner, ruleHeaders, ruleValues)
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

func parseConditions(conditions []string) *data.Conditions {
	zap.S().Debugln("Validating ruleset conditions")
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
	patterns := SplitIgnoringBraces(patternsStr, "|")
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

func (g *APIGetter) parseRules(owner string, headerMap []string, ruleValues []string) []data.Rules {
	rules := make([]data.Rules, 0, len(headerMap))

	for i := 0; i < len(headerMap) && i < len(ruleValues); i++ {
		if ruleValues[i] == "" {
			continue
		}
		header := data.HeaderMap[headerMap[i]]
		rule := data.Rules{
			Type: header,
		}
		if ruleValues[i] != "" {
			parameters := ParseParameters(ruleValues[i])
			rule.Parameters = g.MapToParameters(owner, parameters, header)
		} else {
			zap.S().Debugf("%s does not contain Parameters", header)
		}
		rules = append(rules, rule)
	}
	return rules
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

func ProcessRulesets(ruleset data.RepoRuleset) (data.CreateRuleset, error) {
	createRuleset := data.CreateRuleset{
		Name:         ruleset.Name,
		Target:       ruleset.Target,
		Enforcement:  ruleset.Enforcement,
		BypassActors: ruleset.BypassActors,
		Conditions:   ruleset.Conditions,
		Rules:        make([]data.CreateRules, len(ruleset.Rules)),
	}
	zap.S().Debugf("Removing omitempty from fields if needed from ruleset: %s", ruleset.Name)
	for i, rule := range ruleset.Rules {
		if rule.Parameters == nil {
			createRuleset.Rules[i].Type = rule.Type
			continue
		} else {
			v := reflect.ValueOf(rule.Parameters).Elem()
			t := v.Type()

			newFields := make([]reflect.StructField, v.NumField())
			for j := 0; j < v.NumField(); j++ {
				fieldType := t.Field(j)
				fieldName := fieldType.Name
				if fields, ok := data.NonOmitEmptyFields[rule.Type]; ok {
					if Contains(fields, fieldName) {
						fieldType = UpdateTag(fieldType, "json", fieldName)
					}
				}
				newFields[j] = fieldType
			}
			newStructType := reflect.StructOf(newFields)
			newStruct := reflect.New(newStructType).Elem()
			for j := 0; j < v.NumField(); j++ {
				fieldValue := v.Field(j)
				if fieldValue.Kind() == reflect.Slice && fieldValue.Type().Elem().Kind() == reflect.String {
					newStruct.Field(j).Set(reflect.MakeSlice(fieldValue.Type(), fieldValue.Len(), fieldValue.Cap()))
					for k := 0; k < fieldValue.Len(); k++ {
						newStruct.Field(j).Index(k).Set(fieldValue.Index(k))
					}
				} else {
					newStruct.Field(j).Set(fieldValue)
				}
			}
			createRuleset.Rules[i].Type = rule.Type
			createRuleset.Rules[i].Parameters = newStruct.Interface()
		}
	}
	return createRuleset, nil
}
