package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func writeRepoMappingCSV(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mapping.csv")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return path
}

func TestLoadRepoMapping(t *testing.T) {
	tests := []struct {
		name    string
		path    func(t *testing.T) string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "valid headered csv",
			path: func(t *testing.T) string {
				return writeRepoMappingCSV(t, "source,target\nold,new\nlegacy,modern\n")
			},
			want: map[string]string{"old": "new", "legacy": "modern"},
		},
		{
			name: "header order and case insensitive",
			path: func(t *testing.T) string {
				return writeRepoMappingCSV(t, "TARGET,SOURCE\nnew,old\n")
			},
			want: map[string]string{"old": "new"},
		},
		{
			name: "trims whitespace",
			path: func(t *testing.T) string {
				return writeRepoMappingCSV(t, " source , target \n old-repo , new-repo \n")
			},
			want: map[string]string{"old-repo": "new-repo"},
		},
		{
			name: "empty path",
			path: func(t *testing.T) string { return "" },
			want: map[string]string{},
		},
		{
			name:    "missing file",
			path:    func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing.csv") },
			wantErr: true,
		},
		{
			name: "duplicate source",
			path: func(t *testing.T) string {
				return writeRepoMappingCSV(t, "source,target\nold,new\nold,newer\n")
			},
			wantErr: true,
		},
		{
			name: "missing source header",
			path: func(t *testing.T) string {
				return writeRepoMappingCSV(t, "from,target\nold,new\n")
			},
			wantErr: true,
		},
		{
			name: "missing target header",
			path: func(t *testing.T) string {
				return writeRepoMappingCSV(t, "source,to\nold,new\n")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadRepoMapping(tt.path(t))
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadRepoMapping() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got == nil {
				t.Fatalf("LoadRepoMapping() returned nil map")
			}
			if len(got) != len(tt.want) {
				t.Errorf("LoadRepoMapping() returned %d mappings, want %d", len(got), len(tt.want))
			}
			for source, wantTarget := range tt.want {
				if got[source] != wantTarget {
					t.Errorf("LoadRepoMapping()[%q] = %q, want %q", source, got[source], wantTarget)
				}
			}
		})
	}
}

func TestResolveTargetRepo(t *testing.T) {
	mapping := map[string]string{"old": "new"}
	if got := ResolveTargetRepo(mapping, "old"); got != "new" {
		t.Errorf("ResolveTargetRepo() = %q, want %q", got, "new")
	}
	if got := ResolveTargetRepo(mapping, "unmapped"); got != "unmapped" {
		t.Errorf("ResolveTargetRepo() = %q, want %q", got, "unmapped")
	}
	if got := ResolveTargetRepo(nil, "safe"); got != "safe" {
		t.Errorf("ResolveTargetRepo() = %q, want %q", got, "safe")
	}
}
