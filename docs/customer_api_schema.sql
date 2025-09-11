-- Customer API Database Schema Design
-- Extends existing benchmark_metrics table

-- Users table for customer accounts
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

-- API Keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_prefix TEXT NOT NULL,                    -- First 8 chars for display (sk_12345678...)
    key_hash TEXT NOT NULL,                      -- SHA-256 hash of full key
    name TEXT NOT NULL,                          -- Customer-defined name ("Production", "Development")
    is_active BOOLEAN DEFAULT 1,
    last_used_at INTEGER,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    expires_at INTEGER,                          -- Optional expiration
    permissions TEXT DEFAULT 'read,recommend',   -- JSON array of permissions
    rate_limit_override INTEGER                  -- Optional custom rate limit
);

-- Usage tracking table
CREATE TABLE IF NOT EXISTS api_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id INTEGER NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,                      -- '/v1/recommend', '/v1/generate'
    prompt_category TEXT,                        -- From classifier: 'coding', 'math', etc.
    recommended_model TEXT,                      -- Model that was recommended
    tokens_estimated INTEGER,                    -- Token count estimate
    response_time_ms INTEGER,                    -- API response time
    status_code INTEGER,                         -- HTTP status (200, 429, 500, etc.)
    error_message TEXT,                          -- If error occurred
    timestamp INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
    date_bucket TEXT NOT NULL,                   -- YYYY-MM-DD for daily quotas
    hour_bucket TEXT NOT NULL                    -- YYYY-MM-DD-HH for hourly limits
);

-- Plan limits configuration
CREATE TABLE IF NOT EXISTS plan_limits (
    plan_type TEXT PRIMARY KEY,
    requests_per_hour INTEGER NOT NULL,
    requests_per_day INTEGER NOT NULL,
    requests_per_month INTEGER NOT NULL,
    max_api_keys INTEGER DEFAULT 5,
    can_generate BOOLEAN DEFAULT 0,             -- Can use /generate endpoint (costs more)
    priority_support BOOLEAN DEFAULT 0,
    rate_limit_burst INTEGER DEFAULT 10,        -- Burst allowance
    created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_usage_user_date ON api_usage(user_id, date_bucket);
CREATE INDEX IF NOT EXISTS idx_usage_user_hour ON api_usage(user_id, hour_bucket);
CREATE INDEX IF NOT EXISTS idx_usage_key ON api_usage(api_key_id);
CREATE INDEX IF NOT EXISTS idx_usage_timestamp ON api_usage(timestamp);

-- Initial plan data
INSERT OR REPLACE INTO plan_limits (plan_type, requests_per_hour, requests_per_day, requests_per_month, max_api_keys, can_generate) VALUES
('free', 100, 1000, 10000, 2, 0),
('starter', 1000, 10000, 100000, 5, 1),
('pro', 5000, 50000, 500000, 10, 1),
('enterprise', 20000, 200000, 2000000, 50, 1);

-- Triggers to update timestamps
CREATE TRIGGER IF NOT EXISTS update_users_timestamp 
    AFTER UPDATE ON users
    BEGIN
        UPDATE users SET updated_at = strftime('%s', 'now') WHERE id = NEW.id;
    END;