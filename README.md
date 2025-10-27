# SQLite FactStore for Mangle

This package provides a persistent, high-performance `FactStore` implementation for the Google Mangle Datalog engine, backed by SQLite.

It is designed as a durable, concurrent-safe, and efficient storage layer for Mangle facts (`ast.Atom`), making it suitable for applications where facts must persist beyond the life of the process.

## Features

*   **Persistent Storage**: Uses SQLite to store facts on disk or in memory.
*   **High Performance**: Optimized for speed with `WAL` mode, prepared statements, and a `WITHOUT ROWID` schema.
*   **Efficient Querying**: Stores atom arguments as a binary `JSONB` blob, allowing for fast, indexed pattern matching using SQLite's native JSON functions.
*   **Concurrent Safe**: All operations (`Add`, `Contains`, `Remove`, `GetFacts`) are safe for concurrent use by multiple goroutines.
*   **Full `FactStore` Compliance**: Implements the `factstore.FactStoreWithRemove` interface, integrating seamlessly with Mangle.
*   **Type Fidelity**: Uses a custom JSON marshaller to ensure that all Mangle `ast.Constant` types (including lists, maps, and structs) are serialized and deserialed without losing type information.
*   **Fast Deduplication**: A pre-calculated 64-bit hash (`atom_hash`) for each fact serves as the primary key, enabling extremely fast lookups and atomic, concurrent-safe insertions.

## Schema Design

The fact store uses a single, highly-optimized table to store all facts.

```sql
CREATE TABLE facts (
    predicate TEXT NOT NULL,
    atom_hash BIGINT NOT NULL,
    args BLOB NOT NULL,
    PRIMARY KEY(atom_hash)
) WITHOUT ROWID;

CREATE INDEX idx_predicate ON facts(predicate);
```

### Key Design Choices:

*   **`predicate`**: A simple text key in the format `"symbol_arity"` (e.g., `"parent_2"`). An index on this column makes `GetFacts` queries for a specific predicate very fast.
*   **`atom_hash`**: A unique `BIGINT` hash calculated from the predicate and all arguments. It serves as the `PRIMARY KEY`, and its uniqueness is used for atomic `INSERT ON CONFLICT DO NOTHING` operations, guaranteeing both deduplication and concurrency safety.
*   **`args`**: A `BLOB` column storing the atom's arguments as a binary JSON array (`JSONB`). This is more efficient for storage and querying than text-based JSON.
*   **`WITHOUT ROWID`**: This SQLite optimization makes the table an "index-organized table." The `atom_hash` primary key *is* the table, eliminating a layer of indirection and reducing storage space and lookup time.

## Usage

Here is a basic example of how to create and use the `DBFactStore`.

```go
package main

import (
	"fmt"
	"log"

	"github.com/google/mangle/ast"
	"github.com/google/mangle/parse"
	"twinfer/factstoredb" 
)

func main() {
	// For a persistent, file-based store:
	// store, err := dbfactstore.FactStoreSQLite("./my_facts.db")

	// For an in-memory store (useful for testing):
	store, err := dbfactstore.FactStoreSQLite(":memory:")
	if err != nil {
		log.Fatalf("Failed to create fact store: %v", err)
	}
	defer store.Close()

	// Mangle atoms are created from parsed strings.
	// The `evalAtom` helper evaluates expressions like numbers.
	evalAtom := func(s string) ast.Atom {
		term, _ := parse.Term(s)
		a, _ := functional.EvalAtom(term.(ast.Atom), nil)
		return a
	}

	// 1. Add facts
	// Add returns `true` if the fact was new.
	store.Add(evalAtom("parent(/john, /mary)"))
	store.Add(evalAtom("parent(/john, /bob)"))
	store.Add(evalAtom("age(/mary, 30)"))

	// Adding a duplicate returns `false`.
	wasNew := store.Add(evalAtom("age(/mary, 30)"))
	fmt.Printf("Adding duplicate fact was new? %v\n", wasNew)

	// 2. Check if a fact exists
	fact := evalAtom("parent(/john, /mary)")
	if store.Contains(fact) {
		fmt.Printf("Store contains: %v\n", fact)
	}

	// 3. Query for facts with a pattern
	// 'X' is a variable that matches any value.
	pattern, _ := parse.Term("parent(/john, X)")
	fmt.Println("Children of /john:")
	err = store.GetFacts(pattern.(ast.Atom), func(fact ast.Atom) error {
		// This callback is invoked for each matching fact.
		fmt.Printf("- %v\n", fact)
		return nil
	})
	if err != nil {
		log.Fatalf("GetFacts failed: %v", err)
	}

	// 4. Remove a fact
	// Remove returns `true` if the fact existed and was removed.
	wasRemoved := store.Remove(evalAtom("parent(/john, /bob)"))
	fmt.Printf("Fact 'parent(/john, /bob)' was removed? %v\n", wasRemoved)
	fmt.Printf("Fact count is now: %d\n", store.EstimateFactCount())
}
```

## Performance

As a persistent store, `DBFactStore` has higher latency than Mangle's built-in in-memory stores due to disk I/O and serialization overhead. However, it is heavily optimized for its role.

*   **Writes (`Add`, `Remove`)**: Very fast due to the indexed `atom_hash` and atomic `INSERT ON CONFLICT`.
*   **Reads (`Contains`)**: Extremely fast, as it's a primary key lookup on the hash.
*   **Queries (`GetFacts`)**: Performance depends on the pattern.
    *   Queries on a fully-grounded atom are as fast as `Contains`.
    *   Queries on a predicate (`predicate(X, Y)`) are fast due to the `idx_predicate` index.
    *   Queries with bound arguments (`predicate(/a, X)`) use `json_extract` to filter, which is efficient on the binary JSONB format.

Here is a sample from the benchmark results, comparing against Mangle's `IndexedInMemory` store.

```
goos: darwin
goarch: arm64
cpu: Apple M4

BenchmarkAdd/SQLite-10                      192112          5652 ns/op
BenchmarkAdd/IndexedInMemory-10           18983912          62.03 ns/op

BenchmarkContains/SQLite-10                 347124          3402 ns/op
BenchmarkContains/IndexedInMemory-10      26702194          47.38 ns/op

BenchmarkGetFactsPartialMatch/SQLite-10      10000        104919 ns/op
BenchmarkGetFactsPartialMatch/IndexedInMemory-10    27808917          42.95 ns/op
```

While slower than pure in-memory options, it provides the crucial benefit of persistence, making it ideal for stateful applications.

## Dependencies

*   `github.com/google/mangle`
*   `modernc.org/sqlite` (a pure Go SQLite driver)
*   `github.com/go-json-experiment/json` (for high-performance JSON operations)

## Development

### Running Tests

To run the full test suite:
```bash
go test -v ./...
```

### Running Benchmarks

To run the benchmarks and compare performance against Mangle's standard stores:
```bash
go test -bench=. ./...
```