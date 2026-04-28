package picker

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/vstratful/sanity-cli/internal/config"
)

// InstanceItem wraps a named Instance for display in a picker.
type InstanceItem struct {
	Name     string
	Instance *config.Instance
	Current  bool
}

func (i InstanceItem) Title() string {
	if i.Current {
		return i.Name + " (current)"
	}
	return i.Name
}

func (i InstanceItem) Description() string {
	return fmt.Sprintf("project=%s dataset=%s perspective=%s",
		i.Instance.ProjectID, i.Instance.Dataset, i.Instance.EffectivePerspective())
}

func (i InstanceItem) FilterValue() string {
	return i.Name + " " + i.Instance.ProjectID + " " + i.Instance.Dataset
}

// BuildInstanceItems converts a config to a sorted slice of list items.
func BuildInstanceItems(cfg *config.Config) []list.Item {
	items := make([]list.Item, 0, len(cfg.Instances))
	for name, inst := range cfg.Instances {
		items = append(items, InstanceItem{
			Name:     name,
			Instance: inst,
			Current:  name == cfg.CurrentInstance,
		})
	}
	return items
}
