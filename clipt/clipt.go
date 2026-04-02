package clipt

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"github.com/struki84/datum/clipt/tui"
	"github.com/struki84/datum/clipt/tui/schema"
	"github.com/struki84/datum/clipt/tui/style"
)

func Render(providers []schema.ChatProvider, options ...Option) {
	config := schema.Config{
		Cmds:      tui.DefaultCmds,
		Providers: providers,
		Style:     style.Default(style.Dark),
		Debug: struct {
			Log  bool
			Path string
		}{
			Log:  false,
			Path: "",
		},
	}

	for _, opt := range options {
		opt(&config)
	}

	if config.Debug.Log {
		file, err := tea.LogToFile(config.Debug.Path, "debug")
		if err != nil {
			log.Fatal(err)
		}

		defer file.Close()
	}

	os.Setenv("COLORTERM", "truecolor")
	termenv.ColorProfile()

	app := tea.NewProgram(
		tui.NewLayout(config),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := app.Run(); err != nil {
		fmt.Printf("There's been an error while starting clipt tui: %v", err)
		os.Exit(1)
	}
}
