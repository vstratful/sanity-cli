package cmd

import (
	"sort"

	"github.com/spf13/cobra"
	"github.com/vstratful/sanity-cli/internal/config"
)

var instanceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured instances as JSON",
	RunE:  runInstanceList,
}

func init() {
	instanceCmd.AddCommand(instanceListCmd)
}

type instanceSummary struct {
	Name        string `json:"name"`
	ProjectID   string `json:"project_id"`
	Dataset     string `json:"dataset"`
	APIVersion  string `json:"api_version"`
	Perspective string `json:"perspective"`
	UseCDN      bool   `json:"use_cdn"`
	TokenHint   string `json:"token_hint"`
	Current     bool   `json:"current"`
}

func runInstanceList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return emitError("config_load_failed", err.Error(), nil)
	}
	names := make([]string, 0, len(cfg.Instances))
	for name := range cfg.Instances {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]instanceSummary, 0, len(names))
	for _, name := range names {
		inst := cfg.Instances[name]
		out = append(out, instanceSummary{
			Name:        name,
			ProjectID:   inst.ProjectID,
			Dataset:     inst.Dataset,
			APIVersion:  inst.EffectiveAPIVersion(),
			Perspective: inst.EffectivePerspective(),
			UseCDN:      inst.UseCDN,
			TokenHint:   config.RedactToken(inst.Token),
			Current:     cfg.CurrentInstance == name,
		})
	}
	return emitSuccess(map[string]interface{}{
		"current_instance": cfg.CurrentInstance,
		"instances":        out,
	})
}
