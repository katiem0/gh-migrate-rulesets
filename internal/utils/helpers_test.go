package utils

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
)

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		item  string
		want  bool
	}{
		{
			name:  "item exists",
			slice: []string{"apple", "banana", "orange"},
			item:  "banana",
			want:  true,
		},
		{
			name:  "item does not exist",
			slice: []string{"apple", "banana", "orange"},
			item:  "grape",
			want:  false,
		},
		{
			name:  "empty slice",
			slice: []string{},
			item:  "test",
			want:  false,
		},
		{
			name:  "nil slice",
			slice: nil,
			item:  "test",
			want:  false,
		},
		{
			name:  "empty item in non-empty slice",
			slice: []string{"", "test"},
			item:  "",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Contains(tt.slice, tt.item); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitIgnoringBraces(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		delimiter string
		want      []string
	}{
		{
			name:      "simple split",
			s:         "a|b|c",
			delimiter: "|",
			want:      []string{"a", "b", "c"},
		},
		{
			name:      "split ignoring braces",
			s:         "a|{b|c}|d",
			delimiter: "|",
			want:      []string{"a", "{b|c}", "d"},
		},
		{
			name:      "nested braces",
			s:         "a|{b{c|d}e}|f",
			delimiter: "|",
			want:      []string{"a", "{b{c|d}e}", "f"},
		},
		{
			name:      "empty string",
			s:         "",
			delimiter: "|",
			want:      []string{""},
		},
		{
			name:      "no delimiter present",
			s:         "abcdef",
			delimiter: "|",
			want:      []string{"abcdef"},
		},
		{
			name:      "delimiter at start and end",
			s:         "|a|b|",
			delimiter: "|",
			want:      []string{"", "a", "b", ""},
		},
		{
			name:      "multiple character delimiter",
			s:         "a::b::c",
			delimiter: "::",
			want:      []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitIgnoringBraces(tt.s, tt.delimiter)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitIgnoringBraces() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateTag(t *testing.T) {
	tests := []struct {
		name  string
		field reflect.StructField
		key   string
		value string
		want  string
	}{
		{
			name: "remove omitempty",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  reflect.StructTag(`json:"test_field,omitempty"`),
			},
			key:   "json",
			value: "test_field",
			want:  `json:"test_field"`,
		},
		{
			name: "no existing tag",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  reflect.StructTag(""),
			},
			key:   "json",
			value: "TestField",
			want:  `json:"TestField"`,
		},
		{
			name: "keep other options",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  reflect.StructTag(`json:"test_field,omitempty,string"`),
			},
			key:   "json",
			value: "test_field",
			want:  `json:"test_field,string"`,
		},
		{
			name: "multiple tags present",
			field: reflect.StructField{
				Name: "TestField",
				Tag:  reflect.StructTag(`json:"test_field,omitempty" xml:"test"`),
			},
			key:   "json",
			value: "test_field",
			want:  `json:"test_field"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UpdateTag(tt.field, tt.key, tt.value)
			if string(got.Tag) != tt.want {
				t.Errorf("UpdateTag() = %v, want %v", got.Tag, tt.want)
			}
		})
	}
}

func TestWriteErrorRulesetsToCSV(t *testing.T) {
	tests := []struct {
		name          string
		errorRulesets []data.ErrorRulesets
		wantErr       bool
	}{
		{
			name: "write multiple errors",
			errorRulesets: []data.ErrorRulesets{
				{Source: "org/repo1", RulesetName: "test-ruleset", Error: "validation error"},
				{Source: "org/repo2", RulesetName: "another-ruleset", Error: "creation failed"},
			},
			wantErr: false,
		},
		{
			name: "error with repo-level source",
			errorRulesets: []data.ErrorRulesets{
				{Source: "target-org/new-repo", RulesetName: "rename-ruleset", Error: "Repository does not exist"},
			},
			wantErr: false,
		},
		{
			name:          "empty error list",
			errorRulesets: []data.ErrorRulesets{},
			wantErr:       false,
		},
		{
			name: "error with special characters",
			errorRulesets: []data.ErrorRulesets{
				{Source: "org/repo", RulesetName: "test,ruleset", Error: "error with, comma"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileName := t.TempDir() + "/test_errors.csv"

			err := WriteErrorRulesetsToCSV(tt.errorRulesets, fileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteErrorRulesetsToCSV() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if _, err := os.Stat(fileName); os.IsNotExist(err) {
					t.Errorf("WriteErrorRulesetsToCSV() did not create file")
				}

				content, err := os.ReadFile(fileName)
				if err != nil {
					t.Errorf("Failed to read created file: %v", err)
				}

				contentStr := string(content)
				if !strings.Contains(contentStr, "Source,RulesetName,Error") {
					t.Error("CSV file missing headers")
				}

				for _, errorRuleset := range tt.errorRulesets {
					if !strings.Contains(contentStr, errorRuleset.RulesetName) {
						t.Errorf("CSV file missing ruleset %s", errorRuleset.RulesetName)
					}
					if errorRuleset.RulesetName != "" && !strings.Contains(contentStr, errorRuleset.Source+",") {
						t.Errorf("CSV file missing Source %q", errorRuleset.Source)
					}
				}
			}
		})
	}
}

func TestSafeExecute(t *testing.T) {
	t.Run("returns nil when function succeeds", func(t *testing.T) {
		called := false
		err := SafeExecute(func() error {
			called = true
			return nil
		}, "success case")
		if err != nil {
			t.Errorf("SafeExecute() = %v, want nil", err)
		}
		if !called {
			t.Error("SafeExecute() did not invoke the function")
		}
	})

	t.Run("propagates the function's error", func(t *testing.T) {
		want := "boom"
		err := SafeExecute(func() error {
			return errors.New(want)
		}, "error case")
		if err == nil || err.Error() != want {
			t.Errorf("SafeExecute() = %v, want %q", err, want)
		}
	})

	t.Run("recovers from panic and returns an error", func(t *testing.T) {
		err := SafeExecute(func() error {
			panic("kaboom")
		}, "panic case")
		if err == nil {
			t.Fatal("SafeExecute() = nil, want error from recovered panic")
		}
		if !strings.Contains(err.Error(), "panic in panic case") {
			t.Errorf("SafeExecute() = %q, want it to mention the panic context", err.Error())
		}
	})
}
