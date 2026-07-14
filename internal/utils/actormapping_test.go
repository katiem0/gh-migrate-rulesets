package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "actors.csv")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write temp csv: %v", err)
	}
	return path
}

func TestLoadActorMapping(t *testing.T) {
	t.Run("empty path returns empty map", func(t *testing.T) {
		got, err := LoadActorMapping("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty map, got %d entries", len(got))
		}
	})

	t.Run("parses rows and ignores extra columns", func(t *testing.T) {
		path := writeTempCSV(t, "actor_type,source_id,source_name,target_id\nRepositoryRole,4,Write,9\nTeam,100,platform,200\n")
		got, err := LoadActorMapping(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["repositoryrole:4"] != 9 {
			t.Errorf("repositoryrole:4 = %d, want 9", got["repositoryrole:4"])
		}
		if got["team:100"] != 200 {
			t.Errorf("team:100 = %d, want 200", got["team:100"])
		}
	})

	t.Run("actor_type is matched case-insensitively", func(t *testing.T) {
		path := writeTempCSV(t, "actor_type,source_id,target_id\nrepositoryrole,4,9\n")
		got, err := LoadActorMapping(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got[actorMappingKey("RepositoryRole", 4)] != 9 {
			t.Errorf("expected case-insensitive key match, got %v", got)
		}
	})

	t.Run("blank target rows are skipped", func(t *testing.T) {
		path := writeTempCSV(t, "actor_type,source_id,target_id\nRepositoryRole,5,\n")
		got, err := LoadActorMapping(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected blank-target row skipped, got %d entries", len(got))
		}
	})

	t.Run("missing headers errors", func(t *testing.T) {
		path := writeTempCSV(t, "actor_type,source_id\nRepositoryRole,4\n")
		if _, err := LoadActorMapping(path); err == nil {
			t.Error("expected error for missing target_id header")
		}
	})

	t.Run("non-numeric ids error", func(t *testing.T) {
		path := writeTempCSV(t, "actor_type,source_id,target_id\nRepositoryRole,four,9\n")
		if _, err := LoadActorMapping(path); err == nil {
			t.Error("expected error for non-numeric source_id")
		}
	})

	t.Run("duplicate keys error", func(t *testing.T) {
		path := writeTempCSV(t, "actor_type,source_id,target_id\nRepositoryRole,4,9\nRepositoryRole,4,10\n")
		if _, err := LoadActorMapping(path); err == nil {
			t.Error("expected error for duplicate actor mapping")
		}
	})
}
