package utils

import (
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
)

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func SplitIgnoringBraces(s, delimiter string) []string {
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

func WriteErrorRulesetsToCSV(errorRulesets []data.ErrorRulesets, fileName string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"Source", "RulesetName", "Error"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}
	for _, errorRuleset := range errorRulesets {
		record := []string{errorRuleset.Source, errorRuleset.RulesetName, errorRuleset.Error}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}
	return nil
}
