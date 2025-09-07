package storage

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
)

func InitSQLite(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	_, _ = db.Exec(`PRAGMA journal_mode=WAL;`)
	
	// Create benchmark_metrics table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS benchmark_metrics(
			model_id TEXT NOT NULL,
			source TEXT NOT NULL,
			metric TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT,
			task TEXT,
			difficulty TEXT,
			ts INTEGER,
			PRIMARY KEY(model_id, source, metric)
		);
	`)
	if err != nil {
		return nil, err
	}
	
	return db, nil
}
