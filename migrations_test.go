package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMigrations(t *testing.T) {
	tempDir := t.TempDir()

	testFiles := map[string]string{
		"001_create_users.up.sql":   "create table users (id serial primary key);",
		"001_create_users.down.sql": "drop table users;",
		"002_add_posts.up.sql":      "create table posts (id serial primary key, user_id integer);",
		"003_no_down.up.sql":        "create index idx_users_id on users(id);",
	}

	for filename, content := range testFiles {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	migrations, err := ParseMigrations(tempDir)
	require.NoError(t, err)
	assert.Len(t, migrations, 3)

	migration1 := migrations[0]
	assert.Equal(t, "001_create_users", migration1.Name)
	assert.NotEmpty(t, migration1.DownFile)

	migration3 := migrations[2]
	assert.Equal(t, "003_no_down", migration3.Name)
	assert.Empty(t, migration3.DownFile)
}

func TestParseMigrationsEmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	migrations, err := ParseMigrations(tempDir)
	require.NoError(t, err)
	assert.Empty(t, migrations)
}

func TestParseMigrationsNonExistentDirectory(t *testing.T) {
	_, err := ParseMigrations("/non/existent/directory")
	assert.Error(t, err)
}

func TestParseMigrationsTimestampNaming(t *testing.T) {
	tempDir := t.TempDir()

	testFiles := map[string]string{
		"20240115120000_create_products.up.sql":   "create table products (id serial primary key);",
		"20240115120000_create_products.down.sql": "drop table products;",
		"20240220143000_add_inventory.up.sql":     "alter table products add column inventory integer;",
	}

	for filename, content := range testFiles {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	migrations, err := ParseMigrations(tempDir)
	require.NoError(t, err)
	assert.Len(t, migrations, 2)
	assert.Equal(t, "20240115120000_create_products", migrations[0].Name)
	assert.Equal(t, "20240220143000_add_inventory", migrations[1].Name)
}

func TestParseMigrationsWithNonSQLFiles(t *testing.T) {
	tempDir := t.TempDir()

	files := map[string]string{
		"readme.md":          "# README",
		"001_test.txt":       "not a migration",
		"002_valid.up.sql":   "CREATE TABLE test();",
		"003_valid.down.sql": "DROP TABLE test();",
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		require.NoError(t, err)
	}

	migrations, err := ParseMigrations(tempDir)
	require.NoError(t, err)
	assert.Len(t, migrations, 1)
	assert.Equal(t, "002_valid", migrations[0].Name)
}

func TestParseMigrationsPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tempDir := t.TempDir()
	defer func() {
		os.Chmod(tempDir, 0755)
	}()

	if err := os.Chmod(tempDir, 0000); err != nil {
		t.Skip("skipping test - cannot change directory permissions")
	}

	_, err := ParseMigrations(tempDir)
	assert.Error(t, err)
}