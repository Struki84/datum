package schema

import "github.com/charmbracelet/bubbles/list"

type Mode int

const (
	Chat Mode = iota
	Debug
	Action
)

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
