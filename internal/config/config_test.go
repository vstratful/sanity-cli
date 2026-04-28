package config

import (
	"os"
	"path/filepath"
	"testing"
)

func withTempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig := GetConfigDir
	GetConfigDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { GetConfigDir = orig })
	return dir
}

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	withTempConfigDir(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CurrentInstance != "" {
		t.Errorf("CurrentInstance=%q, want empty", cfg.CurrentInstance)
	}
	if cfg.Instances == nil {
		t.Error("Instances should be initialized to empty map, got nil")
	}
	if len(cfg.Instances) != 0 {
		t.Errorf("Instances len=%d, want 0", len(cfg.Instances))
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := withTempConfigDir(t)
	in := &Config{
		CurrentInstance:  "prod",
		DefaultProjectID: "abc123",
		Instances: map[string]*Instance{
			"prod": {
				ProjectID:   "abc123",
				Dataset:     "production",
				Token:       "skXXXX",
				APIVersion:  "2024-10-01",
				UseCDN:      true,
				Perspective: "published",
			},
			"staging": {
				ProjectID: "abc123",
				Dataset:   "staging",
				Token:     "skYYYY",
			},
		},
	}
	if err := Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// File mode should be 0600.
	info, err := os.Stat(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config.json perm=%o, want 0600", perm)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.CurrentInstance != "prod" {
		t.Errorf("CurrentInstance=%q, want prod", got.CurrentInstance)
	}
	if got.DefaultProjectID != "abc123" {
		t.Errorf("DefaultProjectID=%q, want abc123", got.DefaultProjectID)
	}
	if len(got.Instances) != 2 {
		t.Errorf("Instances len=%d, want 2", len(got.Instances))
	}
	prod, ok := got.Instances["prod"]
	if !ok {
		t.Fatal("prod instance missing")
	}
	if prod.Token != "skXXXX" || prod.Dataset != "production" || !prod.UseCDN {
		t.Errorf("prod = %+v", prod)
	}
}

func TestInstanceValidate(t *testing.T) {
	cases := []struct {
		name    string
		inst    *Instance
		wantErr bool
	}{
		{"complete", &Instance{ProjectID: "p", Dataset: "d", Token: "t"}, false},
		{"missing project", &Instance{Dataset: "d", Token: "t"}, true},
		{"missing dataset", &Instance{ProjectID: "p", Token: "t"}, true},
		{"missing token", &Instance{ProjectID: "p", Dataset: "d"}, true},
		{"nil", nil, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.inst.Validate()
			if (err != nil) != c.wantErr {
				t.Errorf("Validate err=%v, wantErr=%v", err, c.wantErr)
			}
		})
	}
}

func TestInstanceEffectiveDefaults(t *testing.T) {
	inst := &Instance{}
	if inst.EffectiveAPIVersion() != DefaultAPIVersion {
		t.Errorf("EffectiveAPIVersion=%q, want %q", inst.EffectiveAPIVersion(), DefaultAPIVersion)
	}
	if inst.EffectivePerspective() != DefaultPerspective {
		t.Errorf("EffectivePerspective=%q, want %q", inst.EffectivePerspective(), DefaultPerspective)
	}
	inst.APIVersion = "2024-01-01"
	inst.Perspective = "drafts"
	if inst.EffectiveAPIVersion() != "2024-01-01" {
		t.Error("EffectiveAPIVersion should respect override")
	}
	if inst.EffectivePerspective() != "drafts" {
		t.Error("EffectivePerspective should respect override")
	}
}

func TestRedactToken(t *testing.T) {
	cases := map[string]string{
		"":                "",
		"abc":             "****",
		"abcdefgh":        "****",
		"sk1234567890":    "sk12...7890",
		"sk-very-long-tk": "sk-v...g-tk",
	}
	for in, want := range cases {
		if got := RedactToken(in); got != want {
			t.Errorf("RedactToken(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestGetSchemasDir(t *testing.T) {
	dir := withTempConfigDir(t)
	got, err := GetSchemasDir()
	if err != nil {
		t.Fatalf("GetSchemasDir: %v", err)
	}
	want := filepath.Join(dir, "schemas")
	if got != want {
		t.Errorf("GetSchemasDir=%q, want %q", got, want)
	}
}
