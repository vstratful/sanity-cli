// Package tui provides terminal UI components.
package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

var (
	ErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	HelpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	TitleStyle        = lipgloss.NewStyle().MarginLeft(2)
	ItemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	SelectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	PaginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	HelpListStyle     = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)
