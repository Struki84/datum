package main

import (
	"github.com/struki84/clipt"
	"github.com/struki84/clipt/storage"
	"github.com/struki84/clipt/tui/schema"
)

func main() {
	dbPath := "./basic.db"
	sqlite := *storage.NewSQLite(dbPath)

	providers := []schema.ChatProvider{
		NewAnthropic("claude-opus-4.6", sqlite),
	}

	clipt.Render(
		providers,
		clipt.WithStorage(sqlite),
		clipt.WithDebugLog("debug.log"),
		clipt.WithStyle(CustomStyle(Light)),
		clipt.WithCmds(CustomCmds),
	)
}
