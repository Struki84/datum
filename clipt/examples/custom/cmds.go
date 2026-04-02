package main

import (
	"context"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/struki84/clipt/tui"
	"github.com/struki84/clipt/tui/menu"
	"github.com/struki84/clipt/tui/schema"
)

type ProvidersCmd struct {
	title string
	desc  string
}

func NewProvidersCmd(title, desc string) ProvidersCmd {
	return ProvidersCmd{
		title: title,
		desc:  desc,
	}
}

func (cmd ProvidersCmd) Title() string       { return cmd.title }
func (cmd ProvidersCmd) Description() string { return cmd.desc }
func (cmd ProvidersCmd) FilterValue() string { return cmd.title }
func (cmd ProvidersCmd) Execute(model tea.Model) (tea.Model, tea.Cmd) {
	layout := model.(tui.LayoutView)
	items := []list.Item{}

	for _, provider := range layout.Providers {
		items = append(items, ProviderCmd{provider: provider})
	}

	layout.Menu = layout.Menu.PushMenu(items)
	layout.Chat.Input.SetValue("/")

	return layout, nil
}

type ProviderCmd struct {
	provider schema.ChatProvider
}

func (cmd ProviderCmd) Title() string       { return "/" + cmd.provider.Name() }
func (cmd ProviderCmd) Description() string { return cmd.provider.Description() }
func (cmd ProviderCmd) FilterValue() string { return cmd.provider.Name() }
func (cmd ProviderCmd) Execute(model tea.Model) (tea.Model, tea.Cmd) {
	layout := model.(tui.LayoutView)

	layout.Chat.Provider = cmd.provider
	layout.Chat.Provider.Stream(context.TODO(), func(ctx context.Context, msg schema.Msg) error {
		layout.Chat.Stream <- msg
		return nil
	})

	layout.Menu = layout.Menu.Close()
	layout.Chat.Input.SetValue("")

	return layout, nil
}

type ListThemesCmd struct {
	title string
	desc  string
}

func NewListThemesCmd(title, desc string) ListThemesCmd {
	return ListThemesCmd{
		title: title,
		desc:  desc,
	}
}

func (cmd ListThemesCmd) Title() string       { return cmd.title }
func (cmd ListThemesCmd) Description() string { return cmd.desc }
func (cmd ListThemesCmd) FilterValue() string { return cmd.title }
func (cmd ListThemesCmd) Execute(model tea.Model) (tea.Model, tea.Cmd) {
	layout := model.(tui.LayoutView)
	items := []list.Item{
		SelectThemeCmd{title: "dark theme", desc: "Basic white theme for my chat tui.", scheme: Dark},
		SelectThemeCmd{title: "light theme", desc: "Basic dark theme for my chat tui.", scheme: Light},
	}

	layout.Menu = layout.Menu.PushMenu(items)
	layout.Chat.Input.SetValue("/")

	return layout, nil

}

type SelectThemeCmd struct {
	title  string
	desc   string
	scheme ColorScheme
}

func (cmd SelectThemeCmd) Title() string       { return "/" + cmd.title }
func (cmd SelectThemeCmd) Description() string { return cmd.desc }
func (cmd SelectThemeCmd) FilterValue() string { return cmd.title }
func (cmd SelectThemeCmd) Execute(model tea.Model) (tea.Model, tea.Cmd) {
	layout := model.(tui.LayoutView)
	newStyle := CustomStyle(cmd.scheme)

	layout.Style = newStyle
	layout.Chat.Style = newStyle

	layout.Menu = menu.New(layout.Menu.DefaultItems, newStyle)
	layout.Menu = layout.Menu.Close()

	layout.Chat.Viewport.SetContent(layout.Chat.RenderMsgs())
	layout.Chat.Input.SetValue("")

	return layout, nil
}

var CustomCmds = []list.Item{
	NewProvidersCmd("/providers", "List all available models and agents."),
	tui.NewSessionsCmd("/sessions", "List saved sessions"),
	tui.NewCreateSessionCmd("/new", "Start new session"),
	tui.NewDeleteSessionCmd("/delete", "Delete the current session"),
	tui.NewExitCmd("/exit", "Close the application."),
}
