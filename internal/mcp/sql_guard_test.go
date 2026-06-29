package mcp

import (
	"strings"
	"testing"
)

func TestValidateMCPQueryAllowsReadOnlyViewQueries(t *testing.T) {
	tests := []string{
		"SELECT COUNT(*) FROM metrics",
		"SELECT project, SUM(value) FROM metrics WHERE metric = 'daily_downloads' GROUP BY project",
		"WITH recent AS (SELECT * FROM metrics WHERE date > DATE '2025-01-01') SELECT COUNT(*) FROM recent",
		"SELECT e.type FROM events AS e JOIN projects p ON e.project = p.id",
		`SELECT "project" FROM "metrics"`,
		"SELECT * FROM main.metrics",
	}

	for _, query := range tests {
		t.Run(query, func(t *testing.T) {
			if _, err := validateMCPQuery(query); err != nil {
				t.Fatalf("validateMCPQuery() error = %v", err)
			}
		})
	}
}

func TestValidateMCPQueryRejectsUnsafeSQL(t *testing.T) {
	tests := []struct {
		query   string
		wantErr string
	}{
		{"SELECT content FROM read_text('/tmp/secret')", "read_text"},
		{"SELECT * FROM read_json('/tmp/data.json')", "read_json"},
		{"SELECT * FROM '/tmp/secret.csv'", "file paths"},
		{"SELECT * FROM metrics, '/tmp/secret.csv'", "file paths"},
		{"SELECT * FROM information_schema.tables", "schema"},
		{"COPY (SELECT * FROM metrics) TO '/tmp/out.parquet'", "SELECT or WITH"},
		{"SELECT * FROM metrics; SELECT * FROM events", "single SELECT"},
		{"WITH x AS (SELECT * FROM read_parquet('/tmp/x')) SELECT * FROM x", "read_parquet"},
		{"SELECT getenv('HOME')", "getenv"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			_, err := validateMCPQuery(tt.query)
			if err == nil {
				t.Fatal("validateMCPQuery() error = nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("validateMCPQuery() error = %q, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestPrepareMCPQueryAddsLimitOnlyWhenNeeded(t *testing.T) {
	query, err := prepareMCPQuery("SELECT * FROM metrics", 10)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "LIMIT 10") {
		t.Fatalf("query = %q, want LIMIT 10", query)
	}

	query, err = prepareMCPQuery("SELECT * FROM metrics LIMIT 5", 10)
	if err != nil {
		t.Fatal(err)
	}
	if query != "SELECT * FROM metrics LIMIT 5" {
		t.Fatalf("query = %q, want original query", query)
	}
}
