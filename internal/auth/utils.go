package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	// API key format: sk_live_1234567890abcdef...
	APIKeyLength = 48
	APIKeyPrefix = "sk_"
)

// HashPassword creates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword checks if the provided password matches the hash
func VerifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// GenerateAPIKey creates a new API key with the specified environment
func GenerateAPIKey(environment string) (fullKey, prefix, hash string, err error) {
	// Generate 48 random bytes
	randomBytes := make([]byte, 36) // base64 encoding expands by 4/3
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Create base64 encoded key part (without padding)
	keyPart := strings.TrimRight(base64.URLEncoding.EncodeToString(randomBytes), "=")
	if len(keyPart) > APIKeyLength {
		keyPart = keyPart[:APIKeyLength]
	}

	// Create full key: sk_live_... or sk_test_...
	fullKey = fmt.Sprintf("%s%s_%s", APIKeyPrefix, environment, keyPart)
	
	// Create prefix for display (first 12 characters)
	prefix = fullKey[:12] + "..."

	// Create hash for storage
	h := sha256.Sum256([]byte(fullKey))
	hash = fmt.Sprintf("%x", h)

	return fullKey, prefix, hash, nil
}

// HashAPIKey creates a SHA-256 hash of the API key for secure storage
func HashAPIKey(apiKey string) string {
	h := sha256.Sum256([]byte(apiKey))
	return fmt.Sprintf("%x", h)
}

// GenerateHourBucket creates a bucket string for hourly rate limiting
func GenerateHourBucket(t time.Time) string {
	return t.Format("2006-01-02-15")
}

// GenerateDateBucket creates a bucket string for daily rate limiting
func GenerateDateBucket(t time.Time) string {
	return t.Format("2006-01-02")
}

// ValidateAPIKeyFormat checks if the API key has the correct format
func ValidateAPIKeyFormat(apiKey string) error {
	if !strings.HasPrefix(apiKey, APIKeyPrefix) {
		return fmt.Errorf("invalid API key format: must start with %s", APIKeyPrefix)
	}

	// Remove sk_ prefix
	withoutPrefix := strings.TrimPrefix(apiKey, APIKeyPrefix)
	
	// Find the first underscore to separate environment from key
	envSepIndex := strings.Index(withoutPrefix, "_")
	if envSepIndex == -1 {
		return fmt.Errorf("invalid API key format: expected sk_env_key")
	}

	env := withoutPrefix[:envSepIndex]
	if env != "live" && env != "test" {
		return fmt.Errorf("invalid environment: must be 'live' or 'test'")
	}

	keyPart := withoutPrefix[envSepIndex+1:]
	if len(keyPart) < APIKeyLength {
		return fmt.Errorf("invalid API key length")
	}

	return nil
}