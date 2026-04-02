package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/struki84/clipt/tui/chat"
	"github.com/struki84/clipt/tui/menu"
	"github.com/struki84/clipt/tui/schema"
	"github.com/thanhpk/randstr"
)

type LayoutView struct {
	WindowSize tea.WindowSizeMsg

	Style schema.LayoutStyle
	Menu  menu.ChatMenu
	Chat  chat.ChatView

	Storage   schema.SessionStorage
	Providers []schema.ChatProvider

	Info   string
	Status string
	Mode   schema.Mode
}

func NewLayout(conf schema.Config) LayoutView {
	layout := LayoutView{
		Menu:      menu.New(conf.Cmds, conf.Style),
		Chat:      chat.New(conf.Providers[0], conf.Style),
		Style:     conf.Style,
		Storage:   conf.Storage,
		Providers: conf.Providers,
		Info:      "enter - send | \"/\" - menu",
		Mode:      schema.Chat,
	}

	sessionID := randstr.String(8)
	layout.Chat.Msgs = []schema.Msg{}
	layout.Chat.Session = schema.ChatSession{
		ID:        sessionID,
		Title:     fmt.Sprintf("Session - %s", sessionID),
		CreatedAt: time.Now().Unix(),
	}

	if layout.Storage != nil {
		session, err := layout.Storage.LoadRecentSession()
		if err == nil {
			layout.Chat.Session = session
			layout.Chat.Msgs = session.Msgs
		}
	}

	return layout
}

func (layout LayoutView) Init() tea.Cmd {
	return tea.Batch(layout.Chat.Init())
}

func (layout LayoutView) View() string {
	elements := []string{}

	// Render Chat Header - Session title, date, and info
	// layout.Chat.View() will return only the header section and configure
	// the chat view port and the input, since the layout is dynamic due to
	// open-closing of the menu section
	header := layout.Chat.View()
	elements = append(elements, header)

	inputHeight := layout.Chat.Input.LineInfo().Height + 1
	baseViewportHeight := layout.WindowSize.Height - inputHeight - 7

	// Render Chat viewport and/or chat menu and modify the viewport height based on menu height
	if layout.Menu.Active {
		menuHeight := len(layout.Menu.FilteredItems)
		layout.Chat.Viewport.Height = baseViewportHeight - menuHeight

		vp := layout.Style.Chat.ContentView.Render(layout.Chat.Viewport.View())
		elements = append(elements, vp)
		elements = append(elements, layout.Menu.View())

	} else {
		layout.Chat.Viewport.Height = baseViewportHeight
		vp := lipgloss.PlaceHorizontal(
			layout.WindowSize.Width,
			lipgloss.Center,
			layout.Style.Chat.ContentView.Render(layout.Chat.Viewport.View()),
			lipgloss.WithWhitespaceBackground(lipgloss.Color(layout.Style.WhitespaceBGcolor)),
		)

		elements = append(elements, vp)
	}

	// Render Chat input
	input := lipgloss.PlaceHorizontal(
		layout.WindowSize.Width,
		lipgloss.Center,
		layout.Chat.Input.View(),
		lipgloss.WithWhitespaceBackground(lipgloss.Color(layout.Style.WhitespaceBGcolor)),
	)

	elements = append(elements, input)

	infoLine := layout.Style.InfoLine.Width(layout.WindowSize.Width).Render(layout.Info)
	elements = append(elements, infoLine)

	// Render the status line
	providerType := layout.Style.StatusLine.ProviderType.Render(layout.Chat.Provider.Type().String())
	providerName := layout.Style.StatusLine.ProviderName.Render(layout.Chat.Provider.Name())
	tab := layout.Style.StatusLine.ModeLabel.Render("tab")
	mode := layout.Style.StatusLine.ModeName.Render("CHAT")

	leftPart := lipgloss.JoinHorizontal(lipgloss.Top, providerType, providerName)
	rightPart := lipgloss.JoinHorizontal(lipgloss.Top, tab, mode)

	if layout.Chat.IsLoading {
		loader := layout.Style.StatusLine.Loader.Render(layout.Chat.Loader.View()) + layout.Style.StatusLine.Loader.Render("Working...")
		fillerWidth := layout.WindowSize.Width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart) - lipgloss.Width(loader)
		filler := layout.Style.StatusLine.BaseStyle.Width(fillerWidth).Render("")

		statusLine := lipgloss.JoinHorizontal(lipgloss.Top, leftPart, loader, filler, rightPart)

		elements = append(elements, statusLine)
	} else {

		fillerWidth := layout.WindowSize.Width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart)
		filler := layout.Style.StatusLine.BaseStyle.Width(fillerWidth).Render("")
		statusLine := lipgloss.JoinHorizontal(lipgloss.Top, leftPart, filler, rightPart)

		elements = append(elements, statusLine)
	}

	return layout.Style.ContentView.
		Width(layout.WindowSize.Width).
		Height(layout.WindowSize.Height).
		Render(lipgloss.JoinVertical(lipgloss.Center, elements...))
}

func (layout LayoutView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		layout.WindowSize = msg
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if layout.Menu.Active {
				layout.Menu = layout.Menu.Close()
				layout.Chat.Input.SetValue("")
				return layout, nil
			}
		case tea.KeyCtrlC:
			return layout, tea.Quit
		}
	}

	prompt := layout.Chat.Input.Value()

	layout.Menu.Active = strings.HasPrefix(prompt, "/")
	if layout.Menu.Active {
		layout.Info = "ctrl+j - down | ctrl+k - up"
		layout.Menu.SearchString = strings.TrimPrefix(prompt, "/")
	} else {
		layout.Info = "enter - send | \"/\" - menu"
	}

	menuModel, cmd := layout.Menu.Update(msg)
	layout.Menu = menuModel.(menu.ChatMenu)
	cmds = append(cmds, cmd)

	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyEnter {
		if layout.Menu.Active && len(layout.Menu.FilteredItems) > 0 {
			selected, ok := layout.Menu.List.SelectedItem().(schema.CmdItem)
			if ok && selected != nil {
				return selected.Execute(layout)
			}
		}
	}

	chatModel, cmd := layout.Chat.Update(msg)
	layout.Chat = chatModel.(chat.ChatView)
	cmds = append(cmds, cmd)

	return layout, tea.Batch(cmds...)
}
