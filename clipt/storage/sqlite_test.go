package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/struki84/datum/clipt/tui/schema"
	"github.com/thanhpk/randstr"
)

func TestLoadRecentSession(t *testing.T) {
	// Use a temporary directory for the test database
	tempDir, err := os.MkdirTemp("", "sqlite_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")

	// Initialize SQLite
	sqliteDB := NewSQLite(dbPath)
	if sqliteDB == nil {
		t.Fatal("Failed to initialize SQLite")
	}

	newSession3 := Session{
		SessionID: randstr.String(8),
		Title:     "Test Session 3",
		Msgs:      Messages{{Role: "User", Content: "Hello"}},
	}
	err = sqliteDB.db.Create(&newSession3).Error
	if err != nil {
		t.Fatalf("Failed to create third session: %v", err)
	}

	// Test case 3: Load recent with messages
	session3, err := sqliteDB.LoadRecentSession()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(session3.Msgs) != 1 {
		t.Errorf("Expected 1 message, got %d", len(session3.Msgs))
	}
	if len(session3.Msgs) > 0 && session3.Msgs[0].Content != "Hello" {
		t.Errorf("Expected message content 'Hello', got %s", session3.Msgs[0].Content)
	}

	// Test case 4: Add dummy messages to the recent session and reload
	// err = sqliteDB.SaveSessionMsg(session3.ID, "Dummy human message", "Dummy AI response")
	// if err != nil {
	// 	t.Fatalf("Failed to save dummy messages: %v", err)
	// }

	// Reload the recent session to verify messages
	reloadedSession, err := sqliteDB.LoadRecentSession()
	if err != nil {
		t.Errorf("Expected no error on reload, got %v", err)
	}
	if len(reloadedSession.Msgs) == 3 {
		if reloadedSession.Msgs[0].Role != schema.UserMsg || reloadedSession.Msgs[0].Content != "Hello" {
			t.Errorf("First message mismatch: expected User:Hello, got %s:%s", reloadedSession.Msgs[0].Role, reloadedSession.Msgs[0].Content)
		}
		if reloadedSession.Msgs[1].Role != schema.UserMsg || reloadedSession.Msgs[1].Content != "Dummy human message" {
			t.Errorf("Second message mismatch: expected User:Dummy human message, got %s:%s", reloadedSession.Msgs[1].Role, reloadedSession.Msgs[1].Content)
		}
		if reloadedSession.Msgs[2].Role != schema.AIMsg || reloadedSession.Msgs[2].Content != "Dummy AI response" {
			t.Errorf("Third message mismatch: expected AI:Dummy AI response, got %s:%s", reloadedSession.Msgs[2].Role, reloadedSession.Msgs[2].Content)
		}
	}
}
