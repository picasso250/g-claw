package gateway

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

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
