package cmd

import (
	"testing"

	"github.com/vstratful/sanity-cli/internal/config"
)

func TestEnvInstanceFromTrio(t *testing.T) {
	t.Setenv("SANITY_PROJECT_ID", "p1")
	t.Setenv("SANITY_DATASET", "production")
	t.Setenv("SANITY_TOKEN", "skTEST")
	t.Setenv("SANITY_API_VERSION", "2024-10-01")
	t.Setenv("SANITY_USE_CDN", "true")
	t.Setenv("SANITY_PERSPECTIVE", "drafts")

	inst, ok := envInstance()
	if !ok {
		t.Fatal("envInstance ok=false, want true")
	}
	if inst.ProjectID != "p1" || inst.Dataset != "production" || inst.Token != "skTEST" {
		t.Errorf("inst=%+v", inst)
	}
	if inst.APIVersion != "2024-10-01" || !inst.UseCDN || inst.Perspective != "drafts" {
		t.Errorf("optional fields wrong: %+v", inst)
	}
}

func TestEnvInstanceMissingTrioReturnsFalse(t *testing.T) {
	t.Setenv("SANITY_PROJECT_ID", "p1")
	t.Setenv("SANITY_DATASET", "")
	t.Setenv("SANITY_TOKEN", "skTEST")
	if _, ok := envInstance(); ok {
		t.Error("envInstance ok=true with missing dataset, want false")
	}
}

func TestEnvInstanceDefaultsAPIVersionAndPerspective(t *testing.T) {
	t.Setenv("SANITY_PROJECT_ID", "p1")
	t.Setenv("SANITY_DATASET", "production")
	t.Setenv("SANITY_TOKEN", "skTEST")
	t.Setenv("SANITY_API_VERSION", "")
	t.Setenv("SANITY_PERSPECTIVE", "")
	t.Setenv("SANITY_USE_CDN", "")

	inst, ok := envInstance()
	if !ok {
		t.Fatal("envInstance ok=false")
	}
	if inst.APIVersion != config.DefaultAPIVersion {
		t.Errorf("APIVersion=%q, want default", inst.APIVersion)
	}
	if inst.Perspective != config.DefaultPerspective {
		t.Errorf("Perspective=%q, want default", inst.Perspective)
	}
	if inst.UseCDN {
		t.Error("UseCDN=true, want false (empty env value)")
	}
}

func TestApplyOverrides(t *testing.T) {
	in := &config.Instance{ProjectID: "p", Dataset: "d", Token: "t", APIVersion: "v1", Perspective: "published"}

	apiVersionFlag = ""
	perspectiveFlag = ""
	out := applyOverrides(in)
	if out.APIVersion != "v1" || out.Perspective != "published" {
		t.Errorf("no overrides should preserve values: %+v", out)
	}
	// Originals should not be mutated.
	if &out == &in {
		t.Error("applyOverrides should return a copy")
	}

	apiVersionFlag = "v2"
	perspectiveFlag = "drafts"
	t.Cleanup(func() { apiVersionFlag = ""; perspectiveFlag = "" })
	out = applyOverrides(in)
	if out.APIVersion != "v2" {
		t.Errorf("APIVersion=%q, want v2", out.APIVersion)
	}
	if out.Perspective != "drafts" {
		t.Errorf("Perspective=%q, want drafts", out.Perspective)
	}
	if in.APIVersion != "v1" {
		t.Error("input was mutated; applyOverrides must not mutate the original")
	}
}
