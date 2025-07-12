package providers

import (
	"context"
	"database/sql"
)

// SchemaProvider defines the interface for different schema extraction providers
type SchemaProvider interface {
	// Name returns the provider name for identification
	Name() string
	
	// ExtractSchema extracts the schema using the provider's method
	// The context allows for cancellation and timeout control
	ExtractSchema(ctx context.Context, params ExtractParams) (*SchemaResult, error)
	
	// IsAvailable checks if this provider can be used in the current environment
	IsAvailable() bool
}

// ExtractParams contains parameters needed for schema extraction
type ExtractParams struct {
	// DB is the database connection (used by SQL-based providers)
	DB *sql.DB
	
	// ConnectionString is the full connection string (used by external tools)
	ConnectionString string
	
	// Format specifies the output format
	Format SchemaFormat
}

// SchemaFormat represents the desired output format
type SchemaFormat string

const (
	FormatInfo SchemaFormat = "info" // Human-readable format
	FormatSQL  SchemaFormat = "sql"  // SQL DDL format
)

// SchemaResult contains the extracted schema in the requested format
type SchemaResult struct {
	// Tables contains parsed table information (for info format)
	Tables []Table
	
	// RawSQL contains the raw SQL DDL (for sql format)
	RawSQL string
	
	// Format indicates which format was used
	Format SchemaFormat
}

// ProviderRegistry manages available schema providers
type ProviderRegistry struct {
	providers map[string]SchemaProvider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]SchemaProvider),
	}
}

// Register adds a provider to the registry
func (r *ProviderRegistry) Register(provider SchemaProvider) {
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name
func (r *ProviderRegistry) Get(name string) (SchemaProvider, bool) {
	provider, exists := r.providers[name]
	return provider, exists
}

// ListAvailable returns all available providers
func (r *ProviderRegistry) ListAvailable() []string {
	var available []string
	for name, provider := range r.providers {
		if provider.IsAvailable() {
			available = append(available, name)
		}
	}
	return available
}