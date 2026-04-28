package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vstratful/sanity-cli/internal/config"
)

func withTempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig := config.GetConfigDir
	config.GetConfigDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { config.GetConfigDir = orig })
	return dir
}

func TestCachePathStructure(t *testing.T) {
	dir := withTempConfigDir(t)
	got, err := CachePath("abc", "production")
	if err != nil {
		t.Fatalf("CachePath: %v", err)
	}
	want := filepath.Join(dir, "schemas", "abc-production.json")
	if got != want {
		t.Errorf("CachePath=%q, want %q", got, want)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	withTempConfigDir(t)
	doc := &Doc{
		ProjectID:      "abc",
		Dataset:        "production",
		APIVersion:     "2024-10-01",
		Perspective:    "published",
		IntrospectedAt: "2026-04-28T00:00:00Z",
		SampleSize:     10,
		MaxDepth:       6,
		Types: map[string]*TypeInfo{
			"post": {
				Count:      42,
				SampleSize: 10,
				Fields: map[string]*Field{
					"title": {Type: "string"},
				},
			},
		},
	}
	path, err := Save(doc)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("perm=%o, want 0600", perm)
	}

	got, _, err := Load("abc", "production")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.SampleSize != 10 || got.Types["post"].Count != 42 {
		t.Errorf("round trip lost data: %+v", got)
	}
}
