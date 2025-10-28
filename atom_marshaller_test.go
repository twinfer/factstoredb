package factstoredb

import (
	"strings"
	"testing"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/google/mangle/ast"
)

func TestConstantJSONRoundTrip(t *testing.T) {
	// Test constant JSON round trips by constructing constants directly
	name1, _ := ast.Name("/alice")
	name2, _ := ast.Name("/bob")
	name3, _ := ast.Name("/users/john")

	tests := []struct {
		name     string
		constant ast.Constant
	}{
		// Names
		{name: "name_alice", constant: name1},
		{name: "name_bob", constant: name2},
		{name: "name_with_path", constant: name3},
		// Strings
		{name: "string_simple", constant: ast.String("hello world")},
		{name: "string_empty", constant: ast.String("")},
		{name: "string_special", constant: ast.String(`foo"bar\baz`)},
		// Numbers
		{name: "number_positive", constant: ast.Number(42)},
		{name: "number_negative", constant: ast.Number(-17)},
		{name: "number_zero", constant: ast.Number(0)},
		// Floats
		{name: "float_pi", constant: ast.Float64(3.14159)},
		{name: "float_negative", constant: ast.Float64(-2.5)},
		// Lists
		{name: "list_numbers", constant: ast.List([]ast.Constant{
			ast.Number(1), ast.Number(2), ast.Number(3),
		})},
		{name: "list_mixed", constant: ast.List([]ast.Constant{
			name1, ast.String("bob"), ast.Number(42),
		})},
		{name: "list_empty", constant: ast.List([]ast.Constant{})},
		{name: "list_nested", constant: ast.List([]ast.Constant{
			ast.List([]ast.Constant{ast.Number(1), ast.Number(2)}),
			ast.List([]ast.Constant{ast.Number(3), ast.Number(4)}),
		})},
		// Pairs
		{name: "pair_simple", constant: ast.Pair(&name1, &name2)},
		{name: "pair_mixed", constant: ast.Pair(
			&[]ast.Constant{ast.String("key")}[0],
			&[]ast.Constant{ast.Number(42)}[0],
		)},
		// Bytes
		{name: "bytes_simple", constant: ast.Bytes([]byte{0x01, 0x02, 0x03})},
		{name: "bytes_special", constant: ast.Bytes([]byte{0x80, 0x81, 0x0a, 0x22})}, // non-ASCII, newline, quote
		{name: "bytes_empty", constant: ast.Bytes([]byte{})},
		// Maps
		{name: "map_single", constant: *ast.Map(map[*ast.Constant]*ast.Constant{
			&name1: &[]ast.Constant{ast.Number(1)}[0],
		})},
		{name: "map_multiple", constant: *ast.Map(map[*ast.Constant]*ast.Constant{
			&name1: &[]ast.Constant{ast.Number(1)}[0],
			&name2: &[]ast.Constant{ast.Number(2)}[0],
		})},
		{name: "map_empty", constant: *ast.Map(map[*ast.Constant]*ast.Constant{})},
		// Structs
		{name: "struct_single", constant: *ast.Struct(map[*ast.Constant]*ast.Constant{
			&name1: &[]ast.Constant{ast.String("value1")}[0],
		})},
		{name: "struct_multiple", constant: *ast.Struct(map[*ast.Constant]*ast.Constant{
			&name1: &[]ast.Constant{ast.String("value1")}[0],
			&name2: &[]ast.Constant{ast.Number(42)}[0],
		})},
		{name: "struct_empty", constant: *ast.Struct(map[*ast.Constant]*ast.Constant{})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON using constantJSON
			jsonBytes, err := json.Marshal(constantJSON{tt.constant})
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			t.Logf("Constant: %v -> JSON: %s", tt.constant, string(jsonBytes))

			// Unmarshal from JSON
			var unmarshalled constantJSON
			if err := json.Unmarshal(jsonBytes, &unmarshalled); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify equality using Equals method
			if !tt.constant.Equals(unmarshalled.Constant) {
				t.Errorf("Constants not equal:\n  original=%v (type=%d, hash=%d)\n  unmarshalled=%v (type=%d, hash=%d)",
					tt.constant, tt.constant.Type, tt.constant.Hash(),
					unmarshalled.Constant, unmarshalled.Constant.Type, unmarshalled.Constant.Hash())
			}
		})
	}
}

func TestAtomJSONRoundTrip(t *testing.T) {
	// Test Atom round trips by constructing atoms directly
	alice, _ := ast.Name("/alice")
	bob, _ := ast.Name("/bob")
	test, _ := ast.Name("/test")

	tests := []struct {
		name string
		atom ast.Atom
	}{
		{
			name: "atom_simple",
			atom: ast.Atom{
				Predicate: ast.PredicateSym{Symbol: "user", Arity: 2},
				Args:      []ast.BaseTerm{alice, ast.Number(25)},
			},
		},
		{
			name: "atom_multi_arg",
			atom: ast.Atom{
				Predicate: ast.PredicateSym{Symbol: "person", Arity: 3},
				Args:      []ast.BaseTerm{bob, ast.String("Bob Smith"), ast.Number(30)},
			},
		},
		{
			name: "atom_single_arg",
			atom: ast.Atom{
				Predicate: ast.PredicateSym{Symbol: "active", Arity: 1},
				Args:      []ast.BaseTerm{alice},
			},
		},
		{
			name: "atom_no_args",
			atom: ast.Atom{
				Predicate: ast.PredicateSym{Symbol: "ready", Arity: 0},
				Args:      []ast.BaseTerm{},
			},
		},
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
			name: "atom_mixed_types",
			atom: ast.Atom{
				Predicate: ast.PredicateSym{Symbol: "data", Arity: 5},
				Args: []ast.BaseTerm{
					test,
					ast.String("string"),
					ast.Number(42),
					ast.Float64(3.14),
					ast.List([]ast.Constant{ast.Number(1), ast.Number(2)}),
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

			t.Logf("Atom: %v -> JSON: %s", original, jsonStr)

			// Unmarshal from JSON using the atomEntry structure
			type atomEntry struct {
				Predicate struct {
					Symbol string `json:"symbol"`
					Arity  int    `json:"arity"`
				} `json:"predicate"`
				Args []jsontext.Value `json:"args"`
			}

			var entry atomEntry
			if err := json.Unmarshal([]byte(jsonStr), &entry); err != nil {
				t.Fatalf("Failed to unmarshal atom JSON: %v", err)
			}

			// Reconstruct atom from JSON
			args := make([]ast.BaseTerm, len(entry.Args))
			for i, argJSON := range entry.Args {
				var cj constantJSON
				if err := json.Unmarshal([]byte(argJSON), &cj); err != nil {
					t.Fatalf("Failed to unmarshal arg %d: %v", i, err)
				}
				args[i] = cj.Constant
			}

			reconstructed := ast.Atom{
				Predicate: ast.PredicateSym{
					Symbol: entry.Predicate.Symbol,
					Arity:  entry.Predicate.Arity,
				},
				Args: args,
			}

			// Verify equality
			if original.Predicate.Symbol != reconstructed.Predicate.Symbol {
				t.Errorf("Predicate symbol mismatch: %s != %s",
					original.Predicate.Symbol, reconstructed.Predicate.Symbol)
			}
			if original.Predicate.Arity != reconstructed.Predicate.Arity {
				t.Errorf("Predicate arity mismatch: %d != %d",
					original.Predicate.Arity, reconstructed.Predicate.Arity)
			}
			if len(original.Args) != len(reconstructed.Args) {
				t.Fatalf("Args length mismatch: %d != %d",
					len(original.Args), len(reconstructed.Args))
			}
			for i := range original.Args {
				origConst, ok1 := original.Args[i].(ast.Constant)
				reconConst, ok2 := reconstructed.Args[i].(ast.Constant)
				if !ok1 || !ok2 {
					t.Errorf("Arg %d is not a constant", i)
					continue
				}
				if !origConst.Equals(reconConst) {
					t.Errorf("Arg %d not equal:\n  original=%v (type=%d)\n  reconstructed=%v (type=%d)",
						i, origConst, origConst.Type, reconConst, reconConst.Type)
				}
			}
		})
	}
}

func TestAtomJSONMalformed(t *testing.T) {
	tests := []struct {
		name        string
		jsonInput   string
		expectError string
	}{
		{
			name:        "symbol_as_number",
			jsonInput:   `{"predicate": {"symbol": 123, "arity": 2}, "args": []}`,
			expectError: "expected string for 'symbol'",
		},
		{
			name:        "arity_as_string",
			jsonInput:   `{"predicate": {"symbol": "user", "arity": "two"}, "args": []}`,
			expectError: "expected number for 'arity'",
		},
		{
			name:        "symbol_as_null",
			jsonInput:   `{"predicate": {"symbol": null, "arity": 2}, "args": []}`,
			expectError: "expected string for 'symbol'",
		},
		{
			name:        "arity_as_null",
			jsonInput:   `{"predicate": {"symbol": "user", "arity": null}, "args": []}`,
			expectError: "expected number for 'arity'",
		},
		{
			name:        "symbol_as_bool",
			jsonInput:   `{"predicate": {"symbol": true, "arity": 2}, "args": []}`,
			expectError: "expected string for 'symbol'",
		},
		{
			name:        "arity_as_bool",
			jsonInput:   `{"predicate": {"symbol": "user", "arity": false}, "args": []}`,
			expectError: "expected number for 'arity'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec := jsontext.NewDecoder(strings.NewReader(tt.jsonInput))
			var aj atomJSON
			err := aj.UnmarshalJSONFrom(dec)

			if err == nil {
				t.Fatalf("Expected error containing %q, but got no error", tt.expectError)
			}

			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing %q, got %q", tt.expectError, err.Error())
			}
		})
	}
}

// func TestAtomTextFormatRoundTrip(t *testing.T) {
// 	// Test that atoms can round-trip through text format (String() -> parse.Atom())
// 	alice, _ := ast.Name("/alice")
// 	bob, _ := ast.Name("/bob")
// 	test, _ := ast.Name("/test")

// 	tests := []struct {
// 		name string
// 		atom ast.Atom
// 	}{
// 		{
// 			name: "atom_simple",
// 			atom: ast.Atom{
// 				Predicate: ast.PredicateSym{Symbol: "user", Arity: 2},
// 				Args:      []ast.BaseTerm{alice, ast.Number(25)},
// 			},
// 		},
// 		{
// 			name: "atom_multi_arg",
// 			atom: ast.Atom{
// 				Predicate: ast.PredicateSym{Symbol: "person", Arity: 3},
// 				Args:      []ast.BaseTerm{bob, ast.String("Bob Smith"), ast.Number(30)},
// 			},
// 		},
// 		{
// 			name: "atom_single_arg",
// 			atom: ast.Atom{
// 				Predicate: ast.PredicateSym{Symbol: "active", Arity: 1},
// 				Args:      []ast.BaseTerm{alice},
// 			},
// 		},
// 		{
// 			name: "atom_no_args",
// 			atom: ast.Atom{
// 				Predicate: ast.PredicateSym{Symbol: "ready", Arity: 0},
// 				Args:      []ast.BaseTerm{},
// 			},
// 		},
// 		{
// 			name: "atom_list_arg",
// 			atom: ast.Atom{
// 				Predicate: ast.PredicateSym{Symbol: "tags", Arity: 2},
// 				Args: []ast.BaseTerm{
// 					alice,
// 					ast.List([]ast.Constant{ast.Number(1), ast.Number(2), ast.Number(3)}),
// 				},
// 			},
// 		},
// 		{
// 			name: "atom_mixed_types",
// 			atom: ast.Atom{
// 				Predicate: ast.PredicateSym{Symbol: "data", Arity: 5},
// 				Args: []ast.BaseTerm{
// 					test,
// 					ast.String("string"),
// 					ast.Number(42),
// 					ast.Float64(3.14),
// 					ast.List([]ast.Constant{ast.Number(1), ast.Number(2)}),
// 				},
// 			},
// 		},
// 		{
// 			name: "atom_bytes",
// 			atom: ast.Atom{
// 				Predicate: ast.PredicateSym{Symbol: "binary", Arity: 1},
// 				Args:      []ast.BaseTerm{ast.Bytes([]byte{0x01, 0x02, 0x03})},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			original := tt.atom

// 			// Convert to text representation
// 			textRep := original.String()
// 			t.Logf("Atom text: %s", textRep)

// 			// Wrap as JSON string
// 			jsonStr := "\"" + textRep + "\""
// 			t.Logf("JSON string: %s", jsonStr)

// 			// Unmarshal from JSON string using atomJSON
// 			dec := jsontext.NewDecoder(strings.NewReader(jsonStr))
// 			var unmarshalled atomJSON
// 			if err := unmarshalled.UnmarshalJSONFrom(dec); err != nil {
// 				t.Fatalf("Failed to unmarshal atom from JSON %q: %v", jsonStr, err)
// 			}
// 			parsed := unmarshalled.Atom

// 			// Verify equality
// 			if original.Predicate.Symbol != parsed.Predicate.Symbol {
// 				t.Errorf("Predicate symbol mismatch: %s != %s",
// 					original.Predicate.Symbol, parsed.Predicate.Symbol)
// 			}
// 			if original.Predicate.Arity != parsed.Predicate.Arity {
// 				t.Errorf("Predicate arity mismatch: %d != %d",
// 					original.Predicate.Arity, parsed.Predicate.Arity)
// 			}
// 			if len(original.Args) != len(parsed.Args) {
// 				t.Fatalf("Args length mismatch: %d != %d",
// 					len(original.Args), len(parsed.Args))
// 			}
// 			for i := range original.Args {
// 				origConst, ok1 := original.Args[i].(ast.Constant)
// 				parsedConst, ok2 := parsed.Args[i].(ast.Constant)
// 				if !ok1 || !ok2 {
// 					t.Logf("Arg %d: original type=%T, parsed type=%T", i, original.Args[i], parsed.Args[i])
// 					t.Logf("Arg %d: original=%v, parsed=%v", i, original.Args[i], parsed.Args[i])
// 					if !ok1 {
// 						t.Errorf("Arg %d: original is not a constant (type=%T)", i, original.Args[i])
// 					}
// 					if !ok2 {
// 						t.Errorf("Arg %d: parsed is not a constant (type=%T)", i, parsed.Args[i])
// 					}
// 					continue
// 				}
// 				if !origConst.Equals(parsedConst) {
// 					t.Errorf("Arg %d not equal:\n  original=%v (type=%d, hash=%d)\n  parsed=%v (type=%d, hash=%d)",
// 						i, origConst, origConst.Type, origConst.Hash(),
// 						parsedConst, parsedConst.Type, parsedConst.Hash())
// 				}
// 			}
// 		})
// 	}
// }
