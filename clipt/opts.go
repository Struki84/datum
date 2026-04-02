package clipt

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/struki84/clipt/tui/schema"
)

type Option func(*schema.Config)

func WithConfig(config schema.Config) Option {
	return func(conf *schema.Config) {
		conf = &config
	}
}

func WithStorage(storage schema.SessionStorage) Option {
	return func(conf *schema.Config) {
		conf.Storage = storage
	}
}

func WithCmds(cmds []list.Item) Option {
	return func(conf *schema.Config) {
		conf.Cmds = cmds
	}
}

func WithAddedCmds(cmds []list.Item) Option {
	return func(conf *schema.Config) {
		conf.Cmds = append(conf.Cmds, cmds...)
	}
}

func WithStyle(style schema.LayoutStyle) Option {
	return func(conf *schema.Config) {
		conf.Style = style
	}
}

func WithDebugLog(path string) Option {
	return func(conf *schema.Config) {
		conf.Debug.Log = true
		conf.Debug.Path = path
	}
}
