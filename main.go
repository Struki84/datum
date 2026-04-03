package main

import (
	"github.com/struki84/datum/agents"
	"github.com/struki84/datum/clipt"
	"github.com/struki84/datum/clipt/storage"
	"github.com/struki84/datum/clipt/tui/schema"
	"github.com/struki84/datum/clipt/tui/style"
)

func main() {
	dbPath := "./basic.db"
	sqlite := *storage.NewSQLite(dbPath)

	models := []schema.ChatProvider{
		agents.NewChatAgent("openai/gpt-5.4", sqlite),
	}

	clipt.Render(
		models,
		clipt.WithStorage(sqlite),
		clipt.WithDebugLog("debug.log"),
		clipt.WithStyle(style.Default(style.CatppuccinMocha)),
	)
}
