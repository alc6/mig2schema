package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupPostgreSQL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Run("successful_setup", func(t *testing.T) {
		db, err := SetupPostgreSQL(ctx)
		require.NoError(t, err)
		defer db.Close(ctx)

		assert.NotNil(t, db.DB)
		assert.NotNil(t, db.Container)
		assert.NotEmpty(t, db.ConnStr)
		assert.NoError(t, db.DB.Ping())
	})
}

func TestDatabaseRunMigrations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := SetupPostgreSQL(ctx)
	require.NoError(t, err)
	defer db.Close(ctx)

	t.Run("successful_migration", func(t *testing.T) {
		migrations := []Migration{
			{
				Name:   "001_test",
				UpFile: "testdata/001_test.up.sql",
			},
		}

		testDir := t.TempDir()
		migrationContent := "CREATE TABLE test_table (id SERIAL PRIMARY KEY);"
		testFile := filepath.Join(testDir, "001_test.up.sql")
		require.NoError(t, os.WriteFile(testFile, []byte(migrationContent), 0644))

		migrations[0].UpFile = testFile

		require.NoError(t, db.RunMigrations(migrations))

		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'test_table'
		)`
		require.NoError(t, db.DB.QueryRow(query).Scan(&exists))
		assert.True(t, exists)
	})

	t.Run("migration_file_not_found", func(t *testing.T) {
		migrations := []Migration{
			{
				Name:   "nonexistent",
				UpFile: "/nonexistent/file.sql",
			},
		}

		err := db.RunMigrations(migrations)
		assert.Error(t, err)
	})
}

func TestDatabaseClose(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Run("close_valid_database", func(t *testing.T) {
		db, err := SetupPostgreSQL(ctx)
		require.NoError(t, err)

		assert.NoError(t, db.Close(ctx))
	})

	t.Run("close_nil_database", func(t *testing.T) {
		db := &Database{}
		assert.NoError(t, db.Close(ctx))
	})
}
