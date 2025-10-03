-- RouteLLM PostgreSQL Database Schema
-- Production-ready schema for Cloud SQL PostgreSQL 15

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table for customer accounts
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    company_name VARCHAR(255),
    plan_type VARCHAR(50) DEFAULT 'free' CHECK(plan_type IN ('free', 'beta', 'starter', 'pro', 'enterprise')),
    status VARCHAR(50) DEFAULT 'pending' CHECK(status IN ('pending', 'active', 'suspended', 'cancelled')),
    beta_access BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    email_verified_at TIMESTAMP WITH TIME ZONE,
    last_login_at TIMESTAMP WITH TIME ZONE,

    -- OAuth fields
    github_id VARCHAR(255) UNIQUE,
    google_id VARCHAR(255) UNIQUE,
    oauth_provider VARCHAR(50),
    avatar_url TEXT,

    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Waitlist table
CREATE TABLE IF NOT EXISTS waitlist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    full_name VARCHAR(255),
    company VARCHAR(255),
    use_case TEXT,
    position INTEGER NOT NULL,
    status VARCHAR(50) DEFAULT 'waiting' CHECK(status IN ('waiting', 'invited', 'registered', 'expired')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    invited_at TIMESTAMP WITH TIME ZONE,
    registered_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}'::jsonb
);

-- API Keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_prefix VARCHAR(20) NOT NULL,  -- First 8 chars for display (sk_12345678...)
    key_hash VARCHAR(255) NOT NULL,   -- bcrypt hash of full key
    name VARCHAR(255) NOT NULL,       -- Customer-defined name
    is_active BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    permissions TEXT[] DEFAULT ARRAY['read', 'recommend'],
    rate_limit_override INTEGER,
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Usage tracking table
CREATE TABLE IF NOT EXISTS api_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id UUID REFERENCES api_keys(id) ON DELETE SET NULL,
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10),
    prompt_category VARCHAR(100),
    recommended_model VARCHAR(255),
    tokens_estimated INTEGER,
    response_time_ms INTEGER,
    status_code INTEGER,
    error_message TEXT,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    date_bucket DATE NOT NULL DEFAULT CURRENT_DATE,
    hour_bucket TIMESTAMP NOT NULL DEFAULT date_trunc('hour', CURRENT_TIMESTAMP),
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Monthly usage summary for faster rate limit checks
CREATE TABLE IF NOT EXISTS monthly_usage_summary (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    year_month VARCHAR(7) NOT NULL,  -- YYYY-MM
    total_requests INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    unique_endpoints INTEGER DEFAULT 0,
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, year_month)
);

-- Plan limits configuration
CREATE TABLE IF NOT EXISTS plan_limits (
    plan_type VARCHAR(50) PRIMARY KEY,
    requests_per_hour INTEGER NOT NULL,
    requests_per_day INTEGER NOT NULL,
    requests_per_month INTEGER NOT NULL,
    max_api_keys INTEGER DEFAULT 5,
    can_generate BOOLEAN DEFAULT FALSE,
    priority_support BOOLEAN DEFAULT FALSE,
    rate_limit_burst INTEGER DEFAULT 10,
    max_tokens_per_request INTEGER DEFAULT 4000,
    features JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Sessions table for JWT refresh tokens
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) NOT NULL,
    device_info VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_accessed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_plan ON users(plan_type, status);
CREATE INDEX IF NOT EXISTS idx_users_beta ON users(beta_access, is_active);
CREATE INDEX IF NOT EXISTS idx_users_github ON users(github_id) WHERE github_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_google ON users(google_id) WHERE google_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_waitlist_email ON waitlist(email);
CREATE INDEX IF NOT EXISTS idx_waitlist_position ON waitlist(position, status);
CREATE INDEX IF NOT EXISTS idx_waitlist_created ON waitlist(created_at);

CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(key_prefix);

CREATE INDEX IF NOT EXISTS idx_usage_user_date ON api_usage(user_id, date_bucket);
CREATE INDEX IF NOT EXISTS idx_usage_user_hour ON api_usage(user_id, hour_bucket);
CREATE INDEX IF NOT EXISTS idx_usage_key ON api_usage(api_key_id);
CREATE INDEX IF NOT EXISTS idx_usage_timestamp ON api_usage(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_usage_endpoint ON api_usage(endpoint, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_monthly_summary_user ON monthly_usage_summary(user_id, year_month);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(refresh_token_hash);

-- Function to get next waitlist position
CREATE OR REPLACE FUNCTION get_next_waitlist_position() RETURNS INTEGER AS $$
BEGIN
    RETURN COALESCE((SELECT MAX(position) FROM waitlist), 0) + 1;
END;
$$ LANGUAGE plpgsql;

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers to update timestamps
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_plan_limits_updated_at BEFORE UPDATE ON plan_limits
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sessions_last_accessed BEFORE UPDATE ON sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Initial plan data
INSERT INTO plan_limits (plan_type, requests_per_hour, requests_per_day, requests_per_month, max_api_keys, can_generate, max_tokens_per_request, features) VALUES
('free', 10, 100, 500, 1, FALSE, 2000, '{"support": "community", "analytics": false}'::jsonb),
('beta', 100, 1000, 1000, 3, TRUE, 4000, '{"support": "email", "analytics": true, "early_access": true}'::jsonb),
('starter', 1000, 10000, 100000, 5, TRUE, 8000, '{"support": "email", "analytics": true, "custom_models": false}'::jsonb),
('pro', 5000, 50000, 500000, 10, TRUE, 16000, '{"support": "priority", "analytics": true, "custom_models": true, "webhooks": true}'::jsonb),
('enterprise', 20000, 200000, 2000000, 50, TRUE, 32000, '{"support": "dedicated", "analytics": true, "custom_models": true, "webhooks": true, "sla": true}'::jsonb)
ON CONFLICT (plan_type) DO UPDATE SET
    requests_per_hour = EXCLUDED.requests_per_hour,
    requests_per_day = EXCLUDED.requests_per_day,
    requests_per_month = EXCLUDED.requests_per_month,
    max_api_keys = EXCLUDED.max_api_keys,
    can_generate = EXCLUDED.can_generate,
    max_tokens_per_request = EXCLUDED.max_tokens_per_request,
    features = EXCLUDED.features,
    updated_at = CURRENT_TIMESTAMP;

-- Create view for active users with usage stats
CREATE OR REPLACE VIEW v_user_stats AS
SELECT
    u.id,
    u.email,
    u.full_name,
    u.plan_type,
    u.status,
    u.beta_access,
    u.created_at,
    u.last_login_at,
    COUNT(DISTINCT ak.id) as api_keys_count,
    COALESCE(SUM(mus.total_requests), 0) as total_requests_this_month,
    pl.requests_per_month as monthly_limit,
    CASE
        WHEN pl.requests_per_month > 0 THEN
            (COALESCE(SUM(mus.total_requests), 0)::FLOAT / pl.requests_per_month::FLOAT * 100)::INTEGER
        ELSE 0
    END as usage_percentage
FROM users u
LEFT JOIN api_keys ak ON u.id = ak.user_id AND ak.is_active = TRUE
LEFT JOIN monthly_usage_summary mus ON u.id = mus.user_id
    AND mus.year_month = TO_CHAR(CURRENT_DATE, 'YYYY-MM')
LEFT JOIN plan_limits pl ON u.plan_type = pl.plan_type
WHERE u.is_active = TRUE
GROUP BY u.id, u.email, u.full_name, u.plan_type, u.status, u.beta_access,
         u.created_at, u.last_login_at, pl.requests_per_month;

-- Comments for documentation
COMMENT ON TABLE users IS 'User accounts with authentication and plan information';
COMMENT ON TABLE waitlist IS 'Waitlist for beta access with position tracking';
COMMENT ON TABLE api_keys IS 'API keys for programmatic access';
COMMENT ON TABLE api_usage IS 'Detailed API usage logs for analytics and rate limiting';
COMMENT ON TABLE monthly_usage_summary IS 'Aggregated monthly usage for fast rate limit checks';
COMMENT ON TABLE plan_limits IS 'Configuration for different subscription plans';
COMMENT ON TABLE sessions IS 'User sessions for JWT refresh token management';
