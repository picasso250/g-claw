package gateway

import (
	"database/sql"
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

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS email_states (
		uid INTEGER PRIMARY KEY,
		sender TEXT,
		subject TEXT,
		state INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	return db, err
}

func LookupEmailState(db *sql.DB, uid uint32) (int, error) {
	var state int
	err := db.QueryRow("SELECT state FROM email_states WHERE uid = ?", uid).Scan(&state)
	return state, err
}

func SaveEmailState(db *sql.DB, uid uint32, sender, subject string, state int) error {
	_, err := db.Exec(
		"INSERT INTO email_states (uid, sender, subject, state) VALUES (?, ?, ?, ?)",
		uid,
		sender,
		subject,
		state,
	)
	return err
}
