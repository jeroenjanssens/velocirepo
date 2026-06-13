package store

import (
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
)

type SchemaColumn struct {
	Table    string
	Column   string
	Type     string
	Nullable string
}

func QueryLive(dataDir, query string) ([]map[string]interface{}, error) {
	db, err := openLiveDB(dataDir)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	return queryRows(db, query)
}

func SchemaLive(dataDir string) ([]SchemaColumn, error) {
	db, err := openLiveDB(dataDir)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT 'metrics' AS table_name, column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name = 'metrics' ORDER BY ordinal_position")
	if err != nil {
		return nil, fmt.Errorf("query schema: %w", err)
	}
	defer rows.Close()

	var cols []SchemaColumn
	for rows.Next() {
		var c SchemaColumn
		if err := rows.Scan(&c.Table, &c.Column, &c.Type, &c.Nullable); err != nil {
			return nil, fmt.Errorf("scan schema row: %w", err)
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

func openLiveDB(dataDir string) (*sql.DB, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("open in-memory duckdb: %w", err)
	}

	absDir, err := filepath.Abs(dataDir)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("resolve data dir: %w", err)
	}

	glob := filepath.ToSlash(filepath.Join(absDir, "*", "*", "*.jsonl"))
	prefix := filepath.ToSlash(absDir)
	query := fmt.Sprintf(`CREATE OR REPLACE VIEW metrics AS
		SELECT
			project_id AS project,
			string_split(replace(replace(filename, '\', '/'), '%s/', ''), '/')[1] AS source,
			metric,
			CAST(date AS DATE) AS date,
			CAST(value AS BIGINT) AS value,
			tags
		FROM read_json('%s',
			format='newline_delimited',
			filename=true,
			columns={metric: 'VARCHAR', project_id: 'VARCHAR', date: 'VARCHAR', value: 'BIGINT', tags: 'JSON'})`,
		prefix, glob)

	if _, err := db.Exec(query); err != nil {
		slog.Debug("live view creation failed, using empty view", "error", err)
		_, err2 := db.Exec(`CREATE VIEW metrics (project, source, metric, date, value, tags) AS
			SELECT NULL::VARCHAR, NULL::VARCHAR, NULL::VARCHAR, NULL::DATE, NULL::BIGINT, NULL::JSON
			WHERE false`)
		if err2 != nil {
			db.Close()
			return nil, fmt.Errorf("create empty view: %w", err2)
		}
	}

	return db, nil
}

func queryRows(db *sql.DB, query string) ([]map[string]interface{}, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = values[i]
		}
		results = append(results, row)
	}
	return results, rows.Err()
}
