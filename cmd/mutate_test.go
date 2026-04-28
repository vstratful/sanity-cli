package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadMutationsValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.json")
	const body = `[
	  {"create": {"_type":"note","title":"a"}},
	  {"patch": {"id":"x","set":{"title":"b"}}},
	  {"delete": {"id":"y"}}
	]`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	muts, err := readMutations(path)
	if err != nil {
		t.Fatalf("readMutations: %v", err)
	}
	if len(muts) != 3 {
		t.Errorf("len=%d, want 3", len(muts))
	}

	counts, first := summarizeMutations(muts)
	if counts["create"] != 1 || counts["patch"] != 1 || counts["delete"] != 1 {
		t.Errorf("counts=%v, want one each", counts)
	}
	if first == nil {
		t.Error("first nil, want first mutation")
	}
}

func TestReadMutationsRejectsBadInput(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"not array", `{"create":{"_type":"note"}}`},
		{"empty", ``},
		{"empty array", `[]`},
		{"non-object element", `["create"]`},
		{"no recognised key", `[{"upsert":{"_type":"note"}}]`},
		{"two recognised keys", `[{"create":{},"patch":{}}]`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "m.json")
			if err := os.WriteFile(path, []byte(c.body), 0o600); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
			if _, err := readMutations(path); err == nil {
				t.Errorf("expected error for %q", c.name)
			}
		})
	}
}

func TestReadMutationsAllValidKinds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.json")
	body := `[
	  {"create":{"_type":"note"}},
	  {"createOrReplace":{"_type":"note","_id":"a"}},
	  {"createIfNotExists":{"_type":"note","_id":"b"}},
	  {"patch":{"id":"c","set":{"x":1}}},
	  {"delete":{"id":"d"}}
	]`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := readMutations(path); err != nil {
		t.Errorf("readMutations: %v", err)
	}
}

func TestEnvTrue(t *testing.T) {
	cases := map[string]bool{
		"":      false,
		"0":     false,
		"false": false,
		"FALSE": false,
		"no":    false,
		"NO":    false,
		"1":     true,
		"true":  true,
		"TRUE":  true,
		"yes":   true,
		"x":     true, // anything else non-empty is truthy
	}
	const key = "SANITY_CLI_TEST_ENVTRUE"
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			t.Setenv(key, in)
			if got := envTrue(key); got != want {
				t.Errorf("envTrue(%q)=%v, want %v", in, got, want)
			}
		})
	}
}
