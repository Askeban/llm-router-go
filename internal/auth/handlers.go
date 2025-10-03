package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type Handlers struct {
	service       *Service
	jwtManager    *JWTManager
	githubOAuth   *oauth2.Config
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type WaitlistRequest struct {
	Email    string `json:"email" binding:"required,email"`
	FullName string `json:"fullName" binding:"required"`
	Company  string `json:"company"`
	UseCase  string `json:"useCase"`
}

type GitHubOAuthRequest struct {
	Code string `json:"code" binding:"required"`
}

func NewHandlers(service *Service, jwtManager *JWTManager) *Handlers {
	// Setup GitHub OAuth config
	githubOAuth := &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}

	return &Handlers{
		service:     service,
		jwtManager:  jwtManager,
		githubOAuth: githubOAuth,
	}
}

// Register handles user registration
func (h *Handlers) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Check if beta is full
	betaFull, err := h.service.IsBetaFull()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Service temporarily unavailable",
		})
		return
	}

	if betaFull {
		// Add to waitlist instead
		entry, err := h.service.AddToWaitlist(req.Email, req.FullName, "", "")
		if err != nil {
			if strings.Contains(err.Error(), "duplicate") {
				c.JSON(http.StatusConflict, gin.H{
					"error": "Email already registered or on waitlist",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to join waitlist",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"waitlist": true,
			"position": entry.Position,
			"message":  "Beta is full. You've been added to the waitlist.",
		})
		return
	}

	// Create user
	user, err := h.service.CreateUser(req.Email, req.Password, req.FullName)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Email already registered",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create account",
		})
		return
	}

	// Generate JWT token
	token, err := h.jwtManager.Generate(user.ID, user.Email, user.PlanType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate token",
		})
		return
	}

	// Generate refresh token
	refreshToken, err := h.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate refresh token",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"token":   token,
		"refresh_token": refreshToken,
		"user":    user,
	})
}

// Login handles user login
func (h *Handlers) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Verify credentials
	user, err := h.service.VerifyPassword(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid email or password",
		})
		return
	}

	// Check if user is active
	if !user.IsActive || user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Account is suspended or inactive",
		})
		return
	}

	// Generate JWT token
	token, err := h.jwtManager.Generate(user.ID, user.Email, user.PlanType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate token",
		})
		return
	}

	// Generate refresh token
	refreshToken, err := h.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"token":   token,
		"refresh_token": refreshToken,
		"user":    user,
	})
}

// Waitlist handles adding users to waitlist
func (h *Handlers) Waitlist(c *gin.Context) {
	var req WaitlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	entry, err := h.service.AddToWaitlist(req.Email, req.FullName, req.Company, req.UseCase)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Email already on waitlist",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to join waitlist",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"position": entry.Position,
		"message":  fmt.Sprintf("You're #%d on the waitlist", entry.Position),
	})
}

// GitHubOAuth handles GitHub OAuth authentication
func (h *Handlers) GitHubOAuth(c *gin.Context) {
	var req GitHubOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Exchange code for token
	token, err := h.githubOAuth.Exchange(context.Background(), req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Failed to exchange GitHub code",
		})
		return
	}

	// Get user info from GitHub
	client := h.githubOAuth.Client(context.Background(), token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get GitHub user info",
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read GitHub response",
		})
		return
	}

	var githubUser struct {
		ID        int64  `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.Unmarshal(body, &githubUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse GitHub user data",
		})
		return
	}

	// If email is not public, fetch it from emails endpoint
	if githubUser.Email == "" {
		emailResp, err := client.Get("https://api.github.com/user/emails")
		if err == nil {
			defer emailResp.Body.Close()
			var emails []struct {
				Email    string `json:"email"`
				Primary  bool   `json:"primary"`
				Verified bool   `json:"verified"`
			}
			emailBody, _ := io.ReadAll(emailResp.Body)
			if json.Unmarshal(emailBody, &emails) == nil {
				for _, email := range emails {
					if email.Primary && email.Verified {
						githubUser.Email = email.Email
						break
					}
				}
			}
		}
	}

	if githubUser.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "GitHub account must have a verified email address",
		})
		return
	}

	// Use login as name if name is empty
	if githubUser.Name == "" {
		githubUser.Name = githubUser.Login
	}

	// Create or get user
	user, err := h.service.CreateOrGetUserByGitHub(
		fmt.Sprintf("%d", githubUser.ID),
		githubUser.Email,
		githubUser.Name,
		githubUser.AvatarURL,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create or retrieve user",
		})
		return
	}

	// Generate JWT token
	jwtToken, err := h.jwtManager.Generate(user.ID, user.Email, user.PlanType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate token",
		})
		return
	}

	// Generate refresh token
	refreshToken, err := h.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"token":   jwtToken,
		"refresh_token": refreshToken,
		"user":    user,
	})
}

// AuthMiddleware validates JWT tokens
func (h *Handlers) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Verify token
		claims, err := h.jwtManager.Verify(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_plan", claims.Plan)

		c.Next()
	}
}

// GetProfile returns the current user's profile
func (h *Handlers) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not found in context",
		})
		return
	}

	user, err := h.service.GetUserByID(userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetUsage returns user's API usage statistics
func (h *Handlers) GetUsage(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not found in context",
		})
		return
	}

	usage, err := h.service.GetUserUsage(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get usage statistics",
		})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// Logout handles user logout (placeholder for now)
func (h *Handlers) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out successfully",
	})
}

// RefreshToken handles token refresh
func (h *Handlers) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Verify refresh token (simplified - in production, check against database)
	// For now, just generate new tokens
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Token refresh not fully implemented yet",
	})
}

// ListAPIKeys returns user's API keys
func (h *Handlers) ListAPIKeys(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"api_keys": []interface{}{},
		"message": "API key management coming soon",
	})
}

// CreateAPIKey creates a new API key for the user
func (h *Handlers) CreateAPIKey(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "API key creation coming soon",
	})
}

// APIKeyMiddleware validates API keys (placeholder)
func (h *Handlers) APIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
