package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vstratful/sanity-cli/internal/config"
)

// version is injected at build time by GoReleaser.
var version = "dev"

// Persistent root flags.
var (
	timeout         time.Duration
	pretty          bool
	instanceFlag    string
	apiVersionFlag  string
	perspectiveFlag string
)

var rootCmd = &cobra.Command{
	Use:   "sanity-cli",
	Short: "Agent-friendly CLI for interacting with a Sanity.io project",
	Long: `sanity-cli lets a coding agent interact with a Sanity.io instance.

It captures credentials per named instance (project + dataset + token + apiVersion),
supports multiple instances with a switcher, runs GROQ reads and mutations,
introspects the live dataset to produce a schema document, and uploads assets.

Output is JSON-first by default; pass --pretty for indented JSON.

Examples (agent/script-friendly — flag-based):
  sanity-cli instance add prod --project <id> --dataset <name> --token <sk...> --current
  sanity-cli instance list --pretty
  sanity-cli query '*[_type=="post"][0..2]{_id,title}' --pretty
  sanity-cli schema introspect --pretty
  sanity-cli mutate ./mutations.json --confirm

Run 'sanity-cli agent-setup' for a structured guide intended for AI agents.

Interactive alternative for humans: 'sanity-cli init' (prompts on stdin).`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.Version = version
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 2*time.Minute, "HTTP timeout (e.g. 30s, 2m)")
	rootCmd.PersistentFlags().BoolVar(&pretty, "pretty", false, "Indent JSON output")
	rootCmd.PersistentFlags().StringVar(&instanceFlag, "instance", "", "Use a specific named instance from config")
	rootCmd.PersistentFlags().StringVar(&apiVersionFlag, "api-version", "", "Override instance api_version (e.g. 2024-10-01)")
	rootCmd.PersistentFlags().StringVar(&perspectiveFlag, "perspective", "", "Override instance perspective (published|drafts|previewDrafts)")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// resolveInstance picks the active instance using this precedence:
//  1. SANITY_PROJECT_ID + SANITY_DATASET + SANITY_TOKEN env trio (ephemeral, not saved)
//  2. --instance flag
//  3. SANITY_CLI_INSTANCE env var
//  4. config.current_instance
//  5. exactly one configured instance
//  6. interactive first-run prompt (no instances configured)
//
// Returns the resolved instance, the loaded config (nil for env-ephemeral),
// the source string (for diagnostics), and an error.
func resolveInstance() (*config.Instance, *config.Config, string, error) {
	if env, ok := envInstance(); ok {
		return env, nil, "env-ephemeral", nil
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to load config: %w", err)
	}

	if instanceFlag != "" {
		inst, ok := cfg.Instances[instanceFlag]
		if !ok {
			return nil, cfg, "", fmt.Errorf("instance %q not found in config", instanceFlag)
		}
		return applyOverrides(inst), cfg, "flag", nil
	}

	if name := os.Getenv("SANITY_CLI_INSTANCE"); name != "" {
		inst, ok := cfg.Instances[name]
		if !ok {
			return nil, cfg, "", fmt.Errorf("instance %q (from SANITY_CLI_INSTANCE) not found in config", name)
		}
		return applyOverrides(inst), cfg, "env", nil
	}

	if cfg.CurrentInstance != "" {
		if inst, ok := cfg.Instances[cfg.CurrentInstance]; ok {
			return applyOverrides(inst), cfg, "current", nil
		}
	}

	if len(cfg.Instances) == 1 {
		for _, inst := range cfg.Instances {
			return applyOverrides(inst), cfg, "only", nil
		}
	}

	if len(cfg.Instances) == 0 {
		inst, err := config.PromptForInstance(config.DefaultInstanceName)
		if err != nil {
			return nil, cfg, "", err
		}
		cfg.Instances[config.DefaultInstanceName] = inst
		cfg.CurrentInstance = config.DefaultInstanceName
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
		}
		return applyOverrides(inst), cfg, "first-run", nil
	}

	return nil, cfg, "", fmt.Errorf("multiple instances configured; pass --instance, set SANITY_CLI_INSTANCE, or run `sanity-cli instance switch`")
}

// envInstance returns an ephemeral Instance built from environment variables
// when the required trio is present.
func envInstance() (*config.Instance, bool) {
	project := os.Getenv("SANITY_PROJECT_ID")
	dataset := os.Getenv("SANITY_DATASET")
	token := os.Getenv("SANITY_TOKEN")
	if project == "" || dataset == "" || token == "" {
		return nil, false
	}
	apiVersion := os.Getenv("SANITY_API_VERSION")
	if apiVersion == "" {
		apiVersion = config.DefaultAPIVersion
	}
	useCDN := false
	if v := strings.ToLower(os.Getenv("SANITY_USE_CDN")); v == "true" || v == "1" {
		useCDN = true
	}
	perspective := os.Getenv("SANITY_PERSPECTIVE")
	if perspective == "" {
		perspective = config.DefaultPerspective
	}
	inst := &config.Instance{
		ProjectID:   project,
		Dataset:     dataset,
		Token:       token,
		APIVersion:  apiVersion,
		UseCDN:      useCDN,
		Perspective: perspective,
	}
	return applyOverrides(inst), true
}

// applyOverrides folds in --api-version and --perspective onto the instance.
// Returns a copy so we don't mutate the value owned by the config map.
func applyOverrides(in *config.Instance) *config.Instance {
	cp := *in
	if apiVersionFlag != "" {
		cp.APIVersion = apiVersionFlag
	}
	if perspectiveFlag != "" {
		cp.Perspective = perspectiveFlag
	}
	return &cp
}
