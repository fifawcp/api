package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ncondes/fifawcp/internal/infrastructure/logging"
)

// RunMigrations applies all pending *.up.sql migration files found at the
// given path. The path may be a file:// URL (e.g. "file:///app/cmd/db/migrations")
// or a plain directory path. Applied migrations are tracked in a
// schema_migrations table so each file is executed at most once.
func RunMigrations(db *sql.DB, path string, logger logging.Logger) error {
	// Strip the file:// scheme if present so we have a plain directory path.
	dir := strings.TrimPrefix(path, "file://")

	logger.Debug("Migration runner created", "path", dir)

	// Ensure the schema_migrations tracking table exists.
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Collect all *.up.sql files in the migrations directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory %q: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	// Determine which migrations have already been applied.
	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to scan migration version: %w", err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating applied migrations: %w", err)
	}

	// Count pending migrations for the log.
	pending := 0
	for _, f := range files {
		if !applied[f] {
			pending++
		}
	}
	logger.Debug("Migrations pending", "count", pending)

	if pending == 0 {
		logger.Info("No pending migrations — database schema is up to date")
		return nil
	}

	// Apply each pending migration inside its own transaction.
	for _, f := range files {
		if applied[f] {
			continue
		}

		logger.Debug("Applying migration", "file", f)

		content, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return fmt.Errorf("failed to read migration file %q: %w", f, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %q: %w", f, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to execute migration %q: %w", f, err)
		}

		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (version) VALUES ($1)`, f,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to record migration %q: %w", f, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %q: %w", f, err)
		}

		logger.Info("Migration applied", "file", f)
	}

	logger.Info("All migrations completed successfully", "applied", pending)
	return nil
}
