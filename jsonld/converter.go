package jsonld

import (
	"fmt"

	"github.com/go-json-experiment/json/jsontext"
	"github.com/google/mangle/ast"
)

// ConstantJSONLD is a wrapper around ast.Constant that implements
// json.MarshalerTo and json.UnmarshalerFrom for JSON-LD serialization.
type ConstantJSONLD struct {
	ast.Constant
}

// AtomJSONLD is a wrapper around ast.Atom that implements json.MarshalerTo
// and json.UnmarshalerFrom for JSON-LD serialization using hybrid arity-based mapping:
//
// - Arity 0: {"@context": {...}, "@type": "predicate"}
// - Arity 1: {"@context": {...}, "@id": arg0, "@type": "predicate"}
// - Arity 2: {"@context": {...}, "@id": arg0, "predicate": arg1}
// - Arity 3+: {"@context": {...}, "@type": "predicate", "arg0": val, "arg1": val, ...}
type AtomJSONLD struct {
	ast.Atom
}

// AtomsJSONLD is a wrapper around []ast.Atom that implements json.MarshalerTo
// and json.UnmarshalerFrom for JSON-LD serialization using @graph container.
type AtomsJSONLD struct {
	Atoms []ast.Atom
}

// MarshalJSONTo implements json.MarshalerTo for ConstantJSONLD.
// Converts Mangle constants to JSON-LD values.
func (c ConstantJSONLD) MarshalJSONTo(enc *jsontext.Encoder) error {
	switch c.Type {
	case ast.NameType:
		// Names: JSON string with "/" prefix (e.g., "/alice")
		sym, err := c.NameValue()
		if err != nil {
			return fmt.Errorf("failed to get name value: %w", err)
		}
		return enc.WriteToken(jsontext.String(sym))

	case ast.StringType:
		// Strings: plain JSON string (e.g., "hello")
		str, err := c.StringValue()
		if err != nil {
			return fmt.Errorf("failed to get string value: %w", err)
		}
		return enc.WriteToken(jsontext.String(str))

	case ast.BytesType:
		// Bytes: JSON string with b"..." format
		escaped, err := ast.Escape(c.Symbol, true /* isBytes */)
		if err != nil {
			return fmt.Errorf("failed to escape bytes: %w", err)
		}
		bytesStr := fmt.Sprintf(`b"%s"`, escaped)
		return enc.WriteToken(jsontext.String(bytesStr))

	case ast.NumberType:
		// Numbers: JSON integer (e.g., 42)
		num, err := c.NumberValue()
		if err != nil {
			return fmt.Errorf("failed to get number value: %w", err)
		}
		return enc.WriteToken(jsontext.Int(num))

	case ast.Float64Type:
		// Floats: JSON number (e.g., 3.14)
		flt, err := c.Float64Value()
		if err != nil {
			return fmt.Errorf("failed to get float64 value: %w", err)
		}
		return enc.WriteToken(jsontext.Float(flt))

	case ast.ListShape:
		// Lists: {"@list": [elem1, elem2, ...]}
		if err := enc.WriteToken(jsontext.BeginObject); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String("@list")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.BeginArray); err != nil {
			return err
		}
		_, err := c.ListValues(
			func(elem ast.Constant) error {
				return (ConstantJSONLD{elem}).MarshalJSONTo(enc)
			},
			func() error { return nil },
		)
		if err != nil {
			return fmt.Errorf("failed to serialize list: %w", err)
		}
		if err := enc.WriteToken(jsontext.EndArray); err != nil {
			return err
		}
		return enc.WriteToken(jsontext.EndObject)

	case ast.PairShape:
		// Pairs: {"fn:pair": [fst, snd]}
		fst, snd, err := c.PairValue()
		if err != nil {
			return fmt.Errorf("failed to get pair value: %w", err)
		}
		if err := enc.WriteToken(jsontext.BeginObject); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String("fn:pair")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.BeginArray); err != nil {
			return err
		}
		if err := (ConstantJSONLD{fst}).MarshalJSONTo(enc); err != nil {
			return err
		}
		if err := (ConstantJSONLD{snd}).MarshalJSONTo(enc); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.EndArray); err != nil {
			return err
		}
		return enc.WriteToken(jsontext.EndObject)

	case ast.MapShape:
		// Maps: {"fn:map": [key1, val1, key2, val2, ...]}
		var args []ast.Constant
		_, err := c.MapValues(
			func(key, val ast.Constant) error {
				args = append(args, key, val)
				return nil
			},
			func() error { return nil },
		)
		if err != nil {
			return fmt.Errorf("failed to iterate map: %w", err)
		}

		if err := enc.WriteToken(jsontext.BeginObject); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String("fn:map")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.BeginArray); err != nil {
			return err
		}
		for _, arg := range args {
			if err := (ConstantJSONLD{arg}).MarshalJSONTo(enc); err != nil {
				return err
			}
		}
		if err := enc.WriteToken(jsontext.EndArray); err != nil {
			return err
		}
		return enc.WriteToken(jsontext.EndObject)

	case ast.StructShape:
		// Structs: {"fn:struct": [label1, val1, label2, val2, ...]}
		var args []ast.Constant
		_, err := c.StructValues(
			func(label, val ast.Constant) error {
				args = append(args, label, val)
				return nil
			},
			func() error { return nil },
		)
		if err != nil {
			return fmt.Errorf("failed to iterate struct: %w", err)
		}

		if err := enc.WriteToken(jsontext.BeginObject); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String("fn:struct")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.BeginArray); err != nil {
			return err
		}
		for _, arg := range args {
			if err := (ConstantJSONLD{arg}).MarshalJSONTo(enc); err != nil {
				return err
			}
		}
		if err := enc.WriteToken(jsontext.EndArray); err != nil {
			return err
		}
		return enc.WriteToken(jsontext.EndObject)

	default:
		return fmt.Errorf("unknown constant type: %d", c.Type)
	}
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom for ConstantJSONLD.
func (c *ConstantJSONLD) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}

	switch tok.Kind() {
	case 'n': // null
		return fmt.Errorf("null values are not supported for constants")

	case '"': // string
		str := tok.String()
		if len(str) > 0 && str[0] == '/' {
			// Name: starts with "/"
			constant, err := ast.Name(str)
			if err != nil {
				return fmt.Errorf("failed to create name from %q: %w", str, err)
			}
			c.Constant = constant
		} else if len(str) >= 3 && str[:2] == `b"` && str[len(str)-1] == '"' {
			// Bytes: starts with b" and ends with "
			escapedContent := str[2 : len(str)-1]
			unescaped, err := ast.Unescape(escapedContent, true /* isBytes */)
			if err != nil {
				return fmt.Errorf("failed to unescape bytes: %w", err)
			}
			c.Constant = ast.Bytes([]byte(unescaped))
		} else {
			// String: regular string
			c.Constant = ast.String(str)
		}

	case '0': // number
		if tok.Float() != float64(int64(tok.Float())) {
			// Float
			c.Constant = ast.Float64(tok.Float())
		} else {
			// Integer
			c.Constant = ast.Number(tok.Int())
		}

	case '{': // object (could be @list, fn:pair, fn:map, fn:struct)
		// Read the key
		keyTok, err := dec.ReadToken()
		if err != nil {
			return fmt.Errorf("failed to read object key: %w", err)
		}
		if keyTok.Kind() != '"' {
			return fmt.Errorf("expected string key, got %c", keyTok.Kind())
		}
		key := keyTok.String()

		switch key {
		case "@list":
			// List: {"@list": [...]}
			if tok, err := dec.ReadToken(); err != nil || tok.Kind() != '[' {
				return fmt.Errorf("expected array for @list")
			}
			var elems []ast.Constant
			for dec.PeekKind() != ']' {
				var elem ConstantJSONLD
				if err := elem.UnmarshalJSONFrom(dec); err != nil {
					return fmt.Errorf("failed to unmarshal list element: %w", err)
				}
				elems = append(elems, elem.Constant)
			}
			if _, err := dec.ReadToken(); err != nil {
				return fmt.Errorf("failed to read array end: %w", err)
			}
			if tok, err := dec.ReadToken(); err != nil || tok.Kind() != '}' {
				return fmt.Errorf("expected object end")
			}
			c.Constant = ast.List(elems)

		case "fn:pair", "fn:map", "fn:struct":
			// Read the args array
			if tok, err := dec.ReadToken(); err != nil || tok.Kind() != '[' {
				return fmt.Errorf("expected array for %s", key)
			}
			var args []ast.Constant
			for dec.PeekKind() != ']' {
				var arg ConstantJSONLD
				if err := arg.UnmarshalJSONFrom(dec); err != nil {
					return fmt.Errorf("failed to unmarshal arg: %w", err)
				}
				args = append(args, arg.Constant)
			}
			if _, err := dec.ReadToken(); err != nil {
				return fmt.Errorf("failed to read array end: %w", err)
			}
			if tok, err := dec.ReadToken(); err != nil || tok.Kind() != '}' {
				return fmt.Errorf("expected object end")
			}

			switch key {
			case "fn:pair":
				if len(args) != 2 {
					return fmt.Errorf("fn:pair expects 2 args, got %d", len(args))
				}
				c.Constant = ast.Pair(&args[0], &args[1])

			case "fn:map":
				if len(args)%2 != 0 {
					return fmt.Errorf("fn:map expects even number of args, got %d", len(args))
				}
				kvMap := make(map[*ast.Constant]*ast.Constant)
				for i := 0; i < len(args); i += 2 {
					kvMap[&args[i]] = &args[i+1]
				}
				mapConst := ast.Map(kvMap)
				c.Constant = *mapConst

			case "fn:struct":
				if len(args)%2 != 0 {
					return fmt.Errorf("fn:struct expects even number of args, got %d", len(args))
				}
				kvMap := make(map[*ast.Constant]*ast.Constant)
				for i := 0; i < len(args); i += 2 {
					kvMap[&args[i]] = &args[i+1]
				}
				structConst := ast.Struct(kvMap)
				c.Constant = *structConst
			}

		default:
			return fmt.Errorf("unknown JSON-LD object key: %s", key)
		}

	case '[': // bare array (shouldn't happen in JSON-LD)
		return fmt.Errorf("bare arrays not supported; use @list")

	default:
		return fmt.Errorf("unexpected JSON token kind: %c", tok.Kind())
	}

	return nil
}

// MarshalJSONTo implements json.MarshalerTo for AtomJSONLD.
// Uses hybrid arity-based mapping to JSON-LD.
func (a AtomJSONLD) MarshalJSONTo(enc *jsontext.Encoder) error {
	// Begin document object
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}

	// Write @context
	if err := enc.WriteToken(jsontext.String("@context")); err != nil {
		return err
	}
	if err := writeContext(enc); err != nil {
		return err
	}

	// Arity-based routing
	switch a.Predicate.Arity {
	case 0:
		// Arity 0: {"@context": {...}, "@type": "predicate"}
		if err := enc.WriteToken(jsontext.String("@type")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String(a.Predicate.Symbol)); err != nil {
			return err
		}

	case 1:
		// Arity 1: {"@context": {...}, "@id": arg0, "@type": "predicate"}
		arg0, ok := a.Args[0].(ast.Constant)
		if !ok {
			return fmt.Errorf("arg0 is not a constant")
		}

		if err := enc.WriteToken(jsontext.String("@id")); err != nil {
			return err
		}
		if err := (ConstantJSONLD{arg0}).MarshalJSONTo(enc); err != nil {
			return err
		}

		if err := enc.WriteToken(jsontext.String("@type")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String(a.Predicate.Symbol)); err != nil {
			return err
		}

	case 2:
		// Arity 2: {"@context": {...}, "@id": arg0, "predicate": arg1}
		arg0, ok := a.Args[0].(ast.Constant)
		if !ok {
			return fmt.Errorf("arg0 is not a constant")
		}
		arg1, ok := a.Args[1].(ast.Constant)
		if !ok {
			return fmt.Errorf("arg1 is not a constant")
		}

		if err := enc.WriteToken(jsontext.String("@id")); err != nil {
			return err
		}
		if err := (ConstantJSONLD{arg0}).MarshalJSONTo(enc); err != nil {
			return err
		}

		if err := enc.WriteToken(jsontext.String(a.Predicate.Symbol)); err != nil {
			return err
		}
		if err := (ConstantJSONLD{arg1}).MarshalJSONTo(enc); err != nil {
			return err
		}

	default:
		// Arity 3+: {"@context": {...}, "@type": "predicate", "arg0": val, "arg1": val, ...}
		if err := enc.WriteToken(jsontext.String("@type")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String(a.Predicate.Symbol)); err != nil {
			return err
		}

		for i, arg := range a.Args {
			c, ok := arg.(ast.Constant)
			if !ok {
				return fmt.Errorf("arg%d is not a constant", i)
			}

			if err := enc.WriteToken(jsontext.String(fmt.Sprintf("arg%d", i))); err != nil {
				return err
			}
			if err := (ConstantJSONLD{c}).MarshalJSONTo(enc); err != nil {
				return err
			}
		}
	}

	// End document object
	return enc.WriteToken(jsontext.EndObject)
}

// marshalAtomTo writes a single atom to the encoder without the surrounding object braces or context.
// This allows it to be reused by both AtomJSONLD and AtomsJSONLD.
func marshalAtomTo(enc *jsontext.Encoder, atom ast.Atom) error {
	// Arity-based routing
	switch atom.Predicate.Arity {
	case 0:
		// Arity 0: {"@type": "predicate"}
		if err := enc.WriteToken(jsontext.String("@type")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String(atom.Predicate.Symbol)); err != nil {
			return err
		}

	case 1:
		// Arity 1: {"@id": arg0, "@type": "predicate"}
		arg0, ok := atom.Args[0].(ast.Constant)
		if !ok {
			return fmt.Errorf("arg0 is not a constant")
		}

		if err := enc.WriteToken(jsontext.String("@id")); err != nil {
			return err
		}
		if err := (ConstantJSONLD{arg0}).MarshalJSONTo(enc); err != nil {
			return err
		}

		if err := enc.WriteToken(jsontext.String("@type")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String(atom.Predicate.Symbol)); err != nil {
			return err
		}

	case 2:
		// Arity 2: {"@id": arg0, "predicate": arg1}
		arg0, ok := atom.Args[0].(ast.Constant)
		if !ok {
			return fmt.Errorf("arg0 is not a constant")
		}
		arg1, ok := atom.Args[1].(ast.Constant)
		if !ok {
			return fmt.Errorf("arg1 is not a constant")
		}

		if err := enc.WriteToken(jsontext.String("@id")); err != nil {
			return err
		}
		if err := (ConstantJSONLD{arg0}).MarshalJSONTo(enc); err != nil {
			return err
		}

		if err := enc.WriteToken(jsontext.String(atom.Predicate.Symbol)); err != nil {
			return err
		}
		if err := (ConstantJSONLD{arg1}).MarshalJSONTo(enc); err != nil {
			return err
		}

	default:
		// Arity 3+: {"@type": "predicate", "arg0": val, "arg1": val, ...}
		if err := enc.WriteToken(jsontext.String("@type")); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String(atom.Predicate.Symbol)); err != nil {
			return err
		}

		for i, arg := range atom.Args {
			c, ok := arg.(ast.Constant)
			if !ok {
				return fmt.Errorf("arg%d is not a constant", i)
			}

			if err := enc.WriteToken(jsontext.String(fmt.Sprintf("arg%d", i))); err != nil {
				return err
			}
			if err := (ConstantJSONLD{c}).MarshalJSONTo(enc); err != nil {
				return err
			}
		}
	}
	return nil
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom for AtomJSONLD.
// Detects arity from JSON-LD structure and reconstructs the atom.
func (a *AtomJSONLD) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	// Expect object start
	tok, err := dec.ReadToken()
	if err != nil {
		return fmt.Errorf("failed to read atom start: %w", err)
	}
	if tok.Kind() != '{' {
		return fmt.Errorf("expected object start '{', got %c", tok.Kind())
	}

	// Use shared unmarshal logic
	atom, err := unmarshalAtomFrom(dec)
	if err != nil {
		return err
	}

	a.Atom = atom
	return nil
}

// MarshalJSONTo implements json.MarshalerTo for AtomsJSONLD.
// Serializes multiple atoms as a JSON-LD @graph.
func (aa AtomsJSONLD) MarshalJSONTo(enc *jsontext.Encoder) error {
	// Begin document object
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}

	// Write @context
	if err := enc.WriteToken(jsontext.String("@context")); err != nil {
		return err
	}
	if err := writeContext(enc); err != nil {
		return err
	}

	// Write @graph
	if err := enc.WriteToken(jsontext.String("@graph")); err != nil {
		return err
	}

	// Write atoms array
	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		return err
	}

	for _, atom := range aa.Atoms {
		// Write each atom without its own @context (shared at document level)
		if err := marshalAtomObject(enc, atom); err != nil {
			return err
		}
	}

	// End @graph array
	if err := enc.WriteToken(jsontext.EndArray); err != nil {
		return err
	}

	// End document object
	return enc.WriteToken(jsontext.EndObject)
}

// marshalAtomObject writes a complete atom object '{...}' to the encoder.
func marshalAtomObject(enc *jsontext.Encoder, atom ast.Atom) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := marshalAtomTo(enc, atom); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom for AtomsJSONLD.
func (aa *AtomsJSONLD) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	// Expect object start
	tok, err := dec.ReadToken()
	if err != nil {
		return fmt.Errorf("failed to read document start: %w", err)
	}
	if tok.Kind() != '{' {
		return fmt.Errorf("expected object start '{', got %c", tok.Kind())
	}

	var atoms []ast.Atom

	// Read object properties
	for dec.PeekKind() != '}' {
		keyTok, err := dec.ReadToken()
		if err != nil {
			return fmt.Errorf("failed to read key: %w", err)
		}
		if keyTok.Kind() != '"' {
			return fmt.Errorf("expected string key, got %c", keyTok.Kind())
		}
		key := keyTok.String()

		switch key {
		case "@context":
			// Skip context
			if err := dec.SkipValue(); err != nil {
				return fmt.Errorf("failed to skip @context: %w", err)
			}

		case "@graph":
			// Read array of atoms
			tok, err := dec.ReadToken()
			if err != nil {
				return fmt.Errorf("failed to read @graph: %w", err)
			}
			if tok.Kind() != '[' {
				return fmt.Errorf("expected array for @graph, got %c", tok.Kind())
			}

			for dec.PeekKind() != ']' {
				// unmarshalAtomFrom expects the opening '{' to be read.
				if tok, err := dec.ReadToken(); err != nil || tok.Kind() != '{' {
					return fmt.Errorf("expected atom object start, got %v", tok)
				}
				atom, err := unmarshalAtomFrom(dec)
				if err != nil {
					return fmt.Errorf("failed to unmarshal atom from @graph: %w", err)
				}

				atoms = append(atoms, atom)
			}

			// Read @graph array end
			if _, err := dec.ReadToken(); err != nil {
				return fmt.Errorf("failed to read @graph array end: %w", err)
			}

		default:
			// Skip unknown properties
			if err := dec.SkipValue(); err != nil {
				return fmt.Errorf("failed to skip unknown property %s: %w", key, err)
			}
		}
	}

	// Read document object end
	if _, err := dec.ReadToken(); err != nil {
		return fmt.Errorf("failed to read document end: %w", err)
	}

	aa.Atoms = atoms
	return nil
}

// writeContext writes the default JSON-LD context to the encoder.
func writeContext(enc *jsontext.Encoder) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}

	// @vocab
	if err := enc.WriteToken(jsontext.String("@vocab")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(MangleNamespace)); err != nil {
		return err
	}

	// xsd
	if err := enc.WriteToken(jsontext.String("xsd")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("http://www.w3.org/2001/XMLSchema#")); err != nil {
		return err
	}

	// arg0-arg9
	for i := range 10 {
		argKey := fmt.Sprintf("arg%d", i)
		if err := enc.WriteToken(jsontext.String(argKey)); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String(MangleNamespace + argKey)); err != nil {
			return err
		}
	}

	// fn:pair
	if err := enc.WriteToken(jsontext.String("fn:pair")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("@id")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(MangleNamespace + "pair")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("@container")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("@list")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.EndObject); err != nil {
		return err
	}

	// fn:map
	if err := enc.WriteToken(jsontext.String("fn:map")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("@id")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(MangleNamespace + "map")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("@container")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("@list")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.EndObject); err != nil {
		return err
	}

	// fn:struct
	if err := enc.WriteToken(jsontext.String("fn:struct")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("@id")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String(MangleNamespace + "struct")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("@container")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.String("@list")); err != nil {
		return err
	}
	if err := enc.WriteToken(jsontext.EndObject); err != nil {
		return err
	}

	return enc.WriteToken(jsontext.EndObject)
}

// unmarshalAtomFrom reads an atom's properties from the decoder and reconstructs it.
// It assumes the opening '{' has been read.
func unmarshalAtomFrom(dec *jsontext.Decoder) (ast.Atom, error) {
	var id *ast.Constant
	var typeVal string
	properties := make(map[string]ast.Constant)

	// Read object properties
	for dec.PeekKind() != '}' {
		keyTok, err := dec.ReadToken()
		if err != nil {
			return ast.Atom{}, fmt.Errorf("failed to read key: %w", err)
		}
		if keyTok.Kind() != '"' {
			return ast.Atom{}, fmt.Errorf("expected string key, got %c", keyTok.Kind())
		}
		key := keyTok.String()

		switch key {
		case "@context":
			// Skip context (we don't need to parse it)
			if err := dec.SkipValue(); err != nil {
				return ast.Atom{}, fmt.Errorf("failed to skip @context: %w", err)
			}

		case "@id":
			var val ConstantJSONLD
			if err := val.UnmarshalJSONFrom(dec); err != nil {
				return ast.Atom{}, fmt.Errorf("failed to unmarshal @id: %w", err)
			}
			id = &val.Constant

		case "@type":
			tok, err := dec.ReadToken()
			if err != nil {
				return ast.Atom{}, fmt.Errorf("failed to read @type: %w", err)
			}
			if tok.Kind() != '"' {
				return ast.Atom{}, fmt.Errorf("expected string for @type, got %c", tok.Kind())
			}
			typeVal = tok.String()

		default:
			// This is a property (either arity 2 predicate or argN)
			var val ConstantJSONLD
			if err := val.UnmarshalJSONFrom(dec); err != nil {
				return ast.Atom{}, fmt.Errorf("failed to unmarshal property %s: %w", key, err)
			}
			properties[key] = val.Constant
		}
	}

	// Read object end
	if _, err := dec.ReadToken(); err != nil {
		return ast.Atom{}, fmt.Errorf("failed to read object end: %w", err)
	}

	// Reconstruct atom
	return reconstructAtom(id, typeVal, properties)
}

// reconstructAtom builds an ast.Atom from parsed JSON-LD properties.
// Detects arity based on the presence of @id, @type, and other properties.
func reconstructAtom(id *ast.Constant, typeVal string, properties map[string]ast.Constant) (ast.Atom, error) {
	if typeVal != "" && id == nil && len(properties) == 0 {
		// Arity 0: has @type only
		return ast.Atom{
			Predicate: ast.PredicateSym{Symbol: typeVal, Arity: 0},
			Args:      []ast.BaseTerm{},
		}, nil
	} else if typeVal != "" && id != nil && len(properties) == 0 {
		// Arity 1: has @id and @type
		return ast.Atom{
			Predicate: ast.PredicateSym{Symbol: typeVal, Arity: 1},
			Args:      []ast.BaseTerm{*id},
		}, nil
	} else if id != nil && typeVal == "" && len(properties) == 1 {
		// Arity 2: has @id and one property
		var predicate string
		var object ast.Constant
		for k, v := range properties {
			predicate = k
			object = v
		}
		return ast.Atom{
			Predicate: ast.PredicateSym{Symbol: predicate, Arity: 2},
			Args:      []ast.BaseTerm{*id, object},
		}, nil
	} else if typeVal != "" && len(properties) > 0 {
		// Arity 3+: has @type and arg0, arg1, ...
		args := make([]ast.BaseTerm, len(properties))
		for i := 0; i < len(properties); i++ {
			argKey := fmt.Sprintf("arg%d", i)
			val, ok := properties[argKey]
			if !ok {
				return ast.Atom{}, fmt.Errorf("missing %s for arity %d predicate", argKey, len(properties))
			}
			args[i] = val
		}
		return ast.Atom{
			Predicate: ast.PredicateSym{Symbol: typeVal, Arity: len(properties)},
			Args:      args,
		}, nil
	}

	return ast.Atom{}, fmt.Errorf("cannot determine atom arity from JSON-LD structure")
}
