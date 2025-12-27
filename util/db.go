package util

import (
	"database/sql"
	"fmt"
)

func InitDB(db *sql.DB) error {
	createGamesTableQuery := `
		CREATE TABLE IF NOT EXISTS games (
			id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			name TEXT
		);
	`
	createPointsTableQuery := `
		CREATE TABLE IF NOT EXISTS points (
			id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			discordid TEXT,
			points INTEGER
		);
	`

	createNameHistoryTableQuery := `
		CREATE TABLE IF NOT EXISTS names (
			id INTEGER GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			guild_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			new_display_name TEXT NOT NULL,
			changed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`

	_, err := db.Exec(createGamesTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create games table. %w", err)
	}

	_, err = db.Exec(createPointsTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create points table. %w", err)
	}

	_, err = db.Exec(createNameHistoryTableQuery)
	if err != nil {
		return fmt.Errorf("Failed to creeate name history table. %w", err)
	}

	return nil

}
