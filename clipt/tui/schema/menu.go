package schema

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type CmdItem interface {
	list.Item

	Title() string
	Description() string
	Execute(tea.Model) (tea.Model, tea.Cmd)
}
