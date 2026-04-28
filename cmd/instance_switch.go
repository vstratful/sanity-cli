package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/vstratful/sanity-cli/internal/config"
	"github.com/vstratful/sanity-cli/internal/tui/picker"
)

var instanceSwitchCmd = &cobra.Command{
	Use:   "switch [name]",
	Short: "Set the current instance (interactive picker if no name given)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInstanceSwitch,
}

func init() {
	instanceCmd.AddCommand(instanceSwitchCmd)
}

func runInstanceSwitch(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return emitError("config_load_failed", err.Error(), nil)
	}
	if len(cfg.Instances) == 0 {
		return emitError("no_instances",
			"no instances configured; run `sanity-cli init` or `sanity-cli instance add`", nil)
	}

	var target string
	if len(args) == 1 {
		target = args[0]
		if _, ok := cfg.Instances[target]; !ok {
			return emitError("instance_not_found", "no such instance", map[string]string{"name": target})
		}
	} else {
		picked, err := pickInstance(cfg)
		if err != nil {
			return emitError("picker_failed", err.Error(), nil)
		}
		if picked == "" {
			return emitError("cancelled", "selection cancelled", nil)
		}
		target = picked
	}

	cfg.CurrentInstance = target
	if err := config.Save(cfg); err != nil {
		return emitError("config_save_failed", err.Error(), nil)
	}
	return emitSuccess(map[string]interface{}{
		"current_instance": target,
	})
}

func pickInstance(cfg *config.Config) (string, error) {
	items := picker.BuildInstanceItems(cfg)
	model := picker.New(picker.Config{
		Title:  "Select an instance",
		Items:  items,
		Width:  80,
		Height: 20,
	})
	prog := tea.NewProgram(model)
	final, err := prog.Run()
	if err != nil {
		return "", fmt.Errorf("running picker: %w", err)
	}
	m, ok := final.(picker.Model)
	if !ok {
		return "", fmt.Errorf("unexpected picker model type")
	}
	if !m.Chosen {
		return "", nil
	}
	sel := m.SelectedItem()
	if sel == nil {
		return "", nil
	}
	item, ok := sel.(picker.InstanceItem)
	if !ok {
		return "", fmt.Errorf("unexpected selected item type")
	}
	return item.Name, nil
}
