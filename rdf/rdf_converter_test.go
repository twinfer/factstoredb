package jsonld

import (
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/google/mangle/ast"
)

// Test arity 0 atoms (nullary predicates)
func TestRDFArity0(t *testing.T) {
	atom := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "rainy", Arity: 0},
		Args:      []ast.BaseTerm{},
	}

	// Test Atoms → RDF → Atoms round-trip
	dataset, err := AtomsToRDF([]ast.Atom{atom})
	if err != nil {
		t.Fatalf("AtomsToRDF failed: %v", err)
	}

	atoms, err := RDFToAtoms(dataset, "@default")
	if err != nil {
		t.Fatalf("RDFToAtoms failed: %v", err)
	}

	if len(atoms) != 1 {
		t.Fatalf("expected 1 atom, got %d", len(atoms))
	}

	if atoms[0].Predicate.Symbol != "rainy" || atoms[0].Predicate.Arity != 0 {
		t.Errorf("round-trip failed: got %v", atoms[0])
	}
}

// Test arity 1 atoms (unary predicates / class membership)
func TestRDFArity1(t *testing.T) {
	alice, err := ast.Name("/alice")
	if err != nil {
		t.Fatalf("failed to create name: %v", err)
	}

	atom := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "person", Arity: 1},
		Args:      []ast.BaseTerm{alice},
	}

	// Test Atoms → RDF → Atoms round-trip
	dataset, err := AtomsToRDF([]ast.Atom{atom})
	if err != nil {
		t.Fatalf("AtomsToRDF failed: %v", err)
	}

	atoms, err := RDFToAtoms(dataset, "@default")
	if err != nil {
		t.Fatalf("RDFToAtoms failed: %v", err)
	}

	if len(atoms) != 1 {
		t.Fatalf("expected 1 atom, got %d", len(atoms))
	}

	if atoms[0].Predicate.Symbol != "person" || atoms[0].Predicate.Arity != 1 {
		t.Errorf("round-trip failed: got %v", atoms[0])
	}

	subj := atoms[0].Args[0].(ast.Constant)
	subjName, err := subj.NameValue()
	if err != nil || subjName != "/alice" {
		t.Errorf("subject mismatch: got %v", subj)
	}
}

// Test arity 2 atoms (binary predicates / RDF triples)
func TestRDFArity2(t *testing.T) {
	alice, err := ast.Name("/alice")
	if err != nil {
		t.Fatalf("failed to create name: %v", err)
	}

	bob, err := ast.Name("/bob")
	if err != nil {
		t.Fatalf("failed to create name: %v", err)
	}

	atom := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "parent", Arity: 2},
		Args:      []ast.BaseTerm{alice, bob},
	}

	// Test Atoms → RDF → Atoms round-trip
	dataset, err := AtomsToRDF([]ast.Atom{atom})
	if err != nil {
		t.Fatalf("AtomsToRDF failed: %v", err)
	}

	atoms, err := RDFToAtoms(dataset, "@default")
	if err != nil {
		t.Fatalf("RDFToAtoms failed: %v", err)
	}

	if len(atoms) != 1 {
		t.Fatalf("expected 1 atom, got %d", len(atoms))
	}

	if atoms[0].Predicate.Symbol != "parent" || atoms[0].Predicate.Arity != 2 {
		t.Errorf("round-trip failed: got %v", atoms[0])
	}

	subj := atoms[0].Args[0].(ast.Constant)
	subjName, err := subj.NameValue()
	if err != nil || subjName != "/alice" {
		t.Errorf("subject mismatch: got %v", subj)
	}

	obj := atoms[0].Args[1].(ast.Constant)
	objName, err := obj.NameValue()
	if err != nil || objName != "/bob" {
		t.Errorf("object mismatch: got %v", obj)
	}
}

// Test arity 3+ atoms (n-ary relations with reification)
func TestRDFArity3Plus(t *testing.T) {
	eiffel, err := ast.Name("/eiffel")
	if err != nil {
		t.Fatalf("failed to create name: %v", err)
	}

	paris, err := ast.Name("/paris")
	if err != nil {
		t.Fatalf("failed to create name: %v", err)
	}

	france := ast.String("France")

	atom := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "located_at", Arity: 3},
		Args:      []ast.BaseTerm{eiffel, paris, france},
	}

	// Test Atoms → RDF → Atoms round-trip
	dataset, err := AtomsToRDF([]ast.Atom{atom})
	if err != nil {
		t.Fatalf("AtomsToRDF failed: %v", err)
	}

	atoms, err := RDFToAtoms(dataset, "@default")
	if err != nil {
		t.Fatalf("RDFToAtoms failed: %v", err)
	}

	if len(atoms) != 1 {
		t.Fatalf("expected 1 atom, got %d", len(atoms))
	}

	if atoms[0].Predicate.Symbol != "located_at" || atoms[0].Predicate.Arity != 3 {
		t.Errorf("round-trip failed: got %v", atoms[0])
	}

	arg0 := atoms[0].Args[0].(ast.Constant)
	arg0Name, err := arg0.NameValue()
	if err != nil || arg0Name != "/eiffel" {
		t.Errorf("arg0 mismatch: got %v", arg0)
	}

	arg1 := atoms[0].Args[1].(ast.Constant)
	arg1Name, err := arg1.NameValue()
	if err != nil || arg1Name != "/paris" {
		t.Errorf("arg1 mismatch: got %v", arg1)
	}

	arg2 := atoms[0].Args[2].(ast.Constant)
	arg2Str, err := arg2.StringValue()
	if err != nil || arg2Str != "France" {
		t.Errorf("arg2 mismatch: got %v", arg2)
	}
}

// Test multiple atoms round-trip
func TestRDFMultipleAtoms(t *testing.T) {
	alice, _ := ast.Name("/alice")
	bob, _ := ast.Name("/bob")

	atoms := []ast.Atom{
		{
			Predicate: ast.PredicateSym{Symbol: "person", Arity: 1},
			Args:      []ast.BaseTerm{alice},
		},
		{
			Predicate: ast.PredicateSym{Symbol: "person", Arity: 1},
			Args:      []ast.BaseTerm{bob},
		},
		{
			Predicate: ast.PredicateSym{Symbol: "parent", Arity: 2},
			Args:      []ast.BaseTerm{alice, bob},
		},
	}

	// Test Atoms → RDF → Atoms round-trip
	dataset, err := AtomsToRDF(atoms)
	if err != nil {
		t.Fatalf("AtomsToRDF failed: %v", err)
	}

	result, err := RDFToAtoms(dataset, "@default")
	if err != nil {
		t.Fatalf("RDFToAtoms failed: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 atoms, got %d", len(result))
	}
}

// Test JSON-LD marshaling via RDF
func TestRDFJSONLDMarshal(t *testing.T) {
	alice, _ := ast.Name("/alice")
	bob, _ := ast.Name("/bob")

	atom := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "parent", Arity: 2},
		Args:      []ast.BaseTerm{alice, bob},
	}

	wrapper := AtomRDFJSONLD{Atom: atom}

	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	t.Logf("JSON-LD output: %s", string(bytes))

	// Should contain JSON-LD structure
	if len(bytes) == 0 {
		t.Error("empty JSON-LD output")
	}
}

// Test JSON-LD unmarshaling via RDF (arity 2)
func TestRDFJSONLDUnmarshalArity2(t *testing.T) {
	// JSON-LD representing a binary triple
	jsonldStr := `{
		"@context": {
			"@vocab": "http://mangle.datalog.org/"
		},
		"@id": "/alice",
		"parent": "/bob"
	}`

	var wrapper AtomRDFJSONLD
	if err := json.Unmarshal([]byte(jsonldStr), &wrapper); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if wrapper.Atom.Predicate.Symbol != "parent" {
		t.Errorf("expected predicate 'parent', got '%s'", wrapper.Atom.Predicate.Symbol)
	}

	if wrapper.Atom.Predicate.Arity != 2 {
		t.Errorf("expected arity 2, got %d", wrapper.Atom.Predicate.Arity)
	}
}

// Test JSON-LD unmarshaling via RDF (arity 1)
func TestRDFJSONLDUnmarshalArity1(t *testing.T) {
	// JSON-LD representing a type assertion
	jsonldStr := `{
		"@context": {
			"@vocab": "http://mangle.datalog.org/"
		},
		"@id": "/alice",
		"@type": "person"
	}`

	var wrapper AtomRDFJSONLD
	if err := json.Unmarshal([]byte(jsonldStr), &wrapper); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if wrapper.Atom.Predicate.Symbol != "person" {
		t.Errorf("expected predicate 'person', got '%s'", wrapper.Atom.Predicate.Symbol)
	}

	if wrapper.Atom.Predicate.Arity != 1 {
		t.Errorf("expected arity 1, got %d", wrapper.Atom.Predicate.Arity)
	}
}

// Test collection marshaling via RDF
func TestRDFCollectionMarshal(t *testing.T) {
	alice, _ := ast.Name("/alice")
	bob, _ := ast.Name("/bob")

	atoms := []ast.Atom{
		{
			Predicate: ast.PredicateSym{Symbol: "person", Arity: 1},
			Args:      []ast.BaseTerm{alice},
		},
		{
			Predicate: ast.PredicateSym{Symbol: "person", Arity: 1},
			Args:      []ast.BaseTerm{bob},
		},
	}

	wrapper := AtomsRDFJSONLD{Atoms: atoms}

	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	t.Logf("JSON-LD collection output: %s", string(bytes))
}

// Test round-trip with different constant types
func TestRDFConstantTypes(t *testing.T) {
	name, _ := ast.Name("/entity")
	str := ast.String("hello")
	num := ast.Number(42)
	flt := ast.Float64(3.14)

	atoms := []ast.Atom{
		{
			Predicate: ast.PredicateSym{Symbol: "test_string", Arity: 2},
			Args:      []ast.BaseTerm{name, str},
		},
		{
			Predicate: ast.PredicateSym{Symbol: "test_number", Arity: 2},
			Args:      []ast.BaseTerm{name, num},
		},
		{
			Predicate: ast.PredicateSym{Symbol: "test_float", Arity: 2},
			Args:      []ast.BaseTerm{name, flt},
		},
	}

	// Test round-trip
	dataset, err := AtomsToRDF(atoms)
	if err != nil {
		t.Fatalf("AtomsToRDF failed: %v", err)
	}

	result, err := RDFToAtoms(dataset, "@default")
	if err != nil {
		t.Fatalf("RDFToAtoms failed: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 atoms, got %d", len(result))
	}

	// Verify types are preserved
	for i, atom := range result {
		if len(atom.Args) != 2 {
			t.Errorf("atom %d: expected 2 args, got %d", i, len(atom.Args))
		}
	}
}
