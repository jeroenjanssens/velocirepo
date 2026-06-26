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

func BuildDB(dataDir string, projects []ProjectInfo, indicators []IndicatorDef) error {
	absDir, err := filepath.Abs(dataDir)
	if err != nil {
		return fmt.Errorf("resolve data dir: %w", err)
	}

	dbPath := filepath.Join(absDir, "velocirepo.duckdb")

	os.Remove(dbPath)
	os.Remove(dbPath + ".wal")

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("open duckdb file: %w", err)
	}
	defer db.Close()

	if err := createGitHubEventsViewRelative(db, absDir); err != nil {
		return err
	}
	if err := createMetricsViewRelative(db, absDir); err != nil {
		return err
	}
	if err := createYouTubeIndexViewRelative(db, absDir); err != nil {
		return err
	}
	if err := createProjectsTable(db, projects); err != nil {
		return err
	}
	if err := createIndicatorsView(db, indicators); err != nil {
		return err
	}

	return nil
}

func createGitHubEventsViewRelative(db *sql.DB, absDir string) error {
	glob := "github/*/*.jsonl"
	absGlob := filepath.ToSlash(filepath.Join(absDir, glob))

	if !globHasMatches(absGlob) {
		slog.Debug("no github event files found, creating empty view")
		return createEmptyGitHubEventsView(db)
	}

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
		escapeSQLString(absGlob))

	if _, err := db.Exec(query); err != nil {
		slog.Debug("github_events view creation failed, using empty view", "error", err)
		return createEmptyGitHubEventsView(db)
	}
	return nil
}

func createMetricsViewRelative(db *sql.DB, absDir string) error {
	query := metricsViewSQL(absDir)
	if _, err := db.Exec(query); err != nil {
		slog.Debug("metrics view creation failed, using empty view", "error", err)
		return createEmptyMetricsView(db)
	}
	return nil
}

func createYouTubeIndexViewRelative(db *sql.DB, absDir string) error {
	glob := "youtube/*/index.jsonl"
	absGlob := filepath.ToSlash(filepath.Join(absDir, glob))

	if !globHasMatches(absGlob) {
		slog.Debug("no youtube index files found, creating empty view")
		return createEmptyYouTubeIndexView(db)
	}

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
		escapeSQLString(absGlob))

	if _, err := db.Exec(query); err != nil {
		slog.Debug("youtube_index view creation failed, using empty view", "error", err)
		return createEmptyYouTubeIndexView(db)
	}
	return nil
}

func createProjectsTable(db *sql.DB, projects []ProjectInfo) error {
	_, err := db.Exec(`DROP TABLE IF EXISTS projects`)
	if err != nil {
		return fmt.Errorf("drop projects table: %w", err)
	}

	_, err = db.Exec(`CREATE TABLE projects (
		id VARCHAR,
		name VARCHAR,
		description VARCHAR,
		color VARCHAR,
		tags VARCHAR[],
		website VARCHAR,
		logo VARCHAR
	)`)
	if err != nil {
		return fmt.Errorf("create projects table: %w", err)
	}

	if len(projects) == 0 {
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

	query := `INSERT INTO projects VALUES ` + strings.Join(rows, ", ")
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("insert projects: %w", err)
	}

	return nil
}

func globHasMatches(pattern string) bool {
	matches, err := filepath.Glob(pattern)
	return err == nil && len(matches) > 0
}
