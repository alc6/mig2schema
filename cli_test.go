package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cli integration test in short mode")
	}

	if !isDockerAvailable() {
		t.Skip("docker not available, skipping integration test")
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

	t.Run("info_mode", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		resetCommand()

		os.Args = []string{"mig2schema", tempDir}
		err := rootCmd.Execute()
		require.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		assert.Contains(t, output, "=== DATABASE SCHEMA ===")
		assert.Contains(t, output, "Table: users")
		assert.Contains(t, output, "Table: posts")
	})

	t.Run("extract_mode", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		resetCommand()

		os.Args = []string{"mig2schema", "-e", tempDir}
		err := rootCmd.Execute()
		require.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		assert.Contains(t, output, "create table users")
		assert.Contains(t, output, "create table posts")
		assert.Contains(t, output, "create index")
	})

}

func TestCLIErrorHandling(t *testing.T) {
	resetCommand()
	cmd := rootCmd
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err)

	resetCommand()
	cmd = rootCmd
	cmd.SetArgs([]string{"some-directory"})
	err = cmd.ParseFlags([]string{})
	assert.NoError(t, err)
}

func TestCLIMCPMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping mcp mode test in short mode")
	}

	resetCommand()

	os.Args = []string{"mig2schema", "--mcp"}

	cmd := rootCmd
	cmd.SetArgs([]string{"--mcp"})
	err := cmd.ParseFlags([]string{"--mcp"})
	require.NoError(t, err)
	assert.True(t, mcpMode)
}
