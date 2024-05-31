package util

import (
	"database/sql"
	"fmt"
)

func InitDB(db *sql.DB) error {
	createGamesTableQuery := `
		CREATE TABLE IF NOT EXISTS games (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			name TEXT
		);
	`
	createPointsTableQuery := `
		CREATE TABLE IF NOT EXISTS points (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			discordid TEXT,
			points INTEGER
		);
	`

	_, err := db.Exec(createGamesTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create games table. %w", err)
	}

	_, err = db.Exec(createPointsTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create points table. %w", err)
	}

	return nil

}
