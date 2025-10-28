package factstoredb

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/google/mangle/ast"
)

func TestParseConstant(t *testing.T) {
	name1, _ := ast.Name("/alice")
	name2, _ := ast.Name("/bob")
	name3, _ := ast.Name("/users/john")
	num1 := ast.Number(1)
	num2 := ast.Number(2)
	num42 := ast.Number(42)
	str1 := ast.String("value")

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
		{name: "number_positive", constant: num42},
		{name: "number_negative", constant: ast.Number(-17)},
		{name: "number_zero", constant: ast.Number(0)},
		// Floats
		{name: "float_pi", constant: ast.Float64(3.14159)},
		{name: "float_negative", constant: ast.Float64(-2.5)},
		// Bytes
		{name: "bytes_simple", constant: ast.Bytes([]byte{0x01, 0x02, 0x03})},
		{name: "bytes_special", constant: ast.Bytes([]byte{0x80, 0x81, 0x0a, 0x22})}, // non-ASCII, newline, quote
		{name: "bytes_empty", constant: ast.Bytes([]byte{})},
		// Lists
		{name: "list_numbers", constant: ast.List([]ast.Constant{
			ast.Number(1), ast.Number(2), ast.Number(3),
		})},
		{name: "list_mixed", constant: ast.List([]ast.Constant{
			name1, ast.String("bob"), num42,
		})},
		{name: "list_empty", constant: ast.List([]ast.Constant{})},
		{name: "list_nested", constant: ast.List([]ast.Constant{
			ast.List([]ast.Constant{ast.Number(1), ast.Number(2)}),
			ast.List([]ast.Constant{ast.Number(3), ast.Number(4)}),
		})},
		// Pairs
		{name: "pair_simple", constant: ast.Pair(&name1, &name2)},
		{name: "pair_mixed", constant: ast.Pair(&str1, &num42)},
		// Maps
		{name: "map_single", constant: *ast.Map(map[*ast.Constant]*ast.Constant{
			&name1: &num1,
		})},
		{name: "map_multiple", constant: *ast.Map(map[*ast.Constant]*ast.Constant{
			&name1: &num1,
			&name2: &num2,
		})},
		{name: "map_empty", constant: *ast.Map(map[*ast.Constant]*ast.Constant{})},
		// Structs
		{name: "struct_single", constant: *ast.Struct(map[*ast.Constant]*ast.Constant{
			&name1: &str1,
		})},
		{name: "struct_multiple", constant: *ast.Struct(map[*ast.Constant]*ast.Constant{
			&name1: &str1,
			&name2: &num42,
		})},
		{name: "struct_empty", constant: *ast.Struct(map[*ast.Constant]*ast.Constant{})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputStr := tt.constant.String()
			t.Logf("Parsing string: %q", inputStr)

			parsedConstant, err := ParseConstantFromString(inputStr)
			if err != nil {
				t.Fatalf("Failed to parse %q: %v", inputStr, err)
			}

			if !tt.constant.Equals(parsedConstant) {
				t.Errorf("Parsed constant not equal to original:\n  original=%v (type=%d, hash=%d)\n  parsed=%v (type=%d, hash=%d)",
					tt.constant, tt.constant.Type, tt.constant.Hash(),
					parsedConstant, parsedConstant.Type, parsedConstant.Hash())
			}
		})
	}
}

func TestParseConstantFromReader(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ast.Constant
		wantErr bool
	}{
		{
			name:  "simple_number_from_strings_reader",
			input: "42",
			want:  ast.Number(42),
		},
		{
			name:  "string_from_bytes_buffer",
			input: `"hello world"`,
			want:  ast.String("hello world"),
		},
		{
			name:  "list_from_reader",
			input: "[1, 2, 3]",
			want:  ast.List([]ast.Constant{ast.Number(1), ast.Number(2), ast.Number(3)}),
		},
		{
			name:  "map_with_trailing_comma",
			input: "[/alice : 1, /bob : 2,]",
			want: func() ast.Constant {
				alice, _ := ast.Name("/alice")
				bob, _ := ast.Name("/bob")
				one := ast.Number(1)
				two := ast.Number(2)
				return *ast.Map(map[*ast.Constant]*ast.Constant{
					&alice: &one,
					&bob:   &two,
				})
			}(),
		},
		{
			name:    "invalid_input",
			input:   "not a valid constant",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with strings.Reader
			t.Run("strings.Reader", func(t *testing.T) {
				reader := strings.NewReader(tt.input)
				got, err := ParseConstantFromReader(reader)
				if (err != nil) != tt.wantErr {
					t.Errorf("ParseConstantFromReader() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && !tt.want.Equals(got) {
					t.Errorf("ParseConstantFromReader() = %v, want %v", got, tt.want)
				}
			})

			// Test with bytes.Buffer
			t.Run("bytes.Buffer", func(t *testing.T) {
				buffer := bytes.NewBufferString(tt.input)
				got, err := ParseConstantFromReader(buffer)
				if (err != nil) != tt.wantErr {
					t.Errorf("ParseConstantFromReader() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr && !tt.want.Equals(got) {
					t.Errorf("ParseConstantFromReader() = %v, want %v", got, tt.want)
				}
			})
		})
	}
}

func TestParseConstantFromFile(t *testing.T) {
	// Test reading from an actual file
	tempFile, err := os.CreateTemp("", "constant-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	testContent := `{/name : "Alice", /age : 30, /active : fn:pair(1, 2)}`
	if _, err := tempFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Open and read
	file, err := os.Open(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to open temp file: %v", err)
	}
	defer file.Close()

	constant, err := ParseConstantFromReader(file)
	if err != nil {
		t.Fatalf("ParseConstantFromReader() failed: %v", err)
	}

	// Parse the same content as string to compare
	expected, err := ParseConstantFromString(testContent)
	if err != nil {
		t.Fatalf("ParseConstantFromString() failed: %v", err)
	}

	if !expected.Equals(constant) {
		t.Errorf("File parsing doesn't match string parsing:\n  from file=%v\n  from string=%v", constant, expected)
	}

	// Verify we can access struct values (confirms it's a struct)
	structValues := false
	_, err2 := constant.StructValues(
		func(key, val ast.Constant) error {
			structValues = true
			return nil
		},
		func() error { return nil },
	)
	if err2 != nil {
		t.Errorf("Failed to access struct values: %v", err2)
	}
	if !structValues {
		t.Errorf("Expected struct to have values, but got none")
	}
}

func TestParseConstantFromReaderLargeInput(t *testing.T) {
	// Test with a large list to ensure efficient streaming
	var b strings.Builder
	b.WriteString("[")
	for i := range 1000 {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(string(rune('0' + (i % 10))))
	}
	b.WriteString("]")

	reader := strings.NewReader(b.String())
	constant, err := ParseConstantFromReader(reader)
	if err != nil {
		t.Fatalf("ParseConstantFromReader() failed on large input: %v", err)
	}

	// Verify the list has 1000 elements by iterating
	count := 0
	_, err2 := constant.ListValues(
		func(elem ast.Constant) error {
			count++
			return nil
		},
		func() error { return nil },
	)
	if err2 != nil {
		t.Errorf("Failed to iterate list: %v", err2)
	}
	if count != 1000 {
		t.Errorf("Expected 1000 elements in list, got %d", count)
	}
}
