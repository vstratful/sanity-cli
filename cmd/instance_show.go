package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vstratful/sanity-cli/internal/config"
)

var instanceShowToken bool

var instanceShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show instance details (token redacted by default)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInstanceShow,
}

func init() {
	instanceShowCmd.Flags().BoolVar(&instanceShowToken, "show-token", false, "Print the full token (use with care)")
	instanceCmd.AddCommand(instanceShowCmd)
}

func runInstanceShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return emitError("config_load_failed", err.Error(), nil)
	}
	var name string
	if len(args) == 1 {
		name = args[0]
	} else {
		name = cfg.CurrentInstance
		if name == "" {
			return emitError("no_current_instance",
				"no instance specified and no current_instance set", nil)
		}
	}
	inst, ok := cfg.Instances[name]
	if !ok {
		return emitError("instance_not_found", "no such instance", map[string]string{"name": name})
	}
	token := config.RedactToken(inst.Token)
	if instanceShowToken {
		token = inst.Token
	}
	return emitSuccess(map[string]interface{}{
		"name":        name,
		"project_id":  inst.ProjectID,
		"dataset":     inst.Dataset,
		"api_version": inst.EffectiveAPIVersion(),
		"perspective": inst.EffectivePerspective(),
		"use_cdn":     inst.UseCDN,
		"token":       token,
		"current":     cfg.CurrentInstance == name,
	})
}
