package tui

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/struki84/clipt/tui/schema"
	"github.com/thanhpk/randstr"
)

type ProvidersCmd struct {
	title  string
	desc   string
	filter schema.ProviderType
}

func NewProvidersCmd(title, desc string, filter schema.ProviderType) ProvidersCmd {

	return ProvidersCmd{
		title:  title,
		desc:   desc,
		filter: filter,
	}
}

func (cmd ProvidersCmd) Title() string       { return cmd.title }
func (cmd ProvidersCmd) Description() string { return cmd.desc }
func (cmd ProvidersCmd) FilterValue() string { return cmd.title }
func (cmd ProvidersCmd) Execute(model tea.Model) (tea.Model, tea.Cmd) {
	layout := model.(LayoutView)
	items := []list.Item{}

	for _, provider := range layout.Providers {
		if provider.Type() == cmd.filter {
			items = append(items, ProviderCmd{provider: provider})
		}
	}

	layout.Menu = layout.Menu.PushMenu(items)
	layout.Chat.Input.SetValue("/")

	return layout, nil
}

type ProviderCmd struct {
	provider schema.ChatProvider
}

func NewProviderCmd(provider schema.ChatProvider) ProviderCmd {
	return ProviderCmd{
		provider: provider,
	}
}

func (cmd ProviderCmd) Title() string       { return "/" + cmd.provider.Name() }
func (cmd ProviderCmd) Description() string { return cmd.provider.Description() }
func (cmd ProviderCmd) FilterValue() string { return cmd.provider.Name() }
func (cmd ProviderCmd) Execute(model tea.Model) (tea.Model, tea.Cmd) {
	layout := model.(LayoutView)

	layout.Chat.Provider = cmd.provider
	layout.Chat.Provider.Stream(context.TODO(), func(ctx context.Context, msg schema.Msg) error {
		layout.Chat.Stream <- msg
		return nil
	})

	layout.Menu = layout.Menu.Close()
	layout.Chat.Input.SetValue("")

	return layout, nil
}

type SessionsCmd struct {
	title string
	desc  string
}

func NewSessionsCmd(title, desc string) SessionsCmd {
	return SessionsCmd{
		title: title,
		desc:  desc,
	}
}

func (cmd SessionsCmd) Title() string       { return cmd.title }
func (cmd SessionsCmd) Description() string { return cmd.desc }
func (cmd SessionsCmd) FilterValue() string { return cmd.title }
func (cmd SessionsCmd) Execute(model tea.Model) (tea.Model, tea.Cmd) {
	layout := model.(LayoutView)
	items := []list.Item{}

	if layout.Storage != nil {
		sessions := layout.Storage.ListSessions()

		for _, session := range sessions {
			items = append(items, SessionCmd{session: session})
		}
	}

	layout.Menu = layout.Menu.PushMenu(items)
	layout.Chat.Input.SetValue("/")

	return layout, nil
}

type SessionCmd struct {
	session schema.ChatSession
}

func (cmd SessionCmd) NewSessionCmd(session schema.ChatSession) SessionCmd {
	return SessionCmd{
		session: session,
	}
}

func (cmd SessionCmd) Title() string { return "/" + cmd.session.Title }
func (cmd SessionCmd) Description() string {
	return time.Unix(cmd.session.CreatedAt, 0).Format("2 Jan 2006")
}
func (cmd SessionCmd) FilterValue() string { return cmd.session.Title }
func (cmd SessionCmd) Execute(model tea.Model) (tea.Model, tea.Cmd) {
	layout := model.(LayoutView)
	layout.Chat.Session = cmd.session
	layout.Chat.Msgs = cmd.session.Msgs
	layout.Chat.Viewport.SetContent(layout.Chat.RenderMsgs())
	layout.Chat.Viewport.GotoBottom()
	layout.Chat.Input.SetValue("")

	layout.Menu = layout.Menu.Close()

	return layout, nil
}

type CreateSessionCmd struct {
	title string
	desc  string
}

func NewCreateSessionCmd(title, desc string) CreateSessionCmd {
	return CreateSessionCmd{
		title: title,
		desc:  desc,
	}
}

func (cmd CreateSessionCmd) Title() string       { return cmd.title }
func (cmd CreateSessionCmd) Description() string { return cmd.desc }
func (cmd CreateSessionCmd) FilterValue() string { return cmd.title }
func (cmd CreateSessionCmd) Execute(model tea.Model) (tea.Model, tea.Cmd) {
	layout := model.(LayoutView)
	layout.Chat.Msgs = []schema.Msg{}
	session := schema.ChatSession{}

	if layout.Storage != nil {
		session, _ = layout.Storage.NewSession()
	} else {
		sessionID := randstr.String(8)
		session = schema.ChatSession{
			ID:        sessionID,
			Title:     fmt.Sprintf("Session - %s", sessionID),
			CreatedAt: time.Now().Unix(),
		}
	}

	layout.Chat.Session = session
	layout.Chat.Viewport.SetContent(layout.Chat.RenderMsgs())
	layout.Chat.Viewport.GotoBottom()
	layout.Chat.Input.SetValue("")

	layout.Menu = layout.Menu.Close()

	return layout, nil

}

type DeleteSessionCmd struct {
	title string
	desc  string
}

func NewDeleteSessionCmd(title, desc string) DeleteSessionCmd {
	return DeleteSessionCmd{
		title: title,
		desc:  desc,
	}
}

func (cmd DeleteSessionCmd) Title() string       { return cmd.title }
func (cmd DeleteSessionCmd) Description() string { return cmd.desc }
func (cmd DeleteSessionCmd) FilterValue() string { return cmd.title }
func (cmd DeleteSessionCmd) Execute(model tea.Model) (tea.Model, tea.Cmd) {
	layout := model.(LayoutView)

	sessionID := randstr.String(8)
	newSession := schema.ChatSession{
		ID:        sessionID,
		Title:     fmt.Sprintf("Session - %s", sessionID),
		CreatedAt: time.Now().Unix(),
	}

	if layout.Storage != nil {
		err := layout.Storage.DeleteSession(layout.Chat.Session.ID)
		if err != nil {
			log.Printf("Error while deleting sessions: %s", err)
		}

		newSession, _ = layout.Storage.LoadRecentSession()
	}

	layout.Chat.Msgs = []schema.Msg{}
	layout.Chat.Session = newSession

	layout.Chat.Viewport.SetContent(layout.Chat.RenderMsgs())
	layout.Menu.Close()

	layout.Chat.Input.SetValue("")

	return layout, nil
}

type ExitCmd struct {
	title string
	desc  string
}

func NewExitCmd(title, desc string) ExitCmd {
	return ExitCmd{
		title: title,
		desc:  desc,
	}
}

func (cmd ExitCmd) Title() string       { return cmd.title }
func (cmd ExitCmd) Description() string { return cmd.desc }
func (cmd ExitCmd) FilterValue() string { return cmd.title }
func (cmd ExitCmd) Execute(model tea.Model) (tea.Model, tea.Cmd) {
	return model, tea.Quit
}

var DefaultCmds = []list.Item{
	ProvidersCmd{title: "/models", desc: "List available models", filter: schema.LLM},
	ProvidersCmd{title: "/agents", desc: "List available agents", filter: schema.Agent},
	SessionsCmd{title: "/sessions", desc: "List saved sessions"},
	CreateSessionCmd{title: "/new", desc: "Start new session"},
	DeleteSessionCmd{title: "/delete", desc: "Delete the current session"},
	ExitCmd{title: "/exit", desc: "Close tui chat app"},
}
