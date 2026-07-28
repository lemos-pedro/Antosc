package database

import (
	"database/sql"
	"fmt"

	appmigrations "towercore/migrations"
)

func Migrate(db *sql.DB) error {
	scripts, err := appmigrations.Scripts()
	if err != nil {
		return err
	}

	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
    name TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	for _, script := range scripts {
		var applied bool
		if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = $1)`, script.Name).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", script.Name, err)
		}
		if applied {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", script.Name, err)
		}
		if _, err := tx.Exec(script.SQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", script.Name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (name) VALUES ($1)`, script.Name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", script.Name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", script.Name, err)
		}
	}

	return nil
}
