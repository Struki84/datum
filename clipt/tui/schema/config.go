package schema

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
)

type Mode int

const (
	Chat Mode = iota
	Debug
	Action
	Voice
)

func (mode Mode) String() string {
	switch mode {
	case Chat:
		return "Chat"
	case Debug:
		return "Debug"
	case Action:
		return "Action"
	case Voice:
		return "Speach"
	default:
		return fmt.Sprintf("Mode(%d)", mode)
	}
}

type Config struct {
	Providers []ChatProvider
	Style     LayoutStyle
	Storage   SessionStorage
	Cmds      []list.Item

	Debug struct {
		Log  bool
		Path string
	}
}
