package jsonld

import (
	"fmt"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/google/mangle/ast"
	"github.com/piprate/json-gold/ld"
)

// AtomRDFJSONLD is a wrapper around ast.Atom that implements json.MarshalerTo
// and json.UnmarshalerFrom using RDF as an intermediate representation.
// This provides full RDF compatibility for semantic web interoperability.
type AtomRDFJSONLD struct {
	ast.Atom
}

// AtomsRDFJSONLD is a wrapper around []ast.Atom that implements json.MarshalerTo
// and json.UnmarshalerFrom using RDF as an intermediate representation.
type AtomsRDFJSONLD struct {
	Atoms []ast.Atom
}

// MarshalJSONTo implements json.MarshalerTo for AtomRDFJSONLD.
// Converts the atom to RDF, then serializes as JSON-LD using json-gold.
func (a AtomRDFJSONLD) MarshalJSONTo(enc *jsontext.Encoder) error {
	// Convert atom to RDF dataset
	dataset, err := AtomsToRDF([]ast.Atom{a.Atom})
	if err != nil {
		return fmt.Errorf("failed to convert atom to RDF: %w", err)
	}

	// Convert RDF dataset to JSON-LD using json-gold
	proc := ld.NewJsonLdProcessor()
	opts := ld.NewJsonLdOptions("")
	opts.UseNativeTypes = true

	// FromRDF expects the dataset directly, not serialized
	jsonLdDocs, err := proc.FromRDF(dataset, opts)
	if err != nil {
		return fmt.Errorf("failed to convert RDF to JSON-LD: %w", err)
	}

	// FromRDF returns a slice of documents. For a single atom, we expect one document.
	docs, ok := jsonLdDocs.([]interface{})
	if !ok || len(docs) == 0 {
		return fmt.Errorf("expected a single JSON-LD document, but got none or wrong type")
	}
	if len(docs) > 1 {
		// This case might be valid if the atom expands to multiple top-level nodes.
		// For now, we assume a single atom maps to a single top-level node.
	}

	// This requires a temporary buffer as json.Marshal is used.
	// A more advanced approach would be to write a custom marshaler
	// for the json-gold document structure.
	return json.MarshalEncode(enc, docs[0])
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom for AtomRDFJSONLD.
// Parses JSON-LD, converts to RDF, then converts to ast.Atom.
func (a *AtomRDFJSONLD) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	// Read the JSON value
	bytes, err := dec.ReadValue()
	if err != nil {
		return fmt.Errorf("failed to read JSON value: %w", err)
	}

	// The json-gold processor expects a map[string]interface{} for a single object.
	var jsonLdDoc map[string]interface{}
	if err := json.Unmarshal(bytes, &jsonLdDoc); err != nil {
		return fmt.Errorf("failed to unmarshal JSON-LD: %w", err)
	}

	// Convert JSON-LD to RDF using json-gold
	proc := ld.NewJsonLdProcessor()
	opts := ld.NewJsonLdOptions("")

	rdfDatasetRaw, err := proc.ToRDF(jsonLdDoc, opts)
	if err != nil {
		return fmt.Errorf("failed to convert JSON-LD to RDF: %w", err)
	}

	// Type assert to *ld.RDFDataset
	dataset, ok := rdfDatasetRaw.(*ld.RDFDataset)
	if !ok {
		return fmt.Errorf("unexpected RDF dataset type: %T", rdfDatasetRaw)
	}

	// Convert RDF to atoms
	atoms, err := RDFToAtoms(dataset, "@default")
	if err != nil {
		return fmt.Errorf("failed to convert RDF to atoms: %w", err)
	}

	if len(atoms) == 0 {
		return fmt.Errorf("no atoms found in JSON-LD document")
	}

	if len(atoms) > 1 {
		return fmt.Errorf("expected single atom, got %d atoms", len(atoms))
	}

	a.Atom = atoms[0]
	return nil
}

// MarshalJSONTo implements json.MarshalerTo for AtomsRDFJSONLD.
// Converts the atoms to RDF, then serializes as JSON-LD with @graph.
func (aa AtomsRDFJSONLD) MarshalJSONTo(enc *jsontext.Encoder) error {
	// Convert atoms to RDF dataset
	dataset, err := AtomsToRDF(aa.Atoms)
	if err != nil {
		return fmt.Errorf("failed to convert atoms to RDF: %w", err)
	}

	// Convert RDF dataset to JSON-LD using json-gold
	proc := ld.NewJsonLdProcessor()
	opts := ld.NewJsonLdOptions("")
	opts.UseNativeTypes = true

	// FromRDF expects the dataset directly, not serialized
	jsonLdDoc, err := proc.FromRDF(dataset, opts)
	if err != nil {
		return fmt.Errorf("failed to convert RDF to JSON-LD: %w", err)
	}

	return json.MarshalEncode(enc, jsonLdDoc) // For collections, the slice is the correct format
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom for AtomsRDFJSONLD.
// Parses JSON-LD with @graph, converts to RDF, then converts to []ast.Atom.
func (aa *AtomsRDFJSONLD) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	// Read the JSON value
	var jsonLdDoc interface{}
	bytes, err := dec.ReadValue()
	if err != nil {
		return fmt.Errorf("failed to read JSON value: %w", err)
	}

	// The processor can handle a map (for @graph) or a slice directly.
	if err := json.Unmarshal(bytes, &jsonLdDoc); err != nil {
		return fmt.Errorf("failed to unmarshal JSON-LD: %w", err)
	}

	// Convert JSON-LD to RDF using json-gold
	proc := ld.NewJsonLdProcessor()
	opts := ld.NewJsonLdOptions("")

	rdfDatasetRaw, err := proc.ToRDF(jsonLdDoc, opts)
	if err != nil {
		return fmt.Errorf("failed to convert JSON-LD to RDF: %w", err)
	}

	// Type assert to *ld.RDFDataset
	dataset, ok := rdfDatasetRaw.(*ld.RDFDataset)
	if !ok {
		return fmt.Errorf("unexpected RDF dataset type: %T", rdfDatasetRaw)
	}

	// Convert RDF to atoms
	atoms, err := RDFToAtoms(dataset, "@default")
	if err != nil {
		return fmt.Errorf("failed to convert RDF to atoms: %w", err)
	}

	aa.Atoms = atoms
	return nil
}
