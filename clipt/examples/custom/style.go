package main

import (
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/struki84/clipt/tui/schema"
)

type ColorScheme int

const (
	Light ColorScheme = iota
	Dark
)

func CustomStyle(scheme ColorScheme) (style schema.LayoutStyle) {

	schemes := map[ColorScheme]map[string]string{
		// Rosé Pine Dawn
		Light: {
			"primaryBGcolor":               "#FAF4ED",
			"secondaryBGcolor":             "#F2E9E1",
			"tertiaryBGcolor":              "#E4DFDE",
			"primaryFGcolor":               "#907AA9",
			"secondaryFGcolor":             "#575279",
			"tertiaryFGcolor":              "#9893A5",
			"statusLineFGcolor":            "#6E6A86",
			"providerNameBGcolor":          "#D6CFC7",
			"menuDescFGcolor":              "#797593",
			"chatMsgErrBorderFGcolor":      "#B4637A",
			"chatMsgInternalBorderFGcolor": "#D7827E",
		},
		// Rosé Pine
		Dark: {
			"primaryBGcolor":               "#191724",
			"secondaryBGcolor":             "#1F1D2E",
			"tertiaryBGcolor":              "#26233A",
			"primaryFGcolor":               "#C4A7E7",
			"secondaryFGcolor":             "#E0DEF4",
			"tertiaryFGcolor":              "#908CAA",
			"statusLineFGcolor":            "#6E6A86",
			"providerNameBGcolor":          "#2A2740",
			"menuDescFGcolor":              "#6E6A86",
			"chatMsgErrBorderFGcolor":      "#EB6F92",
			"chatMsgInternalBorderFGcolor": "#EBBCBA",
		},
	}

	var (
		// reused background and foreground colors
		primaryBGcolor   = schemes[scheme]["primaryBGcolor"]
		secondaryBGcolor = schemes[scheme]["secondaryBGcolor"]
		tertiaryBGcolor  = schemes[scheme]["tertiaryBGcolor"]

		primaryFGcolor   = schemes[scheme]["primaryFGcolor"]
		secondaryFGcolor = schemes[scheme]["secondaryFGcolor"]
		tertiaryFGcolor  = schemes[scheme]["tertiaryFGcolor"]

		statusLineFGcolor            = schemes[scheme]["statusLineFGcolor"]
		providerNameBGcolor          = schemes[scheme]["providerNameBGcolor"]
		menuDescFGcolor              = schemes[scheme]["menuDescFGcolor"]
		chatMsgErrBorderFGcolor      = schemes[scheme]["chatMsgErrBorderFGcolor"]
		chatMsgInternalBorderFGcolor = schemes[scheme]["chatMsgInternalBorderFGcolor"]
	)

	// Glamour Styling - WIP
	// Glamour is used for rendering mardkown
	if scheme == Light {
		style.Chat.Msg.Glamour = styles.LightStyleConfig
	} else {
		style.Chat.Msg.Glamour = styles.DarkStyleConfig
	}

	zeroUint := uint(0)
	style.Chat.Msg.Glamour.Document.Margin = &zeroUint
	style.WhitespaceBGcolor = primaryBGcolor

	// Main container view
	style.ContentView = lipgloss.NewStyle().
		Background(lipgloss.Color(primaryBGcolor))

	// Infoline and status line
	style.InfoLine = lipgloss.NewStyle().
		Background(lipgloss.Color(primaryBGcolor)).
		Foreground(lipgloss.Color(tertiaryFGcolor)).
		Padding(0, 2, 0, 2).
		MarginBottom(1).
		Align(lipgloss.Left)

	style.StatusLine.BaseStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(tertiaryBGcolor)).
		Foreground(lipgloss.Color(statusLineFGcolor))

	style.StatusLine.ModeLabel = lipgloss.NewStyle().
		Background(lipgloss.Color(tertiaryBGcolor)).
		Foreground(lipgloss.Color(tertiaryFGcolor)).
		PaddingRight(1)

	style.StatusLine.ModeName = lipgloss.NewStyle().
		Background(lipgloss.Color(primaryFGcolor)).
		Foreground(lipgloss.Color(tertiaryBGcolor)).
		PaddingLeft(1).
		PaddingRight(1).
		BorderStyle(lipgloss.ThickBorder()).
		BorderBackground(lipgloss.Color(tertiaryBGcolor)).
		BorderForeground(lipgloss.Color(primaryFGcolor)).
		BorderLeft(true).
		BorderRight(false).
		BorderTop(false).
		BorderBottom(false)

	style.StatusLine.ProviderType = lipgloss.NewStyle().
		Background(lipgloss.Color(primaryFGcolor)).
		Foreground(lipgloss.Color(tertiaryBGcolor)).
		PaddingLeft(1).
		PaddingRight(1).
		BorderStyle(lipgloss.ThickBorder()).
		BorderBackground(lipgloss.Color(tertiaryBGcolor)).
		BorderForeground(lipgloss.Color(primaryFGcolor)).
		BorderRight(true)

	style.StatusLine.ProviderName = lipgloss.NewStyle().
		Background(lipgloss.Color(providerNameBGcolor)).
		Foreground(lipgloss.Color(secondaryFGcolor)).
		PaddingLeft(1).
		PaddingRight(1).
		BorderStyle(lipgloss.ThickBorder()).
		BorderBackground(lipgloss.Color(tertiaryBGcolor)).
		BorderForeground(lipgloss.Color(providerNameBGcolor)).
		BorderLeft(true)

	style.StatusLine.Loader = lipgloss.NewStyle().
		Background(lipgloss.Color(tertiaryBGcolor)).
		Foreground(lipgloss.Color(primaryFGcolor))

	// Chat menu
	style.Menu.ContentView = lipgloss.NewStyle().
		Background(lipgloss.Color(secondaryBGcolor)).
		BorderStyle(lipgloss.ThickBorder()).
		BorderBackground(lipgloss.Color(primaryBGcolor)).
		BorderForeground(lipgloss.Color(secondaryFGcolor)).
		BorderLeft(true).
		BorderRight(true).
		BorderTop(false).
		BorderBottom(false).
		PaddingLeft(1).
		PaddingRight(1)

	style.Menu.ItemNormal = lipgloss.NewStyle().
		Background(lipgloss.Color(secondaryBGcolor)).
		Foreground(lipgloss.Color(secondaryFGcolor)).
		Padding(0).
		Width(30)

	style.Menu.ItemSelected = lipgloss.NewStyle().
		Background(lipgloss.Color(secondaryBGcolor)).
		Foreground(lipgloss.Color(primaryFGcolor)).
		Padding(0).
		Width(30)

	style.Menu.Description = lipgloss.NewStyle().
		Background(lipgloss.Color(secondaryBGcolor)).
		Foreground(lipgloss.Color(menuDescFGcolor)).
		Width(60)

	// Chat view - viewport, input, and messages
	style.Chat.Header = lipgloss.NewStyle().
		Background(lipgloss.Color(primaryBGcolor)).
		Foreground(lipgloss.Color(secondaryFGcolor)).
		BorderStyle(lipgloss.ThickBorder()).
		BorderBackground(lipgloss.Color(primaryBGcolor)).
		BorderForeground(lipgloss.Color(secondaryBGcolor)).
		BorderRight(true).
		PaddingLeft(1).
		PaddingRight(1).
		BorderLeft(true).
		MarginTop(1).
		MarginBackground(lipgloss.Color(primaryBGcolor))

	style.Chat.ContentView = lipgloss.NewStyle().
		Background(lipgloss.Color(primaryBGcolor))

	style.Chat.Msg.AI = lipgloss.NewStyle().
		Background(lipgloss.Color(primaryBGcolor)).
		MarginLeft(3).
		MarginRight(3).
		MarginBackground(lipgloss.Color(primaryBGcolor))

	style.Chat.Msg.User = lipgloss.NewStyle().
		Background(lipgloss.Color(tertiaryBGcolor)).
		Foreground(lipgloss.Color(secondaryFGcolor)).
		BorderStyle(lipgloss.ThickBorder()).
		BorderBackground(lipgloss.Color(primaryBGcolor)).
		BorderForeground(lipgloss.Color(primaryFGcolor)).
		BorderLeft(true).
		BorderRight(true).
		BorderTop(false).
		BorderBottom(false).
		Padding(1).
		Margin(1, 2, 1, 2).
		MarginBackground(lipgloss.Color(primaryBGcolor)).
		Align(lipgloss.Left)

	style.Chat.Msg.Sys = lipgloss.NewStyle().
		Background(lipgloss.Color(tertiaryBGcolor)).
		Foreground(lipgloss.Color(secondaryFGcolor)).
		BorderBackground(lipgloss.Color(primaryBGcolor)).
		BorderStyle(lipgloss.ThickBorder()).
		BorderBackground(lipgloss.Color(primaryBGcolor)).
		BorderForeground(lipgloss.Color(tertiaryBGcolor)).
		BorderLeft(true).
		BorderRight(true).
		BorderTop(false).
		BorderBottom(false).
		Padding(1).
		Margin(1, 2, 1, 2).
		Align(lipgloss.Left)

	style.Chat.Msg.Err = lipgloss.NewStyle().
		Background(lipgloss.Color(tertiaryBGcolor)).
		Foreground(lipgloss.Color(secondaryFGcolor)).
		BorderStyle(lipgloss.ThickBorder()).
		BorderBackground(lipgloss.Color(primaryBGcolor)).
		BorderForeground(lipgloss.Color(chatMsgErrBorderFGcolor)).
		BorderLeft(true).
		BorderRight(true).
		BorderTop(false).
		BorderBottom(false).
		Padding(1).
		Margin(1, 2, 1, 2).
		Align(lipgloss.Left)

	style.Chat.Msg.Internal = lipgloss.NewStyle().
		Background(lipgloss.Color(tertiaryBGcolor)).
		Foreground(lipgloss.Color(secondaryFGcolor)).
		BorderStyle(lipgloss.ThickBorder()).
		BorderBackground(lipgloss.Color(primaryBGcolor)).
		BorderForeground(lipgloss.Color(chatMsgInternalBorderFGcolor)).
		BorderLeft(true).
		BorderRight(true).
		BorderTop(false).
		BorderBottom(false).
		Padding(1).
		Margin(1, 2, 1, 2).
		MarginBackground(lipgloss.Color(primaryBGcolor)).
		Align(lipgloss.Left)

	style.Chat.Input = lipgloss.NewStyle().
		Background(lipgloss.Color(secondaryBGcolor)).
		Foreground(lipgloss.Color(secondaryFGcolor)).
		BorderStyle(lipgloss.ThickBorder()).
		BorderBackground(lipgloss.Color(primaryBGcolor)).
		BorderForeground(lipgloss.Color(secondaryFGcolor)).
		BorderLeft(true).
		BorderRight(true).
		BorderTop(false).
		BorderBottom(false).
		Padding(1, 1, 0, 1)

	return style
}
