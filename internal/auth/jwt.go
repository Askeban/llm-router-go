package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenExpiry  = 1 * time.Hour
	RefreshTokenExpiry = 7 * 24 * time.Hour
)

// JWTManager handles JWT token operations
type JWTManager struct {
	secretKey []byte
}

// NewJWTManager creates a new JWT manager
func NewJWTManager() *JWTManager {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-secret-key-change-this-in-production" // Default for development
	}
	return &JWTManager{
		secretKey: []byte(secret),
	}
}

// GenerateTokenPair generates both access and refresh tokens
func (j *JWTManager) GenerateTokenPair(user User) (string, string, error) {
	// Generate access token
	accessToken, err := j.generateToken(user, AccessTokenExpiry, "access")
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := j.generateToken(user, RefreshTokenExpiry, "refresh")
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// generateToken creates a JWT token with the specified expiry and type
func (j *JWTManager) generateToken(user User, expiry time.Duration, tokenType string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":   user.ID,
		"email":     user.Email,
		"plan_type": user.PlanType,
		"type":      tokenType,
		"iat":       now.Unix(),
		"exp":       now.Add(expiry).Unix(),
		"iss":       "llm-router-api",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// ValidateAccessToken validates an access token and returns the claims
func (j *JWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	return j.validateToken(tokenString, "access")
}

// ValidateRefreshToken validates a refresh token and returns the claims
func (j *JWTManager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return j.validateToken(tokenString, "refresh")
}

// validateToken validates a JWT token and returns the claims
func (j *JWTManager) validateToken(tokenString, expectedType string) (*Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims format")
	}

	// Verify token type
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != expectedType {
		return nil, fmt.Errorf("invalid token type")
	}

	// Extract claims
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid user_id in token")
	}

	email, ok := claims["email"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid email in token")
	}

	planType, ok := claims["plan_type"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid plan_type in token")
	}

	return &Claims{
		UserID:   int(userID),
		Email:    email,
		PlanType: planType,
	}, nil
}

// RefreshAccessToken generates a new access token using a valid refresh token
func (j *JWTManager) RefreshAccessToken(refreshToken string, authService *Service) (string, error) {
	// Validate refresh token
	claims, err := j.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Get user info to ensure they're still active
	user, err := authService.GetUserByEmail(claims.Email)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}

	if user.Status != "active" {
		return "", fmt.Errorf("user account is not active")
	}

	// Generate new access token
	accessToken, err := j.generateToken(*user, AccessTokenExpiry, "access")
	if err != nil {
		return "", fmt.Errorf("failed to generate new access token: %w", err)
	}

	return accessToken, nil
}