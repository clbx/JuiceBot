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
		return fmt.Errorf("Failed to create name history table. %w", err)
	}

	return nil

}

type NameDBEntry struct {
	GuildID        string
	UserID         string
	NewDisplayName string
}

func AddNameEntry(db *sql.DB, entry NameDBEntry) error {
	addNameEntryQuery := `INSERT INTO names (guild_id, user_id, new_display_name) VALUES ($1, $2, $3)`
	_, err := db.Exec(addNameEntryQuery, entry.GuildID, entry.UserID, entry.NewDisplayName)
	if err != nil {
		return fmt.Errorf("Failed to write name change update to the DB. %w", err)
	}
	return nil
}

type NameHistoryEntry struct {
	ID             int
	GuildID        string
	UserID         string
	NewDisplayName string
	ChangedAt      string
}

func GetNameHistory(db *sql.DB, guildID string, userID string) ([]NameHistoryEntry, error) {
	query := `SELECT id, guild_id, user_id, new_display_name, TO_CHAR(changed_at, 'Mon DD, YYYY HH24:MI') FROM names WHERE guild_id = $1 AND user_id = $2 ORDER BY changed_at DESC`
	rows, err := db.Query(query, guildID, userID)
	if err != nil {
		return nil, fmt.Errorf("Failed to query name history. %w", err)
	}
	defer rows.Close()

	var history []NameHistoryEntry
	for rows.Next() {
		var entry NameHistoryEntry
		err := rows.Scan(&entry.ID, &entry.GuildID, &entry.UserID, &entry.NewDisplayName, &entry.ChangedAt)
		if err != nil {
			return nil, fmt.Errorf("Failed to scan name history row. %w", err)
		}
		history = append(history, entry)
	}

	return history, nil
}
