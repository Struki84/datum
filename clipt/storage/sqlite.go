package storage

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/struki84/datum/clipt/tui/schema"
	"github.com/thanhpk/randstr"
)

type Session struct {
	gorm.Model
	SessionID string
	Title     string
	Msgs      Messages `gorm:"type:jsonb;column:msgs"`
}

type Messages []Message

type Message struct {
	Role      string
	Content   string
	Timestamp int64
}

func (m Messages) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *Messages) Scan(src any) error {
	var bytes []byte
	switch v := src.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("could not scan type into Messages")
	}
	return json.Unmarshal(bytes, m)
}

type SQLite struct {
	db     *gorm.DB
	path   string
	record Session
}

func NewSQLite(dbPath string) *SQLite {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Printf("Error connecting to DB: %v", err)
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Error getting DB: %v", err)
		return nil
	}

	sqlDB.Exec("PRAGMA foreign_keys = ON;")
	sqlDB.Exec("PRAGMA journal_mode = WAL;")

	err = db.AutoMigrate(Session{})
	if err != nil {
		log.Printf("Error migrating DB: %v", err)
		return nil
	}

	return &SQLite{
		db:   db,
		path: dbPath,
	}
}

func (sql SQLite) NewSession() (schema.ChatSession, error) {
	sessionID := randstr.String(8)

	sql.record = Session{
		SessionID: sessionID,
		Title:     fmt.Sprintf("Session - %s", sessionID),
		Msgs:      Messages{},
	}

	err := sql.db.Create(&sql.record).Error

	if err != nil {
		return schema.ChatSession{}, fmt.Errorf("Error creating new session, %v", err)
	}

	return schema.ChatSession{
		ID:        sessionID,
		Title:     sql.record.Title,
		Msgs:      []schema.Msg{},
		CreatedAt: sql.record.CreatedAt.Unix(),
	}, nil
}

func (sql SQLite) ListSessions() []schema.ChatSession {
	sessions := []Session{}
	err := sql.db.Find(&sessions).Error
	if err != nil {
		return []schema.ChatSession{}
	}

	list := []schema.ChatSession{}
	for _, session := range sessions {
		msgs := []schema.Msg{}
		for _, msg := range session.Msgs {
			msgs = append(msgs, schema.Msg{
				Role:    schema.EnumRole(msg.Role),
				Content: msg.Content,
			})
		}

		list = append(list, schema.ChatSession{
			ID:        session.SessionID,
			Title:     session.Title,
			Msgs:      msgs,
			CreatedAt: session.CreatedAt.Unix(),
		})
	}

	return list
}

func (sql SQLite) LoadRecentSession() (schema.ChatSession, error) {
	sessions := []Session{}
	err := sql.db.Order("updated_at DESC").Limit(1).Find(&sessions).Error
	if err != nil {
		return schema.ChatSession{}, fmt.Errorf("Error loading recent sessions, %v", err)
	}

	if len(sessions) > 0 {
		session := sessions[0]

		msgs := []schema.Msg{}
		for _, msg := range session.Msgs {
			msgs = append(msgs, schema.Msg{
				Role:      schema.EnumRole(msg.Role),
				Content:   msg.Content,
				Timestamp: msg.Timestamp,
			})
		}

		sql.record = session

		return schema.ChatSession{
			ID:        sql.record.SessionID,
			Title:     sql.record.Title,
			Msgs:      msgs,
			CreatedAt: sql.record.CreatedAt.Unix(),
		}, nil

	}

	sessionID := randstr.String(8)
	sql.record = Session{
		SessionID: sessionID,
		Title:     fmt.Sprintf("Session - %s", sessionID),
		Msgs:      Messages{},
	}

	err = sql.db.Save(&sql.record).Error
	if err != nil {
		return schema.ChatSession{}, fmt.Errorf("Error loading recent sessions, %v", err)
	}

	return schema.ChatSession{
		ID:        sql.record.SessionID,
		Title:     sql.record.Title,
		Msgs:      []schema.Msg{},
		CreatedAt: sql.record.CreatedAt.Unix(),
	}, nil
}

func (sql SQLite) LoadSession(sessionID string) (schema.ChatSession, error) {
	err := sql.db.Where("session_id = ?", sessionID).Find(&sql.record).Error
	if err != nil {
		return schema.ChatSession{}, fmt.Errorf("Error loading session: %v", err)
	}

	msgs := []schema.Msg{}
	for _, msg := range sql.record.Msgs {
		msgs = append(msgs, schema.Msg{
			Role:      schema.EnumRole(msg.Role),
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
		})
	}

	return schema.ChatSession{
		ID:        sql.record.SessionID,
		Title:     sql.record.Title,
		Msgs:      msgs,
		CreatedAt: sql.record.CreatedAt.Unix(),
	}, nil
}

func (sql SQLite) SaveSession(session schema.ChatSession) (schema.ChatSession, error) {
	msgs := Messages{}
	for i, msg := range session.Msgs {
		msgs[i] = Message{
			Role:      msg.Role.String(),
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
		}
	}

	sql.record = Session{
		SessionID: session.ID,
		Title:     session.Title,
		Msgs:      msgs,
	}

	err := sql.db.Save(&sql.record).Error
	if err != nil {
		return schema.ChatSession{}, fmt.Errorf("Can't save session, %v", err)
	}

	return session, nil
}

func (sql SQLite) DeleteSession(sessionID string) error {
	err := sql.db.Where("session_id = ?", sessionID).Delete(&sql.record)
	if err != nil {
		return fmt.Errorf("Error deleting session: %v ", err)
	}
	return nil
}

func (sql SQLite) SaveMsg(sessionID string, msg schema.Msg) error {
	err := sql.db.Where("session_id = ?", sessionID).Find(&sql.record).Error
	if err != nil {
		return fmt.Errorf("Error loading session: %v", err)
	}

	message := Message{
		Role:      msg.Role.String(),
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
	}

	sql.record.Msgs = append(sql.record.Msgs, message)

	err = sql.db.Save(&sql.record).Error
	if err != nil {
		return fmt.Errorf("Can't save session, %v", err)
	}

	return nil
}

func (sql SQLite) LoadMsgs(sessionID string) (string, error) {
	result := []string{}
	err := sql.db.Where("session_id = ?", sessionID).Find(&sql.record).Error
	if err != nil {
		return "", fmt.Errorf("Error loading session: %v", err)
	}

	if sql.record.Msgs != nil {
		for _, msg := range sql.record.Msgs {
			result = append(result, fmt.Sprintf("%s: %s", msg.Role, msg.Content))
		}
	}

	return strings.Join(result, "\n"), nil
}
