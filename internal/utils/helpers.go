package utils

import (
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"runtime/debug"
	"strings"

	"github.com/katiem0/gh-migrate-rulesets/internal/data"
	"go.uber.org/zap"
)

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// csvFieldValue safely returns the trimmed value at index from a CSV record,
// returning an empty string when the index is out of range.
func csvFieldValue(record []string, index int) string {
	if index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}

// isEmptyCSVRecord reports whether every field in a CSV record is blank.
func isEmptyCSVRecord(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func SplitIgnoringBraces(s, delimiter string) []string {
	var result []string
	var currentSegment strings.Builder
	inBraces := false

	for i := 0; i < len(s); i++ {
		char := s[i]

		switch char {
		case '{':
			inBraces = true
		case '}':
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
	defer func() {
		if err := file.Close(); err != nil {
			zap.S().Errorf("Error closing file: %v", err)
		}
	}()

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

func SafeExecute(fn func() error, context string) error {
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				zap.S().Errorf("Panic recovered in %s: %v\nStack trace:\n%s",
					context, r, string(debug.Stack()))
				// Convert panic to error so caller knows something went wrong
				err = fmt.Errorf("panic in %s: %v", context, r)
			}
		}()
		err = fn()
	}()
	return err
}
