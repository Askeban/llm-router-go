package auth

import (
	"database/sql"
	"fmt"
	"time"
)

// Service handles authentication operations
type Service struct {
	db *sql.DB
}

// NewService creates a new auth service
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// CreateUser creates a new user account
func (s *Service) CreateUser(req RegisterRequest) (*User, error) {
	// Hash the password
	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert user into database
	query := `
		INSERT INTO users (email, password_hash, company_name, first_name, last_name, status)
		VALUES (?, ?, ?, ?, ?, 'active')
		RETURNING id, email, company_name, first_name, last_name, plan_type, status, created_at, updated_at
	`

	var user User
	var createdAtUnix, updatedAtUnix int64

	err = s.db.QueryRow(query, req.Email, hashedPassword, req.CompanyName, req.FirstName, req.LastName).Scan(
		&user.ID, &user.Email, &user.CompanyName, &user.FirstName, &user.LastName,
		&user.PlanType, &user.Status, &createdAtUnix, &updatedAtUnix,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Convert Unix timestamps to time.Time
	user.CreatedAt = time.Unix(createdAtUnix, 0)
	user.UpdatedAt = time.Unix(updatedAtUnix, 0)

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (s *Service) GetUserByEmail(email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, company_name, first_name, last_name, 
		       plan_type, status, created_at, updated_at, email_verified_at, last_login_at
		FROM users WHERE email = ?
	`

	var user User
	var createdAtUnix, updatedAtUnix int64
	var emailVerifiedAtUnix, lastLoginAtUnix sql.NullInt64

	err := s.db.QueryRow(query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.CompanyName, &user.FirstName, &user.LastName,
		&user.PlanType, &user.Status, &createdAtUnix, &updatedAtUnix, &emailVerifiedAtUnix, &lastLoginAtUnix,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Convert Unix timestamps to time.Time
	user.CreatedAt = time.Unix(createdAtUnix, 0)
	user.UpdatedAt = time.Unix(updatedAtUnix, 0)
	
	if emailVerifiedAtUnix.Valid {
		t := time.Unix(emailVerifiedAtUnix.Int64, 0)
		user.EmailVerifiedAt = &t
	}
	
	if lastLoginAtUnix.Valid {
		t := time.Unix(lastLoginAtUnix.Int64, 0)
		user.LastLoginAt = &t
	}

	return &user, nil
}

// UpdateLastLogin updates the user's last login time
func (s *Service) UpdateLastLogin(userID int) error {
	query := `UPDATE users SET last_login_at = strftime('%s', 'now') WHERE id = ?`
	_, err := s.db.Exec(query, userID)
	return err
}

// CreateAPIKey creates a new API key for a user
func (s *Service) CreateAPIKey(userID int, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	// Check if user has reached API key limit
	planLimits, err := s.GetPlanLimits(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check plan limits: %w", err)
	}

	// Count existing API keys
	var keyCount int
	err = s.db.QueryRow("SELECT COUNT(*) FROM api_keys WHERE user_id = ? AND is_active = 1", userID).Scan(&keyCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count API keys: %w", err)
	}

	if keyCount >= planLimits.MaxAPIKeys {
		return nil, fmt.Errorf("API key limit reached for plan %s (max: %d)", planLimits.PlanType, planLimits.MaxAPIKeys)
	}

	// Generate new API key
	fullKey, prefix, hash, err := GenerateAPIKey(req.Environment)
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Insert API key into database
	query := `
		INSERT INTO api_keys (user_id, key_prefix, key_hash, name, is_active, created_at)
		VALUES (?, ?, ?, ?, 1, strftime('%s', 'now'))
		RETURNING created_at
	`

	var createdAtUnix int64
	err = s.db.QueryRow(query, userID, prefix, hash, req.Name).Scan(&createdAtUnix)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	return &CreateAPIKeyResponse{
		APIKey:    fullKey,
		KeyPrefix: prefix,
		Name:      req.Name,
		CreatedAt: time.Unix(createdAtUnix, 0),
		Warning:   "This key will only be shown once. Store it securely.",
	}, nil
}

// ValidateAPIKey validates an API key and returns the associated user
func (s *Service) ValidateAPIKey(apiKey string) (*User, *APIKey, error) {
	// Validate format first
	if err := ValidateAPIKeyFormat(apiKey); err != nil {
		return nil, nil, fmt.Errorf("invalid API key format: %w", err)
	}

	// Hash the key for lookup
	hash := HashAPIKey(apiKey)

	// Query for API key and user info
	query := `
		SELECT 
			ak.id, ak.user_id, ak.key_prefix, ak.name, ak.is_active, ak.expires_at,
			u.id, u.email, u.plan_type, u.status
		FROM api_keys ak
		JOIN users u ON ak.user_id = u.id
		WHERE ak.key_hash = ? AND ak.is_active = 1 AND u.status = 'active'
	`

	var key APIKey
	var user User
	var expiresAtUnix sql.NullInt64

	err := s.db.QueryRow(query, hash).Scan(
		&key.ID, &key.UserID, &key.KeyPrefix, &key.Name, &key.IsActive, &expiresAtUnix,
		&user.ID, &user.Email, &user.PlanType, &user.Status,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, fmt.Errorf("invalid or inactive API key")
		}
		return nil, nil, fmt.Errorf("failed to validate API key: %w", err)
	}

	// Check expiration
	if expiresAtUnix.Valid {
		expiresAt := time.Unix(expiresAtUnix.Int64, 0)
		key.ExpiresAt = &expiresAt
		if time.Now().After(expiresAt) {
			return nil, nil, fmt.Errorf("API key has expired")
		}
	}

	// Update last used timestamp
	go s.updateAPIKeyLastUsed(key.ID)

	return &user, &key, nil
}

// updateAPIKeyLastUsed updates the last_used_at timestamp for an API key
func (s *Service) updateAPIKeyLastUsed(apiKeyID int) {
	query := `UPDATE api_keys SET last_used_at = strftime('%s', 'now') WHERE id = ?`
	s.db.Exec(query, apiKeyID)
}

// GetPlanLimits retrieves the plan limits for a user
func (s *Service) GetPlanLimits(userID int) (*PlanLimits, error) {
	query := `
		SELECT pl.plan_type, pl.requests_per_hour, pl.requests_per_day, pl.requests_per_month,
		       pl.max_api_keys, pl.can_generate, pl.priority_support, pl.rate_limit_burst
		FROM users u
		JOIN plan_limits pl ON u.plan_type = pl.plan_type
		WHERE u.id = ?
	`

	var limits PlanLimits
	err := s.db.QueryRow(query, userID).Scan(
		&limits.PlanType, &limits.RequestsPerHour, &limits.RequestsPerDay, &limits.RequestsPerMonth,
		&limits.MaxAPIKeys, &limits.CanGenerate, &limits.PrioritySupport, &limits.RateLimitBurst,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan limits: %w", err)
	}

	return &limits, nil
}

// ListAPIKeys returns all API keys for a user
func (s *Service) ListAPIKeys(userID int) ([]APIKey, error) {
	query := `
		SELECT id, user_id, key_prefix, name, is_active, last_used_at, created_at, expires_at
		FROM api_keys
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var key APIKey
		var lastUsedUnix, createdAtUnix sql.NullInt64
		var expiresAtUnix sql.NullInt64

		err := rows.Scan(
			&key.ID, &key.UserID, &key.KeyPrefix, &key.Name, &key.IsActive,
			&lastUsedUnix, &createdAtUnix, &expiresAtUnix,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}

		if createdAtUnix.Valid {
			key.CreatedAt = time.Unix(createdAtUnix.Int64, 0)
		}
		if lastUsedUnix.Valid {
			t := time.Unix(lastUsedUnix.Int64, 0)
			key.LastUsedAt = &t
		}
		if expiresAtUnix.Valid {
			t := time.Unix(expiresAtUnix.Int64, 0)
			key.ExpiresAt = &t
		}

		keys = append(keys, key)
	}

	return keys, nil
}

// RecordAPIUsage logs an API request for analytics and rate limiting
func (s *Service) RecordAPIUsage(usage APIUsage) error {
	now := time.Now()
	usage.Timestamp = now
	usage.DateBucket = GenerateDateBucket(now)
	usage.HourBucket = GenerateHourBucket(now)

	query := `
		INSERT INTO api_usage (user_id, api_key_id, endpoint, prompt_category, recommended_model,
		                       tokens_estimated, response_time_ms, status_code, error_message,
		                       timestamp, date_bucket, hour_bucket)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, strftime('%s', 'now'), ?, ?)
	`

	_, err := s.db.Exec(query,
		usage.UserID, usage.APIKeyID, usage.Endpoint, usage.PromptCategory, usage.RecommendedModel,
		usage.TokensEstimated, usage.ResponseTimeMs, usage.StatusCode, usage.ErrorMessage,
		usage.DateBucket, usage.HourBucket,
	)
	if err != nil {
		return fmt.Errorf("failed to record API usage: %w", err)
	}

	return nil
}