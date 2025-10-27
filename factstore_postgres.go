package factstoredb

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// NewPostgresFactStore creates a new PostgreSQL-backed FactStore.
// It accepts a standard PostgreSQL connection string.
func NewPostgresFactStore(connStr string, opts ...StoreOption) (*FactStoreDB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(4)

	// PostgreSQL does not use PRAGMAs, so options are ignored for now,
	// but the signature is kept for consistency.

	store := &FactStoreDB{
		db:      db,
		dialect: postgresDialect{},
	}

	if err := store.initSchemaAndStatements(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema for PostgreSQL: %w", err)
	}

	return store, nil
}
