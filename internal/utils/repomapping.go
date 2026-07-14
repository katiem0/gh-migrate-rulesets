package utils

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"go.uber.org/zap"
)

func LoadRepoMapping(path string) (map[string]string, error) {
	repoMapping := map[string]string{}
	if path == "" {
		return repoMapping, nil
	}

	f, err := os.Open(path)
	zap.S().Debugf("Opening up repo mapping file %s", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open repo mapping file: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			zap.S().Errorf("Error closing file: %v", err)
		}
	}()

	csvReader := csv.NewReader(f)
	csvReader.FieldsPerRecord = -1

	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read repo mapping headers: %w", err)
	}

	sourceIndex, targetIndex := -1, -1
	for i, column := range header {
		switch strings.ToLower(strings.TrimSpace(column)) {
		case "source":
			sourceIndex = i
		case "target":
			targetIndex = i
		}
	}
	if sourceIndex == -1 || targetIndex == -1 {
		return nil, fmt.Errorf("repo mapping file must contain source and target headers")
	}

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read repo mapping record: %w", err)
		}
		if isEmptyCSVRecord(record) {
			continue
		}

		sourceRepo := csvFieldValue(record, sourceIndex)
		targetRepo := csvFieldValue(record, targetIndex)
		if sourceRepo == "" || targetRepo == "" {
			return nil, fmt.Errorf("repo mapping requires non-empty source and target values")
		}
		if _, ok := repoMapping[sourceRepo]; ok {
			return nil, fmt.Errorf("duplicate source repo mapping for %s", sourceRepo)
		}
		repoMapping[sourceRepo] = targetRepo
	}

	return repoMapping, nil
}

func ResolveTargetRepo(mapping map[string]string, sourceRepo string) string {
	if targetRepo, ok := mapping[sourceRepo]; ok {
		return targetRepo
	}
	return sourceRepo
}

func csvFieldValue(record []string, index int) string {
	if index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}

func isEmptyCSVRecord(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}
