package factstoredb

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/go-json-experiment/json/jsontext"

	"github.com/google/mangle/ast"
	"github.com/google/mangle/factstore"
)

// Counter for generating unique in-memory database names
var inMemoryDBCounter atomic.Uint64

// DBFactStore implements the Mangle FactStore interface using SQLite
// as the backing storage. It uses a single table schema with predicate and
// args columns, where args is stored as JSON for efficient querying.
type FactStoreDB struct {
	db *sql.DB
	// dialect handles SQL syntax differences between databases.
	dialect dialect
	// Prepared statements for performance
	addStmt      *sql.Stmt
	removeStmt   *sql.Stmt
	containsStmt *sql.Stmt
}

// Verify that DBFactStore implements the FactStoreWithRemove interface
var _ factstore.FactStoreWithRemove = (*FactStoreDB)(nil)

// FactStore Interface Methods

// Add adds a fact to the store and returns true if it didn't exist before.
func (s *FactStoreDB) Add(atom ast.Atom) bool {
	// // FactStore only stores grounded atoms (no variables)
	// if !atom.IsGround() {
	// 	return false
	// }
	// Convert atom to row format
	// also verifies all args are constants
	predicate := predicateToKey(atom.Predicate)
	atomHash, args, err := atomToRowForInsert(atom)
	if err != nil {
		// Cannot store atoms with non-constant args
		return false
	}

	// Execute INSERT ON CONFLICT DO NOTHING - concurrent safe and atomic
	// The UNIQUE(atom_hash) constraint handles deduplication
	res, err := s.addStmt.Exec(predicate, atomHash, args)
	if err != nil {
		log.Printf("DBFactStore failed to execute add statement: %v", err)
		return false
	}

	// Check if row was actually inserted (rowsAffected=0 means already existed)
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false
	}

	return rowsAffected > 0
}

// Contains returns true if given atom is already present in store.
func (s *FactStoreDB) Contains(atom ast.Atom) bool {
	// Convert atom to canonical form for lookup
	atomHash, _, err := atomToRowForInsert(atom)
	if err != nil {
		log.Printf("DBFactStore failed to process atom for Contains: %v", err)
		return false
	}

	// Use prepared statement for fast lookup by atom_hash
	var count int
	err = s.containsStmt.QueryRow(atomHash).Scan(&count)
	if err != nil {
		log.Printf("DBFactStore failed to execute contains statement: %v", err)
		return false
	}

	return count > 0
}

// Remove removes a fact from the store and returns true if that fact was present.
func (s *FactStoreDB) Remove(atom ast.Atom) bool {
	// Convert atom to canonical form for removal
	atomHash, _, err := atomToRowForInsert(atom)
	if err != nil {
		log.Printf("DBFactStore failed to process atom for Remove: %v", err)
		return false
	}

	// Execute the prepared statement - simple atom_hash match
	result, err := s.removeStmt.Exec(atomHash)
	if err != nil {
		log.Printf("DBFactStore failed to execute remove statement: %v", err)
		return false
	}

	// Check if any rows were deleted
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("DBFactStore failed to get rows affected after remove: %v", err)
		return false
	}

	return rowsAffected > 0
}

// GetFacts returns a stream of facts that match a given atom.
// The atom may contain variables (wildcards) for pattern matching.
func (s *FactStoreDB) GetFacts(pattern ast.Atom, callback func(ast.Atom) error) error {
	// Build SQL query based on pattern using strings.Builder
	var queryBuf strings.Builder
	var params []any

	// Get the dialect-specific base query.
	queryBuf.WriteString(s.dialect.getFactsBaseSQL())

	// Filter by predicate key in "symbol_arity" format (e.g., "person_1")
	// This is much faster than LIKE pattern matching
	predicateKey := predicateToKey(pattern.Predicate)
	params = append(params, predicateKey)

	// For each argument, if it's a constant, add a filter using json_extract
	// json_extract works efficiently on JSONB binary format without parsing overhead
	for i, arg := range pattern.Args {
		// Check if arg is a constant (grounded) vs variable
		if constant, ok := arg.(ast.Constant); ok {
			// Serialize the constant to JSON in the same format we use for storage
			var buf strings.Builder
			enc := jsontext.NewEncoder(&buf)
			if err := (constantJSON{constant}).MarshalJSONTo(enc); err != nil {
				return fmt.Errorf("failed to marshal pattern arg: %w", err)
			}

			// Trim trailing newline that jsontext.Encoder adds
			jsonStr := strings.TrimSuffix(buf.String(), "\n")

			queryBuf.WriteString(s.dialect.getFactsFragment(i, &params))
			params = append(params, s.dialect.jsonParam(jsonStr))
		}
		// If it's a variable, don't filter (wildcard)
	}

	query := queryBuf.String()
	rows, err := s.db.Query(query, params...)
	if err != nil {
		return fmt.Errorf("failed to query facts: %w", err)
	}
	defer rows.Close()

	// Process each row and call callback
	for rows.Next() {
		var predicateStr string
		var argsJSON string

		if err := rows.Scan(&predicateStr, &argsJSON); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Unmarshal directly to ast.Atom in a single efficient operation
		// This skips parse.BaseTerm and functional.EvalAtom for better performance
		// and avoids intermediate allocations
		reconstructedAtom, err := unmarshalAtom(pattern.Predicate, argsJSON)
		if err != nil {
			return fmt.Errorf("failed to unmarshal atom: %w", err)
		}

		// Call callback with canonical atom (already in canonical form)
		if err := callback(reconstructedAtom); err != nil {
			return err
		}
	}

	return rows.Err()
}

// ListPredicates lists predicates available in this store.
func (s *FactStoreDB) ListPredicates() []ast.PredicateSym {
	rows, err := s.db.Query(`SELECT DISTINCT predicate FROM facts`)
	if err != nil {
		log.Printf("DBFactStore failed to query for predicates: %v", err)
		return nil
	}
	defer rows.Close()

	var predicates []ast.PredicateSym
	for rows.Next() {
		var predicateKey string
		if err := rows.Scan(&predicateKey); err != nil {
			log.Printf("DBFactStore failed to scan predicate row: %v", err)
			continue
		}

		// Parse "symbol_arity" format (e.g., "person_2" or "my_predicate_3")
		pred, err := keyToPredicate(predicateKey)
		if err != nil {
			log.Printf("DBFactStore %v", err)
			continue
		}

		predicates = append(predicates, pred)
	}
	if err := rows.Err(); err != nil {
		log.Printf("DBFactStore error iterating predicate rows: %v", err)
	}

	return predicates
}

// EstimateFactCount returns the estimated number of facts in the store.
func (s *FactStoreDB) EstimateFactCount() int {
	// Using a simple query string is more efficient than the builder for this case.
	const query = "SELECT COUNT(*) FROM facts"
	var count int
	// QueryRow is safe here as there are no parameters.
	if err := s.db.QueryRow(query).Scan(&count); err != nil {
		log.Printf("DBFactStore failed to estimate fact count: %v", err)
		return 0
	}

	return count
}

// Merge merges contents of given store into this store.
// Uses optimized bulk insertion with multi-row INSERTs for large merges.
func (s *FactStoreDB) Merge(other factstore.ReadOnlyFactStore) {
	// Step 1: Collect all facts from the other store
	var facts []ast.Atom
	predicates := other.ListPredicates()

	for _, predicate := range predicates {
		// Use ast.NewQuery to create a query atom with all variables
		queryAtom := ast.NewQuery(predicate)

		// Collect all facts matching this predicate
		_ = other.GetFacts(queryAtom, func(atom ast.Atom) error {
			facts = append(facts, atom)
			return nil
		})
	}

	// Nothing to merge
	if len(facts) == 0 {
		return
	}

	// Use optimized batch insert for better performance
	if err := s.batchInsertFacts(facts); err != nil {
		log.Printf("DBFactStore failed to batch insert facts: %v", err)
	}
}

// batchInsertFacts inserts a slice of facts using optimized multi-row INSERT statements.
// This is significantly faster than individual INSERTs, especially for large batches.
func (s *FactStoreDB) batchInsertFacts(facts []ast.Atom) error {
	const batchSize = 500 // Optimal batch size balancing SQL parsing vs transaction size

	// Pre-compute all rows outside transaction to minimize lock time
	type row struct {
		predicate string
		atomHash  int64
		args      string
	}
	rows := make([]row, 0, len(facts))
	for _, fact := range facts {
		predicate := predicateToKey(fact.Predicate)
		atomHash, args, err := atomToRowForInsert(fact)
		if err != nil {
			// Skip non-grounded atoms (shouldn't happen in a proper FactStore)
			continue
		}
		rows = append(rows, row{predicate, atomHash, args})
	}

	if len(rows) == 0 {
		return nil
	}

	// Begin transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback is a no-op if Commit succeeds

	// Process rows in batches using multi-row INSERT
	for i := 0; i < len(rows); i += batchSize {
		end := min(i+batchSize, len(rows))
		batch := rows[i:end]

		// Generate the dialect-specific multi-row INSERT statement.
		sql := s.dialect.batchInsertSQL(len(batch))

		// Pre-allocate params slice
		params := make([]any, 0, len(batch)*3)
		for _, r := range batch {
			params = append(params, r.predicate, r.atomHash, r.args)
		}

		// Execute batch insert
		if _, err := tx.Exec(sql, params...); err != nil {
			return fmt.Errorf("failed to execute batch insert: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WriteTo writes all facts from the store to w in JSON format.
// It implements the io.WriterTo interface.
// Facts are streamed directly to the writer without intermediate buffering,
// making this efficient for large fact stores.
// Returns the number of bytes written and any error encountered.
func (s *FactStoreDB) WriteTo(w io.Writer) (int64, error) {
	// Wrap writer to count bytes
	cw := &countingWriter{w: w}

	// Create JSON encoder
	enc := jsontext.NewEncoder(cw)

	// Write opening array: [
	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		return cw.count, err
	}

	// Get and sort predicates for deterministic output
	predicates := s.ListPredicates()
	sort.Slice(predicates, func(i, j int) bool {
		a := predicates[i]
		b := predicates[j]
		return a.Symbol < b.Symbol || (a.Symbol == b.Symbol && a.Arity < b.Arity)
	})

	// Stream facts directly using atomJSON
	for _, pred := range predicates {
		if err := s.GetFacts(ast.NewQuery(pred), func(atom ast.Atom) error {
			// Marshal entire atom using atomJSON
			if err := (atomJSON{atom}).MarshalJSONTo(enc); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return cw.count, fmt.Errorf("failed to get facts for %v: %w", pred, err)
		}
	}

	// Write closing array: ]
	if err := enc.WriteToken(jsontext.EndArray); err != nil {
		return cw.count, err
	}

	return cw.count, nil
}

// ReadFrom reads facts from a JSON stream (r) and bulk-inserts them into the store.
// It implements the io.ReaderFrom interface.
// The expected format is a JSON array of atom objects, the same format produced by WriteTo.
// This method is memory-efficient as it streams and processes atoms in batches.
// Returns the number of bytes read and any error encountered.
func (s *FactStoreDB) ReadFrom(r io.Reader) (int64, error) {
	// Wrap reader to count bytes read
	cr := &countingReader{r: r}
	dec := jsontext.NewDecoder(cr)

	// Expect the start of a JSON array: [
	tok, err := dec.ReadToken()
	if err != nil {
		return cr.count, fmt.Errorf("failed to read opening token: %w", err)
	}
	if tok.Kind() != '[' {
		return cr.count, fmt.Errorf("expected JSON array start '[', got %c", tok.Kind())
	}

	const batchSize = 500 // Match batchInsertFacts batch size
	var atomBatch []ast.Atom

	// Loop through the array, decoding one atom at a time
	for dec.PeekKind() != ']' {
		// Unmarshal one atom object using our efficient streaming unmarshaller.
		// This avoids intermediate allocations and re-parsing.
		var aj atomJSON
		if err := aj.UnmarshalJSONFrom(dec); err != nil {
			return cr.count, fmt.Errorf("failed to unmarshal atom from stream: %w", err)
		}

		atomBatch = append(atomBatch, aj.Atom)

		// When batch is full, insert it and clear the slice
		if len(atomBatch) >= batchSize {
			if err := s.batchInsertFacts(atomBatch); err != nil {
				return cr.count, fmt.Errorf("failed to insert batch: %w", err)
			}
			atomBatch = atomBatch[:0] // Reset slice while keeping capacity
		}
	}

	// Insert any remaining facts in the last batch
	if len(atomBatch) > 0 {
		if err := s.batchInsertFacts(atomBatch); err != nil {
			return cr.count, fmt.Errorf("failed to insert final batch: %w", err)
		}
	}

	// Expect the end of the JSON array: ]
	tok, err = dec.ReadToken()
	if err != nil {
		return cr.count, fmt.Errorf("failed to read closing token: %w", err)
	}
	if tok.Kind() != ']' {
		return cr.count, fmt.Errorf("expected JSON array end ']', got %c", tok.Kind())
	}

	return cr.count, nil
}

// countingWriter wraps an io.Writer and counts bytes written.
type countingWriter struct {
	w     io.Writer
	count int64
}

func (cw *countingWriter) Write(p []byte) (n int, err error) {
	n, err = cw.w.Write(p)
	cw.count += int64(n)
	return n, err
}

// countingReader wraps an io.Reader and counts bytes read.
type countingReader struct {
	r     io.Reader
	count int64
}

func (cr *countingReader) Read(p []byte) (n int, err error) {
	n, err = cr.r.Read(p)
	cr.count += int64(n)
	return n, err
}

// Close closes the database connection.
func (s *FactStoreDB) Close() error {
	if s.addStmt != nil {
		s.addStmt.Close()
	}
	if s.removeStmt != nil {
		s.removeStmt.Close()
	}
	if s.containsStmt != nil {
		s.containsStmt.Close()
	}
	return s.db.Close()
}

// Helper Functions

// predicateToKey converts a PredicateSym to the database key format "symbol_arity".
// For example: PredicateSym{Symbol: "person", Arity: 2} -> "person_2"
func predicateToKey(p ast.PredicateSym) string {
	return p.Symbol + "_" + strconv.Itoa(p.Arity)
}

// keyToPredicate parses a database key in "symbol_arity" format back to PredicateSym.
// For example: "person_2" -> PredicateSym{Symbol: "person", Arity: 2}
func keyToPredicate(key string) (ast.PredicateSym, error) {
	lastUnderscore := strings.LastIndex(key, "_")
	if lastUnderscore == -1 {
		return ast.PredicateSym{}, fmt.Errorf("invalid predicate key format: %q", key)
	}
	symbol := key[:lastUnderscore]
	arity, err := strconv.Atoi(key[lastUnderscore+1:])
	if err != nil {
		return ast.PredicateSym{}, fmt.Errorf("invalid arity in predicate key %q: %w", key, err)
	}
	return ast.PredicateSym{Symbol: symbol, Arity: arity}, nil
}

// szudzikElegantPair implements Szudzik's elegant pairing function.
// See http://szudzik.com/ElegantPairing.pdf
func szudzikElegantPair(fst, snd uint64) uint64 {
	if fst >= snd {
		return fst*fst + fst + snd
	}
	return snd*snd + fst
}

// pair represents a key-value pair for sorting map/struct entries.
type pair struct {
	key ast.Constant
	val ast.Constant
}

// sortablePairs allows sorting of map/struct entries by key hash.
type sortablePairs []pair

func (p sortablePairs) Len() int           { return len(p) }
func (p sortablePairs) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p sortablePairs) Less(i, j int) bool { return p[i].key.Hash() < p[j].key.Hash() }

// getSortedConstants extracts key-value pairs from a map or struct,
// sorts them by key, and returns a flat slice of [key1, val1, key2, val2, ...].
func getSortedConstants(c ast.Constant) ([]ast.Constant, error) {
	var pairs sortablePairs
	var err error

	if c.Type == ast.MapShape {
		_, err = c.MapValues(func(k, v ast.Constant) error {
			pairs = append(pairs, pair{k, v})
			return nil
		}, func() error { return nil })
	} else { // ast.StructShape
		_, err = c.StructValues(func(k, v ast.Constant) error {
			pairs = append(pairs, pair{k, v})
			return nil
		}, func() error { return nil })
	}
	if err != nil {
		return nil, err
	}
	sort.Sort(pairs)
	return flattenPairs(pairs), nil
}

// atomToRowForInsert converts an ast.Atom to atom_hash and args for insertion.
// Assumes all args are already ast.Constant (not variables or other BaseTerms).
// Returns an error if any arg is not a constant.
// Callers can compute the predicate key using predicateToKey(atom.Predicate).
func atomToRowForInsert(atom ast.Atom) (int64, string, error) {

	// Compute hash starting with predicate
	h := fnv.New64a()
	h.Write([]byte(atom.Predicate.Symbol))
	predicateHash := h.Sum64()
	hashResult := szudzikElegantPair(predicateHash, uint64(atom.Predicate.Arity))

	// Marshal constants to JSON while also computing the hash in a single pass.
	var buf strings.Builder
	enc := jsontext.NewEncoder(&buf)
	if err := enc.WriteToken(jsontext.BeginArray); err != nil {
		return 0, "", fmt.Errorf("failed to write array start for JSON args: %w", err)
	}

	for _, arg := range atom.Args {
		c, ok := arg.(ast.Constant)
		if !ok {
			return 0, "", fmt.Errorf("evaluation produced something that is not a value: %v %T", arg, arg)
		}
		if err := (constantJSON{c}).MarshalJSONTo(enc); err != nil {
			return 0, "", fmt.Errorf("failed to marshal arg to JSON: %w", err)
		}
		// For maps and structs, we need an order-insensitive hash.
		// We get the key-value pairs, sort them by key, and then hash them in order.
		if c.Type == ast.MapShape || c.Type == ast.StructShape {
			sorted, err := getSortedConstants(c)
			if err != nil {
				return 0, "", fmt.Errorf("failed to sort map/struct for hashing: %w", err)
			}
			for _, part := range sorted {
				partHash := szudzikElegantPair(part.Hash(), uint64(part.Type))
				hashResult = szudzikElegantPair(hashResult, partHash)
			}
		} else {
			// For other types, hash directly.
			argHashWithType := szudzikElegantPair(c.Hash(), uint64(c.Type))
			hashResult = szudzikElegantPair(hashResult, argHashWithType)
		}
	}

	if err := enc.WriteToken(jsontext.EndArray); err != nil {
		return 0, "", fmt.Errorf("failed to write array end for JSON args: %w", err)
	}

	// Cast to int64 for database/sql compatibility - BIGINT will interpret the bit pattern correctly
	atomHash := int64(hashResult)

	return atomHash, buf.String(), nil
}

// flattenPairs converts a slice of sorted pairs into a flat slice of constants.
func flattenPairs(pairs []pair) []ast.Constant {
	flat := make([]ast.Constant, 0, len(pairs)*2)
	for _, p := range pairs {
		flat = append(flat, p.key, p.val)
	}
	return flat
}
