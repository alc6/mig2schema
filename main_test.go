package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/alc6/mig2schema/providers"
)

func TestMigrationToSchema(t *testing.T) {
	if !isDockerAvailable() {
		t.Skip("docker not available, skipping migration to schema test")
	}

	tempDir := t.TempDir()

	migrationContent := map[string]string{
		"001_create_users.up.sql": `
			create table users (
				id serial primary key,
				email varchar(255) not null unique,
				created_at timestamp default current_timestamp
			);
		`,
		"002_create_posts.up.sql": `
			create table posts (
				id serial primary key,
				title varchar(255) not null,
				user_id integer not null references users(id),
				created_at timestamp default current_timestamp
			);
			create index idx_posts_user_id on posts(user_id);
		`,
	}

	for filename, content := range migrationContent {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	migrations, err := ParseMigrations(tempDir)
	require.NoError(t, err)
	assert.Len(t, migrations, 2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db, err := SetupPostgreSQL(ctx)
	require.NoError(t, err)
	defer func() {
		if err := db.Close(ctx); err != nil {
			t.Logf("failed to cleanup database: %v", err)
		}
	}()

	err = db.RunMigrations(migrations)
	require.NoError(t, err)

	schema, err := ExtractSchema(db.DB)
	require.NoError(t, err)
	assert.Len(t, schema, 2)

	var usersTable *providers.Table
	for i := range schema {
		if schema[i].Name == "users" {
			usersTable = &schema[i]
			break
		}
	}
	require.NotNil(t, usersTable, "users table not found in schema")
	assert.GreaterOrEqual(t, len(usersTable.Columns), 3)

	hasPrimaryKey := false
	for _, col := range usersTable.Columns {
		if col.IsPrimaryKey {
			hasPrimaryKey = true
			break
		}
	}
	assert.True(t, hasPrimaryKey, "users table should have a primary key")
}

func TestFormatSchemaOutputModes(t *testing.T) {
	tables := []providers.Table{
		{
			Name: "test_table",
			Columns: []providers.Column{
				{
					Name:         "id",
					DataType:     "integer",
					IsNullable:   false,
					IsPrimaryKey: true,
				},
			},
		},
	}

	infoOutput := FormatSchema(tables)
	assert.NotEmpty(t, infoOutput)

	sqlOutput := FormatSchemaAsSQL(tables)
	assert.NotEmpty(t, sqlOutput)
	assert.NotEqual(t, infoOutput, sqlOutput)
}

func TestRun(t *testing.T) {
	t.Run("run_function_help", func(t *testing.T) {
		resetCommand()
		cmd := rootCmd
		cmd.SetArgs([]string{"--help"})
		err := cmd.Execute()
		t.Logf("help command result: %v", err)
	})

	t.Run("run_function_no_args", func(t *testing.T) {
		resetCommand()
		cmd := rootCmd
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		assert.Error(t, err)
	})
}

func TestProcessSchemaUnit(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("migration_directory_does_not_exist", func(t *testing.T) {
		mockReader := &MockMigrationReader{}
		mockDB := &MockDatabaseManager{}
		mockExtractor := &MockSchemaExtractor{}

		err := processSchema("/non/existent/path", mockReader, mockDB, mockExtractor)
		if err == nil {
			t.Fatal("expected error for non-existent directory")
		}
		if err.Error() != "migration directory does not exist: /non/existent/path" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("no_migrations_found", func(t *testing.T) {
		mockReader := &MockMigrationReader{
			DiscoverMigrationsFunc: func(dir string) ([]Migration, error) {
				return []Migration{}, nil
			},
		}
		mockDB := &MockDatabaseManager{}
		mockExtractor := &MockSchemaExtractor{}

		err := processSchema(tempDir, mockReader, mockDB, mockExtractor)
		if err == nil {
			t.Fatal("expected error when no migrations found")
		}
		if err.Error() != fmt.Sprintf("no migration files found in directory: %s", tempDir) {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("migration_discovery_error", func(t *testing.T) {
		mockReader := &MockMigrationReader{
			DiscoverMigrationsFunc: func(dir string) ([]Migration, error) {
				return nil, fmt.Errorf("failed to read directory")
			},
		}
		mockDB := &MockDatabaseManager{}
		mockExtractor := &MockSchemaExtractor{}

		err := processSchema(tempDir, mockReader, mockDB, mockExtractor)
		if err == nil {
			t.Fatal("expected error when migration discovery fails")
		}
	})

	t.Run("database_setup_error", func(t *testing.T) {
		mockReader := &MockMigrationReader{
			DiscoverMigrationsFunc: func(dir string) ([]Migration, error) {
				return []Migration{{Name: "001_test", UpFile: "001_test.up.sql"}}, nil
			},
		}
		mockDB := &MockDatabaseManager{
			SetupFunc: func(ctx context.Context) error {
				return fmt.Errorf("failed to connect to database")
			},
		}
		mockExtractor := &MockSchemaExtractor{}

		err := processSchema(tempDir, mockReader, mockDB, mockExtractor)
		if err == nil {
			t.Fatal("expected error when database setup fails")
		}
		if !mockDB.SetupCalled {
			t.Error("expected Setup to be called")
		}
	})

	t.Run("migration_execution_error", func(t *testing.T) {
		mockReader := &MockMigrationReader{
			DiscoverMigrationsFunc: func(dir string) ([]Migration, error) {
				return []Migration{{Name: "001_test", UpFile: "001_test.up.sql"}}, nil
			},
		}
		mockDB := &MockDatabaseManager{
			RunMigrationsFunc: func(migrations []Migration) error {
				return fmt.Errorf("SQL syntax error")
			},
		}
		mockExtractor := &MockSchemaExtractor{}

		err := processSchema(tempDir, mockReader, mockDB, mockExtractor)
		if err == nil {
			t.Fatal("expected error when migration execution fails")
		}
		if !mockDB.SetupCalled {
			t.Error("expected Setup to be called")
		}
		if !mockDB.RunMigrationsCalled {
			t.Error("expected RunMigrations to be called")
		}
		if !mockDB.CloseCalled {
			t.Error("expected Close to be called even on error")
		}
	})

	t.Run("schema_extraction_error", func(t *testing.T) {
		mockReader := &MockMigrationReader{
			DiscoverMigrationsFunc: func(dir string) ([]Migration, error) {
				return []Migration{{Name: "001_test", UpFile: "001_test.up.sql"}}, nil
			},
		}
		mockDB := &MockDatabaseManager{
			GetDBFunc: func() *sql.DB {
				return &sql.DB{} // Return a non-nil DB
			},
		}
		mockExtractor := &MockSchemaExtractor{
			ExtractSchemaFunc: func(db *sql.DB) ([]providers.Table, error) {
				return nil, fmt.Errorf("failed to query information_schema")
			},
		}

		err := processSchema(tempDir, mockReader, mockDB, mockExtractor)
		if err == nil {
			t.Fatal("expected error when schema extraction fails")
		}
		if !mockDB.GetDBCalled {
			t.Error("expected GetDB to be called")
		}
	})

	t.Run("successful_execution_info_mode", func(t *testing.T) {
		originalExtractMode := extractMode
		extractMode = false
		defer func() { extractMode = originalExtractMode }()

		testMigrations := []Migration{{Name: "001_test", UpFile: "001_test.up.sql"}}
		testSchema := []providers.Table{
			{
				Name: "users",
				Columns: []providers.Column{
					{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
					{Name: "email", DataType: "varchar", IsNullable: false},
				},
			},
		}

		mockReader := &MockMigrationReader{
			DiscoverMigrationsFunc: func(dir string) ([]Migration, error) {
				return testMigrations, nil
			},
		}
		mockDB := &MockDatabaseManager{
			GetDBFunc: func() *sql.DB {
				return &sql.DB{}
			},
		}
		mockExtractor := &MockSchemaExtractor{
			ExtractSchemaFunc: func(db *sql.DB) ([]providers.Table, error) {
				return testSchema, nil
			},
			FormatSchemaFunc: func(tables []providers.Table) string {
				return "Table: users\nColumns:\n  - id integer NOT NULL (PRIMARY KEY)\n  - email varchar NOT NULL\n"
			},
		}

		err := processSchema(tempDir, mockReader, mockDB, mockExtractor)
		require.NoError(t, err)
		assert.True(t, mockDB.SetupCalled)
		assert.True(t, mockDB.RunMigrationsCalled)
		assert.True(t, mockDB.GetDBCalled)
		assert.True(t, mockDB.CloseCalled)
	})

	t.Run("successful_execution_extract_mode", func(t *testing.T) {
		originalExtractMode := extractMode
		extractMode = true
		defer func() { extractMode = originalExtractMode }()

		testMigrations := []Migration{{Name: "001_test", UpFile: "001_test.up.sql"}}
		testSchema := []providers.Table{
			{
				Name: "users",
				Columns: []providers.Column{
					{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
					{Name: "email", DataType: "varchar", IsNullable: false},
				},
			},
		}

		mockReader := &MockMigrationReader{
			DiscoverMigrationsFunc: func(dir string) ([]Migration, error) {
				return testMigrations, nil
			},
		}
		mockDB := &MockDatabaseManager{
			GetDBFunc: func() *sql.DB {
				return &sql.DB{}
			},
		}
		mockExtractor := &MockSchemaExtractor{
			ExtractSchemaFunc: func(db *sql.DB) ([]providers.Table, error) {
				return testSchema, nil
			},
			FormatSchemaAsSQLFunc: func(tables []providers.Table) string {
				return "create table users (\n    id integer not null,\n    email varchar not null,\n    primary key (id)\n);\n"
			},
		}

		err := processSchema(tempDir, mockReader, mockDB, mockExtractor)
		require.NoError(t, err)
	})
}

func resetCommand() {
	extractMode = false
	mcpMode = false
	rootCmd.ResetFlags()
	rootCmd.Flags().BoolVarP(&extractMode, "extract", "e", false, "Extract schema as SQL CREATE statements")
	rootCmd.Flags().BoolVar(&mcpMode, "mcp", false, "Run as Model Context Protocol server")
}

func isDockerAvailable() bool {
	return true
}
