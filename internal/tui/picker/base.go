// Package picker provides reusable list picker components.
package picker

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vstratful/sanity-cli/internal/tui"
)

// Item is the interface for items that can be displayed in a picker.
type Item interface {
	list.Item
	Title() string
	Description() string
}

// ItemDelegate renders items in the picker list.
type ItemDelegate struct{}

func (d ItemDelegate) Height() int                             { return 2 }
func (d ItemDelegate) Spacing() int                            { return 1 }
func (d ItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d ItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(Item)
	if !ok {
		return
	}

	title := i.Title()
	desc := i.Description()

	if index == m.Index() {
		title = tui.SelectedItemStyle.Render("> " + title)
		desc = tui.SelectedItemStyle.Render("  " + desc)
	} else {
		title = tui.ItemStyle.Render(title)
		desc = tui.ItemStyle.Render(desc)
	}

	fmt.Fprintf(w, "%s\n%s", title, desc)
}

// Model is the Bubble Tea model for a generic picker.
type Model struct {
	List     list.Model
	Err      error
	Width    int
	Height   int
	Quitting bool
	Chosen   bool
}

// Config holds configuration for creating a new picker.
type Config struct {
	Title  string
	Items  []list.Item
	Width  int
	Height int
}

// New creates a new picker Model.
func New(cfg Config) Model {
	height := cfg.Height
	if height < 6 {
		height = 12
	}
	width := cfg.Width
	if width < 20 {
		width = 60
	}
	l := list.New(cfg.Items, ItemDelegate{}, width, height-2)
	l.Title = cfg.Title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = tui.TitleStyle
	l.Styles.PaginationStyle = tui.PaginationStyle
	l.Styles.HelpStyle = tui.HelpListStyle

	return Model{
		List:   l,
		Width:  width,
		Height: height,
	}
}

// Init initializes the picker.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the picker.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.List.SetWidth(msg.Width)
		m.List.SetHeight(msg.Height - 2)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.Quitting = true
			return m, tea.Quit
		case "enter":
			m.Chosen = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.List, cmd = m.List.Update(msg)
	return m, cmd
}

// View renders the picker.
func (m Model) View() string {
	if m.Quitting && !m.Chosen {
		return ""
	}
	if m.Err != nil {
		return tui.ErrorStyle.Render(fmt.Sprintf("\n   Error: %s\n", m.Err.Error()))
	}
	return m.List.View()
}

// SelectedItem returns the currently selected item.
func (m Model) SelectedItem() list.Item {
	return m.List.SelectedItem()
}
