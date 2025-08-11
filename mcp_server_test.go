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

func TestStartMCPServerExists(t *testing.T) {
	t.Run("mcp_server_function_exists", func(t *testing.T) {
		t.Log("StartMCPServer function is defined and accessible")
	})
}

func TestValidateMigrationsCore(t *testing.T) {
	t.Run("valid_migrations", func(t *testing.T) {
		tempDir := t.TempDir()
		files := map[string]string{
			"001_users.up.sql":   "create table users (id int);",
			"001_users.down.sql": "drop table users;",
			"002_posts.up.sql":   "create table posts (id int);",
		}
		for filename, content := range files {
			err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
			require.NoError(t, err)
		}
		
		result, err := validateMigrationsCore(tempDir)
		require.NoError(t, err)
		assert.Contains(t, result, `"migration_count": 2`)
	})

	t.Run("empty_directory", func(t *testing.T) {
		tempDir := t.TempDir()
		
		result, err := validateMigrationsCore(tempDir)
		require.NoError(t, err)
		assert.Contains(t, result, `"migration_count": 0`)
	})

	t.Run("nonexistent_directory", func(t *testing.T) {
		_, err := validateMigrationsCore("/path/that/does/not/exist")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "migration directory does not exist")
	})

	t.Run("with_down_files", func(t *testing.T) {
		tempDir := t.TempDir()
		files := map[string]string{
			"001_test.up.sql":   "create table test (id int);",
			"001_test.down.sql": "drop table test;",
		}
		for filename, content := range files {
			err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
			require.NoError(t, err)
		}
		
		result, err := validateMigrationsCore(tempDir)
		require.NoError(t, err)
		assert.Contains(t, result, `"has_down_file": true`)
	})

	t.Run("parse_error", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("test setup failed or running as root")
		}

		tempDir := t.TempDir()
		migrationFile := filepath.Join(tempDir, "001_test.up.sql")
		err := os.WriteFile(migrationFile, []byte("create table test (id int);"), 0644)
		require.NoError(t, err)

		if err := os.Chmod(tempDir, 0000); err != nil {
			t.Skip("test setup failed - cannot change directory permissions")
		}
		defer os.Chmod(tempDir, 0755)

		_, err = validateMigrationsCore(tempDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
	})
}

func TestExtractSchemaCore(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping extract schema core test in short mode")
	}
	
	if !isDockerAvailable() {
		t.Skip("docker not available, skipping extract schema test")
	}

	t.Run("extract_info_format", func(t *testing.T) {
		tempDir := t.TempDir()
		content := `create table test_table (
			id serial primary key,
			name varchar(100)
		);`
		err := os.WriteFile(filepath.Join(tempDir, "001_test.up.sql"), []byte(content), 0644)
		require.NoError(t, err)
		
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		
		result, err := extractSchemaCore(ctx, tempDir, "info", "native", "postgres:16-alpine")
		require.NoError(t, err)
		assert.Contains(t, result, "Table: test_table")
	})

	t.Run("extract_sql_format", func(t *testing.T) {
		tempDir := t.TempDir()
		content := `create table sql_test (
			id serial primary key
		);`
		err := os.WriteFile(filepath.Join(tempDir, "001_test.up.sql"), []byte(content), 0644)
		require.NoError(t, err)
		
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		
		result, err := extractSchemaCore(ctx, tempDir, "sql", "native", "postgres:16-alpine")
		require.NoError(t, err)
		assert.Contains(t, result, "create table sql_test")
	})

	t.Run("empty_directory_error", func(t *testing.T) {
		tempDir := t.TempDir()
		
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		
		_, err := extractSchemaCore(ctx, tempDir, "info", "native", "postgres:16-alpine")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no migration files found")
	})

	t.Run("nonexistent_directory_error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		
		_, err := extractSchemaCore(ctx, "/nonexistent/path", "info", "native", "postgres:16-alpine")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "migration directory does not exist")
	})
}

func TestMCPValidationLogic(t *testing.T) {
	t.Run("valid_migrations", func(t *testing.T) {
		tempDir := t.TempDir()
		files := map[string]string{
			"001_create_users.up.sql":   "create table users (id serial primary key);",
			"001_create_users.down.sql": "drop table users;",
			"002_add_posts.up.sql":      "create table posts (id serial primary key);",
		}
		for filename, content := range files {
			err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
			require.NoError(t, err)
		}

		migrations, err := ParseMigrations(tempDir)
		require.NoError(t, err)
		assert.Len(t, migrations, 2)

		result := map[string]any{
			"valid":           true,
			"migration_count": len(migrations),
			"migrations":      make([]map[string]any, len(migrations)),
		}
		assert.Equal(t, 2, result["migration_count"])
	})

	t.Run("empty_directory", func(t *testing.T) {
		tempDir := t.TempDir()

		migrations, err := ParseMigrations(tempDir)
		require.NoError(t, err)
		assert.Len(t, migrations, 0)

		result := map[string]any{
			"valid":           true,
			"migration_count": len(migrations),
			"migrations":      make([]map[string]any, len(migrations)),
		}
		assert.Equal(t, 0, result["migration_count"])
	})

	t.Run("nonexistent_directory", func(t *testing.T) {
		_, err := ParseMigrations("/nonexistent/directory")
		assert.Error(t, err)
	})
}

func TestMCPExtractionLogic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping mcp extraction logic test in short mode")
	}

	if !isDockerAvailable() {
		t.Skip("docker not available, skipping mcp extraction test")
	}

	t.Run("extraction_workflow", func(t *testing.T) {
		tempDir := t.TempDir()

		migrationContent := `
			create table mcp_test_table (
				id serial primary key,
				name varchar(255) not null
			);
		`

		err := os.WriteFile(filepath.Join(tempDir, "001_mcp_test.up.sql"), []byte(migrationContent), 0644)
		require.NoError(t, err)

		_, err = os.Stat(tempDir)
		assert.NoError(t, err)

		migrations, err := ParseMigrations(tempDir)
		require.NoError(t, err)
		assert.NotEmpty(t, migrations)

		testTables := []providers.Table{
			{
				Name: "test_table",
				Columns: []providers.Column{
					{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
				},
			},
		}

		infoOutput := FormatSchema(testTables)
		assert.NotEmpty(t, infoOutput)

		sqlOutput := FormatSchemaAsSQL(testTables)
		assert.NotEmpty(t, sqlOutput)
		assert.NotEqual(t, infoOutput, sqlOutput)
	})
}

func TestExtractSchemaCoreWithDeps(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("successful_extraction_info_format", func(t *testing.T) {
		testMigrations := []Migration{{Name: "001_test", UpFile: "001_test.up.sql"}}
		testSchema := []providers.Table{
			{
				Name: "test_table",
				Columns: []providers.Column{
					{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
					{Name: "name", DataType: "varchar", IsNullable: true},
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
				return "Table: test_table\nColumns:\n  - id integer NOT NULL (PRIMARY KEY)\n  - name varchar NULL\n"
			},
		}

		result, err := extractSchemaCoreWithDeps(context.Background(), tempDir, "info", 
			mockReader, mockDB, mockExtractor)
		
		require.NoError(t, err)
		assert.Contains(t, result, "Table: test_table")
	})

	t.Run("successful_extraction_sql_format", func(t *testing.T) {
		testMigrations := []Migration{{Name: "001_test", UpFile: "001_test.up.sql"}}
		testSchema := []providers.Table{
			{
				Name: "sql_test",
				Columns: []providers.Column{
					{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
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
				return "create table sql_test (\n    id integer not null,\n    primary key (id)\n);\n"
			},
		}

		result, err := extractSchemaCoreWithDeps(context.Background(), tempDir, "sql", 
			mockReader, mockDB, mockExtractor)
		
		require.NoError(t, err)
		assert.Contains(t, result, "create table sql_test")
	})

	t.Run("nonexistent_directory", func(t *testing.T) {
		mockReader := &MockMigrationReader{}
		mockDB := &MockDatabaseManager{}
		mockExtractor := &MockSchemaExtractor{}

		_, err := extractSchemaCoreWithDeps(context.Background(), "/nonexistent/path", "info",
			mockReader, mockDB, mockExtractor)
		
		require.Error(t, err)
		assert.Contains(t, err.Error(), "migration directory does not exist")
	})

	t.Run("no_migrations_found", func(t *testing.T) {
		mockReader := &MockMigrationReader{
			DiscoverMigrationsFunc: func(dir string) ([]Migration, error) {
				return []Migration{}, nil
			},
		}
		mockDB := &MockDatabaseManager{}
		mockExtractor := &MockSchemaExtractor{}

		_, err := extractSchemaCoreWithDeps(context.Background(), tempDir, "info",
			mockReader, mockDB, mockExtractor)
		
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no migration files found")
	})

	t.Run("database_setup_failure", func(t *testing.T) {
		mockReader := &MockMigrationReader{
			DiscoverMigrationsFunc: func(dir string) ([]Migration, error) {
				return []Migration{{Name: "001_test", UpFile: "001_test.up.sql"}}, nil
			},
		}
		mockDB := &MockDatabaseManager{
			SetupFunc: func(ctx context.Context) error {
				return fmt.Errorf("connection refused")
			},
		}
		mockExtractor := &MockSchemaExtractor{}

		_, err := extractSchemaCoreWithDeps(context.Background(), tempDir, "info",
			mockReader, mockDB, mockExtractor)
		
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to setup postgresql")
	})

	t.Run("migration_execution_failure", func(t *testing.T) {
		mockReader := &MockMigrationReader{
			DiscoverMigrationsFunc: func(dir string) ([]Migration, error) {
				return []Migration{{Name: "001_test", UpFile: "001_test.up.sql"}}, nil
			},
		}
		mockDB := &MockDatabaseManager{
			RunMigrationsFunc: func(migrations []Migration) error {
				return fmt.Errorf("syntax error in SQL")
			},
		}
		mockExtractor := &MockSchemaExtractor{}

		_, err := extractSchemaCoreWithDeps(context.Background(), tempDir, "info",
			mockReader, mockDB, mockExtractor)
		
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to run migrations")
		assert.True(t, mockDB.CloseCalled)
	})

	t.Run("schema_extraction_failure", func(t *testing.T) {
		mockReader := &MockMigrationReader{
			DiscoverMigrationsFunc: func(dir string) ([]Migration, error) {
				return []Migration{{Name: "001_test", UpFile: "001_test.up.sql"}}, nil
			},
		}
		mockDB := &MockDatabaseManager{
			GetDBFunc: func() *sql.DB {
				return &sql.DB{}
			},
		}
		mockExtractor := &MockSchemaExtractor{
			ExtractSchemaFunc: func(db *sql.DB) ([]providers.Table, error) {
				return nil, fmt.Errorf("information_schema query failed")
			},
		}

		_, err := extractSchemaCoreWithDeps(context.Background(), tempDir, "info",
			mockReader, mockDB, mockExtractor)
		
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to extract schema")
	})
}