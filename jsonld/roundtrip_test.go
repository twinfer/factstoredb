package jsonld

import (
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/google/mangle/ast"
)

// TestRoundTripArity0 tests that arity 0 atoms survive round-trip conversion
func TestRoundTripArity0(t *testing.T) {
	original := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "rainy", Arity: 0},
		Args:      []ast.BaseTerm{},
	}

	// Marshal
	wrapper := AtomJSONLD{Atom: original}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var result AtomJSONLD
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify
	if !atomsEqual(original, result.Atom) {
		t.Errorf("Round-trip failed: expected %v, got %v", original, result.Atom)
	}
}

// TestRoundTripArity1 tests that arity 1 atoms survive round-trip conversion
func TestRoundTripArity1(t *testing.T) {
	alice, _ := ast.Name("/alice")
	original := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "person", Arity: 1},
		Args:      []ast.BaseTerm{alice},
	}

	wrapper := AtomJSONLD{Atom: original}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result AtomJSONLD
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !atomsEqual(original, result.Atom) {
		t.Errorf("Round-trip failed: expected %v, got %v", original, result.Atom)
	}
}

// TestRoundTripArity2 tests that arity 2 atoms survive round-trip conversion
func TestRoundTripArity2(t *testing.T) {
	alice, _ := ast.Name("/alice")
	bob, _ := ast.Name("/bob")
	original := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "parent", Arity: 2},
		Args:      []ast.BaseTerm{alice, bob},
	}

	wrapper := AtomJSONLD{Atom: original}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result AtomJSONLD
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !atomsEqual(original, result.Atom) {
		t.Errorf("Round-trip failed: expected %v, got %v", original, result.Atom)
	}
}

// TestRoundTripArity3Plus tests that arity 3+ atoms survive round-trip conversion
func TestRoundTripArity3Plus(t *testing.T) {
	eiffel, _ := ast.Name("/eiffel")
	paris, _ := ast.Name("/paris")
	france := ast.String("France")

	original := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "located_at", Arity: 3},
		Args:      []ast.BaseTerm{eiffel, paris, france},
	}

	wrapper := AtomJSONLD{Atom: original}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result AtomJSONLD
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !atomsEqual(original, result.Atom) {
		t.Errorf("Round-trip failed: expected %v, got %v", original, result.Atom)
	}
}

// TestRoundTripWithMixedTypes tests atoms with different constant types
func TestRoundTripWithMixedTypes(t *testing.T) {
	name, _ := ast.Name("/item")
	str := ast.String("description")
	num := ast.Number(42)
	flt := ast.Float64(3.14)

	original := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "item_info", Arity: 4},
		Args:      []ast.BaseTerm{name, str, num, flt},
	}

	wrapper := AtomJSONLD{Atom: original}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result AtomJSONLD
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !atomsEqual(original, result.Atom) {
		t.Errorf("Round-trip failed: expected %v, got %v", original, result.Atom)
	}
}

// TestRoundTripWithList tests atoms containing list constants
func TestRoundTripWithList(t *testing.T) {
	list := ast.List([]ast.Constant{
		ast.Number(1),
		ast.Number(2),
		ast.Number(3),
	})
	name, _ := ast.Name("/collection")

	original := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "has_items", Arity: 2},
		Args:      []ast.BaseTerm{name, list},
	}

	wrapper := AtomJSONLD{Atom: original}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result AtomJSONLD
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !atomsEqual(original, result.Atom) {
		t.Errorf("Round-trip failed: expected %v, got %v", original, result.Atom)
	}
}

// TestRoundTripWithPair tests atoms containing pair constants
func TestRoundTripWithPair(t *testing.T) {
	fst := ast.String("key")
	snd := ast.Number(100)
	pair := ast.Pair(&fst, &snd)
	name, _ := ast.Name("/mapping")

	original := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "has_pair", Arity: 2},
		Args:      []ast.BaseTerm{name, pair},
	}

	wrapper := AtomJSONLD{Atom: original}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result AtomJSONLD
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !atomsEqual(original, result.Atom) {
		t.Errorf("Round-trip failed: expected %v, got %v", original, result.Atom)
	}
}

// TestRoundTripWithMap tests atoms containing map constants
func TestRoundTripWithMap(t *testing.T) {
	key1 := ast.String("name")
	val1 := ast.String("Alice")
	key2 := ast.String("age")
	val2 := ast.Number(30)

	kvMap := make(map[*ast.Constant]*ast.Constant)
	kvMap[&key1] = &val1
	kvMap[&key2] = &val2

	mapConst := ast.Map(kvMap)
	name, _ := ast.Name("/person_data")

	original := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "has_map", Arity: 2},
		Args:      []ast.BaseTerm{name, *mapConst},
	}

	wrapper := AtomJSONLD{Atom: original}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result AtomJSONLD
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// For maps, we check arity and predicate since map key order is not guaranteed
	if result.Atom.Predicate != original.Predicate {
		t.Errorf("Predicate mismatch: expected %v, got %v", original.Predicate, result.Atom.Predicate)
	}
	if len(result.Atom.Args) != len(original.Args) {
		t.Errorf("Args length mismatch: expected %d, got %d", len(original.Args), len(result.Atom.Args))
	}
}

// TestRoundTripWithStruct tests atoms containing struct constants
func TestRoundTripWithStruct(t *testing.T) {
	label1 := ast.String("field1")
	val1 := ast.Number(42)
	label2 := ast.String("field2")
	val2 := ast.String("value")

	kvMap := make(map[*ast.Constant]*ast.Constant)
	kvMap[&label1] = &val1
	kvMap[&label2] = &val2

	structConst := ast.Struct(kvMap)
	name, _ := ast.Name("/record")

	original := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "has_struct", Arity: 2},
		Args:      []ast.BaseTerm{name, *structConst},
	}

	wrapper := AtomJSONLD{Atom: original}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result AtomJSONLD
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// For structs, we check arity and predicate since struct field order is not guaranteed
	if result.Atom.Predicate != original.Predicate {
		t.Errorf("Predicate mismatch: expected %v, got %v", original.Predicate, result.Atom.Predicate)
	}
	if len(result.Atom.Args) != len(original.Args) {
		t.Errorf("Args length mismatch: expected %d, got %d", len(original.Args), len(result.Atom.Args))
	}
}

// TestRoundTripCollection tests that collections of atoms survive round-trip
func TestRoundTripCollection(t *testing.T) {
	alice, _ := ast.Name("/alice")
	bob, _ := ast.Name("/bob")
	charlie, _ := ast.Name("/charlie")

	original := []ast.Atom{
		{
			Predicate: ast.PredicateSym{Symbol: "person", Arity: 1},
			Args:      []ast.BaseTerm{alice},
		},
		{
			Predicate: ast.PredicateSym{Symbol: "parent", Arity: 2},
			Args:      []ast.BaseTerm{alice, bob},
		},
		{
			Predicate: ast.PredicateSym{Symbol: "parent", Arity: 2},
			Args:      []ast.BaseTerm{bob, charlie},
		},
	}

	// Marshal
	wrapper := AtomsJSONLD{Atoms: original}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var result AtomsJSONLD
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify length
	if len(result.Atoms) != len(original) {
		t.Fatalf("Length mismatch: expected %d atoms, got %d", len(original), len(result.Atoms))
	}

	// Verify each atom
	for i := range original {
		if !atomsEqual(original[i], result.Atoms[i]) {
			t.Errorf("Atom %d mismatch: expected %v, got %v", i, original[i], result.Atoms[i])
		}
	}
}

// TestRoundTripBytes tests atoms containing bytes constants
func TestRoundTripBytes(t *testing.T) {
	bytes := ast.Bytes([]byte("hello\nworld"))
	name, _ := ast.Name("/data")

	original := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "has_bytes", Arity: 2},
		Args:      []ast.BaseTerm{name, bytes},
	}

	wrapper := AtomJSONLD{Atom: original}
	jsonBytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result AtomJSONLD
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !atomsEqual(original, result.Atom) {
		t.Errorf("Round-trip failed: expected %v, got %v", original, result.Atom)
	}
}

// Helper function to compare atoms
func atomsEqual(a, b ast.Atom) bool {
	if a.Predicate != b.Predicate {
		return false
	}
	if len(a.Args) != len(b.Args) {
		return false
	}
	for i := range a.Args {
		aConst, aOk := a.Args[i].(ast.Constant)
		bConst, bOk := b.Args[i].(ast.Constant)
		if !aOk || !bOk {
			return false
		}
		if !aConst.Equals(bConst) {
			return false
		}
	}
	return true
}
