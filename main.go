package main

import (
	"github.com/struki84/datum/agents"
	"github.com/struki84/datum/clipt"
	"github.com/struki84/datum/clipt/storage"
	"github.com/struki84/datum/clipt/tui/schema"
	"github.com/struki84/datum/clipt/tui/style"
)

func main() {

	// sentry := files.NewFileSentry("./files", client)
	//
	// err = sentry.ScanFiles(ctx)
	// if err != nil {
	// 	log.Println("Error scanning for files:", err)
	// 	return
	// }
	//
	// go func() {
	// 	err = sentry.WatchFiles(ctx)
	// 	if err != nil {
	// 		log.Println("Error watching files:", err)
	// 	}
	// }()

	dbPath := "./basic.db"
	sqlite := *storage.NewSQLite(dbPath)

	models := []schema.ChatProvider{
		agents.NewChatAgent("openai/gpt-4.1", sqlite),
		agents.NewVoiceAgent(agents.NewChatAgent("openai/gpt-5.4-mini", sqlite)),
	}

	clipt.Render(
		models,
		clipt.WithStorage(sqlite),
		clipt.WithDebugLog("debug.log"),
		clipt.WithStyle(style.Default(style.CatppuccinMocha)),
	)
}
