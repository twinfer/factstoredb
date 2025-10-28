package factstoredb

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/scanner"

	"github.com/google/mangle/ast"
)

const (
	fnPrefix = "fn:"
	fnPair   = "fn:pair"
	fnMap    = "fn:map"
	fnStruct = "fn:struct"
)

// initConstantScanner initializes and configures a scanner for parsing ast.Constant values.
// It configures the scanner to recognize Mangle's identifiers and operators.
func initConstantScanner(r io.Reader) *scanner.Scanner {
	var s scanner.Scanner
	s.Init(r)
	// Configure scanner to recognize Mangle's identifiers and operators.
	// This is crucial for "fn:pair" to be tokenized as a single ident.
	s.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanFloats | scanner.ScanInts | scanner.ScanRawStrings
	s.IsIdentRune = func(ch rune, i int) bool {
		// Default ident runes, plus ':' for fn: symbols.
		return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' || (ch >= '0' && ch <= '9' && i > 0) || ch == ':' || ch == '.'
	}
	return &s
}

// ParseConstantFromReader parses a constant from an io.Reader.
// This is useful for reading constants from files, network connections, or other streams.
// The caller is responsible for managing the lifecycle of the reader (e.g., closing files).
func ParseConstantFromReader(r io.Reader) (ast.Constant, error) {
	s := initConstantScanner(r)
	return parseConstantRecursive(s)
}

// ParseConstantFromString parses a string representation of an ast.Constant
// into an ast.Constant object. For reading from files or streams, use ParseConstantFromReader.
func ParseConstantFromString(input string) (ast.Constant, error) {
	return ParseConstantFromReader(strings.NewReader(input))
}

// isNameChar returns true if the given rune is a valid character in a name/path.
func isNameChar(ch rune) bool {
	return ch == '/' || ch == '_' ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9')
}

// parseError creates a formatted error message with scanner position information.
func parseError(s *scanner.Scanner, format string, args ...any) error {
	pos := s.Pos()
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("parse error at %s: %s", pos, msg)
}

// parseConstantRecursive uses a text/scanner to parse a string representation of an
// ast.Constant into an ast.Constant object. This is a simple recursive descent parser.
func parseConstantRecursive(s *scanner.Scanner) (ast.Constant, error) {
	tok := s.Scan()
	text := s.TokenText()

	switch tok {
	case scanner.Ident: // Handles 'b' for bytes, and 'fn:...' for functional terms
		// Handle cases like `b"..."` for bytes.
		if text == "b" && s.Peek() == '"' {
			tok = s.Scan() // Scan the string part that follows 'b'
			if tok != scanner.String {
				return ast.Constant{}, fmt.Errorf("expected string after 'b' for bytes literal, got %q", s.TokenText())
			}
			escapedContent, err := strconv.Unquote(s.TokenText())
			if err != nil {
				return ast.Constant{}, fmt.Errorf("could not unquote bytes string %q: %w", s.TokenText(), err)
			}
			unescaped, err := ast.Unescape(escapedContent, true /* isBytes */)
			if err != nil {
				return ast.Constant{}, fmt.Errorf("failed to unescape bytes: %w", err)
			}
			return ast.Bytes([]byte(unescaped)), nil
		}

		// Handle functional terms like fn:pair, fn:map, fn:struct
		if strings.HasPrefix(text, fnPrefix) {
			symbol := text
			if s.Scan() != '(' { // Expect '('
				return ast.Constant{}, fmt.Errorf("expected '(' after function symbol %q", symbol)
			}

			var args []ast.Constant
			if s.Peek() != ')' { // Not an empty arg list
				for {
					arg, err := parseConstantRecursive(s)
					if err != nil {
						return ast.Constant{}, fmt.Errorf("failed to parse arg for %q: %w", symbol, err)
					}
					args = append(args, arg)
					if s.Peek() == ',' {
						s.Scan() // consume ','
					} else {
						break
					}
				}
			}
			if s.Scan() != ')' { // Expect ')'
				return ast.Constant{}, fmt.Errorf("expected ')' to end function call %q", symbol)
			}

			switch symbol {
			case fnPair:
				if len(args) != 2 {
					return ast.Constant{}, fmt.Errorf("%s expects 2 args, got %d", fnPair, len(args))
				}
				return ast.Pair(&args[0], &args[1]), nil
			case fnMap:
				if len(args)%2 != 0 {
					return ast.Constant{}, fmt.Errorf("%s expects even number of args, got %d", fnMap, len(args))
				}
				kvMap := make(map[*ast.Constant]*ast.Constant)
				for i := 0; i < len(args); i += 2 {
					key := args[i]
					val := args[i+1]
					kvMap[&key] = &val
				}
				return *ast.Map(kvMap), nil
			case fnStruct:
				if len(args)%2 != 0 {
					return ast.Constant{}, fmt.Errorf("%s expects even number of args, got %d", fnStruct, len(args))
				}
				kvMap := make(map[*ast.Constant]*ast.Constant)
				for i := 0; i < len(args); i += 2 {
					key := args[i]
					val := args[i+1]
					kvMap[&key] = &val
				}
				return *ast.Struct(kvMap), nil
			default:
				return ast.Constant{}, parseError(s, "unknown function symbol: %s", symbol)
			}
		}
		// If not 'b' and not 'fn:', then it's an unexpected ident.
		return ast.Constant{}, parseError(s, "unexpected identifier: %s", text)

	case scanner.String:
		// Raw string content, includes quotes.
		unquoted, err := strconv.Unquote(text)
		if err != nil {
			return ast.Constant{}, fmt.Errorf("could not unquote string %q: %w", text, err)
		}
		return ast.String(unquoted), nil

	case scanner.Int:
		val, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return ast.Constant{}, fmt.Errorf("could not parse int %q: %w", text, err)
		}
		return ast.Number(val), nil

	case scanner.Float:
		val, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return ast.Constant{}, fmt.Errorf("could not parse float %q: %w", text, err)
		}
		return ast.Float64(val), nil

	case '-': // Negative number or float
		nextTok := s.Scan()
		nextText := s.TokenText()
		switch nextTok {
		case scanner.Int:
			val, err := strconv.ParseInt("-"+nextText, 10, 64)
			if err != nil {
				return ast.Constant{}, fmt.Errorf("could not parse negative int %q: %w", "-"+nextText, err)
			}
			return ast.Number(val), nil
		case scanner.Float:
			val, err := strconv.ParseFloat("-"+nextText, 64)
			if err != nil {
				return ast.Constant{}, fmt.Errorf("could not parse negative float %q: %w", "-"+nextText, err)
			}
			return ast.Float64(val), nil
		default:
			return ast.Constant{}, parseError(s, "expected number after '-', got %q", nextText)
		}

	case '/': // This is how we identify a Name.
		// scanner.Scan() stops at operators, so we need to read the rest of the name.
		// The default scanner tokenizes this as a sequence of '/', 'ident', '/', 'ident'.
		// We need to piece them together.
		var nameBuilder strings.Builder
		nameBuilder.WriteString("/") // Start with the initial '/'
		for {
			s.Scan()
			nameBuilder.WriteString(s.TokenText())
			// A name is a path, so it can be followed by another '/' or something else.
			// We stop when the next character is not part of a path-like identifier.
			if !isNameChar(s.Peek()) {
				break
			}
		}
		name := nameBuilder.String()
		c, err := ast.Name(name)
		if err != nil {
			return ast.Constant{}, fmt.Errorf("failed to create name from %q: %w", name, err)
		}
		return c, nil

	case '[':
		// This can be a List `[1, 2]` or a Map `[/a:1, /b:2]`.
		// We need to look ahead to distinguish them. A map will have a ':' after the first element.
		if s.Peek() == ']' { // Empty list
			s.Scan() // Consume ']'
			return ast.List(nil), nil
		}

		// Tentatively parse the first element.
		firstElem, err := parseConstantRecursive(s)
		if err != nil {
			return ast.Constant{}, fmt.Errorf("failed to parse first element in list/map: %w", err)
		}

		// Check the next token to decide if it's a map or a list.
		tokAfterFirst := s.Scan()
		tokText := s.TokenText()
		// Note: ':' might be scanned as an Ident (due to IsIdentRune config) or as the rune ':'
		isColon := tokAfterFirst == ':' || (tokAfterFirst == scanner.Ident && tokText == ":")

		if isColon {
			// It's a map.
			firstVal, err := parseConstantRecursive(s)
			if err != nil {
				return ast.Constant{}, fmt.Errorf("failed to parse map value for key %v: %w", firstElem, err)
			}
			kvMap := make(map[*ast.Constant]*ast.Constant)
			firstKey := firstElem
			kvMap[&firstKey] = &firstVal

			// Continue parsing the rest of the map.
			for s.Peek() == ',' {
				s.Scan()             // consume ','
				if s.Peek() == ']' { // Handle trailing comma
					break
				}
				key, err := parseConstantRecursive(s)
				if err != nil {
					return ast.Constant{}, fmt.Errorf("failed to parse subsequent map key: %w", err)
				}
				tok := s.Scan()
				tokText := s.TokenText()
				if tok != ':' && !(tok == scanner.Ident && tokText == ":") {
					return ast.Constant{}, fmt.Errorf("expected ':' after map key %v", key)
				}
				val, err := parseConstantRecursive(s)
				if err != nil {
					return ast.Constant{}, fmt.Errorf("failed to parse map value for key %v: %w", key, err)
				}
				kvMap[&key] = &val
			}
			if tok := s.Scan(); tok != ']' {
				return ast.Constant{}, fmt.Errorf("expected ']' to end map, got %q", s.TokenText())
			}
			return *ast.Map(kvMap), nil
		} else if tokAfterFirst == ',' {
			// It's a list with more elements.
			elems := []ast.Constant{firstElem}
			// The first comma was already consumed. Now we parse the rest of the elements.
			for {
				if s.Peek() == ']' { // Handle trailing comma
					break
				}
				elem, err := parseConstantRecursive(s)
				if err != nil {
					return ast.Constant{}, fmt.Errorf("failed to parse list element: %w", err)
				}
				elems = append(elems, elem)

				if s.Peek() == ',' {
					s.Scan() // consume comma before next element
				} else {
					break // No more commas, so no more elements.
				}
			}
			if s.Scan() != ']' {
				return ast.Constant{}, fmt.Errorf("expected ']' to end list, got %q", s.TokenText())
			}
			return ast.List(elems), nil
		} else if tokAfterFirst == ']' {
			// It's a list with a single element.
			return ast.List([]ast.Constant{firstElem}), nil
		} else {
			return ast.Constant{}, fmt.Errorf("unexpected token %q after first element in list/map", s.TokenText())
		}

	case '{': // Struct
		if s.Peek() == '}' { // Empty struct
			s.Scan() // Consume '}'
			return *ast.Struct(make(map[*ast.Constant]*ast.Constant)), nil
		}

		kvMap := make(map[*ast.Constant]*ast.Constant)
		for {
			parsedKey, err := parseConstantRecursive(s)
			if err != nil {
				return ast.Constant{}, fmt.Errorf("failed to parse struct key: %w", err)
			}

			tok := s.Scan()
			tokText := s.TokenText()
			if tok != ':' && !(tok == scanner.Ident && tokText == ":") {
				return ast.Constant{}, fmt.Errorf("expected ':' after struct key %v", parsedKey)
			}

			parsedVal, err := parseConstantRecursive(s)
			if err != nil {
				return ast.Constant{}, fmt.Errorf("failed to parse struct value for key %v: %w", parsedKey, err)
			}
			// Copy to new variables before taking addresses to avoid pointer aliasing
			key := parsedKey
			val := parsedVal
			kvMap[&key] = &val

			if s.Peek() != ',' {
				break
			}
			s.Scan()             // consume ','
			if s.Peek() == '}' { // Handle trailing comma
				break
			}
		}

		if tok := s.Scan(); tok != '}' {
			return ast.Constant{}, fmt.Errorf("expected '}' to end struct, got %q", s.TokenText())
		}
		return *ast.Struct(kvMap), nil

	case scanner.EOF:
		return ast.Constant{}, parseError(s, "unexpected EOF")

	default:
		return ast.Constant{}, parseError(s, "unhandled token: %q (%s)", text, scanner.TokenString(tok))
	}
}
