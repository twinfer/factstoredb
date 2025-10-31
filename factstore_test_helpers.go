package factstoredb

import (
	"fmt"
	"strings"
	"testing"

	"bitbucket.org/creachadair/stringset"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/factstore"
	"github.com/google/mangle/functional"
	"github.com/google/mangle/parse"
)

// Test helper functions (following Mangle's pattern)

// atom parses a string into an ast.Atom using Mangle's parser
func atom(s string) ast.Atom {
	term, err := parse.Term(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse %q: %v", s, err))
	}
	return term.(ast.Atom)
}

// evalAtom parses and evaluates a string into an ast.Atom
func evalAtom(s string) ast.Atom {
	term, err := parse.Term(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse %q: %v", s, err))
	}
	eval, err := functional.EvalAtom(term.(ast.Atom), nil)
	if err != nil {
		panic(fmt.Sprintf("failed to eval %q: %v", s, err))
	}
	return eval
}

// evalConstant parses a string and evaluates it into an ast.Constant.
// This is useful for creating complex constants like maps or lists for tests.
func evalConstant(s string) ast.Constant {
	term, err := parse.Term(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse %q: %v", s, err))
	}
	baseTerm, ok := term.(ast.BaseTerm)
	if !ok {
		panic(fmt.Sprintf("expected a base term, but got %T for %q", term, s))
	}
	eval, err := functional.EvalExpr(baseTerm, nil)
	if err != nil {
		panic(fmt.Sprintf("failed to eval %q: %v", s, err))
	}
	return eval.(ast.Constant)
}

// runAddContainsRemoveTest tests comprehensive add/contains operations.
func runAddContainsRemoveTest(t *testing.T, store factstore.FactStoreWithRemove) {
	// Test atoms covering all constant types
	tests := []ast.Atom{
		// 0-arity
		atom("baz()"),
		// Simple names
		atom("foo(/bar)"),
		atom("foo(/zzz)"),
		// Multiple args
		atom("bar(/abc)"),
		atom("bar(/bar,/baz)"),
		atom("bar(/bar,/def)"),
		atom("bar(/abc,/def)"),
		// Numbers
		evalAtom("num(42)"),
		evalAtom("num(1)"),
		evalAtom("num(-123)"),
		// Float64
		evalAtom("flt(3.14)"),
		evalAtom("flt(-2.71)"),
		// Strings
		evalAtom(`str("hello")`),
		evalAtom(`str("world")`),
		// Lists
		evalAtom("lst([/abc])"),
		evalAtom("lst([/abc, /def])"),
		evalAtom("bar([/abc],1,/def)"),
		evalAtom("bar([/abc, /def],1,/def)"),
		evalAtom("bar([/def, /abc],1,/def)"),
		// Maps
		evalAtom("qaz([/abc : 123,  /def : 345], 1, /def)"),
		evalAtom("qaz([/abc : 456,  /def : 789], 1, /def)"),
		// Structs
		evalAtom("baz({/abc : 1,  /def : 2}, 1, /def)"),
		evalAtom("baz({/abc : 1,  /def : 3}, 1, /def)"),
		// Special characters - unicode
		evalAtom("uni('=$')"),
		// Special characters - newlines in strings
		evalAtom(`special("\n/bar")`),
		// Binary data
		evalAtom(`bin(b'\x80\x81')`),
	}

	for _, testAtom := range tests {
		t.Run(testAtom.String(), func(t *testing.T) {
			// First add should return true
			if got := store.Add(testAtom); !got {
				t.Errorf("Add(%v)=%v want %v", testAtom, got, true)
			}

			// Should be contained
			if !store.Contains(testAtom) {
				t.Errorf("Contains(%v)=false want true", testAtom)
			}

			// Second add should return false (already exists)
			if got := store.Add(testAtom); got {
				t.Errorf("Add(%v)=%v want %v (second add)", testAtom, got, false)
			}
		})
	}

	// Verify final count
	if got, want := store.EstimateFactCount(), len(tests); got != want {
		t.Errorf("EstimateFactCount() = %d want %d", got, want)
	}
}

// runOrderInsensitiveHashingTest verifies that maps and structs with different
// key-value orderings are treated as the same fact.
func runOrderInsensitiveHashingTest(t *testing.T, store factstore.FactStoreWithRemove) {
	map1 := evalAtom(`data([/a:1, /b:"foo"])`)
	map2 := evalAtom(`data([/b:"foo", /a:1])`) // Same as map1, different order

	struct1 := evalAtom(`data({/x: [1,2], /y: /z})`)
	struct2 := evalAtom(`data({/y: /z, /x: [1,2]})`) // Same as struct1, different order

	// Test maps
	if !store.Add(map1) {
		t.Errorf("Add(%v) should return true for new map", map1)
	}
	if store.Add(map2) {
		t.Errorf("Add(%v) should return false for equivalent map", map2)
	}
	if !store.Contains(map1) {
		t.Errorf("Contains(%v) should be true", map1)
	}
	if !store.Contains(map2) {
		t.Errorf("Contains(%v) should be true for equivalent map", map2)
	}
	if count := store.EstimateFactCount(); count != 1 {
		t.Errorf("EstimateFactCount() = %d, want 1 after adding equivalent maps", count)
	}

	// Test structs
	if !store.Add(struct1) {
		t.Errorf("Add(%v) should return true for new struct", struct1)
	}
	if store.Add(struct2) {
		t.Errorf("Add(%v) should return false for equivalent struct", struct2)
	}
	if !store.Contains(struct1) {
		t.Errorf("Contains(%v) should be true", struct1)
	}
	if !store.Contains(struct2) {
		t.Errorf("Contains(%v) should be true for equivalent struct", struct2)
	}
	if count := store.EstimateFactCount(); count != 2 {
		t.Errorf("EstimateFactCount() = %d, want 2 after adding structs", count)
	}

	// Test removal
	if !store.Remove(map2) {
		t.Errorf("Remove(%v) should return true for equivalent map", map2)
	}
	if store.Contains(map1) {
		t.Errorf("Contains(%v) should be false after removing equivalent map", map1)
	}
}

// runGetFactsPatternMatchingTest tests pattern matching with variables.
func runGetFactsPatternMatchingTest(t *testing.T, store factstore.FactStoreWithRemove) {
	// Add test facts
	testFacts := []ast.Atom{
		atom("baz()"),
		atom("foo(/bar)"),
		atom("foo(/zzz)"),
		atom("bar(/bar,/baz)"),
		atom("bar(/bar,/def)"),
		atom("bar(/abc,/def)"),
		evalAtom("bar([/abc],1,/def)"),
		evalAtom("bar([/abc, /def],1,/def)"),
	}

	for _, f := range testFacts {
		store.Add(f)
	}

	// Table-driven tests for pattern matching
	tests := []struct {
		pattern string
		want    stringset.Set
	}{
		{
			pattern: "baz()",
			want:    stringset.New("baz()"),
		},
		{
			pattern: "baz(X)",
			want:    stringset.New(), // 0-arity, doesn't match with 1 arg
		},
		{
			pattern: "baaaaz()",
			want:    stringset.New(), // Non-existent predicate
		},
		{
			pattern: "foo(/bar)",
			want:    stringset.New("foo(/bar)"),
		},
		{
			pattern: "foo(/abc)",
			want:    stringset.New(), // Non-existent fact
		},
		{
			pattern: "fooooo(/bar)",
			want:    stringset.New(), // Non-existent predicate
		},
		{
			pattern: "foo(X)",
			want:    stringset.New("foo(/bar)", "foo(/zzz)"),
		},
		{
			pattern: "bar(/bar,X)",
			want:    stringset.New("bar(/bar,/baz)", "bar(/bar,/def)"),
		},
		{
			pattern: "bar(X,Y)",
			want:    stringset.New("bar(/bar,/baz)", "bar(/bar,/def)", "bar(/abc,/def)"),
		},
		{
			pattern: "bar(X, /def)",
			want:    stringset.New("bar(/bar,/def)", "bar(/abc,/def)"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := stringset.New()
			err := store.GetFacts(atom(tt.pattern), func(fact ast.Atom) error {
				got.Add(fact.String())
				return nil
			})
			if err != nil {
				t.Fatalf("GetFacts(%q) error: %v", tt.pattern, err)
			}

			if !got.Equals(tt.want) {
				t.Errorf("GetFacts(%q) = %v want %v", tt.pattern, got, tt.want)
			}
		})
	}
}

// runNonGroundedAtomsTest tests that non-grounded atoms cannot be added.
func runNonGroundedAtomsTest(t *testing.T, store factstore.FactStoreWithRemove) {
	// Atoms with variables should not be addable
	tests := []ast.Atom{
		atom("foo(X)"),
		atom("bar(X, /def)"),
		atom("baz(X, Y, Z)"),
	}

	for _, testAtom := range tests {
		t.Run(testAtom.String(), func(t *testing.T) {
			if got := store.Add(testAtom); got {
				t.Errorf("Add(%v)=%v want %v (should not add non-grounded)", testAtom, got, false)
			}

			if store.Contains(testAtom) {
				t.Errorf("Contains(%v)=true want false (should not contain non-grounded)", testAtom)
			}
		})
	}
}

// runListPredicatesTest tests predicate listing.
func runListPredicatesTest(t *testing.T, store factstore.FactStoreWithRemove) {
	// Initially empty
	predicates := store.ListPredicates()
	if len(predicates) != 0 {
		t.Errorf("Expected 0 predicates, got %d", len(predicates))
	}

	// Add facts with different predicates
	store.Add(atom("parent(/john, /mary)"))
	store.Add(atom("parent(/john, /bob)"))
	store.Add(atom("gender(/john, /male)"))
	store.Add(atom("age(/john, 42)"))

	predicates = store.ListPredicates()
	if len(predicates) != 3 {
		t.Errorf("Expected 3 predicates, got %d", len(predicates))
	}

	// Verify predicates are correct
	predSet := stringset.New()
	for _, p := range predicates {
		predSet.Add(p.String())
	}
	want := stringset.New("parent(A0, A1)", "gender(A0, A1)", "age(A0, A1)")
	if !predSet.Equals(want) {
		t.Errorf("ListPredicates() = %v want %v", predSet, want)
	}
}

// runMergeTest tests merging two stores.
func runMergeTest(t *testing.T, newStore func() (factstore.FactStoreWithRemove, error)) {
	store1, err := newStore()
	if err != nil {
		t.Fatalf("Failed to create store1: %v", err)
	}
	defer store1.(interface{ Close() error }).Close()

	store2, err := newStore()
	if err != nil {
		t.Fatalf("Failed to create store2: %v", err)
	}
	defer store2.(interface{ Close() error }).Close()

	// Add facts to both stores with some overlap
	store1.Add(atom("foo(/bar)"))
	store1.Add(atom("foo(/existing)"))

	store2.Add(atom("foo(/bar)"))   // Duplicate
	store2.Add(atom("foo(/new)"))   // New fact
	store2.Add(atom("baz(/other)")) // Different predicate

	// Merge store2 into store1
	store1.Merge(store2)

	// Verify all unique facts are present
	expectedFacts := []ast.Atom{
		atom("foo(/bar)"),
		atom("foo(/existing)"),
		atom("foo(/new)"),
		atom("baz(/other)"),
	}

	for _, fact := range expectedFacts {
		if !store1.Contains(fact) {
			t.Errorf("After merge, store1 should contain %v", fact)
		}
	}

	// Count should be 4 (no duplicate /bar)
	if got := store1.EstimateFactCount(); got != 4 {
		t.Errorf("EstimateFactCount() = %d want 4", got)
	}
}

// runRoundTripTest tests adding facts, querying all, and verifying completeness.
func runRoundTripTest(t *testing.T, store factstore.FactStoreWithRemove) {
	// Add comprehensive set of facts
	facts := []ast.Atom{
		atom("baz()"),
		atom("foo(/bar)"),
		evalAtom("num(42)"),
		evalAtom(`str("hello")`),
		evalAtom("lst([/a, /b, /c])"),
		evalAtom("map([/key : /value])"),
		evalAtom("struct({/x : 1, /y : 2})"),
		evalAtom(`special("\n\t")`),
		evalAtom("uni('=$')"),
		evalAtom(`bin(b'\x80\x81')`),
	}

	for _, f := range facts {
		store.Add(f)
	}

	// Query all predicates and collect all facts
	retrieved := stringset.New()
	for _, pred := range store.ListPredicates() {
		// Create query with all variables
		query := ast.NewQuery(pred)
		err := store.GetFacts(query, func(fact ast.Atom) error {
			retrieved.Add(fact.String())
			return nil
		})
		if err != nil {
			t.Fatalf("GetFacts error: %v", err)
		}
	}

	// Verify we got all facts back
	want := stringset.New()
	for _, f := range facts {
		want.Add(f.String())
	}

	if !retrieved.Equals(want) {
		t.Errorf("Round trip failed.")
		for fact := range want {
			if !retrieved.Contains(fact) {
				t.Errorf("  Missing fact: %s", fact)
			}
		}
		for fact := range retrieved {
			if !want.Contains(fact) {
				t.Errorf("  Extra fact: %s", fact)
			}
		}
	}
}

// runComplexTypesTest tests all complex constant types.
func runComplexTypesTest(t *testing.T, store factstore.FactStoreWithRemove) {
	tests := []struct {
		name string
		atom ast.Atom
	}{
		{"list-empty", evalAtom("test([])")},
		{"list-single", evalAtom("test([/a])")},
		{"list-multiple", evalAtom("test([/a, /b, /c])")},
		{"list-nested", evalAtom("test([[/a, /b], [/c, /d]])")},
		{"map-single", evalAtom("test([/k : /v])")},
		{"map-multiple", evalAtom("test([/a : 1, /b : 2, /c : 3])")},
		{"struct-empty", evalAtom("test({})")},
		{"struct-single", evalAtom("test({/field : /value})")},
		{"struct-multiple", evalAtom("test({/x : 1, /y : 2, /z : 3})")},
		{"mixed", evalAtom(`test([/a, /b], {/x : 1}, "str")`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !store.Add(tt.atom) {
				t.Errorf("Failed to add %v", tt.atom)
			}

			if !store.Contains(tt.atom) {
				t.Errorf("Store should contain %v", tt.atom)
			}

			// Query and verify
			got := stringset.New()
			pattern := ast.NewQuery(tt.atom.Predicate)
			store.GetFacts(pattern, func(fact ast.Atom) error {
				got.Add(fact.String())
				return nil
			})

			if !got.Contains(tt.atom.String()) {
				t.Errorf("Query didn't return added fact: %v", tt.atom)
			}
		})
	}
}

// runRemoveTest tests the Remove method.
func runRemoveTest(t *testing.T, store factstore.FactStoreWithRemove) {
	tests := []struct {
		name string
		atom ast.Atom
	}{
		{"0-arity", atom("baz()")},
		{"name", atom("foo(/bar)")},
		{"number", evalAtom("num(42)")},
		{"float", evalAtom("flt(3.14)")},
		{"string", evalAtom(`str("hello")`)},
		{"list", evalAtom("lst([/a, /b, /c])")},
		{"map", evalAtom("map([/key : /value])")},
		{"struct", evalAtom("struct({/x : 1, /y : 2})")},
		{"special-chars", evalAtom(`special("\n\t")`)},
		{"binary", evalAtom(`bin(b'\x80\x81')`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add fact
			if !store.Add(tt.atom) {
				t.Errorf("Failed to add %v", tt.atom)
			}

			// Verify it's there
			if !store.Contains(tt.atom) {
				t.Errorf("Store should contain %v after Add", tt.atom)
			}

			// Remove fact - should return true
			if !store.Remove(tt.atom) {
				t.Errorf("Remove(%v) should return true (fact exists)", tt.atom)
			}

			// Verify it's gone
			if store.Contains(tt.atom) {
				t.Errorf("Store should not contain %v after Remove", tt.atom)
			}

			// Remove again - should return false
			if store.Remove(tt.atom) {
				t.Errorf("Remove(%v) should return false (fact already removed)", tt.atom)
			}
		})
	}
}

// runReadWriteTest tests the ReadFrom and WriteTo methods for streaming JSON.
func runReadWriteTest(t *testing.T, newStore func() (*FactStoreDB, error)) {
	// Setup: Create a source store with facts
	store1, err := newStore()
	if err != nil {
		t.Fatalf("Failed to create store1: %v", err)
	}
	defer store1.Close()

	// Add a variety of facts
	facts := []ast.Atom{
		atom("foo(/bar)"),
		atom("foo(/baz)"),
		evalAtom("num(42)"),
		evalAtom(`str("hello")`),
		evalAtom("lst([/a, /b])"),
		evalAtom("map([/key : /value])"),
		evalAtom("struct({/x : 1})"),
		evalAtom(`bin(b'\x01\x02\x03')`),
	}

	for _, fact := range facts {
		store1.Add(fact)
	}

	// Step 1: Export store1 to a buffer using WriteTo
	var buf strings.Builder
	n, err := store1.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	if n == 0 {
		t.Error("WriteTo returned 0 bytes written")
	}
	jsonOutput := buf.String()

	// Step 2: Create a new empty store (store2)
	store2, err := newStore()
	if err != nil {
		t.Fatalf("Failed to create store2: %v", err)
	}
	defer store2.Close()

	// Step 3: Import the JSON into store2 using ReadFrom
	bytesRead, err := store2.ReadFrom(strings.NewReader(jsonOutput))
	if err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}

	// Verify bytes read
	if bytesRead != int64(len(jsonOutput)) {
		t.Errorf("ReadFrom reported %d bytes read, but input was %d bytes", bytesRead, len(jsonOutput))
	}

	// Step 4: Verify that store2 contains all the original facts
	if count := store2.EstimateFactCount(); count != len(facts) {
		t.Errorf("After import, expected %d facts, got %d", len(facts), count)
	}
	for _, fact := range facts {
		if !store2.Contains(fact) {
			t.Errorf("After import, store should contain %v", fact)
		}
	}
}

// runJSONRoundTripTest verifies that the custom JSON marshaller works correctly.
func runJSONRoundTripTest(t *testing.T) {
	// Test Atom round trips by constructing atoms directly
	alice, _ := ast.Name("/alice")
	bob, _ := ast.Name("/bob")

	tests := []struct {
		name string
		atom ast.Atom
	}{
		{
			name: "atom_list_arg",
			atom: ast.Atom{
				Predicate: ast.PredicateSym{Symbol: "tags", Arity: 2},
				Args: []ast.BaseTerm{
					alice,
					ast.List([]ast.Constant{ast.Number(1), ast.Number(2), ast.Number(3)}),
				},
			},
		},
		{
			name: "atom_map_arg",
			atom: ast.Atom{
				Predicate: ast.PredicateSym{Symbol: "data", Arity: 1},
				Args:      []ast.BaseTerm{evalConstant(`[/a:1, /b:"foo"]`)},
			},
		},
		{
			name: "atom_struct_arg",
			atom: ast.Atom{
				Predicate: ast.PredicateSym{Symbol: "config", Arity: 1},
				Args:      []ast.BaseTerm{evalConstant(`{/enabled: /true, /retries: 3}`)},
			},
		},
		{
			// ast.Pair returns a constant, which is a valid BaseTerm.
			// Note: ast.Pair takes pointers to constants.
			// We create them on the fly here.
			name: "atom_pair_arg",
			atom: ast.Atom{
				Predicate: ast.PredicateSym{Symbol: "link", Arity: 1},
				Args: []ast.BaseTerm{
					ast.Pair(&alice, &bob),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := tt.atom

			// Marshal to JSON using atomJSON
			var buf strings.Builder
			enc := jsontext.NewEncoder(&buf)
			if err := (atomJSON{original}).MarshalJSONTo(enc); err != nil {
				t.Fatalf("Failed to marshal atom: %v", err)
			}
			jsonStr := buf.String()

			// Unmarshal from JSON
			var unmarshalled atomJSON
			if err := json.Unmarshal([]byte(jsonStr), &unmarshalled); err != nil {
				t.Fatalf("Failed to unmarshal atom JSON: %v", err)
			}

			// Verify equality
			if !original.Equals(unmarshalled.Atom) {
				t.Errorf("Atoms not equal:\n  original=%v\n  unmarshalled=%v", original, unmarshalled.Atom)
			}
		})
	}
}

// runSuite runs all shared tests for a given store implementation.
func runSuite(t *testing.T, newStore func() (factstore.FactStoreWithRemove, error)) {
	t.Run("AddContainsRemove", func(t *testing.T) {
		store, err := newStore()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		t.Cleanup(func() { store.(interface{ Close() error }).Close() })
		runAddContainsRemoveTest(t, store)
	})

	t.Run("OrderInsensitiveHashing", func(t *testing.T) {
		store, err := newStore()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		t.Cleanup(func() { store.(interface{ Close() error }).Close() })
		runOrderInsensitiveHashingTest(t, store)
	})

	t.Run("GetFactsPatternMatching", func(t *testing.T) {
		store, err := newStore()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		t.Cleanup(func() { store.(interface{ Close() error }).Close() })
		runGetFactsPatternMatchingTest(t, store)
	})

	t.Run("NonGroundedAtoms", func(t *testing.T) {
		store, err := newStore()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		t.Cleanup(func() { store.(interface{ Close() error }).Close() })
		runNonGroundedAtomsTest(t, store)
	})

	t.Run("ListPredicates", func(t *testing.T) {
		store, err := newStore()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		t.Cleanup(func() { store.(interface{ Close() error }).Close() })
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
		t.Cleanup(func() { store.(interface{ Close() error }).Close() })
		runRoundTripTest(t, store)
	})

	t.Run("ComplexTypes", func(t *testing.T) {
		store, err := newStore()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		t.Cleanup(func() { store.(interface{ Close() error }).Close() })
		runComplexTypesTest(t, store)
	})

	t.Run("Remove", func(t *testing.T) {
		store, err := newStore()
		if err != nil {
			t.Fatalf("Failed to create store: %v", err)
		}
		t.Cleanup(func() { store.(interface{ Close() error }).Close() })
		runRemoveTest(t, store)
	})
}
