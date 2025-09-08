package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Handlers contains all auth-related HTTP handlers
type Handlers struct {
	authService *Service
	jwtManager  *JWTManager
}

// NewHandlers creates new auth handlers
func NewHandlers(authService *Service, jwtManager *JWTManager) *Handlers {
	return &Handlers{
		authService: authService,
		jwtManager:  jwtManager,
	}
}

// Register handles user registration
func (h *Handlers) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	// Create user
	user, err := h.authService.CreateUser(req)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusConflict, gin.H{
				"error": gin.H{
					"code":    "email_already_exists",
					"message": "An account with this email already exists",
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "registration_failed",
				"message": "Failed to create account",
			},
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Account created successfully",
		"user":    user,
	})
}

// Login handles user authentication
func (h *Handlers) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	// Get user by email
	user, err := h.authService.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "invalid_credentials",
				"message": "Invalid email or password",
			},
		})
		return
	}

	// Verify password
	if err := VerifyPassword(req.Password, user.PasswordHash); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "invalid_credentials",
				"message": "Invalid email or password",
			},
		})
		return
	}

	// Check account status
	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"code":    "account_inactive",
				"message": "Your account is not active",
			},
		})
		return
	}

	// Generate tokens
	accessToken, refreshToken, err := h.jwtManager.GenerateTokenPair(*user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "token_generation_failed",
				"message": "Failed to generate authentication tokens",
			},
		})
		return
	}

	// Update last login time
	h.authService.UpdateLastLogin(user.ID)

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(AccessTokenExpiry.Seconds()),
		User:         *user,
	})
}

// RefreshToken handles token refresh
func (h *Handlers) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	// Generate new access token
	accessToken, err := h.jwtManager.RefreshAccessToken(req.RefreshToken, h.authService)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "invalid_refresh_token",
				"message": "Invalid or expired refresh token",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"expires_in":   int(AccessTokenExpiry.Seconds()),
	})
}

// CreateAPIKey handles API key creation
func (h *Handlers) CreateAPIKey(c *gin.Context) {
	// Get user from context (set by auth middleware)
	claims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "unauthorized",
				"message": "Authentication required",
			},
		})
		return
	}

	userClaims := claims.(*Claims)

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "invalid_request",
				"message": err.Error(),
			},
		})
		return
	}

	// Create API key
	apiKeyResp, err := h.authService.CreateAPIKey(userClaims.UserID, req)
	if err != nil {
		if strings.Contains(err.Error(), "limit reached") {
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":    "api_key_limit_reached",
					"message": err.Error(),
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "api_key_creation_failed",
				"message": "Failed to create API key",
			},
		})
		return
	}

	c.JSON(http.StatusCreated, apiKeyResp)
}

// ListAPIKeys handles listing user's API keys
func (h *Handlers) ListAPIKeys(c *gin.Context) {
	// Get user from context
	claims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "unauthorized",
				"message": "Authentication required",
			},
		})
		return
	}

	userClaims := claims.(*Claims)

	// Get API keys
	apiKeys, err := h.authService.ListAPIKeys(userClaims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "failed_to_list_keys",
				"message": "Failed to retrieve API keys",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_keys": apiKeys,
	})
}

// GetProfile returns the current user's profile
func (h *Handlers) GetProfile(c *gin.Context) {
	// Get user from context
	claims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "unauthorized",
				"message": "Authentication required",
			},
		})
		return
	}

	userClaims := claims.(*Claims)

	// Get full user profile
	user, err := h.authService.GetUserByEmail(userClaims.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "profile_fetch_failed",
				"message": "Failed to retrieve profile",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

// AuthMiddleware validates JWT tokens for dashboard endpoints
func (h *Handlers) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "missing_authorization",
					"message": "Authorization header is required",
				},
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "invalid_authorization_format",
					"message": "Authorization header must be 'Bearer <token>'",
				},
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := h.jwtManager.ValidateAccessToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "invalid_token",
					"message": "Invalid or expired token",
				},
			})
			c.Abort()
			return
		}

		// Set user claims in context
		c.Set("user", claims)
		c.Next()
	}
}

// APIKeyMiddleware validates API keys for customer API endpoints
func (h *Handlers) APIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "missing_api_key",
					"message": "API key is required",
				},
			})
			c.Abort()
			return
		}

		// Extract API key from "Bearer <api_key>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "invalid_api_key_format",
					"message": "Authorization header must be 'Bearer <api_key>'",
				},
			})
			c.Abort()
			return
		}

		apiKey := parts[1]

		// Validate API key
		user, keyInfo, err := h.authService.ValidateAPIKey(apiKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "invalid_api_key",
					"message": err.Error(),
				},
			})
			c.Abort()
			return
		}

		// Set user and API key info in context
		c.Set("user", user)
		c.Set("api_key", keyInfo)

		// Add user ID for easy access
		c.Set("user_id", user.ID)
		c.Set("api_key_id", keyInfo.ID)

		c.Next()
	}
}

// GetUsage returns usage statistics for the current user
func (h *Handlers) GetUsage(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "unauthorized",
				"message": "Authentication required",
			},
		})
		return
	}

	// Get query parameters for filtering
	period := c.DefaultQuery("period", "last_30_days")
	
	// For now, return a placeholder response
	// This will be implemented with actual usage aggregation in Phase 2
	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"period":  period,
		"message": "Usage analytics will be implemented in Phase 2",
		"total_requests": 0,
		"by_category": gin.H{},
		"by_model": gin.H{},
	})
}