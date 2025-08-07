package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/alc6/mig2schema/providers"
)

// StartMCPServer starts the MCP server for schema extraction
func StartMCPServer() error {
	s := server.NewMCPServer(
		"mig2schema",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	extractSchemaTool := mcp.NewTool("extract_schema",
		mcp.WithDescription("Extract database schema from PostgreSQL migration files using pg_dump"),
		mcp.WithString("migration_directory",
			mcp.Required(),
			mcp.Description("Path to directory containing migration files"),
		),
		mcp.WithString("format",
			mcp.Description("Output format: 'sql' for CREATE statements (default)"),
			mcp.Enum("sql"),
		),
		mcp.WithString("postgres_image",
			mcp.Description("PostgreSQL Docker image to use (default: postgres:16-alpine)"),
		),
	)

	s.AddTool(extractSchemaTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleExtractSchema(ctx, request)
	})

	validateMigrationsTool := mcp.NewTool("validate_migrations",
		mcp.WithDescription("Validate migration files in directory without running them"),
		mcp.WithString("migration_directory",
			mcp.Required(),
			mcp.Description("Path to directory containing migration files"),
		),
		mcp.WithString("postgres_image",
			mcp.Description("PostgreSQL Docker image to use (default: postgres:16-alpine)"),
		),
	)

	s.AddTool(validateMigrationsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleValidateMigrations(ctx, request)
	})

	slog.Info("starting mig2schema mcp server")
	return server.ServeStdio(s)
}

// handleExtractSchema processes the extract_schema tool request
func handleExtractSchema(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	migrationDir, err := request.RequireString("migration_directory")
	if err != nil {
		return mcp.NewToolResultError("migration_directory parameter is required"), nil
	}

	format := request.GetString("format", "sql")
	pgImage := request.GetString("postgres_image", "postgres:16-alpine")
	// Always use pg_dump provider in MCP mode
	providerName := "pg_dump"

	output, err := extractSchemaCore(ctx, migrationDir, format, providerName, pgImage)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("schema extracted successfully:\n\n%s", output)), nil
}

// extractSchemaCore contains the core logic for schema extraction, separated for testing
func extractSchemaCore(ctx context.Context, migrationDir, format, providerName, pgImage string) (string, error) {
	// Initialize provider registry
	registry := providers.NewProviderRegistry()
	registry.Register(providers.NewNativeProvider())
	registry.Register(providers.NewPgDumpProvider())

	provider, exists := registry.Get(providerName)
	if !exists {
		return "", fmt.Errorf("unknown provider: %s", providerName)
	}

	if !provider.IsAvailable() {
		return "", fmt.Errorf("provider '%s' is not available in this environment", providerName)
	}

	migrationReader := NewFileMigrationReader()
	dbManager := NewPostgreSQLManager(pgImage)
	
	return extractSchemaCoreWithProvider(ctx, migrationDir, format, migrationReader, dbManager, provider)
}

// extractSchemaCoreWithProvider is the provider-based extraction function
func extractSchemaCoreWithProvider(ctx context.Context, migrationDir, format string, 
	migrationReader MigrationReader, dbManager DatabaseManager, provider providers.SchemaProvider) (string, error) {
	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		return "", fmt.Errorf("migration directory does not exist: %s", migrationDir)
	}

	migrations, err := migrationReader.DiscoverMigrations(migrationDir)
	if err != nil {
		return "", fmt.Errorf("failed to parse migrations: %v", err)
	}

	if len(migrations) == 0 {
		return "", fmt.Errorf("no migration files found in directory")
	}

	if err := dbManager.Setup(ctx); err != nil {
		return "", fmt.Errorf("failed to setup postgresql: %v", err)
	}
	defer func() {
		if err := dbManager.Close(ctx); err != nil {
			slog.Error("failed to cleanup database", "error", err)
		}
	}()

	if err := dbManager.RunMigrations(migrations); err != nil {
		return "", fmt.Errorf("failed to run migrations: %v", err)
	}

	// Convert format string to SchemaFormat
	var schemaFormat providers.SchemaFormat
	switch format {
	case "sql":
		schemaFormat = providers.FormatSQL
	default:
		schemaFormat = providers.FormatInfo
	}

	// Extract schema using the provider
	params := providers.ExtractParams{
		DB:               dbManager.GetDB(),
		ConnectionString: dbManager.GetConnectionString(),
		Format:           schemaFormat,
	}

	result, err := provider.ExtractSchema(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to extract schema: %v", err)
	}

	// Format output based on result
	var output string
	if schemaFormat == providers.FormatSQL {
		output = result.RawSQL
	} else {
		output = providers.FormatSchemaInfo(result.Tables)
	}

	return output, nil
}

// extractSchemaCoreWithDeps is the testable version with dependency injection
func extractSchemaCoreWithDeps(ctx context.Context, migrationDir, format string, 
	migrationReader MigrationReader, dbManager DatabaseManager, schemaExtractor SchemaExtractor) (string, error) {
	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		return "", fmt.Errorf("migration directory does not exist: %s", migrationDir)
	}

	migrations, err := migrationReader.DiscoverMigrations(migrationDir)
	if err != nil {
		return "", fmt.Errorf("failed to parse migrations: %v", err)
	}

	if len(migrations) == 0 {
		return "", fmt.Errorf("no migration files found in directory")
	}

	if err := dbManager.Setup(ctx); err != nil {
		return "", fmt.Errorf("failed to setup postgresql: %v", err)
	}
	defer func() {
		if err := dbManager.Close(ctx); err != nil {
			slog.Error("failed to cleanup database", "error", err)
		}
	}()

	if err := dbManager.RunMigrations(migrations); err != nil {
		return "", fmt.Errorf("failed to run migrations: %v", err)
	}

	schema, err := schemaExtractor.ExtractSchema(dbManager.GetDB())
	if err != nil {
		return "", fmt.Errorf("failed to extract schema: %v", err)
	}

	var output string
	if format == "sql" {
		output = schemaExtractor.FormatSchemaAsSQL(schema)
	} else {
		output = schemaExtractor.FormatSchema(schema)
	}

	return output, nil
}

// handleValidateMigrations processes the validate_migrations tool request
func handleValidateMigrations(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	migrationDir, err := request.RequireString("migration_directory")
	if err != nil {
		return mcp.NewToolResultError("migration_directory parameter is required"), nil
	}

	output, err := validateMigrationsCore(migrationDir)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("migration validation completed:\n\n%s", output)), nil
}

// validateMigrationsCore contains the core logic for migration validation, separated for testing
func validateMigrationsCore(migrationDir string) (string, error) {
	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		return "", fmt.Errorf("migration directory does not exist: %s", migrationDir)
	}

	migrations, err := ParseMigrations(migrationDir)
	if err != nil {
		return "", fmt.Errorf("failed to parse migrations: %v", err)
	}

	result := map[string]interface{}{
		"valid":           true,
		"migration_count": len(migrations),
		"migrations":      make([]map[string]interface{}, len(migrations)),
	}

	for i, migration := range migrations {
		migrationInfo := map[string]interface{}{
			"name":          migration.Name,
			"up_file":       migration.UpFile,
			"has_down_file": migration.DownFile != "",
		}
		if migration.DownFile != "" {
			migrationInfo["down_file"] = migration.DownFile
		}
		result["migrations"].([]map[string]interface{})[i] = migrationInfo
	}

	jsonOutput, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	return string(jsonOutput), nil
}
