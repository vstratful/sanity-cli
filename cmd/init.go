package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vstratful/sanity-cli/internal/config"
)

var (
	initName string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Configure the first sanity-cli instance interactively",
	Long: `Run an interactive wizard to capture project ID, dataset, token, and API
version. Saves the result as a named instance and marks it as current.

Example:
  sanity-cli init
  sanity-cli init --name staging`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initName, "name", config.DefaultInstanceName, "Name to use for this instance")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return emitError("config_load_failed", err.Error(), nil)
	}

	if _, exists := cfg.Instances[initName]; exists {
		return emitError("instance_exists",
			"an instance with that name already exists; use `instance remove` first or pick another --name",
			map[string]string{"name": initName})
	}

	inst, err := config.PromptForInstance(initName)
	if err != nil {
		return emitError("prompt_failed", err.Error(), nil)
	}

	cfg.Instances[initName] = inst
	cfg.CurrentInstance = initName
	if cfg.DefaultProjectID == "" {
		cfg.DefaultProjectID = inst.ProjectID
	}
	if err := config.Save(cfg); err != nil {
		return emitError("config_save_failed", err.Error(), nil)
	}

	configPath, _ := config.GetConfigPath()
	return emitSuccess(map[string]interface{}{
		"name":        initName,
		"project_id":  inst.ProjectID,
		"dataset":     inst.Dataset,
		"api_version": inst.EffectiveAPIVersion(),
		"current":     true,
		"config_path": configPath,
	})
}
