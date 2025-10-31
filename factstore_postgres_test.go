package factstoredb

import (
	"database/sql"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/mangle/factstore"
)

// runPostgresTests runs a suite of tests against a PostgreSQL-backed store.
// It is skipped unless the POSTGRES_TEST environment variable is set.
func TestPostgresFactStore(t *testing.T) {

	// Start an embedded PostgreSQL server for the test.
	// This downloads and runs a temporary PostgreSQL instance.
	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().Port(5433).Logger(nil).Logger(nil))
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
		store, err := NewFactStorePostgreSQL(connStr)
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

// TestNewFactStorePostgreSQLFromDB tests the FromDB constructor
func TestNewFactStorePostgreSQLFromDB(t *testing.T) {
	// Start an embedded PostgreSQL server for the test.
	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().Port(5433).Logger(nil))
	if err := postgres.Start(); err != nil {
		t.Fatalf("Failed to start embedded-postgres: %v", err)
	}
	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Errorf("Failed to stop embedded-postgres: %v", err)
		}
	}()

	connStr := "postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable"

	// Create a database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create store from the db connection
	store, err := NewFactStorePostgreSQLFromDB(db)
	if err != nil {
		t.Fatalf("Failed to create store from db: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	if store.db == nil {
		t.Fatal("Database connection is nil")
	}

	if store.ownsDB {
		t.Error("Expected store to NOT own the database connection")
	}

	count := store.EstimateFactCount()
	if count != 0 {
		t.Errorf("Expected empty store, got %d facts", count)
	}

	// Verify store works correctly
	atom := evalAtom("test(/foo, 42)")
	if !store.Add(atom) {
		t.Error("Failed to add atom")
	}

	if !store.Contains(atom) {
		t.Error("Store should contain the added atom")
	}

	// Close the store and verify db is still usable
	store.Close()

	// DB should still be open since store doesn't own it
	var result int
	err = db.QueryRow("SELECT COUNT(*) FROM facts").Scan(&result)
	if err != nil {
		t.Errorf("Database should still be usable after store.Close(): %v", err)
	}
	if result != 1 {
		t.Errorf("Expected 1 fact in database, got %d", result)
	}
}
