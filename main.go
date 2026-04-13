package main

import (
	"context"
	"log"

	"github.com/struki84/datum/agents"
	"github.com/struki84/datum/clipt"
	"github.com/struki84/datum/clipt/storage"
	"github.com/struki84/datum/clipt/tui/schema"
	"github.com/struki84/datum/clipt/tui/style"
	"github.com/struki84/datum/tools/library"
)

func main() {

	sentry := library.NewFileSentry("./data")
	ctx := context.Background()

	err := sentry.ScanFiles(ctx)
	if err != nil {
		log.Println("Error scanning for files:", err)
		return
	}

	go func() {
		err = sentry.WatchFiles(ctx)
		if err != nil {
			log.Println("Error watching files:", err)
		}
	}()

	dbPath := "./basic.db"
	sqlite := *storage.NewSQLite(dbPath)

	chatAgent := agents.NewChatAgent("openai/gpt-4.1", sqlite)
	voiceAgent := agents.NewVoiceAgent(agents.NewChatAgent("openai/gpt-4.1", sqlite))

	models := []schema.ChatProvider{
		chatAgent,
		voiceAgent,
	}

	clipt.Render(
		models,
		clipt.WithStorage(sqlite),
		clipt.WithDebugLog("debug.log"),
		clipt.WithStyle(style.Default(style.CatppuccinMocha)),
	)
}
