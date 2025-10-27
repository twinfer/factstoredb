package factstoredb

import (
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
	defer store.Close()

	if store.db == nil {
		t.Fatal("Database connection is nil")
	}

	count := store.EstimateFactCount()
	if count != 0 {
		t.Errorf("Expected empty store, got %d facts", count)
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

// runSuite runs all shared tests for a given store implementation.
func runSuite(t *testing.T, newStore func() (factstore.FactStoreWithRemove, error)) {
	t.Run("AddContainsRemove", func(t *testing.T) {
		store, err := newStore()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		defer store.(interface{ Close() error }).Close()
		runAddContainsRemoveTest(t, store)
	})

	t.Run("OrderInsensitiveHashing", func(t *testing.T) {
		store, err := newStore()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		defer store.(interface{ Close() error }).Close()
		runOrderInsensitiveHashingTest(t, store)
	})

	t.Run("GetFactsPatternMatching", func(t *testing.T) {
		store, err := newStore()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		defer store.(interface{ Close() error }).Close()
		runGetFactsPatternMatchingTest(t, store)
	})

	t.Run("NonGroundedAtoms", func(t *testing.T) {
		store, err := newStore()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		defer store.(interface{ Close() error }).Close()
		runNonGroundedAtomsTest(t, store)
	})

	t.Run("ListPredicates", func(t *testing.T) {
		store, err := newStore()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		defer store.(interface{ Close() error }).Close()
		runListPredicatesTest(t, store)
	})

	t.Run("Merge", func(t *testing.T) {
		runMergeTest(t, newStore)
	})

	t.Run("RoundTrip", func(t *testing.T) {
		store, err := newStore()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		defer store.(interface{ Close() error }).Close()
		runRoundTripTest(t, store)
	})

	t.Run("ComplexTypes", func(t *testing.T) {
		store, err := newStore()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		defer store.(interface{ Close() error }).Close()
		runComplexTypesTest(t, store)
	})

	t.Run("Remove", func(t *testing.T) {
		store, err := newStore()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		defer store.(interface{ Close() error }).Close()
		runRemoveTest(t, store)
	})
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
