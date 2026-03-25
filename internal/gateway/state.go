package gateway

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const (
	StateIgnored   = 2
	StateProcessed = 3
)

func InitDB() (*sql.DB, error) {
	os.MkdirAll(filepath.Dir(DBFile), 0755)
	db, err := sql.Open("sqlite", DBFile)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS message_states (
		source TEXT NOT NULL,
		external_id TEXT NOT NULL,
		sender TEXT,
		subject TEXT,
		state INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (source, external_id)
	)`)
	if err != nil {
		return db, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS feishu_chat_user_cache (
		chat_id TEXT NOT NULL,
		open_id TEXT NOT NULL,
		display_name TEXT NOT NULL,
		refreshed_at_unix INTEGER NOT NULL,
		PRIMARY KEY (chat_id, open_id)
	)`)
	return db, err
}

func LookupMessageState(db *sql.DB, source, externalID string) (int, error) {
	var state int
	err := db.QueryRow("SELECT state FROM message_states WHERE source = ? AND external_id = ?", source, externalID).Scan(&state)
	return state, err
}

func SaveMessageState(db *sql.DB, source, externalID, sender, subject string, state int) error {
	_, err := db.Exec(
		"INSERT INTO message_states (source, external_id, sender, subject, state) VALUES (?, ?, ?, ?, ?)",
		source,
		externalID,
		sender,
		subject,
		state,
	)
	return err
}

func LookupEmailState(db *sql.DB, uid uint32) (int, error) {
	return LookupMessageState(db, "mail", fmt.Sprintf("%d", uid))
}

func SaveEmailState(db *sql.DB, uid uint32, sender, subject string, state int) error {
	return SaveMessageState(db, "mail", fmt.Sprintf("%d", uid), sender, subject, state)
}

type FeishuUserCacheEntry struct {
	ChatID         string
	OpenID         string
	DisplayName    string
	RefreshedAtUTC time.Time
}

func LookupFeishuUserCache(db *sql.DB, chatID, openID string) (*FeishuUserCacheEntry, error) {
	if db == nil {
		return nil, sql.ErrNoRows
	}

	var entry FeishuUserCacheEntry
	var refreshedAtUnix int64
	err := db.QueryRow(
		"SELECT chat_id, open_id, display_name, refreshed_at_unix FROM feishu_chat_user_cache WHERE chat_id = ? AND open_id = ?",
		chatID,
		openID,
	).Scan(&entry.ChatID, &entry.OpenID, &entry.DisplayName, &refreshedAtUnix)
	if err != nil {
		return nil, err
	}
	entry.RefreshedAtUTC = time.Unix(refreshedAtUnix, 0).UTC()
	return &entry, nil
}

func SaveFeishuUserCache(db *sql.DB, entry FeishuUserCacheEntry) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}

	refreshedAt := entry.RefreshedAtUTC
	if refreshedAt.IsZero() {
		refreshedAt = time.Now().UTC()
	}

	_, err := db.Exec(
		`INSERT INTO feishu_chat_user_cache (chat_id, open_id, display_name, refreshed_at_unix)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(chat_id, open_id) DO UPDATE SET
			display_name = excluded.display_name,
			refreshed_at_unix = excluded.refreshed_at_unix`,
		entry.ChatID,
		entry.OpenID,
		entry.DisplayName,
		refreshedAt.Unix(),
	)
	return err
}
