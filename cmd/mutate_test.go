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
		{"single mutation object", `{"create":{"_type":"note"}}`},
		{"empty", ``},
		{"empty array", `[]`},
		{"non-object element", `["create"]`},
		{"no recognised key", `[{"upsert":{"_type":"note"}}]`},
		{"two recognised keys", `[{"create":{},"patch":{}}]`},
		{"unwrapped object with no mutations key", `{"foo":"bar"}`},
		{"not json", `not json at all`},
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

func TestReadMutationsAcceptsWrappedShape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.json")
	body := `{
	  "mutations": [
	    {"patch": {"id": "x", "set": {"title": "B"}}},
	    {"create": {"_type": "note", "title": "A"}}
	  ]
	}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	muts, err := readMutations(path)
	if err != nil {
		t.Fatalf("readMutations: %v", err)
	}
	if len(muts) != 2 {
		t.Errorf("len=%d, want 2", len(muts))
	}
}

func TestReadMutationsErrorMessages(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		mustMatch string
	}{
		{"single object hint", `{"patch":{"id":"x","set":{"k":"v"}}}`, "single mutation object"},
		{"plain object hint", `{"foo":"bar"}`, "expected a JSON array"},
		{"not json", `garbage`, "not valid JSON"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "m.json")
			if err := os.WriteFile(path, []byte(c.body), 0o600); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
			_, err := readMutations(path)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", c.mustMatch)
			}
			if !contains(err.Error(), c.mustMatch) {
				t.Errorf("error %q does not contain %q", err.Error(), c.mustMatch)
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0))
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
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
