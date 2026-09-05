package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed scripts
var Scripts embed.FS

type PostgresMigrator struct {
	db        *DBConnection
	scriptDir fs.FS
	log       *slog.Logger
}

func NewMigrator(connection *DBConnection, scripts fs.FS) *PostgresMigrator {
	return &PostgresMigrator{
		db:        connection,
		scriptDir: scripts,
		log:       slog.Default(),
	}
}

func (m *PostgresMigrator) Run(ctx context.Context) error {
	m.log.Info("Starting migration run")
	if err := m.ensureMigrationTableExists(ctx); err != nil {
		return err
	}

	files, err := fs.ReadDir(m.scriptDir, "scripts")
	if err != nil {
		return fmt.Errorf("failed to read scripts dir, %w", err)
	}

	m.log.Info("found scripts", slog.Int("count", len(files)))
	for _, f := range files {
		if f.IsDir() {
			m.log.Info("skipping dir", slog.String("name", f.Name()))

			continue
		}

		ok, err := m.migrationExists(ctx, f.Name())
		if err != nil {
			return err
		}

		if ok {
			m.log.Info("migration already applied", slog.String("name", f.Name()))

			continue
		}

		err = m.db.WithTransaction(ctx, func(db *DBConnection) error {
			scriptName := f.Name()

			b, err := fs.ReadFile(m.scriptDir, "scripts/"+scriptName)
			if err != nil {
				return fmt.Errorf("failed to read %s from scripts dir, %w", scriptName, err)
			}

			for q := range strings.SplitSeq(string(b), ";") {
				query := strings.TrimSpace(q)
				if query == "" {
					continue
				}

				if _, err := db.Conn.Exec(ctx, query); err != nil {
					return fmt.Errorf("script execution failed for %s, %w", scriptName, err)
				}
			}

			err = m.addMigrated(ctx, scriptName)
			if err != nil {
				return err
			}

			m.log.Info("migration applied", slog.String("name", scriptName))

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func (m *PostgresMigrator) addMigrated(ctx context.Context, name string) error {
	args := pgx.NamedArgs{
		"name": name,
	}
	query := `
INSERT INTO
	migration (name)
VALUES
	(@name)
`
	if _, err := m.db.Conn.Exec(ctx, query, args); err != nil {
		return fmt.Errorf("insert migration entry failed, %w", err)
	}

	return nil
}

func (m *PostgresMigrator) migrationExists(ctx context.Context, name string) (bool, error) {
	args := pgx.NamedArgs{
		"name": name,
	}
	query := `
SELECT
  count(*)
FROM
  migration AS m
WHERE
  m.name = @name
`
	row := m.db.Conn.QueryRow(ctx, query, args)

	var count int
	if err := row.Scan(&count); err != nil {
		return false, fmt.Errorf("find migration entry failed during row scan, %w", err)
	}

	return count > 0, nil
}

func (m *PostgresMigrator) ensureMigrationTableExists(ctx context.Context) error {
	query := `
SELECT
	count(*) 
FROM
	information_schema.TABLES
WHERE 
	(TABLE_SCHEMA = 'public') 
AND (TABLE_NAME = 'migration')`
	row := m.db.Conn.QueryRow(ctx, query)

	var tableCount int
	if err := row.Scan(&tableCount); err != nil {
		return fmt.Errorf("find migration table failed during row scan, %w", err)
	}

	if tableCount > 0 {
		m.log.Info("table 'migration' already created.")

		return nil
	}

	migrationTable := `
CREATE TABLE migration 
(
    name    VARCHAR(100) PRIMARY KEY NOT NULL CHECK ( name <> '' ),
    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
	if _, err := m.db.Conn.Exec(ctx, migrationTable); err != nil {
		return fmt.Errorf("failed to create migration table, %w", err)
	}

	m.log.Info("table 'migration' created.")

	return nil
}
