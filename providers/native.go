package providers

import (
	"context"
	"fmt"
	"log/slog"
)

// NativeProvider uses the existing custom SQL queries to extract schema
type NativeProvider struct{}

// NewNativeProvider creates a new native provider
func NewNativeProvider() SchemaProvider {
	return &NativeProvider{}
}

// Name returns the provider name
func (p *NativeProvider) Name() string {
	return "native"
}

// IsAvailable always returns true for the native provider
func (p *NativeProvider) IsAvailable() bool {
	return true
}

// ExtractSchema extracts the schema using custom SQL queries
func (p *NativeProvider) ExtractSchema(ctx context.Context, params ExtractParams) (*SchemaResult, error) {
	if params.DB == nil {
		return nil, fmt.Errorf("native provider requires database connection")
	}

	slog.Debug("extracting schema using native provider", "format", params.Format)

	// Extract tables using the SQL queries
	tables, err := ExtractSchemaFromDB(params.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to extract schema: %w", err)
	}

	result := &SchemaResult{
		Tables: tables,
		Format: params.Format,
	}

	// Format based on requested format
	switch params.Format {
	case FormatSQL:
		result.RawSQL = FormatSchemaSQL(tables)
	case FormatInfo:
		// For info format, we'll handle formatting at the output layer
		// Just return the tables
	default:
		return nil, fmt.Errorf("unsupported format: %s", params.Format)
	}

	return result, nil
}