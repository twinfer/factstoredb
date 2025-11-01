package jsonld

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/mangle/ast"
	"github.com/piprate/json-gold/ld"
)

// RDF namespace constants
const (
	RDFType      = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
	RDFStatement = "http://www.w3.org/1999/02/22-rdf-syntax-ns#Statement"
	RDFSubject   = "http://www.w3.org/1999/02/22-rdf-syntax-ns#subject"
	RDFPredicate = "http://www.w3.org/1999/02/22-rdf-syntax-ns#predicate"
	RDFObject    = "http://www.w3.org/1999/02/22-rdf-syntax-ns#object"
)

// MangleNamespace is the default namespace URI for Mangle predicates and concepts.
const MangleNamespace = "http://mangle.datalog.org/"

// AtomsToRDF converts a slice of Mangle ast.Atom to an RDF dataset.
// Uses arity-based mapping:
// - Arity 0: generates blank node with rdf:type predicate
// - Arity 1: subject has type
// - Arity 2: direct triple (subject-predicate-object)
// - Arity 3+: reification pattern with additional arguments
func AtomsToRDF(atoms []ast.Atom) (*ld.RDFDataset, error) {
	dataset := ld.NewRDFDataset()
	issuer := ld.NewIdentifierIssuer("_:b")

	for _, atom := range atoms {
		quads, err := atomToRDFQuads(atom, issuer)
		if err != nil {
			return nil, fmt.Errorf("failed to convert atom to RDF: %w", err)
		}

		// Add quads to the default graph
		dataset.Graphs["@default"] = append(dataset.Graphs["@default"], quads...)
	}

	return dataset, nil
}

// atomToRDFQuads converts a single ast.Atom to RDF quads based on arity.
func atomToRDFQuads(atom ast.Atom, issuer *ld.IdentifierIssuer) ([]*ld.Quad, error) {
	predicateIRI := MangleNamespace + atom.Predicate.Symbol
	arity := atom.Predicate.Arity

	switch arity {
	case 0:
		// Arity 0: _:bN rdf:type predicate
		blankNode := ld.NewBlankNode(issuer.GetId(""))
		quad := ld.NewQuad(
			blankNode,
			ld.NewIRI(RDFType),
			ld.NewIRI(predicateIRI),
			"@default",
		)
		return []*ld.Quad{quad}, nil

	case 1:
		// Arity 1: arg0 rdf:type predicate
		subject, err := constantToRDFNode(atom.Args[0].(ast.Constant))
		if err != nil {
			return nil, fmt.Errorf("failed to convert arity 1 subject: %w", err)
		}
		quad := ld.NewQuad(
			subject,
			ld.NewIRI(RDFType),
			ld.NewIRI(predicateIRI),
			"@default",
		)
		return []*ld.Quad{quad}, nil

	case 2:
		// Arity 2: arg0 predicate arg1
		subject, err := constantToRDFNode(atom.Args[0].(ast.Constant))
		if err != nil {
			return nil, fmt.Errorf("failed to convert arity 2 subject: %w", err)
		}
		object, err := constantToRDFNode(atom.Args[1].(ast.Constant))
		if err != nil {
			return nil, fmt.Errorf("failed to convert arity 2 object: %w", err)
		}
		quad := ld.NewQuad(
			subject,
			ld.NewIRI(predicateIRI),
			object,
			"@default",
		)
		return []*ld.Quad{quad}, nil

	default:
		// Arity 3+: Use reification
		return reifyAtom(atom, predicateIRI, issuer)
	}
}

// reifyAtom creates a reified statement for n-ary relations (arity >= 3).
// Pattern:
//
//	_:stmt rdf:type rdf:Statement
//	_:stmt rdf:subject arg0
//	_:stmt rdf:predicate predicateIRI
//	_:stmt rdf:object arg1
//	_:stmt mangle:arg2 arg2
//	_:stmt mangle:arg3 arg3
//	...
func reifyAtom(atom ast.Atom, predicateIRI string, issuer *ld.IdentifierIssuer) ([]*ld.Quad, error) {
	stmtNode := ld.NewBlankNode(issuer.GetId("stmt"))
	var quads []*ld.Quad

	// rdf:type rdf:Statement
	quads = append(quads, ld.NewQuad(
		stmtNode,
		ld.NewIRI(RDFType),
		ld.NewIRI(RDFStatement),
		"@default",
	))

	// rdf:subject arg0
	subject, err := constantToRDFNode(atom.Args[0].(ast.Constant))
	if err != nil {
		return nil, fmt.Errorf("failed to convert reified subject: %w", err)
	}
	quads = append(quads, ld.NewQuad(
		stmtNode,
		ld.NewIRI(RDFSubject),
		subject,
		"@default",
	))

	// rdf:predicate predicateIRI
	quads = append(quads, ld.NewQuad(
		stmtNode,
		ld.NewIRI(RDFPredicate),
		ld.NewIRI(predicateIRI),
		"@default",
	))

	// rdf:object arg1
	object, err := constantToRDFNode(atom.Args[1].(ast.Constant))
	if err != nil {
		return nil, fmt.Errorf("failed to convert reified object: %w", err)
	}
	quads = append(quads, ld.NewQuad(
		stmtNode,
		ld.NewIRI(RDFObject),
		object,
		"@default",
	))

	// Additional arguments: mangle:arg2, mangle:arg3, ...
	for i := 2; i < len(atom.Args); i++ {
		argPredicate := fmt.Sprintf("%sarg%d", MangleNamespace, i)
		argNode, err := constantToRDFNode(atom.Args[i].(ast.Constant))
		if err != nil {
			return nil, fmt.Errorf("failed to convert arg%d: %w", i, err)
		}
		quads = append(quads, ld.NewQuad(
			stmtNode,
			ld.NewIRI(argPredicate),
			argNode,
			"@default",
		))
	}

	return quads, nil
}

// constantToRDFNode converts an ast.Constant to an RDF node.
func constantToRDFNode(c ast.Constant) (ld.Node, error) {
	switch c.Type {
	case ast.NameType:
		// Names are IRIs (e.g., "/alice")
		sym, err := c.NameValue()
		if err != nil {
			return nil, fmt.Errorf("failed to get name value: %w", err)
		}
		return ld.NewIRI(sym), nil

	case ast.StringType:
		// Strings are literals
		str, err := c.StringValue()
		if err != nil {
			return nil, fmt.Errorf("failed to get string value: %w", err)
		}
		return ld.NewLiteral(str, "http://www.w3.org/2001/XMLSchema#string", ""), nil

	case ast.NumberType:
		// Numbers are integer literals
		num, err := c.NumberValue()
		if err != nil {
			return nil, fmt.Errorf("failed to get number value: %w", err)
		}
		return ld.NewLiteral(strconv.FormatInt(num, 10), "http://www.w3.org/2001/XMLSchema#integer", ""), nil

	case ast.Float64Type:
		// Floats are double literals
		flt, err := c.Float64Value()
		if err != nil {
			return nil, fmt.Errorf("failed to get float64 value: %w", err)
		}
		return ld.NewLiteral(fmt.Sprintf("%g", flt), "http://www.w3.org/2001/XMLSchema#double", ""), nil

	case ast.BytesType:
		// Bytes are base64Binary literals
		// The raw bytes are stored in the Symbol field for BytesType.
		encoded := base64.StdEncoding.EncodeToString([]byte(c.Symbol))
		return ld.NewLiteral(encoded, "http://www.w3.org/2001/XMLSchema#base64Binary", ""), nil

	default:
		// For complex types (lists, maps, etc.), use string representation
		return ld.NewLiteral(c.Symbol, "http://www.w3.org/2001/XMLSchema#string", ""), nil
	}
}

// RDFToAtoms converts an RDF dataset to a slice of Mangle ast.Atom.
// Detects arity based on RDF patterns:
// - rdf:type triple with no other properties → arity 0 or 1
// - simple triple → arity 2
// - reification pattern → arity 3+
// - W3C n-ary pattern → detect and reconstruct
func RDFToAtoms(dataset *ld.RDFDataset, graphName string) ([]ast.Atom, error) {
	if graphName == "" {
		graphName = "@default"
	}

	quads := dataset.GetQuads(graphName)
	if quads == nil {
		return []ast.Atom{}, nil
	}

	// Group quads by subject to detect patterns
	subjectQuads := make(map[string][]*ld.Quad)
	for _, quad := range quads {
		subjectKey := nodeToString(quad.Subject)
		subjectQuads[subjectKey] = append(subjectQuads[subjectKey], quad)
	}

	// Process each subject's quads to reconstruct atoms
	var atoms []ast.Atom
	for _, subQuads := range subjectQuads {
		// Try to detect reification pattern first, as it's the most specific.
		if atom, ok := tryDetectReification(subQuads[0].Subject, subQuads); ok {
			atoms = append(atoms, atom)
			continue
		}

		// Try to detect W3C n-ary pattern
		if atom, ok := tryDetectNaryPattern(subQuads[0].Subject, subQuads); ok {
			atoms = append(atoms, atom)
			continue
		}

		// If not a complex pattern, process as simple triples.
		// A single subject can be part of multiple simple triples.
		for _, quad := range subQuads {
			// Skip subjects that are reified statements, as they are not atoms themselves.
			if nodeToString(quad.Predicate) == RDFSubject ||
				nodeToString(quad.Predicate) == RDFPredicate ||
				nodeToString(quad.Predicate) == RDFObject {
				continue
			}

			// Handle rdf:type (arity 0 or 1)
			if nodeToString(quad.Predicate) == RDFType {
				// Don't process rdf:Statement types here; they are handled by reification detection.
				if nodeToString(quad.Object) == RDFStatement {
					continue
				}
				atom, err := rdfTypeToAtom(quad)
				if err != nil {
					return nil, err
				}
				atoms = append(atoms, atom)
			} else { // Handle simple binary triple (arity 2)
				atom, err := simpleTripleToAtom(quad)
				if err != nil {
					return nil, err
				}
				atoms = append(atoms, atom)
			}
		}
	}

	return atoms, nil
}

// tryDetectReification checks if the quads represent a reified statement.
// Returns the atom if detected, false otherwise.
func tryDetectReification(subject ld.Node, quads []*ld.Quad) (ast.Atom, bool) {
	// Look for rdf:Statement type
	hasStatementType := false
	var rdfSubject, rdfPredicate, rdfObject ld.Node
	additionalArgs := make(map[int]ld.Node)

	for _, quad := range quads {
		predIRI := nodeToString(quad.Predicate)

		switch predIRI {
		case RDFType:
			if nodeToString(quad.Object) == RDFStatement {
				hasStatementType = true
			}
		case RDFSubject:
			rdfSubject = quad.Object
		case RDFPredicate:
			rdfPredicate = quad.Object
		case RDFObject:
			rdfObject = quad.Object
		default:
			// Check for mangle:argN pattern
			if after, ok := strings.CutPrefix(predIRI, MangleNamespace+"arg"); ok {
				argNumStr := after
				if argNum, err := strconv.Atoi(argNumStr); err == nil {
					additionalArgs[argNum] = quad.Object
				}
			}
		}
	}

	if !hasStatementType || rdfSubject == nil || rdfPredicate == nil || rdfObject == nil {
		return ast.Atom{}, false
	}

	// Reconstruct the atom
	predicateIRI := nodeToString(rdfPredicate)
	predicateName := strings.TrimPrefix(predicateIRI, MangleNamespace)

	args := make([]ast.BaseTerm, 2+len(additionalArgs))
	var err error

	args[0], err = rdfNodeToConstant(rdfSubject)
	if err != nil {
		return ast.Atom{}, false
	}

	args[1], err = rdfNodeToConstant(rdfObject)
	if err != nil {
		return ast.Atom{}, false
	}

	// Add additional arguments in order
	for i := 2; i < len(args); i++ {
		node, ok := additionalArgs[i]
		if !ok {
			return ast.Atom{}, false // Missing argument
		}
		args[i], err = rdfNodeToConstant(node)
		if err != nil {
			return ast.Atom{}, false
		}
	}

	return ast.Atom{
		Predicate: ast.PredicateSym{Symbol: predicateName, Arity: len(args)},
		Args:      args,
	}, true
}

// tryDetectNaryPattern checks for W3C n-ary relation patterns.
// For now, returns false - can be extended to detect specific patterns.
func tryDetectNaryPattern(subject ld.Node, quads []*ld.Quad) (ast.Atom, bool) {
	// TODO: Implement W3C n-ary pattern detection
	// This would look for patterns like nodes with multiple properties
	// that should be grouped into a single n-ary predicate
	return ast.Atom{}, false
}

// rdfTypeToAtom converts an rdf:type triple to an arity 0 or 1 atom.
func rdfTypeToAtom(quad *ld.Quad) (ast.Atom, error) {
	typeIRI := nodeToString(quad.Object)
	typeName := strings.TrimPrefix(typeIRI, MangleNamespace)

	// Check if subject is a blank node (arity 0) or has identity (arity 1)
	if ld.IsBlankNode(quad.Subject) {
		// Arity 0: nullary predicate
		return ast.Atom{
			Predicate: ast.PredicateSym{Symbol: typeName, Arity: 0},
			Args:      []ast.BaseTerm{},
		}, nil
	}

	// Arity 1: unary predicate with subject
	subject, err := rdfNodeToConstant(quad.Subject)
	if err != nil {
		return ast.Atom{}, err
	}

	return ast.Atom{
		Predicate: ast.PredicateSym{Symbol: typeName, Arity: 1},
		Args:      []ast.BaseTerm{subject},
	}, nil
}

// simpleTripleToAtom converts a simple RDF triple to an arity 2 atom.
func simpleTripleToAtom(quad *ld.Quad) (ast.Atom, error) {
	predicateIRI := nodeToString(quad.Predicate)
	predicateName := strings.TrimPrefix(predicateIRI, MangleNamespace)

	subject, err := rdfNodeToConstant(quad.Subject)
	if err != nil {
		return ast.Atom{}, err
	}

	object, err := rdfNodeToConstant(quad.Object)
	if err != nil {
		return ast.Atom{}, err
	}

	return ast.Atom{
		Predicate: ast.PredicateSym{Symbol: predicateName, Arity: 2},
		Args:      []ast.BaseTerm{subject, object},
	}, nil
}

// rdfNodeToConstant converts an RDF node to an ast.Constant.
func rdfNodeToConstant(node ld.Node) (ast.Constant, error) {
	if ld.IsIRI(node) {
		iri := node.(ld.IRI)
		// IRIs are treated as Mangle names
		return ast.Name(iri.Value)
	}

	if ld.IsLiteral(node) {
		lit := node.(ld.Literal)

		switch lit.Datatype {
		case "http://www.w3.org/2001/XMLSchema#base64Binary":
			decoded, err := base64.StdEncoding.DecodeString(lit.Value)
			if err != nil {
				return ast.Constant{}, fmt.Errorf("failed to decode base64Binary: %w", err)
			}
			return ast.Bytes(decoded), nil

		case "http://www.w3.org/2001/XMLSchema#integer":
			num, err := strconv.ParseInt(lit.Value, 10, 64)
			if err != nil {
				return ast.Constant{}, fmt.Errorf("failed to parse integer: %w", err)
			}
			return ast.Number(num), nil

		case "http://www.w3.org/2001/XMLSchema#double":
			flt, err := strconv.ParseFloat(lit.Value, 64)
			if err != nil {
				return ast.Constant{}, fmt.Errorf("failed to parse float: %w", err)
			}
			return ast.Float64(flt), nil

		case "http://www.w3.org/2001/XMLSchema#string":
			return ast.String(lit.Value), nil

		default:
			// Default to string
			return ast.String(lit.Value), nil
		}
	}

	if ld.IsBlankNode(node) {
		bn := node.(ld.BlankNode)
		// Blank nodes as names
		name, err := ast.Name("_:" + bn.Attribute)
		if err != nil {
			return ast.Constant{}, fmt.Errorf("failed to create blank node name: %w", err)
		}
		return name, nil
	}

	return ast.Constant{}, fmt.Errorf("unknown RDF node type")
}

// nodeToString converts an RDF node to its string representation.
func nodeToString(node ld.Node) string {
	if node == nil {
		return ""
	}

	if ld.IsIRI(node) {
		return node.(ld.IRI).Value
	}

	if ld.IsLiteral(node) {
		return node.(ld.Literal).Value
	}

	if ld.IsBlankNode(node) {
		return "_:" + node.(ld.BlankNode).Attribute
	}

	return ""
}
