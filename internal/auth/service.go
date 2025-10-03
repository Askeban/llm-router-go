package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db *sql.DB
}

type User struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	FullName       string    `json:"full_name"`
	CompanyName    string    `json:"company_name,omitempty"`
	PlanType       string    `json:"plan_type"`
	Status         string    `json:"status"`
	BetaAccess     bool      `json:"beta_access"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
	GitHubID       *string   `json:"github_id,omitempty"`
	AvatarURL      *string   `json:"avatar_url,omitempty"`
}

type WaitlistEntry struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Company   string    `json:"company,omitempty"`
	UseCase   string    `json:"use_case,omitempty"`
	Position  int       `json:"position"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// CreateUser creates a new user with hashed password
func (s *Service) CreateUser(email, password, fullName string) (*User, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Check if beta is full
	betaFull, err := s.IsBetaFull()
	if err != nil {
		return nil, fmt.Errorf("failed to check beta status: %w", err)
	}

	planType := "free"
	betaAccess := false
	if !betaFull {
		planType = "beta"
		betaAccess = true
	}

	user := &User{
		ID:         uuid.New().String(),
		Email:      email,
		FullName:   fullName,
		PlanType:   planType,
		Status:     "active",
		BetaAccess: betaAccess,
		IsActive:   true,
		CreatedAt:  time.Now(),
	}

	query := `
		INSERT INTO users (id, email, password_hash, full_name, plan_type, status, beta_access, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at`

	err = s.db.QueryRow(
		query,
		user.ID, user.Email, string(hashedPassword), user.FullName,
		user.PlanType, user.Status, user.BetaAccess, user.IsActive,
	).Scan(&user.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (s *Service) GetUserByEmail(email string) (*User, error) {
	user := &User{}
	query := `
		SELECT id, email, full_name, company_name, plan_type, status,
		       beta_access, is_active, created_at, email_verified_at,
		       last_login_at, github_id, avatar_url
		FROM users
		WHERE email = $1 AND is_active = TRUE`

	err := s.db.QueryRow(query, email).Scan(
		&user.ID, &user.Email, &user.FullName, &user.CompanyName,
		&user.PlanType, &user.Status, &user.BetaAccess, &user.IsActive,
		&user.CreatedAt, &user.EmailVerifiedAt, &user.LastLoginAt,
		&user.GitHubID, &user.AvatarURL,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *Service) GetUserByID(id string) (*User, error) {
	user := &User{}
	query := `
		SELECT id, email, full_name, company_name, plan_type, status,
		       beta_access, is_active, created_at, email_verified_at,
		       last_login_at, github_id, avatar_url
		FROM users
		WHERE id = $1 AND is_active = TRUE`

	err := s.db.QueryRow(query, id).Scan(
		&user.ID, &user.Email, &user.FullName, &user.CompanyName,
		&user.PlanType, &user.Status, &user.BetaAccess, &user.IsActive,
		&user.CreatedAt, &user.EmailVerifiedAt, &user.LastLoginAt,
		&user.GitHubID, &user.AvatarURL,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// VerifyPassword verifies a user's password
func (s *Service) VerifyPassword(email, password string) (*User, error) {
	var hashedPassword string
	var user User

	query := `
		SELECT id, email, password_hash, full_name, company_name, plan_type,
		       status, beta_access, is_active, created_at
		FROM users
		WHERE email = $1 AND is_active = TRUE`

	err := s.db.QueryRow(query, email).Scan(
		&user.ID, &user.Email, &hashedPassword, &user.FullName, &user.CompanyName,
		&user.PlanType, &user.Status, &user.BetaAccess, &user.IsActive, &user.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("invalid credentials")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Update last login
	_, _ = s.db.Exec("UPDATE users SET last_login_at = $1 WHERE id = $2", time.Now(), user.ID)

	return &user, nil
}

// IsBetaFull checks if beta access is full (100 users)
func (s *Service) IsBetaFull() (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM users
		WHERE beta_access = TRUE AND is_active = TRUE
	`).Scan(&count)

	if err != nil {
		return true, err
	}

	return count >= 100, nil
}

// AddToWaitlist adds a user to the waitlist
func (s *Service) AddToWaitlist(email, fullName, company, useCase string) (*WaitlistEntry, error) {
	entry := &WaitlistEntry{
		ID:       uuid.New().String(),
		Email:    email,
		FullName: fullName,
		Company:  company,
		UseCase:  useCase,
		Status:   "waiting",
	}

	query := `
		INSERT INTO waitlist (id, email, full_name, company, use_case, position, status)
		VALUES ($1, $2, $3, $4, $5, (SELECT get_next_waitlist_position()), $6)
		RETURNING position, created_at`

	err := s.db.QueryRow(
		query,
		entry.ID, entry.Email, entry.FullName, entry.Company, entry.UseCase, entry.Status,
	).Scan(&entry.Position, &entry.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to add to waitlist: %w", err)
	}

	return entry, nil
}

// GetUserUsage gets user's API usage statistics
func (s *Service) GetUserUsage(userID string) (map[string]interface{}, error) {
	// Get current month usage
	var totalRequests, totalTokens int
	yearMonth := time.Now().Format("2006-01")

	err := s.db.QueryRow(`
		SELECT COALESCE(total_requests, 0), COALESCE(total_tokens, 0)
		FROM monthly_usage_summary
		WHERE user_id = $1 AND year_month = $2
	`, userID, yearMonth).Scan(&totalRequests, &totalTokens)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	// Get plan limits
	var planType string
	var monthlyLimit int
	err = s.db.QueryRow(`
		SELECT u.plan_type, pl.requests_per_month
		FROM users u
		JOIN plan_limits pl ON u.plan_type = pl.plan_type
		WHERE u.id = $1
	`, userID).Scan(&planType, &monthlyLimit)

	if err != nil {
		return nil, fmt.Errorf("failed to get plan limits: %w", err)
	}

	usagePercent := 0
	if monthlyLimit > 0 {
		usagePercent = (totalRequests * 100) / monthlyLimit
	}

	return map[string]interface{}{
		"total_requests":  totalRequests,
		"total_tokens":    totalTokens,
		"monthly_limit":   monthlyLimit,
		"usage_percent":   usagePercent,
		"requests_remaining": monthlyLimit - totalRequests,
		"plan_type":       planType,
		"period":          yearMonth,
	}, nil
}

// CreateOrGetUserByGitHub creates or retrieves a user by GitHub ID
func (s *Service) CreateOrGetUserByGitHub(githubID, email, fullName, avatarURL string) (*User, error) {
	// Check if user exists with this GitHub ID
	user := &User{}
	query := `
		SELECT id, email, full_name, plan_type, status, beta_access, is_active, created_at
		FROM users
		WHERE github_id = $1 AND is_active = TRUE`

	err := s.db.QueryRow(query, githubID).Scan(
		&user.ID, &user.Email, &user.FullName, &user.PlanType,
		&user.Status, &user.BetaAccess, &user.IsActive, &user.CreatedAt,
	)

	if err == nil {
		// User exists, update last login
		_, _ = s.db.Exec("UPDATE users SET last_login_at = $1 WHERE id = $2", time.Now(), user.ID)
		return user, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check GitHub user: %w", err)
	}

	// User doesn't exist, create new one
	betaFull, err := s.IsBetaFull()
	if err != nil {
		return nil, err
	}

	planType := "free"
	betaAccess := false
	if !betaFull {
		planType = "beta"
		betaAccess = true
	}

	user = &User{
		ID:         uuid.New().String(),
		Email:      email,
		FullName:   fullName,
		PlanType:   planType,
		Status:     "active",
		BetaAccess: betaAccess,
		IsActive:   true,
		GitHubID:   &githubID,
		AvatarURL:  &avatarURL,
		CreatedAt:  time.Now(),
	}

	insertQuery := `
		INSERT INTO users (id, email, full_name, plan_type, status, beta_access,
		                   is_active, github_id, avatar_url, oauth_provider, password_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'github', '')
		RETURNING created_at`

	err = s.db.QueryRow(
		insertQuery,
		user.ID, user.Email, user.FullName, user.PlanType, user.Status,
		user.BetaAccess, user.IsActive, githubID, avatarURL,
	).Scan(&user.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub user: %w", err)
	}

	return user, nil
}
