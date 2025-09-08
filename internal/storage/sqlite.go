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

	// Create customer management tables
	err = initCustomerTables(db)
	if err != nil {
		return nil, err
	}
	
	return db, nil
}

func initCustomerTables(db *sql.DB) error {
	// Users table for customer accounts
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			company_name TEXT,
			first_name TEXT,
			last_name TEXT,
			plan_type TEXT DEFAULT 'free' CHECK(plan_type IN ('free', 'starter', 'pro', 'enterprise')),
			status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'active', 'suspended', 'cancelled')),
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			email_verified_at INTEGER,
			last_login_at INTEGER
		);
	`)
	if err != nil {
		return err
	}

	// API Keys table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			key_prefix TEXT NOT NULL,
			key_hash TEXT NOT NULL,
			name TEXT NOT NULL,
			is_active BOOLEAN DEFAULT 1,
			last_used_at INTEGER,
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			expires_at INTEGER,
			permissions TEXT DEFAULT 'read,recommend',
			rate_limit_override INTEGER
		);
	`)
	if err != nil {
		return err
	}

	// Usage tracking table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS api_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			api_key_id INTEGER NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
			endpoint TEXT NOT NULL,
			prompt_category TEXT,
			recommended_model TEXT,
			tokens_estimated INTEGER,
			response_time_ms INTEGER,
			status_code INTEGER,
			error_message TEXT,
			timestamp INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
			date_bucket TEXT NOT NULL,
			hour_bucket TEXT NOT NULL
		);
	`)
	if err != nil {
		return err
	}

	// Plan limits configuration
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS plan_limits (
			plan_type TEXT PRIMARY KEY,
			requests_per_hour INTEGER NOT NULL,
			requests_per_day INTEGER NOT NULL,
			requests_per_month INTEGER NOT NULL,
			max_api_keys INTEGER DEFAULT 5,
			can_generate BOOLEAN DEFAULT 0,
			priority_support BOOLEAN DEFAULT 0,
			rate_limit_burst INTEGER DEFAULT 10,
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
		);
	`)
	if err != nil {
		return err
	}

	// Create indexes for performance
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id, is_active);`,
		`CREATE INDEX IF NOT EXISTS idx_usage_user_date ON api_usage(user_id, date_bucket);`,
		`CREATE INDEX IF NOT EXISTS idx_usage_user_hour ON api_usage(user_id, hour_bucket);`,
		`CREATE INDEX IF NOT EXISTS idx_usage_key ON api_usage(api_key_id);`,
		`CREATE INDEX IF NOT EXISTS idx_usage_timestamp ON api_usage(timestamp);`,
	}

	for _, index := range indexes {
		_, err = db.Exec(index)
		if err != nil {
			return err
		}
	}

	// Insert initial plan data
	_, err = db.Exec(`
		INSERT OR REPLACE INTO plan_limits (plan_type, requests_per_hour, requests_per_day, requests_per_month, max_api_keys, can_generate) VALUES
		('free', 100, 1000, 10000, 2, 0),
		('starter', 1000, 10000, 100000, 5, 1),
		('pro', 5000, 50000, 500000, 10, 1),
		('enterprise', 20000, 200000, 2000000, 50, 1);
	`)
	if err != nil {
		return err
	}

	// Create trigger to update timestamps
	_, err = db.Exec(`
		CREATE TRIGGER IF NOT EXISTS update_users_timestamp 
			AFTER UPDATE ON users
			BEGIN
				UPDATE users SET updated_at = strftime('%s', 'now') WHERE id = NEW.id;
			END;
	`)
	return err
}
