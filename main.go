package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	extractMode bool
	mcpMode     bool
)

var rootCmd = &cobra.Command{
	Use:   "mig2schema [migration-directory]",
	Short: "Extract database schema from migration files",
	Long: `mig2schema takes a directory containing PostgreSQL migration files (.up.sql and .down.sql)
and extracts the resulting database schema after running the migrations.

It uses testcontainers to spin up a PostgreSQL instance, runs the migrations,
and then extracts the schema information.

Modes:
  info mode (default): Shows human-readable schema information
  extract mode (-e): Outputs SQL CREATE statements
  mcp mode (--mcp): Run as Model Context Protocol server`,
	Args: func(cmd *cobra.Command, args []string) error {
		if mcpMode {
			return nil
		}
		return cobra.ExactArgs(1)(cmd, args)
	},
	Run: runMig2Schema,
}

func main() {
	if err := run(); err != nil {
		slog.Error("command execution failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))

	if rootCmd.Flags().Lookup("extract") == nil {
		rootCmd.Flags().BoolVarP(&extractMode, "extract", "e", false, "Extract schema as SQL CREATE statements")
	}
	if rootCmd.Flags().Lookup("mcp") == nil {
		rootCmd.Flags().BoolVar(&mcpMode, "mcp", false, "Run as Model Context Protocol server")
	}

	return rootCmd.Execute()
}

func runMig2Schema(cmd *cobra.Command, args []string) {
	if mcpMode {
		slog.Info("starting mcp server")
		if err := StartMCPServer(); err != nil {
			slog.Error("failed to start mcp server", "error", err)
			os.Exit(1)
		}
		return
	}

	migrationDir := args[0]
	
	migrationReader := NewFileMigrationReader()
	dbManager := NewPostgreSQLManager()
	schemaExtractor := NewPostgreSQLSchemaExtractor()
	
	if err := processSchema(migrationDir, migrationReader, dbManager, schemaExtractor); err != nil {
		slog.Error("failed to process schema", "error", err)
		os.Exit(1)
	}
}

func processSchema(migrationDir string, migrationReader MigrationReader, dbManager DatabaseManager, schemaExtractor SchemaExtractor) error {
	slog.Info("processing migration directory", "directory", migrationDir)

	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		return fmt.Errorf("migration directory does not exist: %s", migrationDir)
	}

	ctx := context.Background()

	slog.Info("parsing migration files")
	migrations, err := migrationReader.DiscoverMigrations(migrationDir)
	if err != nil {
		return fmt.Errorf("failed to parse migrations: %w", err)
	}

	if len(migrations) == 0 {
		return fmt.Errorf("no migration files found in directory: %s", migrationDir)
	}

	slog.Info("found migrations", "count", len(migrations))

	slog.Info("setting up database")
	if err := dbManager.Setup(ctx); err != nil {
		return fmt.Errorf("failed to setup database: %w", err)
	}
	defer func() {
		if err := dbManager.Close(ctx); err != nil {
			slog.Error("failed to cleanup", "error", err)
		}
	}()

	slog.Info("running migrations")
	if err := dbManager.RunMigrations(migrations); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("extracting schema")
	schema, err := schemaExtractor.ExtractSchema(dbManager.GetDB())
	if err != nil {
		return fmt.Errorf("failed to extract schema: %w", err)
	}

	if extractMode {
		fmt.Print(schemaExtractor.FormatSchemaAsSQL(schema))
	} else {
		fmt.Println("\n=== DATABASE SCHEMA ===")
		fmt.Print(schemaExtractor.FormatSchema(schema))
	}
	
	return nil
}
