package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vstratful/sanity-cli/internal/config"
)

var instanceRemoveYes bool

var instanceRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a named instance",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstanceRemove,
}

func init() {
	instanceRemoveCmd.Flags().BoolVar(&instanceRemoveYes, "yes", false, "Skip confirmation")
	instanceCmd.AddCommand(instanceRemoveCmd)
}

func runInstanceRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	if !instanceRemoveYes {
		return emitError("confirmation_required",
			"refusing to remove instance without --yes",
			map[string]string{"name": name})
	}
	cfg, err := config.Load()
	if err != nil {
		return emitError("config_load_failed", err.Error(), nil)
	}
	if _, ok := cfg.Instances[name]; !ok {
		return emitError("instance_not_found", "no such instance", map[string]string{"name": name})
	}
	delete(cfg.Instances, name)
	if cfg.CurrentInstance == name {
		cfg.CurrentInstance = ""
		for n := range cfg.Instances {
			cfg.CurrentInstance = n
			break
		}
	}
	if err := config.Save(cfg); err != nil {
		return emitError("config_save_failed", err.Error(), nil)
	}
	return emitSuccess(map[string]interface{}{
		"removed":          name,
		"current_instance": cfg.CurrentInstance,
	})
}
