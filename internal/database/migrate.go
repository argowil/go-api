package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// Migrate runs any SQL files in migrationsDir that have not been applied yet.
// It tracks applied migrations in a schema_migrations table.
//
// dsn must include multiStatements=true so that files with multiple CREATE TABLE
// statements are executed in one shot without client-side splitting.
func Migrate(db *sqlx.DB, dsn, migrationsDir string) error {
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}

	// Open a separate connection that allows multi-statement execution.
	// We keep this separate from the main pool to avoid accidental multi-statement
	// usage in application queries (which would be a SQL-injection risk).
	mdb, err := sql.Open("mysql", dsn+"&multiStatements=true")
	if err != nil {
		return fmt.Errorf("open migration connection: %w", err)
	}
	defer mdb.Close()

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	sort.Strings(files)

	for _, file := range files {
		name := filepath.Base(file)

		var count int
		_ = db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, name).Scan(&count)
		if count > 0 {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		if _, err := mdb.Exec(string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}

		if _, err := db.Exec(`INSERT INTO schema_migrations (name) VALUES (?)`, name); err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		log.Printf("migration applied: %s", name)
	}

	return nil
}

func ensureMigrationsTable(db *sqlx.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name       VARCHAR(255) NOT NULL,
			applied_at DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (name)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	return err
}
