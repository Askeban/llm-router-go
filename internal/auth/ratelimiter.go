package auth

import "github.com/gin-gonic/gin"

type RateLimiter struct{}

func NewRateLimiter(redisAddr string, password string, db int) *RateLimiter {
	return &RateLimiter{}
}

func (r *RateLimiter) RateLimitMiddleware(authService *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
