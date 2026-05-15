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

	for _, script := range scripts {
		if _, err := db.Exec(script.SQL); err != nil {
			return fmt.Errorf("apply migration %s: %w", script.Name, err)
		}
	}

	return nil
}
