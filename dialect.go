package factstoredb

import (
	"strconv"
	"strings"
)

// dialect defines an interface for generating database-specific SQL.
type dialect interface {
	// createTableSQL returns the SQL for creating the 'facts' table.
	createTableSQL() string
	// createIndexSQL returns the SQL for creating the index on the 'predicate' column.
	createIndexSQL() string
	// addSQL returns the SQL for inserting a fact with conflict handling.
	addSQL() string
	// removeSQL returns the SQL for deleting a fact by its hash.
	removeSQL() string
	// containsSQL returns the SQL for checking if a fact exists by its hash.
	containsSQL() string
	// getFactsBaseSQL returns the initial SELECT statement for GetFacts.
	getFactsBaseSQL() string
	// batchInsertSQL builds a multi-row INSERT statement for a given number of rows.
	batchInsertSQL(numRows int) string
	// getFactsFragment appends the SQL fragment for filtering by a constant argument in GetFacts.
	getFactsFragment(index int, params *[]any) string
	// jsonParam prepares a parameter for a JSON comparison.
	jsonParam(jsonStr string) any
}

// --- SQLite Dialect ---

type sqliteDialect struct{}

func (d sqliteDialect) createTableSQL() string {
	return `
		CREATE TABLE IF NOT EXISTS facts (
			predicate TEXT NOT NULL,
			atom_hash BIGINT NOT NULL,
			args BLOB NOT NULL,
			PRIMARY KEY(atom_hash)
		) WITHOUT ROWID;
	`
}

func (d sqliteDialect) createIndexSQL() string {
	return `CREATE INDEX IF NOT EXISTS idx_predicate ON facts(predicate);`
}

func (d sqliteDialect) addSQL() string {
	return `
		INSERT INTO facts (predicate, atom_hash, args)
		VALUES (?, ?, ?)
		ON CONFLICT DO NOTHING
	`
}

func (d sqliteDialect) removeSQL() string {
	return `DELETE FROM facts WHERE atom_hash = ?`
}

func (d sqliteDialect) containsSQL() string {
	return `SELECT COUNT(*) FROM facts WHERE atom_hash = ?`
}

func (d sqliteDialect) getFactsBaseSQL() string {
	return `SELECT predicate, json(args) FROM facts WHERE predicate = ?`
}

func (d sqliteDialect) batchInsertSQL(numRows int) string {
	var sb strings.Builder
	sb.WriteString("INSERT INTO facts (predicate, atom_hash, args) VALUES ")
	for i := range numRows {
		if i > 0 {
			sb.WriteString(",")
		}
		// Each row has 3 placeholders: predicate, atom_hash, args.
		sb.WriteString("(?,?,?)")
	}
	sb.WriteString(" ON CONFLICT DO NOTHING")
	return sb.String()
}

func (d sqliteDialect) getFactsFragment(index int, params *[]any) string {
	// To correctly compare JSON values in SQLite, we must compare their
	// extracted forms. We wrap the parameter in a single-element JSON array
	// and extract from it. This ensures that strings are compared with strings,
	// numbers with numbers, etc., without ambiguity.
	// e.g., json_extract('["/foo"]', '$[0]') == json_extract('["/foo"]', '$[0]').
	// Using a builder is more performant than fmt.Sprintf.
	var sb strings.Builder
	sb.WriteString(" AND json_extract(args, '$[")
	sb.WriteString(strconv.Itoa(index))
	sb.WriteString("]') = json_extract(?, '$[0]')")
	return sb.String()
}

func (d sqliteDialect) jsonParam(jsonStr string) any {
	// Wrap the JSON string in a single-element array `["..."]` for the comparison.
	// Simple concatenation is faster than fmt.Sprintf.
	return "[" + jsonStr + "]"
}

// --- PostgreSQL Dialect ---

type postgresDialect struct{}

func (d postgresDialect) createTableSQL() string {
	return `
		CREATE TABLE IF NOT EXISTS facts (
			predicate TEXT NOT NULL,
			atom_hash BIGINT NOT NULL,
			args JSONB NOT NULL,
			PRIMARY KEY(atom_hash)
		);
	`
}

func (d postgresDialect) createIndexSQL() string {
	// In PostgreSQL, ON CONFLICT needs an index to work, which the PRIMARY KEY provides.
	// This index is for GetFacts performance.
	return `CREATE INDEX IF NOT EXISTS idx_predicate ON facts(predicate);`
}

func (d postgresDialect) addSQL() string {
	// By omitting the ::jsonb cast, we rely on the driver to use the binary
	// protocol for jsonb, which is more efficient than sending text and casting.
	return `
		INSERT INTO facts (predicate, atom_hash, args)
		VALUES ($1, $2, $3)
		ON CONFLICT (atom_hash) DO NOTHING
	`
}

func (d postgresDialect) removeSQL() string {
	return `DELETE FROM facts WHERE atom_hash = $1`
}

func (d postgresDialect) containsSQL() string {
	return `SELECT COUNT(*) FROM facts WHERE atom_hash = $1`
}

func (d postgresDialect) getFactsBaseSQL() string {
	return `SELECT predicate, args::text FROM facts WHERE predicate = $1`
}

func (d postgresDialect) batchInsertSQL(numRows int) string {
	var sb strings.Builder
	sb.WriteString("INSERT INTO facts (predicate, atom_hash, args) VALUES ")
	paramIndex := 1
	for i := range numRows {
		if i > 0 {
			sb.WriteString(",")
		}
		// Each row has 3 placeholders. We omit the ::jsonb cast for binary transfer.
		// Appending directly to the builder is more efficient than fmt.Sprintf.
		sb.WriteString("($")
		sb.WriteString(strconv.Itoa(paramIndex))
		sb.WriteString(", $")
		sb.WriteString(strconv.Itoa(paramIndex + 1))
		sb.WriteString(", $")
		sb.WriteString(strconv.Itoa(paramIndex + 2))
		sb.WriteString(")")
		paramIndex += 3
	}
	// PostgreSQL requires specifying the conflict target column(s).
	sb.WriteString(" ON CONFLICT (atom_hash) DO NOTHING")
	return sb.String()
}

func (d postgresDialect) getFactsFragment(index int, params *[]any) string {
	// The '->' operator extracts a JSON array element as jsonb.
	// We compare it to the parameter, which is sent as a jsonb type.
	// The placeholder is determined by the current length of the params slice.
	var sb strings.Builder
	sb.WriteString(" AND (args -> ")
	sb.WriteString(strconv.Itoa(index))
	sb.WriteString(") = $")
	sb.WriteString(strconv.Itoa(len(*params) + 1))
	return sb.String()
}

func (d postgresDialect) jsonParam(jsonStr string) any {
	return jsonStr
}
