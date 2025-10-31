package factstoredb

import (
	"database/sql"
	"testing"

	"github.com/google/mangle/ast"
	"github.com/google/mangle/factstore"
)

// TestNewFactStoreSQLite tests the constructor
func TestNewFactStoreSQLite(t *testing.T) {
	store, err := NewFactStoreSQLite(":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	if store.db == nil {
		t.Fatal("Database connection is nil")
	}

	if !store.ownsDB {
		t.Error("Expected store to own the database connection")
	}

	count := store.EstimateFactCount()
	if count != 0 {
		t.Errorf("Expected empty store, got %d facts", count)
	}
}

// TestNewFactStoreSQLiteFromDB tests the FromDB constructor
func TestNewFactStoreSQLiteFromDB(t *testing.T) {
	// Create a database connection
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create store from the db connection
	store, err := NewFactStoreSQLiteFromDB(db)
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

// newSQLiteTestStore is a factory for creating in-memory SQLite stores for testing.
func newSQLiteTestStore() (factstore.FactStoreWithRemove, error) {
	return NewFactStoreSQLite(":memory:")
}

// newSQLiteDBStore is a factory for creating DBFactStore for tests that need the concrete type.
func newSQLiteDBStore() (*FactStoreDB, error) {
	return NewFactStoreSQLite(":memory:")
}

// TestSQLiteFactStore runs the full test suite against the SQLite implementation.
func TestSQLiteFactStore(t *testing.T) {
	runSuite(t, newSQLiteTestStore)

	// Run tests that require the concrete DBFactStore type.
	t.Run("ReadWrite", func(t *testing.T) {
		runReadWriteTest(t, newSQLiteDBStore)
	})

	t.Run("JSONRoundTrip", func(t *testing.T) {
		runJSONRoundTripTest(t)
	})

	// This test is specific to the implementation detail of how atoms are stored
	// and is not part of the generic FactStore interface tests.
	t.Run("UnmarshalAtom", func(t *testing.T) {
		pred := ast.PredicateSym{Symbol: "foo", Arity: 2}
		argsJSON := `["/bar", 42]`
		atom, err := unmarshalAtom(pred, argsJSON)
		if err != nil {
			t.Fatalf("unmarshalAtom failed: %v", err)
		}
		if atom.Predicate != pred {
			t.Errorf("Predicate mismatch: got %v, want %v", atom.Predicate, pred)
		}
		if len(atom.Args) != 2 {
			t.Errorf("Expected 2 args, got %d", len(atom.Args))
		}
	})
}
