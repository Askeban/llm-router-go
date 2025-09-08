package auth

import (
	"time"
)

// User represents a customer account
type User struct {
	ID              int       `json:"id" db:"id"`
	Email           string    `json:"email" db:"email"`
	PasswordHash    string    `json:"-" db:"password_hash"`
	CompanyName     *string   `json:"company_name,omitempty" db:"company_name"`
	FirstName       *string   `json:"first_name,omitempty" db:"first_name"`
	LastName        *string   `json:"last_name,omitempty" db:"last_name"`
	PlanType        string    `json:"plan_type" db:"plan_type"`
	Status          string    `json:"status" db:"status"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty" db:"email_verified_at"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
}

// APIKey represents an API key for authentication
type APIKey struct {
	ID                int       `json:"id" db:"id"`
	UserID            int       `json:"user_id" db:"user_id"`
	KeyPrefix         string    `json:"key_prefix" db:"key_prefix"`
	KeyHash           string    `json:"-" db:"key_hash"`
	Name              string    `json:"name" db:"name"`
	IsActive          bool      `json:"is_active" db:"is_active"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	Permissions       string    `json:"permissions" db:"permissions"`
	RateLimitOverride *int      `json:"rate_limit_override,omitempty" db:"rate_limit_override"`
}

// PlanLimits represents the limits for a subscription plan
type PlanLimits struct {
	PlanType         string    `json:"plan_type" db:"plan_type"`
	RequestsPerHour  int       `json:"requests_per_hour" db:"requests_per_hour"`
	RequestsPerDay   int       `json:"requests_per_day" db:"requests_per_day"`
	RequestsPerMonth int       `json:"requests_per_month" db:"requests_per_month"`
	MaxAPIKeys       int       `json:"max_api_keys" db:"max_api_keys"`
	CanGenerate      bool      `json:"can_generate" db:"can_generate"`
	PrioritySupport  bool      `json:"priority_support" db:"priority_support"`
	RateLimitBurst   int       `json:"rate_limit_burst" db:"rate_limit_burst"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// APIUsage tracks API usage for analytics and rate limiting
type APIUsage struct {
	ID               int       `json:"id" db:"id"`
	UserID           int       `json:"user_id" db:"user_id"`
	APIKeyID         int       `json:"api_key_id" db:"api_key_id"`
	Endpoint         string    `json:"endpoint" db:"endpoint"`
	PromptCategory   *string   `json:"prompt_category,omitempty" db:"prompt_category"`
	RecommendedModel *string   `json:"recommended_model,omitempty" db:"recommended_model"`
	TokensEstimated  *int      `json:"tokens_estimated,omitempty" db:"tokens_estimated"`
	ResponseTimeMs   *int      `json:"response_time_ms,omitempty" db:"response_time_ms"`
	StatusCode       int       `json:"status_code" db:"status_code"`
	ErrorMessage     *string   `json:"error_message,omitempty" db:"error_message"`
	Timestamp        time.Time `json:"timestamp" db:"timestamp"`
	DateBucket       string    `json:"date_bucket" db:"date_bucket"`
	HourBucket       string    `json:"hour_bucket" db:"hour_bucket"`
}

// Registration request
type RegisterRequest struct {
	Email       string  `json:"email" binding:"required,email"`
	Password    string  `json:"password" binding:"required,min=8"`
	CompanyName *string `json:"company_name,omitempty"`
	FirstName   *string `json:"first_name,omitempty"`
	LastName    *string `json:"last_name,omitempty"`
}

// Login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login response
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	User         User   `json:"user"`
}

// Create API key request
type CreateAPIKeyRequest struct {
	Name        string `json:"name" binding:"required"`
	Environment string `json:"environment" binding:"required,oneof=live test"`
}

// Create API key response
type CreateAPIKeyResponse struct {
	APIKey    string    `json:"api_key"`
	KeyPrefix string    `json:"key_prefix"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Warning   string    `json:"warning"`
}

// JWT Claims
type Claims struct {
	UserID   int    `json:"user_id"`
	Email    string `json:"email"`
	PlanType string `json:"plan_type"`
}