package factstoredb

import (
	"fmt"
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
	// Use jsonb() to convert JSON text to binary JSONB format
	return `
		INSERT INTO facts (predicate, atom_hash, args)
		VALUES (?, ?, jsonb(?))
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
	for i := 0; i < numRows; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		// Each row has 3 placeholders: predicate, atom_hash, args
		// The args placeholder is wrapped in jsonb() to convert text to binary JSON.
		sb.WriteString("(?,?,jsonb(?))")
	}
	sb.WriteString(" ON CONFLICT DO NOTHING")
	return sb.String()
}

func (d sqliteDialect) getFactsFragment(index int, params *[]any) string {
	// To correctly compare JSON values in SQLite, we must compare their
	// extracted forms. We wrap the parameter in a single-element JSON array
	// and extract from it. This ensures that strings are compared with strings,
	// numbers with numbers, etc., without ambiguity.
	// e.g., json_extract('["/foo"]', '$[0]') == json_extract('["/foo"]', '$[0]')
	return fmt.Sprintf(" AND json_extract(args, '$[%d]') = json_extract(?, '$[0]')", index)
}

func (d sqliteDialect) jsonParam(jsonStr string) any {
	// Wrap the JSON string in a single-element array `["..."]` for the comparison.
	return fmt.Sprintf("[%s]", jsonStr)
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
	// Use a type cast (?::jsonb) and specify the conflict target.
	return `
		INSERT INTO facts (predicate, atom_hash, args)
		VALUES ($1, $2, $3::jsonb)
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
	for i := 0; i < numRows; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		// Each row has 3 placeholders, and the args placeholder needs a type cast.
		sb.WriteString(fmt.Sprintf("($%d, $%d, $%d::jsonb)", paramIndex, paramIndex+1, paramIndex+2))
		paramIndex += 3
	}
	// PostgreSQL requires specifying the conflict target column(s).
	sb.WriteString(" ON CONFLICT (atom_hash) DO NOTHING")
	return sb.String()
}

func (d postgresDialect) getFactsFragment(index int, params *[]any) string {
	// The '->' operator extracts a JSON array element as jsonb.
	// We compare it to the parameter, which is also cast to jsonb.
	// The placeholder is determined by the current length of the params slice.
	return fmt.Sprintf(" AND (args -> %d) = $%d::jsonb", index, len(*params)+1)
}

func (d postgresDialect) jsonParam(jsonStr string) any {
	return jsonStr
}
