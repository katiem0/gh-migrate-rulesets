package utils

import (
	"reflect"
	"testing"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
)

// MockGetter implements the Getter interface for testing
type MockParametersGetter struct{}

func (m *MockParametersGetter) GetRepoByID(repoID int) (*data.RepoInfo, error) {
	return &data.RepoInfo{
		DatabaseId: repoID,
		Name:       "test-repo",
	}, nil
}

func TestGetValidFields(t *testing.T) {
	tests := []struct {
		name     string
		ruleType string
		want     map[string]map[string]struct{}
	}{
		{
			name:     "merge_queue rule type",
			ruleType: "merge_queue",
			want: map[string]map[string]struct{}{
				"CheckResponseTimeoutMinutes":  {},
				"GroupingStrategy":             {},
				"MaxEntriesToBuild":            {},
				"MaxEntriesToMerge":            {},
				"MergeType":                    {},
				"MinEntriesToMerge":            {},
				"MinEntriesToMergeWaitMinutes": {},
			},
		},
		{
			name:     "pull_request rule type",
			ruleType: "pull_request",
			want: map[string]map[string]struct{}{
				"DismissStaleReviewsOnPush":      {},
				"RequireCodeOwnerReview":         {},
				"RequireLastPushApproval":        {},
				"RequiredApprovingReviewCount":   {},
				"RequiredReviewThreadResolution": {},
			},
		},
		{
			name:     "invalid rule type",
			ruleType: "invalid",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetValidFields(tt.ruleType)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetValidFields() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParametersToMap(t *testing.T) {
	g := &APIGetter{}
	// We'll need to override GetRepoByID behavior - use a wrapper approach

	tests := []struct {
		name     string
		params   data.Parameters
		ruleType string
		want     map[string]string
	}{
		{
			name: "pull_request parameters",
			params: data.Parameters{
				RequiredApprovingReviewCount: 2,
				DismissStaleReviewsOnPush:    true,
			},
			ruleType: "pull_request",
			want: map[string]string{
				"RequiredApprovingReviewCount":   "2",
				"DismissStaleReviewsOnPush":      "true",
				"RequireCodeOwnerReview":         "false",
				"RequireLastPushApproval":        "false",
				"RequiredReviewThreadResolution": "false",
			},
		},
		{
			name: "simple parameters with empty slices",
			params: data.Parameters{
				RequiredApprovingReviewCount: 1,
			},
			ruleType: "pull_request",
			want: map[string]string{
				"RequiredApprovingReviewCount":   "1",
				"DismissStaleReviewsOnPush":      "false",
				"RequireCodeOwnerReview":         "false",
				"RequireLastPushApproval":        "false",
				"RequiredReviewThreadResolution": "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For workflows test, we need a properly mocked APIGetter
			if tt.ruleType == "workflows" {
				t.Skip("Skipping workflows test - requires full mock setup")
			}

			got := g.ParametersToMap(tt.params, tt.ruleType)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParametersToMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParametersToMap_WithWorkflows(t *testing.T) {
	t.Run("workflows parameters", func(t *testing.T) {
		// Create a custom implementation that overrides GetRepoByID
		customGetter := &struct {
			*APIGetter
		}{
			APIGetter: &APIGetter{},
		}

		// We can't easily test this without mocking the REST client
		// So we'll just verify the function doesn't panic with nil workflows
		emptyParams := data.Parameters{
			Workflows: []data.Workflows{},
		}

		got := customGetter.ParametersToMap(emptyParams, "workflows")
		if got == nil {
			t.Errorf("ParametersToMap() returned nil, expected empty map")
		}

		// Skip the actual workflow test since it requires REST client mock
		t.Log("Skipping full workflow test - would require REST client mock")
	})
}

func TestParseParameters(t *testing.T) {
	tests := []struct {
		name     string
		paramStr string
		want     map[string]interface{}
	}{
		{
			name:     "empty string",
			paramStr: "",
			want:     nil,
		},
		{
			name:     "simple key-value pair",
			paramStr: "RequiredApprovingReviewCount:2",
			want: map[string]interface{}{
				"RequiredApprovingReviewCount": "2",
			},
		},
		{
			name:     "array value",
			paramStr: "RestrictedFilePaths:[file1.txt file2.txt]",
			want: map[string]interface{}{
				"RestrictedFilePaths": []string{"file1.txt", "file2.txt"},
			},
		},
		{
			name:     "complex object",
			paramStr: "RequiredStatusChecks:{Context=test|IntegrationID=123}",
			want: map[string]interface{}{
				"RequiredStatusChecks": []map[string]string{
					{"Context": "test", "IntegrationID": "123"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseParameters(tt.paramStr)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseParameters() = %v, want %v", got, tt.want)
			}
		})
	}
}
