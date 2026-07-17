package utils

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

// actorMappingKey builds the lookup key for a bypass actor mapping entry. The actor
// type is lowercased so CSV entries match regardless of casing.
func actorMappingKey(actorType string, sourceID int) string {
	return fmt.Sprintf("%s:%d", strings.ToLower(actorType), sourceID)
}

// resolveMappedActorID returns the target actor ID for a source actor if an explicit mapping exists.
func resolveMappedActorID(actorMapping map[string]int, actorType string, sourceID int) (int, bool) {
	targetID, ok := actorMapping[actorMappingKey(actorType, sourceID)]
	return targetID, ok
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

// LoadActorMapping reads a bypass actor mapping CSV keyed by actor_type and source_id.
// It is used to supply target IDs the API cannot resolve by name (e.g. base repository
// roles) or to override name-based resolution when an actor was renamed in the target.
func LoadActorMapping(path string) (map[string]int, error) {
	actorMapping := map[string]int{}
	if path == "" {
		return actorMapping, nil
	}

	f, err := os.Open(path)
	zap.S().Debugf("Opening up actor mapping file %s", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open actor mapping file: %w", err)
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
		return nil, fmt.Errorf("failed to read actor mapping headers: %w", err)
	}

	typeIndex, sourceIndex, targetIndex := -1, -1, -1
	for i, column := range header {
		switch strings.ToLower(strings.TrimSpace(column)) {
		case "actor_type":
			typeIndex = i
		case "source_id":
			sourceIndex = i
		case "target_id":
			targetIndex = i
		}
	}
	if typeIndex == -1 || sourceIndex == -1 || targetIndex == -1 {
		return nil, fmt.Errorf("actor mapping file must contain actor_type, source_id, and target_id headers")
	}

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read actor mapping record: %w", err)
		}
		if isEmptyCSVRecord(record) {
			continue
		}

		actorType := csvFieldValue(record, typeIndex)
		sourceValue := csvFieldValue(record, sourceIndex)
		targetValue := csvFieldValue(record, targetIndex)
		// Blank targets fall back to name-based resolution during create.
		if targetValue == "" {
			zap.S().Debugf("Skipping actor mapping row for %s source_id %s: blank target_id (will auto-resolve)", actorType, sourceValue)
			continue
		}
		if actorType == "" || sourceValue == "" {
			return nil, fmt.Errorf("actor mapping requires non-empty actor_type and source_id values")
		}
		sourceID, err := strconv.Atoi(sourceValue)
		if err != nil {
			return nil, fmt.Errorf("invalid source_id %q in actor mapping: %w", sourceValue, err)
		}
		targetID, err := strconv.Atoi(targetValue)
		if err != nil {
			return nil, fmt.Errorf("invalid target_id %q in actor mapping: %w", targetValue, err)
		}

		key := actorMappingKey(actorType, sourceID)
		if _, ok := actorMapping[key]; ok {
			return nil, fmt.Errorf("duplicate actor mapping for %s source_id %d", actorType, sourceID)
		}
		actorMapping[key] = targetID
	}

	return actorMapping, nil
}
