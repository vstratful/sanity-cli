package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vstratful/sanity-cli/internal/config"
)

var (
	addProject     string
	addDataset     string
	addToken       string
	addAPIVersion  string
	addCDN         bool
	addPerspective string
	addSetCurrent  bool
)

var instanceAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a named instance to the config",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstanceAdd,
}

func init() {
	instanceAddCmd.Flags().StringVar(&addProject, "project", "", "Sanity project ID (required)")
	instanceAddCmd.Flags().StringVar(&addDataset, "dataset", "", "Dataset name (required)")
	instanceAddCmd.Flags().StringVar(&addToken, "token", "", "API token (required)")
	instanceAddCmd.Flags().StringVar(&addAPIVersion, "api-version", config.DefaultAPIVersion, "API version (e.g. 2024-10-01)")
	instanceAddCmd.Flags().BoolVar(&addCDN, "cdn", false, "Use the apicdn.sanity.io host for reads")
	instanceAddCmd.Flags().StringVar(&addPerspective, "perspective", config.DefaultPerspective, "Default perspective (published|drafts|previewDrafts)")
	instanceAddCmd.Flags().BoolVar(&addSetCurrent, "current", false, "Mark this instance as current after adding")
	_ = instanceAddCmd.MarkFlagRequired("project")
	_ = instanceAddCmd.MarkFlagRequired("dataset")
	_ = instanceAddCmd.MarkFlagRequired("token")
	instanceCmd.AddCommand(instanceAddCmd)
}

func runInstanceAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	cfg, err := config.Load()
	if err != nil {
		return emitError("config_load_failed", err.Error(), nil)
	}
	if _, exists := cfg.Instances[name]; exists {
		return emitError("instance_exists", "instance already exists", map[string]string{"name": name})
	}
	inst := &config.Instance{
		ProjectID:   addProject,
		Dataset:     addDataset,
		Token:       addToken,
		APIVersion:  addAPIVersion,
		UseCDN:      addCDN,
		Perspective: addPerspective,
	}
	if err := inst.Validate(); err != nil {
		return emitError("invalid_instance", err.Error(), nil)
	}
	cfg.Instances[name] = inst
	if cfg.DefaultProjectID == "" {
		cfg.DefaultProjectID = inst.ProjectID
	}
	if addSetCurrent || cfg.CurrentInstance == "" {
		cfg.CurrentInstance = name
	}
	if err := config.Save(cfg); err != nil {
		return emitError("config_save_failed", err.Error(), nil)
	}
	return emitSuccess(map[string]interface{}{
		"name":        name,
		"project_id":  inst.ProjectID,
		"dataset":     inst.Dataset,
		"api_version": inst.EffectiveAPIVersion(),
		"perspective": inst.EffectivePerspective(),
		"use_cdn":     inst.UseCDN,
		"current":     cfg.CurrentInstance == name,
	})
}
