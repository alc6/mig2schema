package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/alc6/mig2schema/providers"
)

var (
	extractMode    bool
	mcpMode        bool
	providerName   string
	listProviders  bool
	pgImage        string
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
		if mcpMode || listProviders {
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
	if rootCmd.Flags().Lookup("provider") == nil {
		rootCmd.Flags().StringVarP(&providerName, "provider", "p", "native", "Schema extraction provider (native, pg_dump)")
	}
	if rootCmd.Flags().Lookup("list-providers") == nil {
		rootCmd.Flags().BoolVar(&listProviders, "list-providers", false, "List available schema extraction providers")
	}
	if rootCmd.Flags().Lookup("pg-image") == nil {
		rootCmd.Flags().StringVar(&pgImage, "pg-image", "postgres:16-alpine", "PostgreSQL Docker image to use")
	}

	return rootCmd.Execute()
}

func runMig2Schema(cmd *cobra.Command, args []string) {
	// Initialize provider registry
	registry := providers.NewProviderRegistry()
	registry.Register(providers.NewNativeProvider())
	registry.Register(providers.NewPgDumpProvider())

	if listProviders {
		fmt.Println("Available schema extraction providers:")
		for _, name := range registry.ListAvailable() {
			fmt.Printf("  - %s\n", name)
		}
		return
	}

	if mcpMode {
		slog.Info("starting mcp server")
		if err := StartMCPServer(); err != nil {
			slog.Error("failed to start mcp server", "error", err)
			os.Exit(1)
		}
		return
	}

	migrationDir := args[0]
	
	// Get the selected provider
	provider, exists := registry.Get(providerName)
	if !exists {
		slog.Error("unknown provider", "provider", providerName)
		fmt.Printf("Unknown provider: %s\n", providerName)
		fmt.Println("Use --list-providers to see available providers")
		os.Exit(1)
	}

	if !provider.IsAvailable() {
		slog.Error("provider not available", "provider", providerName)
		fmt.Printf("Provider '%s' is not available in this environment\n", providerName)
		os.Exit(1)
	}

	migrationReader := NewFileMigrationReader()
	dbManager := NewPostgreSQLManager(pgImage)
	
	if err := processSchemaWithProvider(migrationDir, migrationReader, dbManager, provider); err != nil {
		slog.Error("failed to process schema", "error", err)
		os.Exit(1)
	}
}

func processSchemaWithProvider(migrationDir string, migrationReader MigrationReader, dbManager DatabaseManager, provider providers.SchemaProvider) error {
	slog.Info("processing migration directory", "directory", migrationDir, "provider", provider.Name())

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
	
	// Determine format based on extractMode flag
	format := providers.FormatInfo
	if extractMode {
		format = providers.FormatSQL
	}

	// Extract schema using the provider
	params := providers.ExtractParams{
		DB:               dbManager.GetDB(),
		ConnectionString: dbManager.GetConnectionString(),
		Format:           format,
	}

	result, err := provider.ExtractSchema(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to extract schema: %w", err)
	}

	// Output the result
	if extractMode {
		fmt.Print(result.RawSQL)
	} else {
		fmt.Println("\n=== DATABASE SCHEMA ===")
		// Use the native formatter for info mode
		fmt.Print(providers.FormatSchemaInfo(result.Tables))
	}
	
	return nil
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
