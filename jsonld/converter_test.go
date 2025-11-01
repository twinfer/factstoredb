package jsonld

import (
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/google/mangle/ast"
)

// Test arity 0 atoms (nullary predicates)
func TestArity0ToJSONLD(t *testing.T) {
	atom := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "rainy", Arity: 0},
		Args:      []ast.BaseTerm{},
	}

	wrapper := AtomJSONLD{Atom: atom}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal arity 0 atom: %v", err)
	}

	// Unmarshal to check structure
	var doc map[string]interface{}
	if err := json.Unmarshal(bytes, &doc); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Check @type
	typeVal, ok := doc["@type"]
	if !ok {
		t.Fatal("Missing @type in arity 0 conversion")
	}
	if typeVal != "rainy" {
		t.Errorf("Expected @type='rainy', got %v", typeVal)
	}

	// Arity 0 should NOT have @id (as per user's requirement)
	if _, hasID := doc["@id"]; hasID {
		t.Error("Arity 0 should NOT have @id")
	}
}

// Test arity 1 atoms (unary predicates - class membership)
func TestArity1ToJSONLD(t *testing.T) {
	alice, _ := ast.Name("/alice")
	atom := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "person", Arity: 1},
		Args:      []ast.BaseTerm{alice},
	}

	wrapper := AtomJSONLD{Atom: atom}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal arity 1 atom: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(bytes, &doc); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Check @id
	idVal, ok := doc["@id"]
	if !ok {
		t.Fatal("Missing @id in arity 1 conversion")
	}
	if idVal != "/alice" {
		t.Errorf("Expected @id='/alice', got %v", idVal)
	}

	// Check @type
	typeVal, ok := doc["@type"]
	if !ok {
		t.Fatal("Missing @type in arity 1 conversion")
	}
	if typeVal != "person" {
		t.Errorf("Expected @type='person', got %v", typeVal)
	}
}

// Test arity 2 atoms (binary predicates - RDF triples)
func TestArity2ToJSONLD(t *testing.T) {
	alice, _ := ast.Name("/alice")
	bob, _ := ast.Name("/bob")
	atom := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "parent", Arity: 2},
		Args:      []ast.BaseTerm{alice, bob},
	}

	wrapper := AtomJSONLD{Atom: atom}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal arity 2 atom: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(bytes, &doc); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Check @id (subject)
	idVal, ok := doc["@id"]
	if !ok {
		t.Fatal("Missing @id in arity 2 conversion")
	}
	if idVal != "/alice" {
		t.Errorf("Expected @id='/alice', got %v", idVal)
	}

	// Check predicate property (object)
	parentVal, ok := doc["parent"]
	if !ok {
		t.Fatal("Missing 'parent' property in arity 2 conversion")
	}
	if parentVal != "/bob" {
		t.Errorf("Expected parent='/bob', got %v", parentVal)
	}

	// Should NOT have @type for arity 2
	if _, hasType := doc["@type"]; hasType {
		t.Error("Arity 2 should not have @type")
	}
}

// Test arity 3+ atoms (n-ary relations)
func TestArity3ToJSONLD(t *testing.T) {
	eiffel, _ := ast.Name("/eiffel")
	paris, _ := ast.Name("/paris")
	france := ast.String("France")

	atom := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "located_at", Arity: 3},
		Args:      []ast.BaseTerm{eiffel, paris, france},
	}

	wrapper := AtomJSONLD{Atom: atom}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal arity 3 atom: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(bytes, &doc); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Check @type
	typeVal, ok := doc["@type"]
	if !ok {
		t.Fatal("Missing @type in arity 3 conversion")
	}
	if typeVal != "located_at" {
		t.Errorf("Expected @type='located_at', got %v", typeVal)
	}

	// Check arg0
	arg0Val, ok := doc["arg0"]
	if !ok {
		t.Fatal("Missing arg0 in arity 3 conversion")
	}
	if arg0Val != "/eiffel" {
		t.Errorf("Expected arg0='/eiffel', got %v", arg0Val)
	}

	// Check arg1
	arg1Val, ok := doc["arg1"]
	if !ok {
		t.Fatal("Missing arg1 in arity 3 conversion")
	}
	if arg1Val != "/paris" {
		t.Errorf("Expected arg1='/paris', got %v", arg1Val)
	}

	// Check arg2
	arg2Val, ok := doc["arg2"]
	if !ok {
		t.Fatal("Missing arg2 in arity 3 conversion")
	}
	if arg2Val != "France" {
		t.Errorf("Expected arg2='France', got %v", arg2Val)
	}
}

// Test reverse conversion: arity 0
func TestJSONLDToArity0(t *testing.T) {
	jsonStr := `{"@context": {}, "@type": "rainy"}`

	var wrapper AtomJSONLD
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	atom := wrapper.Atom

	if atom.Predicate.Symbol != "rainy" {
		t.Errorf("Expected predicate 'rainy', got %s", atom.Predicate.Symbol)
	}
	if atom.Predicate.Arity != 0 {
		t.Errorf("Expected arity 0, got %d", atom.Predicate.Arity)
	}
	if len(atom.Args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(atom.Args))
	}
}

// Test reverse conversion: arity 1
func TestJSONLDToArity1(t *testing.T) {
	jsonStr := `{"@context": {}, "@id": "/alice", "@type": "person"}`

	var wrapper AtomJSONLD
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	atom := wrapper.Atom

	if atom.Predicate.Symbol != "person" {
		t.Errorf("Expected predicate 'person', got %s", atom.Predicate.Symbol)
	}
	if atom.Predicate.Arity != 1 {
		t.Errorf("Expected arity 1, got %d", atom.Predicate.Arity)
	}
	if len(atom.Args) != 1 {
		t.Errorf("Expected 1 arg, got %d", len(atom.Args))
	}

	arg0, ok := atom.Args[0].(ast.Constant)
	if !ok {
		t.Fatal("arg0 is not a constant")
	}
	name, err := arg0.NameValue()
	if err != nil {
		t.Fatalf("arg0 is not a Name: %v", err)
	}
	if name != "/alice" {
		t.Errorf("Expected arg0='/alice', got %s", name)
	}
}

// Test reverse conversion: arity 2
func TestJSONLDToArity2(t *testing.T) {
	jsonStr := `{"@context": {}, "@id": "/alice", "parent": "/bob"}`

	var wrapper AtomJSONLD
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	atom := wrapper.Atom

	if atom.Predicate.Symbol != "parent" {
		t.Errorf("Expected predicate 'parent', got %s", atom.Predicate.Symbol)
	}
	if atom.Predicate.Arity != 2 {
		t.Errorf("Expected arity 2, got %d", atom.Predicate.Arity)
	}
	if len(atom.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(atom.Args))
	}

	arg0, _ := atom.Args[0].(ast.Constant)
	name0, _ := arg0.NameValue()
	if name0 != "/alice" {
		t.Errorf("Expected arg0='/alice', got %s", name0)
	}

	arg1, _ := atom.Args[1].(ast.Constant)
	name1, _ := arg1.NameValue()
	if name1 != "/bob" {
		t.Errorf("Expected arg1='/bob', got %s", name1)
	}
}

// Test reverse conversion: arity 3+
func TestJSONLDToArity3(t *testing.T) {
	jsonStr := `{"@context": {}, "@type": "located_at", "arg0": "/eiffel", "arg1": "/paris", "arg2": "France"}`

	var wrapper AtomJSONLD
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	atom := wrapper.Atom

	if atom.Predicate.Symbol != "located_at" {
		t.Errorf("Expected predicate 'located_at', got %s", atom.Predicate.Symbol)
	}
	if atom.Predicate.Arity != 3 {
		t.Errorf("Expected arity 3, got %d", atom.Predicate.Arity)
	}
	if len(atom.Args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(atom.Args))
	}

	arg0, _ := atom.Args[0].(ast.Constant)
	name0, _ := arg0.NameValue()
	if name0 != "/eiffel" {
		t.Errorf("Expected arg0='/eiffel', got %s", name0)
	}

	arg1, _ := atom.Args[1].(ast.Constant)
	name1, _ := arg1.NameValue()
	if name1 != "/paris" {
		t.Errorf("Expected arg1='/paris', got %s", name1)
	}

	arg2, _ := atom.Args[2].(ast.Constant)
	str2, _ := arg2.StringValue()
	if str2 != "France" {
		t.Errorf("Expected arg2='France', got %s", str2)
	}
}

// Test constant types
func TestConstantTypes(t *testing.T) {
	tests := []struct {
		name     string
		constant ast.Constant
		expected interface{}
	}{
		{
			name:     "Name",
			constant: mustName("/alice"),
			expected: "/alice",
		},
		{
			name:     "String",
			constant: ast.String("hello"),
			expected: "hello",
		},
		{
			name:     "Number",
			constant: ast.Number(42),
			expected: float64(42), // JSON unmarshals numbers as float64
		},
		{
			name:     "Float64",
			constant: ast.Float64(3.14),
			expected: 3.14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper := ConstantJSONLD{Constant: tt.constant}
			bytes, err := json.Marshal(wrapper)
			if err != nil {
				t.Fatalf("Failed to marshal constant: %v", err)
			}

			// Unmarshal to check value
			var val interface{}
			if err := json.Unmarshal(bytes, &val); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if val != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, val)
			}

			// Round-trip
			var result ConstantJSONLD
			if err := json.Unmarshal(bytes, &result); err != nil {
				t.Fatalf("Failed to unmarshal back: %v", err)
			}

			if !result.Constant.Equals(tt.constant) {
				t.Errorf("Round-trip failed: expected %v, got %v", tt.constant, result.Constant)
			}
		})
	}
}

// Test List constant
func TestListConstant(t *testing.T) {
	list := ast.List([]ast.Constant{
		ast.Number(1),
		ast.Number(2),
		ast.Number(3),
	})

	wrapper := ConstantJSONLD{Constant: list}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal list: %v", err)
	}

	// Unmarshal to check structure
	var valMap map[string]interface{}
	if err := json.Unmarshal(bytes, &valMap); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	listVal, ok := valMap["@list"]
	if !ok {
		t.Fatal("Missing @list property")
	}

	listArray, ok := listVal.([]interface{})
	if !ok {
		t.Fatal("@list should be an array")
	}

	if len(listArray) != 3 {
		t.Errorf("Expected 3 elements, got %d", len(listArray))
	}

	// Round-trip
	var result ConstantJSONLD
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal list back: %v", err)
	}

	// Verify it's a list
	_, err = result.Constant.ListValues(
		func(elem ast.Constant) error { return nil },
		func() error { return nil },
	)
	if err != nil {
		t.Errorf("Round-trip didn't produce a list: %v", err)
	}
}

// Test Pair constant
func TestPairConstant(t *testing.T) {
	fst := ast.Number(1)
	snd := ast.Number(2)
	pair := ast.Pair(&fst, &snd)

	wrapper := ConstantJSONLD{Constant: pair}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal pair: %v", err)
	}

	// Unmarshal to check structure
	var valMap map[string]interface{}
	if err := json.Unmarshal(bytes, &valMap); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	pairVal, ok := valMap["fn:pair"]
	if !ok {
		t.Fatal("Missing fn:pair property")
	}

	pairArray, ok := pairVal.([]interface{})
	if !ok || len(pairArray) != 2 {
		t.Fatal("fn:pair should be an array of 2 elements")
	}

	// Round-trip
	var result ConstantJSONLD
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal pair back: %v", err)
	}

	fstBack, sndBack, err := result.Constant.PairValue()
	if err != nil {
		t.Fatalf("Round-trip didn't produce a pair: %v", err)
	}

	if !fstBack.Equals(fst) || !sndBack.Equals(snd) {
		t.Error("Pair values don't match after round-trip")
	}
}

// Test collection conversion
func TestAtomsToJSONLD(t *testing.T) {
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

	wrapper := AtomsJSONLD{Atoms: atoms}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal atoms: %v", err)
	}

	// Unmarshal to check structure
	var doc map[string]interface{}
	if err := json.Unmarshal(bytes, &doc); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Check for @graph
	graph, ok := doc["@graph"]
	if !ok {
		t.Fatal("Missing @graph in collection")
	}

	graphArray, ok := graph.([]interface{})
	if !ok {
		t.Fatal("@graph should be an array")
	}

	if len(graphArray) != 2 {
		t.Errorf("Expected 2 nodes in graph, got %d", len(graphArray))
	}
}

// Test reverse collection conversion
func TestJSONLDToAtoms(t *testing.T) {
	jsonStr := `{
		"@context": {},
		"@graph": [
			{"@id": "/alice", "@type": "person"},
			{"@id": "/bob", "@type": "person"}
		]
	}`

	var wrapper AtomsJSONLD
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	atoms := wrapper.Atoms

	if len(atoms) != 2 {
		t.Errorf("Expected 2 atoms, got %d", len(atoms))
	}

	for _, atom := range atoms {
		if atom.Predicate.Symbol != "person" {
			t.Errorf("Expected predicate 'person', got %s", atom.Predicate.Symbol)
		}
		if atom.Predicate.Arity != 1 {
			t.Errorf("Expected arity 1, got %d", atom.Predicate.Arity)
		}
	}
}

// Helper function
func mustName(s string) ast.Constant {
	n, err := ast.Name(s)
	if err != nil {
		panic(err)
	}
	return n
}
