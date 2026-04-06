package chat

import (
	"context"
	"fmt"
	"log"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/struki84/datum/clipt/tui/schema"
)

type ChatView struct {
	WindowSize tea.WindowSizeMsg

	Style    schema.LayoutStyle
	Msgs     []schema.Msg
	Stream   chan schema.Msg
	Provider schema.ChatProvider
	Session  schema.ChatSession

	IsLoading bool

	Header   string
	Viewport *viewport.Model
	Input    *textarea.Model
	Loader   spinner.Model
}

func New(provider schema.ChatProvider, style schema.LayoutStyle) ChatView {
	input := textarea.New()
	input.Focus()
	input.CharLimit = 0
	input.KeyMap.InsertNewline.SetEnabled(false)

	view := viewport.New(0, 0)
	loader := spinner.New()
	loader.Spinner = spinner.Dot

	return ChatView{
		Provider:  provider,
		Input:     &input,
		Viewport:  &view,
		Style:     style,
		Loader:    loader,
		IsLoading: false,
		Stream:    make(chan schema.Msg),
	}
}

func (chat ChatView) Init() tea.Cmd {
	cmds := []tea.Cmd{}
	cmds = append(cmds, textarea.Blink)
	cmds = append(cmds, chat.HandleStream)

	chat.Provider.Stream(context.TODO(), func(ctx context.Context, msg schema.Msg) error {
		chat.Stream <- msg
		return nil
	})

	return tea.Batch(cmds...)
}

func (chat ChatView) View() string {
	date := time.Unix(chat.Session.CreatedAt, 0).Format("2 Jan 2006")
	title := fmt.Sprintf("# %s \n%v", chat.Session.Title, date)

	chat.Header = lipgloss.PlaceHorizontal(
		chat.WindowSize.Width,
		lipgloss.Center,
		chat.Style.Chat.Header.Width(chat.WindowSize.Width-6).Render(title),
		lipgloss.WithWhitespaceBackground(lipgloss.Color(chat.Style.WhitespaceBGcolor)),
	)

	chat.Viewport.KeyMap = viewport.KeyMap{
		PageDown: key.NewBinding(key.WithKeys("pgdown")),
		PageUp:   key.NewBinding(key.WithKeys("pgup")),
		Down:     key.NewBinding(key.WithKeys("down")),
		Up:       key.NewBinding(key.WithKeys("up")),
	}

	chat.Input.Prompt = ""
	chat.Input.SetHeight(chat.Input.LineInfo().Height + 1)
	chat.Input.SetWidth(chat.WindowSize.Width - 4)
	chat.Input.FocusedStyle.CursorLine = lipgloss.NewStyle()
	chat.Input.FocusedStyle.Base = chat.Style.Chat.Input
	chat.Input.ShowLineNumbers = false

	return chat.Header
}

func (chat ChatView) RenderMsgs() string {
	var styledMessages []string

	user, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	width := chat.Viewport.Width - 6

	for _, msg := range chat.Msgs {
		switch msg.Role {
		case schema.InternalMsg:
			fullMsg := fmt.Sprintf("%s", msg.Content)
			chatMsg := chat.Style.Chat.Msg.Internal.Width(width).Render(fullMsg)

			styledMessages = append(styledMessages, chatMsg)
		case schema.ErrMsg:
			fullMsg := fmt.Sprintf("%s", msg.Content)
			chatMsg := chat.Style.Chat.Msg.Err.Width(width).Render(fullMsg)

			styledMessages = append(styledMessages, chatMsg)
		case schema.SysMsg:
			fullMsg := fmt.Sprintf("%s", msg.Content)
			chatMsg := chat.Style.Chat.Msg.Sys.Width(width).Render(fullMsg)

			styledMessages = append(styledMessages, chatMsg)
		case schema.UserMsg:
			date := time.Unix(msg.Timestamp, 0).Format("2 Jan | 15:04")
			username := user.Username
			fullMsg := fmt.Sprintf("%s\n%s (%s) ", msg.Content, username, date)
			chatMsg := chat.Style.Chat.Msg.User.Width(width).Render(fullMsg)

			styledMessages = append(styledMessages, chatMsg)

		case schema.AIMsg:
			renderer, _ := glamour.NewTermRenderer(
				glamour.WithStyles(chat.Style.Chat.Msg.Glamour),
				glamour.WithWordWrap(width-6),
			)

			renderedTxt, _ := renderer.Render(msg.Content)

			renderedTxt = replaceResets(renderedTxt, chat.Style.WhitespaceBGcolor)
			chatMsg := chat.Style.Chat.Msg.AI.Width(width).Render(renderedTxt)
			styledMessages = append(styledMessages, chatMsg)
		}
	}

	return lipgloss.PlaceHorizontal(
		chat.WindowSize.Width,
		lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, styledMessages...),
		lipgloss.WithWhitespaceBackground(lipgloss.Color(chat.Style.WhitespaceBGcolor)),
	)
}

func (chat ChatView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		chat.WindowSize = msg
		chat.Viewport.Width = msg.Width
		chat.Viewport.Height = msg.Height - chat.Input.LineInfo().Height - 7
		chat.Viewport.SetContent(chat.RenderMsgs())
		chat.Viewport.GotoBottom()
	case spinner.TickMsg:
		loader, cmd := chat.Loader.Update(msg)
		chat.Loader = loader
		cmds = append(cmds, cmd)
	case schema.Msg:
		if chat.IsLoading && msg.Role == schema.AIMsg {
			chat.IsLoading = false
		}

		lastMsg := chat.Msgs[len(chat.Msgs)-1]
		if msg.Stream {
			if !lastMsg.Stream {
				aiMsg := schema.Msg{
					Stream:    true,
					Content:   "",
					Role:      schema.AIMsg,
					Timestamp: time.Now().Unix(),
				}

				chat.Msgs = append(chat.Msgs, aiMsg)
				lastMsg = aiMsg
			}

			lastMsg.Role = msg.Role
			lastMsg.Content += msg.Content
			lastMsg.Timestamp = msg.Timestamp

			chat.Msgs[len(chat.Msgs)-1] = lastMsg
		} else {
			chat.Msgs = append(chat.Msgs, msg)
		}

		chat.Viewport.SetContent(chat.RenderMsgs())
		chat.Viewport.GotoBottom()

		return chat, chat.HandleStream

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			prompt := chat.Input.Value()
			menuActive := strings.HasPrefix(prompt, "/")
			if !menuActive && chat.Input.Focused() {
				input := chat.Input.Value()

				chat.Input.Reset()
				chat.IsLoading = true

				userMsg := schema.Msg{
					Stream:    false,
					Content:   input,
					Role:      schema.UserMsg,
					Timestamp: time.Now().Unix(),
				}

				chat.Msgs = append(chat.Msgs, userMsg)

				// aiMsg := schema.Msg{
				// 	Stream:    true,
				// 	Content:   "",
				// 	Role:      schema.AIMsg,
				// 	Timestamp: time.Now().Unix(),
				// }
				//
				// chat.Msgs = append(chat.Msgs, aiMsg)
				//

				chat.Viewport.SetContent(chat.RenderMsgs())
				chat.Viewport.GotoBottom()

				go func() {
					err := chat.Provider.Run(context.TODO(), input, chat.Session)
					if err != nil {
						log.Printf("Error: %v", err)
					}
				}()
			}

			return chat, chat.Loader.Tick
		}
	}

	input, cmd := chat.Input.Update(msg)
	chat.Input = &input
	cmds = append(cmds, cmd)

	vp, cmd := chat.Viewport.Update(msg)
	chat.Viewport = &vp
	cmds = append(cmds, cmd)

	return chat, tea.Batch(cmds...)
}

func (chat ChatView) HandleStream() tea.Msg {
	return <-chat.Stream
}

func replaceResets(s string, bgColor string) string {
	// Convert hex to RGB for truecolor background
	r, g, b := hexToRGB(bgColor)
	bgSeq := fmt.Sprintf("\x1b[0m\x1b[48;2;%d;%d;%dm", r, g, b)
	return strings.ReplaceAll(s, "\x1b[0m", bgSeq)
}

func hexToRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	val, _ := strconv.ParseInt(hex, 16, 32)
	return int(val >> 16), int((val >> 8) & 0xFF), int(val & 0xFF)
}
