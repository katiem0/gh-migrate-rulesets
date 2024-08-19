package utils

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"go.uber.org/zap"
)

func ParametersToMap(params data.Parameters, ruleType string) map[string]string {
	result := make(map[string]string)
	v := reflect.ValueOf(params)

	validFields := GetValidFields(ruleType)
	if validFields == nil {
		return result
	}

	for fieldName := range validFields {
		field := v.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}

		if field.Kind() == reflect.Slice && field.Len() == 0 {
			continue
		}

		switch field.Type() {
		case reflect.TypeOf([]data.Workflows{}):
			var workflowStrings []string
			for j := 0; j < field.Len(); j++ {
				workflow := field.Index(j).Interface().(data.Workflows)
				workflowString := fmt.Sprintf("{Path=%s|Ref=%s|RepositoryID=%d|SHA=%s}", workflow.Path, workflow.Ref, workflow.RepositoryID, workflow.SHA)
				workflowStrings = append(workflowStrings, workflowString)
			}
			result[fieldName] = strings.Join(workflowStrings, ";")
		case reflect.TypeOf([]data.CodeScanning{}):
			var codeScanningStrings []string
			for j := 0; j < field.Len(); j++ {
				codeScanning := field.Index(j).Interface().(data.CodeScanning)
				codeScanningString := fmt.Sprintf("{Tool=%s|SecurityAlertsThreshold=%s|AlertsThreshold=%s}", codeScanning.Tool, codeScanning.SecurityAlertsThreshold, codeScanning.AlertsThreshold)
				codeScanningStrings = append(codeScanningStrings, codeScanningString)
			}
			result[fieldName] = strings.Join(codeScanningStrings, ";")
		case reflect.TypeOf([]data.StatusChecks{}):
			var statusCheckStrings []string
			for j := 0; j < field.Len(); j++ {
				statusCheck := field.Index(j).Interface().(data.StatusChecks)
				statusCheckString := fmt.Sprintf("{Context=%s|IntegrationID=%d}", statusCheck.Context, statusCheck.IntegrationID)
				statusCheckStrings = append(statusCheckStrings, statusCheckString)
			}
			result[fieldName] = strings.Join(statusCheckStrings, ";")
		case reflect.TypeOf(true): // Check for boolean type
			result[fieldName] = fmt.Sprintf("%v", field.Bool())
		default:
			result[fieldName] = fmt.Sprintf("%v", field.Interface())
		}
	}
	return result
}

// getValidFields returns the valid fields for a given rule type.
func GetValidFields(ruleType string) map[string]map[string]struct{} {
	validFields := map[string]map[string]map[string]struct{}{
		"merge_queue": {
			"CheckResponseTimeoutMinutes":  {},
			"GroupingStrategy":             {},
			"MaxEntriesToBuild":            {},
			"MaxEntriesToMerge":            {},
			"MergeType":                    {},
			"MinEntriesToMerge":            {},
			"MinEntriesToMergeWaitMinutes": {},
		},
		"required_deployments": {
			"RequiredDeploymentEnvironments": {},
		},
		"pull_request": {
			"DismissStaleReviewsOnPush":      {},
			"RequireCodeOwnerReview":         {},
			"RequireLastPushApproval":        {},
			"RequiredApprovingReviewCount":   {},
			"RequiredReviewThreadResolution": {},
		},
		"required_status_checks": {
			"DoNotEnforceOnCreate": {},
			"RequiredStatusChecks": {
				"Context":       {},
				"IntegrationID": {},
			},
			"StrictRequiredStatusChecksPolicy": {},
		},
		"commit_message_pattern": {
			"Name":     {},
			"Negate":   {},
			"Operator": {},
			"Pattern":  {},
		},
		"commit_author_email_pattern": {
			"Name":     {},
			"Negate":   {},
			"Operator": {},
			"Pattern":  {},
		},
		"committer_email_pattern": {
			"Name":     {},
			"Negate":   {},
			"Operator": {},
			"Pattern":  {},
		},
		"branch_name_pattern": {
			"Name":     {},
			"Negate":   {},
			"Operator": {},
			"Pattern":  {},
		},
		"tag_name_pattern": {
			"Name":     {},
			"Negate":   {},
			"Operator": {},
			"Pattern":  {},
		},
		"file_path_restriction": {
			"RestrictedFilePaths": {},
		},
		"max_file_path_length": {
			"MaxFilePathLength": {},
		},
		"file_extension_restriction": {
			"RestrictedFileExtensions": {},
		},
		"max_file_size": {
			"MaxFileSize": {},
		},
		"workflows": {
			"DoNotEnforceOnCreate": {},
			"Workflows": {
				"Path":         {},
				"Ref":          {},
				"RepositoryID": {},
				"SHA":          {},
			},
		},
		"code_scanning": {
			"CodeScanningTools": {
				"Tool":                    {},
				"SecurityAlertsThreshold": {},
				"AlertsThreshold":         {},
			},
		},
	}
	return validFields[ruleType]
}

func MapToParameters(paramsMap map[string]interface{}, ruleType string) *data.Parameters {
	var params data.Parameters
	validFields := GetValidFields(ruleType)
	if validFields == nil {
		return nil
	}

	workflowsType := reflect.TypeOf([]data.Workflows{})
	codeScanningType := reflect.TypeOf([]data.CodeScanning{})
	statusChecksType := reflect.TypeOf([]data.StatusChecks{})
	stringSliceType := reflect.TypeOf([]string{})
	v := reflect.ValueOf(&params).Elem()

	for fieldName := range validFields {
		value, exists := paramsMap[fieldName]
		if !exists {
			continue
		}

		field := v.FieldByName(fieldName)
		if !field.IsValid() || !field.CanSet() {
			zap.S().Debug("Field is not valid or cannot be set")
			continue
		}
		switch field.Kind() {
		case reflect.Slice:
			if field.Type() == workflowsType {
				parsedValue := parseWorkflows(value)
				if len(parsedValue) > 0 {
					field.Set(reflect.ValueOf(parsedValue))
				}
			} else if field.Type() == codeScanningType {
				parsedValue := parseCodeScanning(value)
				if len(parsedValue) > 0 {
					field.Set(reflect.ValueOf(parsedValue))
				}
			} else if field.Type() == statusChecksType {
				parsedValue := parseStatusChecks(value)
				if len(parsedValue) > 0 {
					field.Set(reflect.ValueOf(parsedValue))
				}
			} else if field.Type() == stringSliceType {
				strSliceValue, ok := value.([]string)
				if ok {
					field.Set(reflect.ValueOf(strSliceValue))
				}
			}
		case reflect.String:
			strValue := value.(string)
			field.SetString(strValue)
		case reflect.Int, reflect.Int64:
			intValue, _ := strconv.ParseInt(value.(string), 10, 64)
			field.SetInt(intValue)
		case reflect.Bool:
			boolValue, _ := strconv.ParseBool(value.(string))
			field.SetBool(boolValue)
		}

	}
	return &params
}

func parseWorkflows(value interface{}) []data.Workflows {
	var workflows []data.Workflows
	v, ok := value.([]map[string]string)
	if !ok {
		zap.S().Error("Invalid type for value")
		return workflows
	}
	for _, workflowMap := range v {
		repositoryID, err := strconv.Atoi(workflowMap["RepositoryID"])
		if err != nil {
			zap.S().Errorf("Invalid RepositoryID:", workflowMap["RepositoryID"])
			continue
		}
		workflow := data.Workflows{
			Path:         workflowMap["Path"],
			Ref:          workflowMap["Ref"],
			RepositoryID: repositoryID,
			SHA:          workflowMap["SHA"],
		}
		workflows = append(workflows, workflow)
	}
	return workflows
}

func parseStatusChecks(value interface{}) []data.StatusChecks {
	var statusChecks []data.StatusChecks
	v, ok := value.([]map[string]string)
	if !ok {
		zap.S().Error("Invalid type for value")
		return statusChecks
	}
	for _, statusMap := range v {
		integrationID, err := strconv.Atoi(statusMap["IntegrationID"])
		if err != nil {
			zap.S().Errorf("Invalid IntegrationID:", statusMap["IntegrationID"])
			continue
		}
		statusCheck := data.StatusChecks{
			Context:       statusMap["Context"],
			IntegrationID: integrationID,
		}
		statusChecks = append(statusChecks, statusCheck)
	}
	return statusChecks
}

func parseCodeScanning(value interface{}) []data.CodeScanning {
	var codeScannings []data.CodeScanning
	v, ok := value.([]map[string]string)
	if !ok {
		zap.S().Error("Invalid type for value")
		return codeScannings
	}

	for _, codeScanMap := range v {
		codeScanning := data.CodeScanning{
			Tool:                    codeScanMap["Tool"],
			SecurityAlertsThreshold: codeScanMap["SecurityAlertsThreshold"],
			AlertsThreshold:         codeScanMap["AlertsThreshold"],
		}
		codeScannings = append(codeScannings, codeScanning)
	}
	return codeScannings
}
