package store

import (
	"fmt"
	"path/filepath"
)

func ExportParquet(dataDir, outPath string) error {
	db, err := openLiveDB(dataDir)
	if err != nil {
		return err
	}
	defer db.Close()

	absOut, err := filepath.Abs(outPath)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}

	query := fmt.Sprintf(`COPY (SELECT * FROM metrics ORDER BY project, source, metric, date) TO '%s' (FORMAT PARQUET)`, escapeSQLString(absOut))
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("export parquet: %w", err)
	}
	return nil
}

func ExportCSV(dataDir, outPath string) error {
	db, err := openLiveDB(dataDir)
	if err != nil {
		return err
	}
	defer db.Close()

	absOut, err := filepath.Abs(outPath)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}

	query := fmt.Sprintf(`COPY (SELECT * FROM metrics ORDER BY project, source, metric, date) TO '%s' (FORMAT CSV, HEADER)`, escapeSQLString(absOut))
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("export csv: %w", err)
	}
	return nil
}
