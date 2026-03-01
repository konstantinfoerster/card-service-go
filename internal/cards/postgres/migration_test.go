package postgres_test

import (
	"context"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"
	"github.com/konstantinfoerster/card-service-go/internal/cards/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrate_RunAllScripts(t *testing.T) {
	ctx := t.Context()
	conn := Connect(ctx, migrationCfg)
	migration := postgres.NewMigrator(conn, postgres.Scripts)

	err := migration.Run(ctx)

	require.NoError(t, err)
	assert.True(t, tableExists(t, ctx, conn, "migration"), "expected table migration to be created")
	assert.Equal(t, 20, tableCount(t, ctx, conn))
}

func TestMigrate_CanBeRunMultipleTimes(t *testing.T) {
	expectedMigrations := []string{"01-migration.sql", "02-migration.sql"}
	ctx := t.Context()
	rawSQL01, err := os.ReadFile("testdata/scripts/01-migration.sql")
	require.NoError(t, err)
	rawSQL02, err := os.ReadFile("testdata/scripts/02-migration.sql")
	require.NoError(t, err)
	fs := fstest.MapFS{
		"scripts/01-migration.sql": &fstest.MapFile{Data: rawSQL01},
		"scripts/02-migration.sql": &fstest.MapFile{Data: rawSQL02},
	}
	migration := postgres.NewMigrator(connection, fs)
	err = migration.Run(ctx)
	require.NoError(t, err)

	err = migration.Run(ctx)
	require.NoError(t, err)
	assert.True(t, tableExists(t, ctx, connection, "migration"), "expected table migration to be created")
	assert.True(t, tableExists(t, ctx, connection, "test_lang"), "expected table lang2 to be created")
	assert.True(t, tableExists(t, ctx, connection, "test_sub_type"), "expected table sub_type2 to be created")
	assert.True(t, tableExists(t, ctx, connection, "test_card_block"), "expected table sub_type2 to be created")
	assert.Equal(t, expectedMigrations, migrationEntrys(t, ctx, connection))
}

func TestMigrate_MissingScriptDir(t *testing.T) {
	ctx := t.Context()
	fs := fstest.MapFS{}
	migration := postgres.NewMigrator(connection, fs)

	err := migration.Run(ctx)

	assert.ErrorContains(t, err, "file does not exist")
}

func TestMigrate_EmptyScriptDir(t *testing.T) {
	ctx := t.Context()
	fs := fstest.MapFS{
		"scripts": &fstest.MapFile{Mode: fs.ModeDir},
	}
	migration := postgres.NewMigrator(connection, fs)

	err := migration.Run(ctx)

	assert.NoError(t, err)
}

func TestMigrate_ErrorDoesNotRevertAppliedScripts(t *testing.T) {
	expectedMigrations := []string{
		"01-migration.sql",
		"02-migration.sql",
		"03-migration.sql",
	}
	ctx := t.Context()
	rawSQL03, err := os.ReadFile("testdata/scripts/03-migration.sql")
	require.NoError(t, err)
	rawSQL04, err := os.ReadFile("testdata/scripts/04-migration.sql")
	require.NoError(t, err)
	fs := fstest.MapFS{
		"scripts/03-migration.sql": &fstest.MapFile{Data: rawSQL03},
		"scripts/04-migration.sql": &fstest.MapFile{Data: rawSQL04},
	}
	migration := postgres.NewMigrator(connection, fs)

	err = migration.Run(ctx)

	require.Error(t, err)
	assert.True(t, tableExists(t, ctx, connection, "test_card_set"), "expected table card_set2 to be created")
	assert.False(t, tableExists(t, ctx, connection, "test_collection"), "expected table lang3 to be rolled back")
	assert.False(t, tableExists(t, ctx, connection, "withError"), "expected table withError to not exist")
	assert.Equal(t, expectedMigrations, migrationEntrys(t, ctx, connection))
}

func tableExists(t *testing.T, ctx context.Context, con *postgres.DBConnection, name string) bool {
	args := pgx.NamedArgs{
		"name": name,
	}
	query := `
SELECT
	count(*) 
FROM
	information_schema.TABLES
WHERE 
	(TABLE_SCHEMA = 'public') 
AND (TABLE_NAME = @name)`
	row := con.Conn.QueryRow(ctx, query, args)

	var tableCount int
	err := row.Scan(&tableCount)
	require.NoError(t, err)

	return tableCount > 0
}

func tableCount(t *testing.T, ctx context.Context, con *postgres.DBConnection) int {
	query := `
SELECT
	count(*) 
FROM
	information_schema.TABLES
WHERE 
	(TABLE_SCHEMA = 'public')`
	row := con.Conn.QueryRow(ctx, query)

	var tableCount int
	err := row.Scan(&tableCount)
	require.NoError(t, err)

	return tableCount
}

func migrationEntrys(t *testing.T, ctx context.Context, con *postgres.DBConnection) []string {
	query := `
SELECT
  name
FROM
  migration
`

	rows, err := con.Conn.Query(ctx, query)
	require.NoError(t, err)
	defer rows.Close()

	var names []string

	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)

		names = append(names, name)
	}

	require.NoError(t, rows.Err())

	return names
}
