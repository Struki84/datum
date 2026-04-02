package schema

import (
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
)

type LayoutStyle struct {
	WhitespaceBGcolor string
	ContentView       lipgloss.Style
	InfoLine          lipgloss.Style

	StatusLine struct {
		BaseStyle    lipgloss.Style
		ProviderType lipgloss.Style
		ProviderName lipgloss.Style
		Loader       lipgloss.Style
		ModeLabel    lipgloss.Style
		ModeName     lipgloss.Style
	}

	Menu struct {
		ContentView  lipgloss.Style
		ItemNormal   lipgloss.Style
		ItemSelected lipgloss.Style
		Description  lipgloss.Style
	}

	Chat struct {
		ContentView lipgloss.Style
		Header      lipgloss.Style
		Input       lipgloss.Style

		Msg struct {
			User     lipgloss.Style
			AI       lipgloss.Style
			Sys      lipgloss.Style
			Err      lipgloss.Style
			Internal lipgloss.Style
			Glamour  ansi.StyleConfig
		}
	}
}
