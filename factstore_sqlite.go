package factstoredb

import (
	"database/sql"
	"fmt"
	"sort"

	_ "modernc.org/sqlite" // SQLite driver
)

// config holds configuration options for the DBFactStore.
type config struct {
	pragmas map[string]string
}

// StoreOption is a function that configures a DBFactStore.
type StoreOption func(*config)

// WithPragma sets a specific SQLite PRAGMA statement.
// For example: WithPragma("synchronous", "NORMAL").
// This will override any default value for the given PRAGMA key.
func WithPragma(key, value string) StoreOption {
	return func(c *config) {
		if c.pragmas == nil {
			c.pragmas = make(map[string]string)
		}
		c.pragmas[key] = value
	}
}

// defaultConfig returns a new config with default PRAGMA settings
// for performance and concurrency.
func defaultConfig() *config {
	return &config{
		pragmas: map[string]string{
			"journal_mode": "WAL",
			"synchronous":  "OFF",
			"cache_size":   "-64000",
			"temp_store":   "MEMORY",
			"mmap_size":    "268435456",
			"busy_timeout": "5000",
			"foreign_keys": "OFF",
			"auto_vacuum":  "INCREMENTAL",
		},
	}
}

// FactStoreSQLite creates a new SQLite-backed FactStore.
// Pass ":memory:" for dbPath to create an in-memory database.
// Optional StoreOption functions can be provided to customize PRAGMA settings.
func NewFactStoreSQLite(dbPath string, opts ...StoreOption) (*FactStoreDB, error) {
	// For in-memory databases, use a unique name with shared cache
	// This allows concurrent connections within the same database while keeping
	// different database instances separate
	if dbPath == ":memory:" {
		// Generate unique name for this in-memory database instance
		id := inMemoryDBCounter.Add(1)
		dbPath = fmt.Sprintf("file:factstore_%d?mode=memory&cache=shared", id)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite: %w", err)
	}

	// Configure connection pool for concurrent access
	// MaxOpenConns: Allow up to 10 concurrent connections for reads/writes
	// MaxIdleConns: Keep 4 idle connections ready for reuse
	// ConnMaxLifetime: Recycle connections after 1 hour to prevent stale connections
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(4)
	// Note: ConnMaxLifetime intentionally not set for in-memory databases

	// Apply default and user-provided options
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Sort keys for deterministic execution order (good for testing)
	keys := make([]string, 0, len(cfg.pragmas))
	for k := range cfg.pragmas {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Apply PRAGMA settings
	for _, key := range keys {
		value := cfg.pragmas[key]
		pragmaSQL := fmt.Sprintf("PRAGMA %s=%s", key, value)
		if _, err := db.Exec(pragmaSQL); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma %q: %w", pragmaSQL, err)
		}
	}

	store := &FactStoreDB{
		db:      db,
		dialect: sqliteDialect{},
	}

	if err := store.initSchemaAndStatements(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchemaAndStatements creates the table, indexes, and prepared statements.
func (s *FactStoreDB) initSchemaAndStatements() error {
	// Create facts table with 3 columns for optimal performance
	// atom_hash: UNIQUE constraint ensures deduplication and concurrent safety
	// args: Stored in a format suitable for the dialect (BLOB for SQLite, JSONB for PG)
	createTableSQL := s.dialect.createTableSQL()
	if _, err := s.db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create facts table: %w", err)
	}

	// Create index on predicate for faster GetFacts queries
	createIndexSQL := s.dialect.createIndexSQL()
	if _, err := s.db.Exec(createIndexSQL); err != nil {
		return fmt.Errorf("failed to create predicate index: %w", err)
	}

	// Prepare statement for Add with ON CONFLICT for concurrent safety
	addSQL := s.dialect.addSQL()
	addStmt, err := s.db.Prepare(addSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare add statement: %w", err)
	}
	s.addStmt = addStmt

	// Prepare statement for Remove - simple atom_hash match (dialect-agnostic)
	removeSQL := s.dialect.removeSQL()
	removeStmt, err := s.db.Prepare(removeSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare remove statement: %w", err)
	}
	s.removeStmt = removeStmt

	// Prepare statement for Contains
	containsSQL := s.dialect.containsSQL()
	containsStmt, err := s.db.Prepare(containsSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare contains statement: %w", err)
	}
	s.containsStmt = containsStmt

	return nil
}
