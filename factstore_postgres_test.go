package factstoredb

import (
	"os"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/mangle/factstore"
)

// runPostgresTests runs a suite of tests against a PostgreSQL-backed store.
// It is skipped unless the POSTGRES_TEST environment variable is set.
func TestPostgresFactStore(t *testing.T) {
	if os.Getenv("POSTGRES_TEST") == "" {
		t.Skip("Skipping PostgreSQL tests. Set POSTGRES_TEST=1 to enable.")
	}

	// Start an embedded PostgreSQL server for the test.
	// This downloads and runs a temporary PostgreSQL instance.
	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().Port(5433))
	if err := postgres.Start(); err != nil {
		t.Fatalf("Failed to start embedded-postgres: %v", err)
	}
	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Errorf("Failed to stop embedded-postgres: %v", err)
		}
	}()

	connStr := "postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable"

	// Factory function for creating a new PostgreSQL-backed store for each test.
	newPostgresStore := func() (factstore.FactStoreWithRemove, error) {
		store, err := NewPostgresFactStore(connStr)
		if err != nil {
			return nil, err
		}
		// Truncate table to ensure a clean state for each test run.
		_, err = store.db.Exec("TRUNCATE facts")
		if err != nil {
			store.Close()
			return nil, err
		}
		return store, nil
	}

	// Run the shared test suite.
	runSuite(t, newPostgresStore)
}
