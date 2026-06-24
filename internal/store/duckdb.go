package store

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/marcboeker/go-duckdb"
)

type SchemaColumn struct {
	Table    string
	Column   string
	Type     string
	Nullable string
}

type ProjectInfo struct {
	ID          string
	Name        string
	Description string
	Color       string
	Tags        []string
	Website     string
	Logo        string
}

func QueryLive(dataDir string, projects []ProjectInfo, indicators []IndicatorDef, query string) ([]map[string]interface{}, []string, error) {
	db, err := openLiveDB(dataDir, projects, indicators)
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()

	return queryRows(db, query)
}

func QueryLiveParquet(dataDir string, projects []ProjectInfo, indicators []IndicatorDef, query string, outPath string) error {
	db, err := openLiveDB(dataDir, projects, indicators)
	if err != nil {
		return err
	}
	defer db.Close()

	copySQL := fmt.Sprintf(`COPY (%s) TO '%s' (FORMAT PARQUET)`, query, escapeSQLString(outPath))
	_, err = db.Exec(copySQL)
	return err
}

func SchemaLive(dataDir string, projects []ProjectInfo, indicators []IndicatorDef) ([]SchemaColumn, error) {
	db, err := openLiveDB(dataDir, projects, indicators)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SELECT table_name, column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name IN ('github_events', 'indicators', 'metrics', 'projects', 'youtube_index') ORDER BY table_name, ordinal_position")
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

func openLiveDB(dataDir string, projects []ProjectInfo, indicators []IndicatorDef) (*sql.DB, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("open in-memory duckdb: %w", err)
	}

	absDir, err := filepath.Abs(dataDir)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("resolve data dir: %w", err)
	}

	if err := createGitHubEventsView(db, absDir); err != nil {
		db.Close()
		return nil, err
	}

	if err := createMetricsView(db, absDir); err != nil {
		db.Close()
		return nil, err
	}

	if err := createYouTubeIndexView(db, absDir); err != nil {
		db.Close()
		return nil, err
	}

	if err := createProjectsView(db, projects); err != nil {
		db.Close()
		return nil, err
	}

	if err := createIndicatorsView(db, indicators); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func createMetricsView(db *sql.DB, absDir string) error {
	entries, err := os.ReadDir(absDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read data dir: %w", err)
	}

	var globs []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "github" {
			continue
		}
		g := filepath.ToSlash(filepath.Join(absDir, entry.Name(), "*", "*.jsonl"))
		globs = append(globs, "'"+escapeSQLString(g)+"'")
	}

	githubAgg := `SELECT
			project,
			'github' AS source,
			github_repo AS target,
			CASE event_type
				WHEN 'star' THEN 'daily_stars'
				WHEN 'fork' THEN 'daily_forks'
				WHEN 'issue_open' THEN 'daily_issues_opened'
				WHEN 'issue_close' THEN 'daily_issues_closed'
				WHEN 'pr_open' THEN 'daily_prs_opened'
				WHEN 'pr_merge' THEN 'daily_prs_merged'
				ELSE 'daily_' || event_type
			END AS metric,
			CAST(datetime AS DATE) AS date,
			COUNT(*) AS value,
			NULL::JSON AS tags
		FROM github_events
		GROUP BY project, github_repo, event_type, CAST(datetime AS DATE)`

	var query string
	if len(globs) > 0 {
		globList := strings.Join(globs, ", ")
		query = fmt.Sprintf(`CREATE OR REPLACE VIEW metrics AS
			SELECT
				project_id AS project,
				source,
				target,
				metric,
				CAST(date AS DATE) AS date,
				CAST(value AS BIGINT) AS value,
				tags
			FROM read_json([%s],
				format='newline_delimited',
				columns={source: 'VARCHAR', metric: 'VARCHAR', project_id: 'VARCHAR', target: 'VARCHAR', date: 'VARCHAR', value: 'BIGINT', tags: 'JSON'})
			UNION ALL
			%s`, globList, githubAgg)
	} else {
		query = fmt.Sprintf(`CREATE OR REPLACE VIEW metrics AS %s`, githubAgg)
	}

	if _, err := db.Exec(query); err != nil {
		slog.Debug("metrics view creation failed, using empty view", "error", err)
		return createEmptyMetricsView(db)
	}
	return nil
}

func createEmptyMetricsView(db *sql.DB) error {
	_, err := db.Exec(`CREATE VIEW metrics (project, source, target, metric, date, value, tags) AS
		SELECT NULL::VARCHAR, NULL::VARCHAR, NULL::VARCHAR, NULL::VARCHAR, NULL::DATE, NULL::BIGINT, NULL::JSON
		WHERE false`)
	if err != nil {
		return fmt.Errorf("create empty metrics view: %w", err)
	}
	return nil
}

func createGitHubEventsView(db *sql.DB, absDir string) error {
	glob := filepath.ToSlash(filepath.Join(absDir, "github", "*", "*.jsonl"))
	query := fmt.Sprintf(`CREATE OR REPLACE VIEW github_events AS
		SELECT
			project_id AS project,
			source,
			event_type,
			github_repo,
			CAST(datetime AS TIMESTAMP) AS datetime,
			"user"
		FROM read_json('%s',
			format='newline_delimited',
			columns={source: 'VARCHAR', event_type: 'VARCHAR', project_id: 'VARCHAR', github_repo: 'VARCHAR', datetime: 'VARCHAR', "user": 'VARCHAR'})`,
		escapeSQLString(glob))

	if _, err := db.Exec(query); err != nil {
		slog.Debug("github_events view creation failed, using empty view", "error", err)
		return createEmptyGitHubEventsView(db)
	}
	return nil
}

func createEmptyGitHubEventsView(db *sql.DB) error {
	_, err := db.Exec(`CREATE VIEW github_events (project, source, event_type, github_repo, datetime, "user") AS
		SELECT NULL::VARCHAR, NULL::VARCHAR, NULL::VARCHAR, NULL::VARCHAR, NULL::TIMESTAMP, NULL::VARCHAR
		WHERE false`)
	if err != nil {
		return fmt.Errorf("create empty github_events view: %w", err)
	}
	return nil
}


func createYouTubeIndexView(db *sql.DB, absDir string) error {
	glob := filepath.ToSlash(filepath.Join(absDir, "youtube", "*", "index.jsonl"))
	query := fmt.Sprintf(`CREATE OR REPLACE VIEW youtube_index AS
		SELECT
			video_id,
			title,
			CAST(published_at AS TIMESTAMP) AS published_at,
			channel,
			duration,
			tags
		FROM read_json('%s',
			format='newline_delimited',
			columns={video_id: 'VARCHAR', title: 'VARCHAR', published_at: 'VARCHAR', channel: 'VARCHAR', duration: 'BIGINT', tags: 'JSON'})`,
		escapeSQLString(glob))

	if _, err := db.Exec(query); err != nil {
		slog.Debug("youtube_index view creation failed, using empty view", "error", err)
		return createEmptyYouTubeIndexView(db)
	}
	return nil
}

func createEmptyYouTubeIndexView(db *sql.DB) error {
	_, err := db.Exec(`CREATE VIEW youtube_index (video_id, title, published_at, channel, duration, tags) AS
		SELECT NULL::VARCHAR, NULL::VARCHAR, NULL::TIMESTAMP, NULL::VARCHAR, NULL::BIGINT, NULL::JSON
		WHERE false`)
	if err != nil {
		return fmt.Errorf("create empty youtube_index view: %w", err)
	}
	return nil
}

func createProjectsView(db *sql.DB, projects []ProjectInfo) error {
	if len(projects) == 0 {
		_, err := db.Exec(`CREATE VIEW projects (id, name, description, color, tags, website, logo) AS
			SELECT NULL::VARCHAR, NULL::VARCHAR, NULL::VARCHAR, NULL::VARCHAR, NULL::VARCHAR[], NULL::VARCHAR, NULL::VARCHAR
			WHERE false`)
		if err != nil {
			return fmt.Errorf("create empty projects view: %w", err)
		}
		return nil
	}

	var rows []string
	for _, p := range projects {
		tags := "NULL::VARCHAR[]"
		if len(p.Tags) > 0 {
			var escaped []string
			for _, t := range p.Tags {
				escaped = append(escaped, "'"+escapeSQLString(t)+"'")
			}
			tags = "[" + strings.Join(escaped, ", ") + "]"
		}
		row := fmt.Sprintf("('%s', '%s', '%s', '%s', %s, '%s', '%s')",
			escapeSQLString(p.ID),
			escapeSQLString(p.Name),
			escapeSQLString(p.Description),
			escapeSQLString(p.Color),
			tags,
			escapeSQLString(p.Website),
			escapeSQLString(p.Logo),
		)
		rows = append(rows, row)
	}

	query := `CREATE OR REPLACE VIEW projects AS
		SELECT * FROM (VALUES ` + strings.Join(rows, ", ") + `) AS t(id, name, description, color, tags, website, logo)`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("create projects view: %w", err)
	}
	return nil
}

type IndicatorDef struct {
	Name        string
	Description string
	Query       string
}

var DefaultIndicators = []IndicatorDef{
	{
		Name:        "growth_rate",
		Description: "28-day growth rate (ratio of current vs prior 28-day sum)",
		Query: `SELECT project, source, target, metric,
	'{{indicator_name}}' AS indicator, date,
	(sum_28d - sum_prior_28d) / NULLIF(sum_prior_28d, 0.0) AS value,
	tags
FROM (
	SELECT *, SUM(value) OVER w AS sum_28d,
		SUM(value) OVER w_prior AS sum_prior_28d
	FROM metrics WHERE metric LIKE 'daily_%'
	WINDOW
		w AS (PARTITION BY project, source, target, metric, tags ORDER BY date ROWS BETWEEN 27 PRECEDING AND CURRENT ROW),
		w_prior AS (PARTITION BY project, source, target, metric, tags ORDER BY date ROWS BETWEEN 55 PRECEDING AND 28 PRECEDING)
) WHERE sum_prior_28d IS NOT NULL`,
	},
	{
		Name:        "trend",
		Description: "28-day linear trend (value per day via regression)",
		Query: `SELECT project, source, target, metric,
	'{{indicator_name}}' AS indicator, date,
	REGR_SLOPE(value, EXTRACT(EPOCH FROM CAST(date AS TIMESTAMP)) / 86400) OVER w AS value,
	tags
FROM metrics WHERE metric LIKE 'daily_%'
WINDOW w AS (PARTITION BY project, source, target, metric, tags ORDER BY date ROWS BETWEEN 27 PRECEDING AND CURRENT ROW)`,
	},
}

func createIndicatorsView(db *sql.DB, indicators []IndicatorDef) error {
	if len(indicators) == 0 {
		return createEmptyIndicatorsView(db)
	}

	var parts []string
	for _, ind := range indicators {
		q := strings.ReplaceAll(ind.Query, "{{indicator_name}}", escapeSQLString(ind.Name))
		parts = append(parts, q)
	}

	query := "CREATE OR REPLACE VIEW indicators AS " + strings.Join(parts, "\nUNION ALL\n")

	if _, err := db.Exec(query); err != nil {
		slog.Debug("indicators view creation failed, using empty view", "error", err)
		return createEmptyIndicatorsView(db)
	}
	return nil
}

func createEmptyIndicatorsView(db *sql.DB) error {
	_, err := db.Exec(`CREATE VIEW indicators (project, source, target, metric, indicator, date, value, tags) AS
		SELECT NULL::VARCHAR, NULL::VARCHAR, NULL::VARCHAR, NULL::VARCHAR, NULL::VARCHAR, NULL::DATE, NULL::DOUBLE, NULL::JSON
		WHERE false`)
	if err != nil {
		return fmt.Errorf("create empty indicators view: %w", err)
	}
	return nil
}

func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func queryRows(db *sql.DB, query string) ([]map[string]interface{}, []string, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, nil, err
		}
		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = values[i]
		}
		results = append(results, row)
	}
	return results, cols, rows.Err()
}
