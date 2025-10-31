package factstoredb

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// NewFactStorePostgreSQL creates a new PostgreSQL-backed FactStore.
// It accepts a standard PostgreSQL connection string.
func NewFactStorePostgreSQL(connStr string) (*FactStoreDB, error) {
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
		ownsDB:  true,
		dialect: postgresDialect{},
	}

	if err := store.initSchemaAndStatements(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema for PostgreSQL: %w", err)
	}

	return store, nil
}

// NewFactStorePostgreSQLFromDB creates a new PostgreSQL-backed FactStore from an existing database connection.
// The caller retains ownership of the db connection and must close it separately.
// The opts parameter is accepted for API consistency but currently ignored (PostgreSQL does not use PRAGMAs).
// Note: This constructor does not configure connection pooling settings (use NewFactStorePostgreSQL for that).
func NewFactStorePostgreSQLFromDB(db *sql.DB) (*FactStoreDB, error) {
	// PostgreSQL does not use PRAGMAs, so options are ignored for now,
	// but the signature is kept for consistency.

	store := &FactStoreDB{
		db:      db,
		ownsDB:  false,
		dialect: postgresDialect{},
	}

	if err := store.initSchemaAndStatements(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema for PostgreSQL: %w", err)
	}

	return store, nil
}
