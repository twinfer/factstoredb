package factstoredb

import (
	"errors"
	"fmt"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/google/mangle/ast"
)

// constantJSON is a wrapper around ast.Constant that implements
// json.MarshalerTo and json.UnmarshalerFrom for efficient serialization
// without losing type information.
type constantJSON struct {
	ast.Constant
}

// atomJSON is a wrapper around ast.Atom that implements json.MarshalerTo
// for serializing atoms with predicate and args structure.
type atomJSON struct {
	ast.Atom
}

// MarshalJSONTo implements json.MarshalerTo for constantJSON.
// Uses native JSON types matching AST text representation.
func (cj constantJSON) MarshalJSONTo(enc *jsontext.Encoder) error {
	switch cj.Type {
	case ast.NameType:
		// Names: JSON string with "/" prefix (e.g., "/alice")
		sym, err := cj.NameValue()
		if err != nil {
			return fmt.Errorf("failed to get name value: %w", err)
		}
		return enc.WriteToken(jsontext.String(sym))

	case ast.StringType:
		// Strings: plain JSON string (e.g., "hello")
		str, err := cj.StringValue()
		if err != nil {
			return fmt.Errorf("failed to get string value: %w", err)
		}
		return enc.WriteToken(jsontext.String(str))

	case ast.BytesType:
		// Bytes: JSON string with "#b" prefix + hex encoding
		hexStr := "#b" + fmt.Sprintf("%x", []byte(cj.Symbol))
		return enc.WriteToken(jsontext.String(hexStr))

	case ast.NumberType:
		// Numbers: JSON integer (e.g., 42)
		num, err := cj.NumberValue()
		if err != nil {
			return fmt.Errorf("failed to get number value: %w", err)
		}
		return enc.WriteToken(jsontext.Int(num))

	case ast.Float64Type:
		// Floats: JSON number (e.g., 3.14)
		flt, err := cj.Float64Value()
		if err != nil {
			return fmt.Errorf("failed to get float64 value: %w", err)
		}
		return enc.WriteToken(jsontext.Float(flt))

	case ast.ListShape:
		// Lists: JSON array [elem1, elem2, ...]
		if err := enc.WriteToken(jsontext.BeginArray); err != nil {
			return err
		}
		_, err := cj.ListValues(
			func(elem ast.Constant) error {
				return constantJSON{elem}.MarshalJSONTo(enc)
			},
			func() error { return nil },
		)
		if err != nil {
			return fmt.Errorf("failed to serialize list: %w", err)
		}
		return enc.WriteToken(jsontext.EndArray)

	case ast.PairShape:
		// Pairs: ApplyFn with fn:pair(fst, snd)
		// {"function": {"symbol": "fn:pair", "arity": 2}, "args": [fst, snd]}
		fst, snd, err := cj.PairValue()
		if err != nil {
			return fmt.Errorf("failed to get pair value: %w", err)
		}
		if err := enc.WriteToken(jsontext.BeginObject); err != nil {
			return err
		}
		// Write "function" key
		if err := enc.WriteToken(jsontext.String("function")); err != nil {
			return err
		}
		// Write function object
		if err := enc.WriteToken(jsontext.BeginObject); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String("symbol")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String("fn:pair")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String("arity")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.Int(2)); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.EndObject); err != nil {
			return err
		}
		// Write "args" key and array
		if err := enc.WriteToken(jsontext.String("args")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.BeginArray); err != nil {
			return err
		}
		if err := (constantJSON{fst}).MarshalJSONTo(enc); err != nil {
			return err
		}
		if err := (constantJSON{snd}).MarshalJSONTo(enc); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.EndArray); err != nil {
			return err
		}
		return enc.WriteToken(jsontext.EndObject)

	case ast.MapShape:
		// Maps: ApplyFn with fn:map(key1, val1, key2, val2, ...)
		// {"function": {"symbol": "fn:map", "arity": N}, "args": [key1, val1, ...]}
		// Collect all key-value pairs into a flat list
		var args []ast.Constant
		_, err := cj.MapValues(
			func(key, val ast.Constant) error {
				args = append(args, key, val)
				return nil
			},
			func() error { return nil },
		)
		if err != nil {
			return fmt.Errorf("failed to iterate map: %w", err)
		}
		arity := len(args) // Total number of arguments (keys + values)

		if err := enc.WriteToken(jsontext.BeginObject); err != nil {
			return err
		}
		// Write "function" key
		if err := enc.WriteToken(jsontext.String("function")); err != nil {
			return err
		}
		// Write function object
		if err := enc.WriteToken(jsontext.BeginObject); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String("symbol")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String("fn:map")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String("arity")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.Int(int64(arity))); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.EndObject); err != nil {
			return err
		}
		// Write "args" key and array
		if err := enc.WriteToken(jsontext.String("args")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.BeginArray); err != nil {
			return err
		}
		for _, arg := range args {
			if err := (constantJSON{arg}).MarshalJSONTo(enc); err != nil {
				return err
			}
		}
		if err := enc.WriteToken(jsontext.EndArray); err != nil {
			return err
		}
		return enc.WriteToken(jsontext.EndObject)

	case ast.StructShape:
		// Structs: ApplyFn with fn:struct(label1, val1, label2, val2, ...)
		// {"function": {"symbol": "fn:struct", "arity": N}, "args": [label1, val1, ...]}
		var args []ast.Constant
		_, err := cj.StructValues(
			func(label, val ast.Constant) error {
				args = append(args, label, val)
				return nil
			},
			func() error { return nil },
		)
		if err != nil {
			return fmt.Errorf("failed to iterate struct: %w", err)
		}
		arity := len(args) // Total number of arguments (labels + values)

		if err := enc.WriteToken(jsontext.BeginObject); err != nil {
			return err
		}
		// Write "function" key
		if err := enc.WriteToken(jsontext.String("function")); err != nil {
			return err
		}
		// Write function object
		if err := enc.WriteToken(jsontext.BeginObject); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String("symbol")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String("fn:struct")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String("arity")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.Int(int64(arity))); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.EndObject); err != nil {
			return err
		}
		// Write "args" key and array
		if err := enc.WriteToken(jsontext.String("args")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.BeginArray); err != nil {
			return err
		}
		for _, arg := range args {
			if err := (constantJSON{arg}).MarshalJSONTo(enc); err != nil {
				return err
			}
		}
		if err := enc.WriteToken(jsontext.EndArray); err != nil {
			return err
		}
		return enc.WriteToken(jsontext.EndObject)

	default:
		return fmt.Errorf("unknown constant type: %d", cj.Type)
	}
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom for constantJSON.
// Parses native JSON types and constructs BaseTerms, then evaluates with functional.EvalExpr.
func (cj *constantJSON) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	// Peek at the next token to determine the type
	tok, err := dec.ReadToken()
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}

	switch tok.Kind() {
	case 'n': // null
		return errors.New("null values are not supported for constants")

	case '"': // string
		str := tok.String()
		// Determine if this is a Name, Bytes, or String
		if len(str) > 0 && str[0] == '/' {
			// Name: starts with "/"
			c, err := ast.Name(str)
			if err != nil {
				return fmt.Errorf("failed to create name from %q: %w", str, err)
			}
			cj.Constant = c
		} else if len(str) > 2 && str[:2] == "#b" {
			// Bytes: starts with "#b" followed by hex
			hexStr := str[2:]
			bytesData := make([]byte, len(hexStr)/2)
			for i := 0; i < len(hexStr); i += 2 {
				var b byte
				_, err := fmt.Sscanf(hexStr[i:i+2], "%02x", &b)
				if err != nil {
					return fmt.Errorf("failed to decode hex bytes: %w", err)
				}
				bytesData[i/2] = b
			}
			cj.Constant = ast.Bytes(bytesData)
		} else {
			// String: regular string
			cj.Constant = ast.String(str)
		}

	case '0', '-': // number (integer or float)
		if tok.Float() != float64(int64(tok.Float())) {
			// Float
			cj.Constant = ast.Float64(tok.Float())
		} else {
			// Integer
			cj.Constant = ast.Number(tok.Int())
		}

	case '[': // array (list)
		var elems []ast.Constant
		for dec.PeekKind() != ']' {
			var elem constantJSON
			if err := elem.UnmarshalJSONFrom(dec); err != nil {
				return fmt.Errorf("failed to unmarshal list element: %w", err)
			}
			elems = append(elems, elem.Constant)
		}
		// Read the closing bracket
		if _, err := dec.ReadToken(); err != nil {
			return fmt.Errorf("failed to read array end: %w", err)
		}
		cj.Constant = ast.List(elems)

	case '{': // object (ApplyFn for pair/map/struct)
		// Read the object fields manually
		var symbol string
		var arity int
		var args []ast.BaseTerm

		// Read object fields
		for {
			// Read key or closing brace
			tok, err := dec.ReadToken()
			if err != nil {
				return fmt.Errorf("failed to read object token: %w", err)
			}
			if tok.Kind() == '}' {
				break
			}
			if tok.Kind() != '"' {
				return fmt.Errorf("expected string key, got %c", tok.Kind())
			}

			key := tok.String()
			switch key {
			case "function":
				// Read the function object
				tok, err := dec.ReadToken()
				if err != nil {
					return fmt.Errorf("failed to read function object: %w", err)
				}
				if tok.Kind() != '{' {
					return fmt.Errorf("expected object for function, got %c", tok.Kind())
				}

				// Read function fields
				for {
					tok, err := dec.ReadToken()
					if err != nil {
						return fmt.Errorf("failed to read function field: %w", err)
					}
					if tok.Kind() == '}' {
						break
					}
					if tok.Kind() != '"' {
						return fmt.Errorf("expected string key, got %c", tok.Kind())
					}

					funcKey := tok.String()
					switch funcKey {
					case "symbol":
						tok, err := dec.ReadToken()
						if err != nil {
							return fmt.Errorf("failed to read symbol: %w", err)
						}
						symbol = tok.String()
					case "arity":
						tok, err := dec.ReadToken()
						if err != nil {
							return fmt.Errorf("failed to read arity: %w", err)
						}
						arity = int(tok.Int())
					default:
						return fmt.Errorf("unexpected function field: %s", funcKey)
					}
				}

			case "args":
				// Read the args array
				tok, err := dec.ReadToken()
				if err != nil {
					return fmt.Errorf("failed to read args array: %w", err)
				}
				if tok.Kind() != '[' {
					return fmt.Errorf("expected array for args, got %c", tok.Kind())
				}

				// Read array elements
				for dec.PeekKind() != ']' {
					var arg constantJSON
					if err := arg.UnmarshalJSONFrom(dec); err != nil {
						return fmt.Errorf("failed to unmarshal arg: %w", err)
					}
					args = append(args, arg.Constant)
				}

				// Read closing bracket
				if _, err := dec.ReadToken(); err != nil {
					return fmt.Errorf("failed to read args array end: %w", err)
				}

			default:
				return fmt.Errorf("unexpected object field: %s", key)
			}
		}

		// Validate arity matches number of args
		if arity != len(args) {
			return fmt.Errorf("arity mismatch: expected %d args, got %d", arity, len(args))
		}

		// Convert args to constants (they should all be constants already)
		constants := make([]ast.Constant, len(args))
		for i, arg := range args {
			c, ok := arg.(ast.Constant)
			if !ok {
				return fmt.Errorf("arg %d is not a constant: %T", i, arg)
			}
			constants[i] = c
		}

		// Construct the constant based on the function symbol
		switch symbol {
		case "fn:pair":
			if len(constants) != 2 {
				return fmt.Errorf("fn:pair expects 2 args, got %d", len(constants))
			}
			cj.Constant = ast.Pair(&constants[0], &constants[1])

		case "fn:map":
			if len(constants)%2 != 0 {
				return fmt.Errorf("fn:map expects even number of args, got %d", len(constants))
			}
			kvMap := make(map[*ast.Constant]*ast.Constant)
			for i := 0; i < len(constants); i += 2 {
				kvMap[&constants[i]] = &constants[i+1]
			}
			mapConst := ast.Map(kvMap)
			cj.Constant = *mapConst

		case "fn:struct":
			if len(constants)%2 != 0 {
				return fmt.Errorf("fn:struct expects even number of args, got %d", len(constants))
			}
			kvMap := make(map[*ast.Constant]*ast.Constant)
			for i := 0; i < len(constants); i += 2 {
				kvMap[&constants[i]] = &constants[i+1]
			}
			structConst := ast.Struct(kvMap)
			cj.Constant = *structConst

		default:
			return fmt.Errorf("unknown function symbol: %s", symbol)
		}

	default:
		return fmt.Errorf("unexpected JSON token kind: %c", tok.Kind())
	}

	return nil
}

// MarshalJSONTo implements json.MarshalerTo for atomJSON.
// Serializes atom as: {"predicate": {"symbol": "...", "arity": N}, "args": [...]}
func (aj atomJSON) MarshalJSONTo(enc *jsontext.Encoder) error {
	// Write opening brace
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}

	// Write "predicate" key
	if err := enc.WriteToken(jsontext.String("predicate")); err != nil {
		return err
	}

	// Write predicate object
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("symbol")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(aj.Predicate.Symbol)); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("arity")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.Int(int64(aj.Predicate.Arity))); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.EndObject); err != nil {
		return err
	}

	// Write "args" key
	if err := enc.WriteToken(jsontext.String("args")); err != nil {
		return err
	}

	// Write args array
	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}
	for _, arg := range aj.Args {
		c, ok := arg.(ast.Constant)
		if !ok {
			return fmt.Errorf("atom arg is not a constant: %T", arg)
		}
		if err := (constantJSON{c}).MarshalJSONTo(enc); err != nil {
			return err
		}
	}
	if err := enc.WriteToken(jsontext.EndArray); err != nil {
		return err
	}

	// Write closing brace
	return enc.WriteToken(jsontext.EndObject)
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom for atomJSON.
// It stream-parses an atom object: {"predicate": {...}, "args": [...]}
func (aj *atomJSON) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	// Expect the start of an object: {
	tok, err := dec.ReadToken()
	if err != nil {
		return fmt.Errorf("failed to read atom start: %w", err)
	}
	if tok.Kind() != '{' {
		return fmt.Errorf("expected atom object start '{', got %c", tok.Kind())
	}

	var symbol string
	var arity int
	var args []ast.BaseTerm

	// Loop through object keys
	for dec.PeekKind() != '}' {
		// Read key
		tok, err := dec.ReadToken()
		if err != nil {
			return fmt.Errorf("failed to read atom key: %w", err)
		}
		if tok.Kind() != '"' {
			return fmt.Errorf("expected string key for atom field, got %c", tok.Kind())
		}
		key := tok.String()

		switch key {
		case "predicate":
			// Expect predicate object: {
			if tok, err = dec.ReadToken(); err != nil || tok.Kind() != '{' {
				return fmt.Errorf("expected predicate object start '{'")
			}
			for dec.PeekKind() != '}' {
				// Read predicate field key
				if tok, err = dec.ReadToken(); err != nil || tok.Kind() != '"' {
					return fmt.Errorf("expected string key for predicate field")
				}
				predKey := tok.String()
				// Read value
				if tok, err = dec.ReadToken(); err != nil {
					return fmt.Errorf("failed to read predicate value")
				}
				switch predKey {
				case "symbol":
					symbol = tok.String()
				case "arity":
					arity = int(tok.Int())
				}
			}
			// Expect predicate object end: }
			if tok, err = dec.ReadToken(); err != nil || tok.Kind() != '}' {
				return fmt.Errorf("expected predicate object end '}'")
			}

		case "args":
			// Expect args array: [
			if tok, err = dec.ReadToken(); err != nil || tok.Kind() != '[' {
				return fmt.Errorf("expected args array start '['")
			}
			for dec.PeekKind() != ']' {
				var cj constantJSON
				if err := cj.UnmarshalJSONFrom(dec); err != nil {
					return fmt.Errorf("failed to unmarshal arg: %w", err)
				}
				args = append(args, cj.Constant)
			}
			// Expect args array end: ]
			if tok, err = dec.ReadToken(); err != nil || tok.Kind() != ']' {
				return fmt.Errorf("expected args array end ']'")
			}

		default:
			// Skip unknown fields
			if err := dec.SkipValue(); err != nil {
				return fmt.Errorf("failed to skip unknown field %q: %w", key, err)
			}
		}
	}

	// Expect atom object end: }
	if _, err := dec.ReadToken(); err != nil {
		return fmt.Errorf("failed to read atom end: %w", err)
	}

	aj.Atom = ast.Atom{Predicate: ast.PredicateSym{Symbol: symbol, Arity: arity}, Args: args}
	return nil
}

// unmarshalAtom unmarshals a predicate string and args JSON directly into an ast.Atom.
// This is more efficient than unmarshalling args separately and then constructing the atom.
// The predicateStr must be in "symbol_arity" format (e.g., "foo_2").
func unmarshalAtom(predicateStr ast.PredicateSym, argsJSON string) (ast.Atom, error) {

	// Unmarshal args directly to []constantJSON
	var jsonConsts []constantJSON
	if err := json.Unmarshal([]byte(argsJSON), &jsonConsts); err != nil {
		return ast.Atom{}, fmt.Errorf("failed to unmarshal args: %w", err)
	}

	// Convert to []ast.BaseTerm in a single pass
	// Note: We allocate a new slice here instead of using a pool because the callback
	// might store the atom (e.g., in Merge), and pooled slices would be reused and
	// cause data corruption
	baseTerms := make([]ast.BaseTerm, len(jsonConsts))
	for i, jc := range jsonConsts {
		baseTerms[i] = jc.Constant
	}

	return ast.Atom{
		Predicate: ast.PredicateSym{Symbol: predicateStr.Symbol, Arity: predicateStr.Arity},
		Args:      baseTerms,
	}, nil
}
