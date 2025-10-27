package factstoredb

import "fmt"

// dialect defines an interface for generating database-specific SQL.
type dialect interface {
	// createTableSQL returns the SQL for creating the 'facts' table.
	createTableSQL() string
	// createIndexSQL returns the SQL for creating the index on the 'predicate' column.
	createIndexSQL() string
	// addSQL returns the SQL for inserting a fact with conflict handling.
	addSQL() string
	// getFactsFragment generates the SQL fragment for filtering by a constant argument in GetFacts.
	getFactsFragment(index int) string
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

func (d sqliteDialect) getFactsFragment(index int) string {
	// json_extract on both sides ensures correct value comparison.
	return fmt.Sprintf(" AND json_extract(args, '$[%d]') = json_extract(?, '$')", index)
}

func (d sqliteDialect) jsonParam(jsonStr string) any {
	return jsonStr
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

func (d postgresDialect) getFactsFragment(index int) string {
	// The '->' operator extracts a JSON array element as jsonb.
	// We compare it to the parameter, which is also cast to jsonb.
	// The placeholder will be e.g., $4, $5...
	return fmt.Sprintf(" AND (args -> %d) = $%d::jsonb", index, index+2)
}

func (d postgresDialect) jsonParam(jsonStr string) any {
	return jsonStr
}
