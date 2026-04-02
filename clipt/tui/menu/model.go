package menu

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/struki84/datum/clipt/tui/schema"
)

type ChatMenu struct {
	WindowSize tea.WindowSizeMsg
	Style      schema.LayoutStyle

	List          *list.Model
	DefaultItems  []list.Item
	CurrentItems  []list.Item
	FilteredItems []list.Item
	SearchString  string

	Active bool
}

func New(cmds []list.Item, style schema.LayoutStyle) ChatMenu {
	list := list.New(cmds, NewMenuDelegate(style), 0, 0)
	list.SetShowTitle(false)
	list.SetShowHelp(false)
	list.SetShowPagination(false)
	list.SetShowFilter(false)
	list.SetShowStatusBar(false)
	list.SetFilteringEnabled(false)
	list.KeyMap.CursorDown = key.NewBinding(key.WithKeys("ctrl+j"))
	list.KeyMap.CursorUp = key.NewBinding(key.WithKeys("ctrl+k"))

	return ChatMenu{
		List:          &list,
		DefaultItems:  cmds,
		CurrentItems:  cmds,
		FilteredItems: cmds,
		Style:         style,
	}
}

func (menu ChatMenu) Init() tea.Cmd {
	return nil
}

func (menu ChatMenu) View() string {
	menuHeight := len(menu.FilteredItems)
	if menuHeight == 0 {
		menuHeight = 1
	}

	if menuHeight > 10 {
		menuHeight = 10
	}

	menu.List.SetSize(menu.WindowSize.Width-4, menuHeight)

	list := menu.Style.Menu.ContentView.
		Height(menuHeight).
		Width(menu.WindowSize.Width - 6).
		Render(menu.List.View())

	return lipgloss.PlaceHorizontal(
		menu.WindowSize.Width,
		lipgloss.Center,
		list,
		lipgloss.WithWhitespaceBackground(lipgloss.Color(menu.Style.WhitespaceBGcolor)),
	)
}

func (menu ChatMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		menu.WindowSize = msg
	}

	if menu.Active {
		menu.FilteredItems = []list.Item{}

		for _, item := range menu.CurrentItems {
			if strings.Contains(strings.ToLower(item.FilterValue()), strings.ToLower(menu.SearchString)) {
				menu.FilteredItems = append(menu.FilteredItems, item)
			}
		}

		menu.List.SetItems(menu.FilteredItems)
	} else {
		menu = menu.Reset()
	}

	list, cmd := menu.List.Update(msg)
	menu.List = &list
	cmds = append(cmds, cmd)

	return menu, tea.Batch(cmds...)
}

func (menu ChatMenu) PushMenu(submenu []list.Item) ChatMenu {
	menu.FilteredItems = submenu
	menu.CurrentItems = submenu
	menu.List.SetItems(submenu)

	return menu
}

func (menu ChatMenu) Reset() ChatMenu {
	menu.SearchString = ""
	menu.CurrentItems = menu.DefaultItems
	menu.FilteredItems = menu.DefaultItems
	menu.List.SetItems(menu.DefaultItems)

	return menu
}

func (menu ChatMenu) Close() ChatMenu {
	menu.Active = false
	menu = menu.Reset()

	return menu
}
