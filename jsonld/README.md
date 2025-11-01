# `jsonld` Package

This package provides a high-performance, streaming-based converter between Google Mangle `ast.Atom` structures and the JSON-LD format. It is designed for efficient, lossless serialization and deserialization, enabling Mangle facts to be used in Linked Data and Semantic Web ecosystems.

The implementation uses wrapper types (`AtomJSONLD`, `AtomsJSONLD`, `ConstantJSONLD`) that implement `json.MarshalerTo` and `json.UnmarshalerFrom` from the `go-json-experiment/json` library for optimal performance.

## Features

*   **High-Performance Streaming**: Leverages `go-json-experiment/json` for low-allocation, high-speed JSON processing.
*   **Idiomatic Go Wrappers**: Simple `AtomJSONLD{atom}` wrappers integrate directly with `json.Marshal` and `json.Unmarshal`.
*   **Arity-Based Mapping**: Uses a clear and predictable set of rules to map Mangle atoms to intuitive JSON-LD structures.
*   **Full Type Fidelity**: Ensures all Mangle `ast.Constant` types, including complex structures like lists, maps, and pairs, are serialized and deserialized without data loss.
*   **Standard JSON-LD Output**: Produces JSON-LD that is compatible with standard tools, including a default `@context`.

## Mapping Rules

The conversion logic follows a hybrid, arity-based approach to produce the most idiomatic JSON-LD for each type of Mangle atom.

### Arity 0 (Nullary Predicates)

An atom with no arguments, like `rainy()`, is treated as a simple type declaration.

**Mangle**: `rainy()`
**JSON-LD**:
```json
{
  "@context": { ... },
  "@type": "rainy"
}
```

### Arity 1 (Unary Predicates / Class Membership)

An atom with one argument, like `person(/alice)`, is treated as an entity with a specific type. This is the standard way to represent class membership in RDF/JSON-LD.

**Mangle**: `person(/alice)`
**JSON-LD**:
```json
{
  "@context": { ... },
  "@id": "/alice",
  "@type": "person"
}
```

### Arity 2 (Binary Predicates / RDF Triples)

An atom with two arguments, like `parent(/alice, /bob)`, maps directly to the standard RDF subject-predicate-object model.

**Mangle**: `parent(/alice, /bob)`
**JSON-LD**:
```json
{
  "@context": { ... },
  "@id": "/alice",
  "parent": "/bob"
}
```

### Arity 3+ (N-ary Relations)

An atom with three or more arguments, like `located_at(/eiffel, /paris, "France")`, is represented as an object with the predicate as its `@type` and arguments mapped to indexed properties (`arg0`, `arg1`, etc.).

**Mangle**: `located_at(/eiffel, /paris, "France")`
**JSON-LD**:
```json
{
  "@context": { ... },
  "@type": "located_at",
  "arg0": "/eiffel",
  "arg1": "/paris",
  "arg2": "France"
}
```

## Mangle Constant Serialization

The `ConstantJSONLD` wrapper ensures that all Mangle `ast.Constant` types are correctly serialized and can be deserialized without losing type information.

*   **Primitive Types**: `Name`, `String`, `Number`, and `Float64` are mapped to their corresponding JSON string and number types.
*   **Bytes**: Serialized as a string with a `b"..."` wrapper (e.g., `b"\x01\x02"`).
*   **Lists**: Serialized using the standard JSON-LD `@list` keyword: `{"@list": [1, 2, 3]}`.
*   **Complex Types**: `Pair`, `Map`, and `Struct` types are serialized using a namespaced `fn:` prefix to avoid conflicts and ensure they can be correctly identified during deserialization.
    *   `fn:pair`: `{"fn:pair": [fst, snd]}`
    *   `fn:map`: `{"fn:map": [key1, val1, key2, val2, ...]}`
    *   `fn:struct`: `{"fn:struct": [label1, val1, label2, val2, ...]}`

## Usage

### Single Atom to JSON-LD

Wrap an `ast.Atom` in `AtomJSONLD` and use `json.Marshal`.

```go
import (
	"fmt"
	"github.com/go-json-experiment/json"
	"github.com/google/mangle/ast"
	"your/project/jsonld"
)

func main() {
	alice, _ := ast.Name("/alice")
	bob, _ := ast.Name("/bob")
	atom := ast.Atom{
		Predicate: ast.PredicateSym{Symbol: "parent", Arity: 2},
		Args:      []ast.BaseTerm{alice, bob},
	}

	wrapper := jsonld.AtomJSONLD{Atom: atom}
	bytes, err := json.Marshal(wrapper)
	if err != nil {
		// handle error
	}
	fmt.Println(string(bytes))
	// Output: {"@context":{...},"@id":"/alice","parent":"/bob"}
}
```

### JSON-LD to Single Atom

Unmarshal JSON-LD data into an `AtomJSONLD` wrapper.

```go
jsonStr := `{"@context": {}, "@id": "/alice", "parent": "/bob"}`

var wrapper jsonld.AtomJSONLD
if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
	// handle error
}

atom := wrapper.Atom
fmt.Println(atom) // Output: parent(/alice, /bob)
```

### Multiple Atoms to JSON-LD (`@graph`)

To serialize a slice of atoms, use the `AtomsJSONLD` wrapper. This will produce a single JSON-LD document containing a `@graph` array.

```go
atoms := []ast.Atom{ atom1, atom2 }
wrapper := jsonld.AtomsJSONLD{Atoms: atoms}
bytes, err := json.Marshal(wrapper)
// ...
```

### JSON-LD (`@graph`) to Multiple Atoms

Unmarshal a document containing a `@graph` into an `AtomsJSONLD` wrapper.

```go
jsonStr := `{
	"@context": {},
	"@graph": [
		{"@id": "/alice", "@type": "person"},
		{"@id": "/bob", "@type": "person"}
	]
}`

var wrapper jsonld.AtomsJSONLD
if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
	// handle error
}

fmt.Println(len(wrapper.Atoms)) // Output: 2
```

## RDF Converter

The `rdf_converter.go` module provides bidirectional conversion between Mangle Datalog atoms and RDF triples, enabling full semantic web interoperability.

### Key Features

*   **RDF Triple Conversion**: Convert Datalog predicates to RDF quads/triples and vice versa
*   **Reification Support**: N-ary relations (arity 3+) use standard RDF reification patterns
*   **W3C N-ary Pattern Detection**: Recognizes and converts W3C n-ary relation patterns
*   **JSON-LD Integration**: Works with `piprate/json-gold` for JSON-LD ↔ RDF conversion

### RDF Mapping Strategy

The converter uses an arity-based approach to map between Datalog and RDF:

**Datalog → RDF (AtomsToRDF)**:
- **Arity 0** (`rainy()`): `_:b1 rdf:type rainy`
- **Arity 1** (`person(/alice)`): `/alice rdf:type person`
- **Arity 2** (`parent(/alice, /bob)`): `/alice parent /bob`
- **Arity 3+** (`located_at(/eiffel, /paris, "France")`): Reification pattern:
  ```
  _:stmt rdf:type rdf:Statement
  _:stmt rdf:subject /eiffel
  _:stmt rdf:predicate located_at
  _:stmt rdf:object /paris
  _:stmt mangle:arg2 "France"
  ```

**RDF → Datalog (RDFToAtoms)**:
- Detects `rdf:type` triples → arity 0 or 1 atoms
- Simple subject-predicate-object → arity 2 atoms
- Reification patterns → arity 3+ atoms
- W3C n-ary patterns → reconstructs original arity

### Usage with JSON-LD

#### Receiving RDF Data from External Systems

```go
import (
	"github.com/go-json-experiment/json"
	"github.com/piprate/json-gold/ld"
	"your/project/jsonld"
)

// Step 1: Parse JSON-LD to RDF using json-gold
jsonLdInput := `{"@id": "/alice", "@type": "person"}`

proc := ld.NewJsonLdProcessor()
opts := ld.NewJsonLdOptions("")

rdfRaw, err := proc.ToRDF(jsonLdInput, opts)
if err != nil {
	// handle error
}

dataset := rdfRaw.(*ld.RDFDataset)

// Step 2: Convert RDF to Datalog atoms
atoms, err := jsonld.RDFToAtoms(dataset, "@default")
if err != nil {
	// handle error
}

// Now you have Datalog atoms ready for reasoning!
// atoms[0] == person(/alice)
```

#### Sending RDF Data to External Systems

```go
// Step 1: Convert Datalog atoms to RDF
alice, _ := ast.Name("/alice")
atom := ast.Atom{
	Predicate: ast.PredicateSym{Symbol: "person", Arity: 1},
	Args:      []ast.BaseTerm{alice},
}

dataset, err := jsonld.AtomsToRDF([]ast.Atom{atom})
if err != nil {
	// handle error
}

// Step 2: Convert RDF to JSON-LD using json-gold
proc := ld.NewJsonLdProcessor()
opts := ld.NewJsonLdOptions("")
opts.UseNativeTypes = true

jsonLdDoc, err := proc.FromRDF(dataset, opts)
if err != nil {
	// handle error
}

// Step 3: Serialize to JSON
bytes, err := json.Marshal(jsonLdDoc)
// Send bytes to external RDF system
```

### Direct RDF Conversion (No JSON-LD)

You can also work directly with RDF without JSON-LD:

```go
// Convert atoms to RDF
atoms := []ast.Atom{/* your atoms */}
dataset, err := jsonld.AtomsToRDF(atoms)

// Convert RDF back to atoms
recoveredAtoms, err := jsonld.RDFToAtoms(dataset, "@default")

// Round-trip preserves structure
assert.Equal(t, atoms, recoveredAtoms)
```

### Choosing Between Direct and RDF Converters

**Use the Direct Converter** (`AtomJSONLD`, `AtomsJSONLD`) when:
- You want simple, performant JSON-LD serialization
- You're working within the Datalog ecosystem
- You don't need full RDF compatibility
- You want predictable, arity-based JSON-LD structure

**Use the RDF Converter** (`AtomsToRDF`, `RDFToAtoms`) when:
- You need to interop with external RDF systems
- You're receiving RDF data from semantic web sources
- You need standard RDF reification for n-ary relations
- You want to leverage RDF tooling (SPARQL, triple stores, etc.)

Both approaches produce valid JSON-LD, but the RDF converter ensures full semantic web compatibility at the cost of additional complexity.