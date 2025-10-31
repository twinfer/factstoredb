package factstoredb

import (
	"fmt"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/factstore"
)

// Benchmark helpers

// prepareTestFacts creates a slice of test atoms for benchmarking
func prepareTestFacts(count int) []ast.Atom {
	facts := make([]ast.Atom, count)
	for i := range count {
		facts[i] = evalAtom(fmt.Sprintf("fact(/f%d, %d)", i, i*2))
	}
	return facts
}

// prepareMixedFacts creates facts with different predicates for pattern matching
func prepareMixedFacts(count int) []ast.Atom {
	facts := make([]ast.Atom, count)
	predicates := []string{"person", "parent", "age", "location"}
	for i := range count {
		pred := predicates[i%len(predicates)]
		var atomStr string
		switch pred {
		case "person":
			atomStr = fmt.Sprintf("person(/p%d)", i)
		case "parent":
			atomStr = fmt.Sprintf("parent(/p%d, /p%d)", i, i+1)
		case "age":
			atomStr = fmt.Sprintf("age(/p%d, %d)", i, 20+i%50)
		case "location":
			atomStr = fmt.Sprintf("location(/p%d, /city%d)", i, i%10)
		}
		facts[i] = evalAtom(atomStr)
	}
	return facts
}

// Benchmark: Add operation

func BenchmarkAdd(b *testing.B) {
	b.Run("SQLite", func(b *testing.B) {
		base, _ := NewFactStoreSQLite(":memory:")
		defer base.Close()
		// Prepare a large pool of facts to cycle through
		facts := prepareTestFacts(100000)
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			base.Add(facts[i%len(facts)])
		}
	})

	// Run Postgres benchmarks
	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().Port(5433).Logger(nil).Logger(nil))
	if err := postgres.Start(); err != nil {
		b.Fatalf("Failed to start embedded-postgres: %v", err)
	}
	defer postgres.Stop()
	connStr := "postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable"

	b.Run("Postgres", func(b *testing.B) {
		base, _ := NewFactStorePostgreSQL(connStr)
		defer base.Close()
		facts := prepareTestFacts(100000)
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			base.Add(facts[i%len(facts)])
		}
	})

	b.Run("SimpleInMemory", func(b *testing.B) {
		base := factstore.NewSimpleInMemoryStore()
		store := factstore.NewConcurrentFactStore(&base)
		facts := prepareTestFacts(100000)
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			store.Add(facts[i%len(facts)])
		}
	})
}

// Benchmark: Add duplicate (already exists)

func BenchmarkAddDuplicate(b *testing.B) {
	fact := evalAtom("test(/duplicate, 42)")

	b.Run("SQLite", func(b *testing.B) {
		base, _ := NewFactStoreSQLite(":memory:")
		defer base.Close()
		base.Add(fact) // Add once
		b.ResetTimer()
		for b.Loop() {
			base.Add(fact)
		}
	})

	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().Port(5433).Logger(nil))
	if err := postgres.Start(); err != nil {
		b.Fatalf("Failed to start embedded-postgres: %v", err)
	}
	defer postgres.Stop()
	connStr := "postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable"

	b.Run("Postgres", func(b *testing.B) {
		base, _ := NewFactStorePostgreSQL(connStr)
		defer base.Close()
		base.Add(fact) // Add once
		b.ResetTimer()
		for b.Loop() {
			base.Add(fact)
		}
	})

	b.Run("SimpleInMemory", func(b *testing.B) {
		base := factstore.NewSimpleInMemoryStore()
		store := factstore.NewConcurrentFactStore(&base)
		store.Add(fact)
		b.ResetTimer()
		for b.Loop() {
			store.Add(fact)
		}
	})
}

// Benchmark: Contains operation

func BenchmarkContains(b *testing.B) {
	facts := prepareTestFacts(10000)

	b.Run("SQLite", func(b *testing.B) {
		base, _ := NewFactStoreSQLite(":memory:")
		defer base.Close()
		for _, f := range facts {
			base.Add(f)
		}
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			base.Contains(facts[i%len(facts)])
		}
	})

	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().Port(5433).Logger(nil))
	if err := postgres.Start(); err != nil {
		b.Fatalf("Failed to start embedded-postgres: %v", err)
	}
	defer postgres.Stop()
	connStr := "postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable"

	b.Run("Postgres", func(b *testing.B) {
		base, _ := NewFactStorePostgreSQL(connStr)
		defer base.Close()
		for _, f := range facts {
			base.Add(f)
		}
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			base.Contains(facts[i%len(facts)])
		}
	})

	b.Run("SimpleInMemory", func(b *testing.B) {
		base := factstore.NewSimpleInMemoryStore()
		store := factstore.NewConcurrentFactStore(&base)
		for _, f := range facts {
			store.Add(f)
		}
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			store.Contains(facts[i%len(facts)])
		}
	})
}

// Benchmark: Remove operation

func BenchmarkRemove(b *testing.B) {
	// Prepare more facts than b.N to avoid removing non-existent facts
	facts := prepareTestFacts(100000)

	b.Run("SQLite", func(b *testing.B) {
		base, _ := NewFactStoreSQLite(":memory:")
		defer base.Close()
		for _, f := range facts {
			base.Add(f)
		}
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			base.Remove(facts[i%len(facts)])
		}
	})

	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().Port(5433).Logger(nil))
	if err := postgres.Start(); err != nil {
		b.Fatalf("Failed to start embedded-postgres: %v", err)
	}
	defer postgres.Stop()
	connStr := "postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable"

	b.Run("Postgres", func(b *testing.B) {
		base, _ := NewFactStorePostgreSQL(connStr)
		defer base.Close()
		for _, f := range facts {
			base.Add(f)
		}
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			base.Remove(facts[i%len(facts)])
		}
	})

	b.Run("SimpleInMemory", func(b *testing.B) {
		base := factstore.NewSimpleInMemoryStore()
		store := factstore.NewConcurrentFactStore(&base)
		for _, f := range facts {
			store.Add(f)
		}
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			store.Remove(facts[i%len(facts)])
		}
	})
}

// Benchmark: GetFacts with pattern matching (all facts of a predicate)

func BenchmarkGetFactsAll(b *testing.B) {
	facts := prepareMixedFacts(1000)

	b.Run("SQLite", func(b *testing.B) {
		base, _ := NewFactStoreSQLite(":memory:")
		defer base.Close()
		for _, f := range facts {
			base.Add(f)
		}
		patternAtom := evalAtom("person(X)")
		b.ResetTimer()
		for b.Loop() {
			count := 0
			base.GetFacts(patternAtom, func(ast.Atom) error {
				count++
				return nil
			})
		}
	})

	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().Port(5433).Logger(nil))
	if err := postgres.Start(); err != nil {
		b.Fatalf("Failed to start embedded-postgres: %v", err)
	}
	defer postgres.Stop()
	connStr := "postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable"

	b.Run("Postgres", func(b *testing.B) {
		base, _ := NewFactStorePostgreSQL(connStr)
		defer base.Close()
		for _, f := range facts {
			base.Add(f)
		}
		patternAtom := evalAtom("person(X)")
		b.ResetTimer()
		for b.Loop() {
			count := 0
			base.GetFacts(patternAtom, func(ast.Atom) error {
				count++
				return nil
			})
		}
	})

	b.Run("SimpleInMemory", func(b *testing.B) {
		base := factstore.NewSimpleInMemoryStore()
		store := factstore.NewConcurrentFactStore(&base)
		patternAtom := evalAtom("person(X)")
		b.ResetTimer()
		for b.Loop() {
			count := 0
			store.GetFacts(patternAtom, func(ast.Atom) error {
				count++
				return nil
			})
		}
	})
}

// Benchmark: GetFacts with partial pattern (one arg bound)

func BenchmarkGetFactsPartialMatch(b *testing.B) {
	facts := prepareMixedFacts(1000)

	b.Run("SQLite", func(b *testing.B) {
		base, _ := NewFactStoreSQLite(":memory:")
		defer base.Close()
		for _, f := range facts {
			base.Add(f)
		}
		patternAtom := evalAtom("parent(/p500, X)")
		b.ResetTimer()
		for b.Loop() {
			count := 0
			base.GetFacts(patternAtom, func(ast.Atom) error {
				count++
				return nil
			})
		}
	})

	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().Port(5433).Logger(nil))
	if err := postgres.Start(); err != nil {
		b.Fatalf("Failed to start embedded-postgres: %v", err)
	}
	defer postgres.Stop()
	connStr := "postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable"

	b.Run("Postgres", func(b *testing.B) {
		base, _ := NewFactStorePostgreSQL(connStr)
		defer base.Close()
		for _, f := range facts {
			base.Add(f)
		}
		patternAtom := evalAtom("parent(/p500, X)")
		b.ResetTimer()
		for b.Loop() {
			count := 0
			base.GetFacts(patternAtom, func(ast.Atom) error {
				count++
				return nil
			})
		}
	})

	b.Run("SimpleInMemory", func(b *testing.B) {
		base := factstore.NewSimpleInMemoryStore()
		store := factstore.NewConcurrentFactStore(&base)
		patternAtom := evalAtom("parent(/p500, X)")
		b.ResetTimer()
		for b.Loop() {
			count := 0
			store.GetFacts(patternAtom, func(ast.Atom) error {
				count++
				return nil
			})
		}
	})
}

// Benchmark: Merge operation

func BenchmarkMerge(b *testing.B) {
	sourceFacts := prepareTestFacts(1000)

	b.Run("SQLite", func(b *testing.B) {
		// Create source store
		sourceBase := factstore.NewSimpleInMemoryStore()
		source := factstore.NewConcurrentFactStore(&sourceBase)
		for _, f := range sourceFacts {
			source.Add(f)
		}

		b.ResetTimer()
		for b.Loop() {
			b.StopTimer()
			targetBase, _ := NewFactStoreSQLite(":memory:")
			b.StartTimer()
			targetBase.Merge(&sourceBase)
			b.StopTimer()
			targetBase.Close()
			b.StartTimer()
		}
	})

	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().Port(5433).Logger(nil))
	if err := postgres.Start(); err != nil {
		b.Fatalf("Failed to start embedded-postgres: %v", err)
	}
	defer postgres.Stop()
	connStr := "postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable"

	sourceBase := factstore.NewSimpleInMemoryStore()
	for _, f := range sourceFacts {
		sourceBase.Add(f)
	}

	b.Run("Postgres", func(b *testing.B) {
		for b.Loop() {
			b.StopTimer()
			targetBase, _ := NewFactStorePostgreSQL(connStr)
			b.StartTimer()
			targetBase.Merge(&sourceBase)
			targetBase.Close()
		}
	})

	b.Run("SimpleInMemory", func(b *testing.B) {
		sourceBase := factstore.NewSimpleInMemoryStore()
		source := factstore.NewConcurrentFactStore(&sourceBase)
		for _, f := range sourceFacts {
			source.Add(f)
		}

		b.ResetTimer()
		for b.Loop() {
			b.StopTimer()
			targetBase := factstore.NewSimpleInMemoryStore()
			target := factstore.NewConcurrentFactStore(&targetBase)
			b.StartTimer()
			target.Merge(&sourceBase)
		}
	})
}

// Benchmark: Mixed workload (Add + Contains + GetFacts)

func BenchmarkMixedWorkload(b *testing.B) {
	facts := prepareTestFacts(1000)

	b.Run("SQLite", func(b *testing.B) {
		base, _ := NewFactStoreSQLite(":memory:")
		defer base.Close()
		patternAtom := evalAtom("fact(X, Y)")
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			// Add
			base.Add(facts[i%len(facts)])
			// Contains
			base.Contains(facts[(i+1)%len(facts)])
			// Query
			if i%10 == 0 {
				base.GetFacts(patternAtom, func(ast.Atom) error { return nil })
			}
		}
	})

	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().Port(5433).Logger(nil))
	if err := postgres.Start(); err != nil {
		b.Fatalf("Failed to start embedded-postgres: %v", err)
	}
	defer postgres.Stop()
	connStr := "postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable"

	b.Run("Postgres", func(b *testing.B) {
		base, _ := NewFactStorePostgreSQL(connStr)
		defer base.Close()
		patternAtom := evalAtom("fact(X, Y)")
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			base.Add(facts[i%len(facts)])
			base.Contains(facts[(i+1)%len(facts)])
			if i%10 == 0 {
				base.GetFacts(patternAtom, func(ast.Atom) error { return nil })
			}
		}
	})

	b.Run("SimpleInMemory", func(b *testing.B) {
		base := factstore.NewSimpleInMemoryStore()
		store := factstore.NewConcurrentFactStore(&base)
		for _, f := range facts {
			store.Add(f)
		}
		patternAtom := evalAtom("fact(X, Y)")
		b.ResetTimer()
		for i := 0; b.Loop(); i++ {
			store.Add(facts[i%len(facts)])
			store.Contains(facts[(i+1)%len(facts)])
			if i%10 == 0 {
				store.GetFacts(patternAtom, func(ast.Atom) error { return nil })
			}
		}
	})
}
