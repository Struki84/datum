package menu

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/struki84/datum/clipt/tui/schema"
)

type MenuDelegate struct {
	Style schema.LayoutStyle
}

func NewMenuDelegate(style schema.LayoutStyle) MenuDelegate {
	return MenuDelegate{
		Style: style,
	}
}

func (delegate MenuDelegate) Height() int                             { return 1 }
func (delegate MenuDelegate) Spacing() int                            { return 0 }
func (delegate MenuDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (delegate MenuDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(schema.CmdItem)
	if !ok {
		return
	}

	titleStyle := delegate.Style.Menu.ItemNormal

	if index == m.Index() {
		titleStyle = delegate.Style.Menu.ItemSelected
	}

	title := titleStyle.Render(i.Title())
	desc := delegate.Style.Menu.Description.Render(i.Description())

	fmt.Fprint(w, title+desc)
}
