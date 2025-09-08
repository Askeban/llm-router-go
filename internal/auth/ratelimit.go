package auth

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter handles API rate limiting using Redis
type RateLimiter struct {
	client *redis.Client
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisAddr, redisPassword string, redisDB int) *RateLimiter {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	return &RateLimiter{
		client: rdb,
	}
}

// RateLimitInfo contains rate limiting information
type RateLimitInfo struct {
	HourlyLimit     int
	DailyLimit      int
	MonthlyLimit    int
	HourlyRemaining int
	DailyRemaining  int
	MonthlyRemaining int
	HourlyResetAt   time.Time
	DailyResetAt    time.Time
	MonthlyResetAt  time.Time
	Allowed         bool
	RetryAfter      time.Duration
}

// CheckRateLimit checks if a user has exceeded their rate limits
func (rl *RateLimiter) CheckRateLimit(ctx context.Context, userID int, limits *PlanLimits) (*RateLimitInfo, error) {
	now := time.Now()
	hourKey := fmt.Sprintf("rate_limit:user:%d:hour:%s", userID, GenerateHourBucket(now))
	dayKey := fmt.Sprintf("rate_limit:user:%d:day:%s", userID, GenerateDateBucket(now))
	monthKey := fmt.Sprintf("rate_limit:user:%d:month:%s", userID, now.Format("2006-01"))

	// Get current usage counts
	pipe := rl.client.Pipeline()
	hourlyCount := pipe.Get(ctx, hourKey)
	dailyCount := pipe.Get(ctx, dayKey)
	monthlyCount := pipe.Get(ctx, monthKey)
	_, _ = pipe.Exec(ctx)

	// Initialize counts (Redis returns error for non-existent keys)
	hourlyUsed := 0
	dailyUsed := 0
	monthlyUsed := 0

	if hourlyCount.Err() == nil {
		if val, err := hourlyCount.Int(); err == nil {
			hourlyUsed = val
		}
	}

	if dailyCount.Err() == nil {
		if val, err := dailyCount.Int(); err == nil {
			dailyUsed = val
		}
	}

	if monthlyCount.Err() == nil {
		if val, err := monthlyCount.Int(); err == nil {
			monthlyUsed = val
		}
	}

	// Calculate remaining limits
	hourlyRemaining := limits.RequestsPerHour - hourlyUsed
	dailyRemaining := limits.RequestsPerDay - dailyUsed
	monthlyRemaining := limits.RequestsPerMonth - monthlyUsed

	// Calculate reset times
	hourlyResetAt := now.Truncate(time.Hour).Add(time.Hour)
	dailyResetAt := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
	monthlyResetAt := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())

	// Check if any limit is exceeded
	allowed := true
	var retryAfter time.Duration

	if hourlyRemaining <= 0 {
		allowed = false
		retryAfter = time.Until(hourlyResetAt)
	} else if dailyRemaining <= 0 {
		allowed = false
		retryAfter = time.Until(dailyResetAt)
	} else if monthlyRemaining <= 0 {
		allowed = false
		retryAfter = time.Until(monthlyResetAt)
	}

	return &RateLimitInfo{
		HourlyLimit:      limits.RequestsPerHour,
		DailyLimit:       limits.RequestsPerDay,
		MonthlyLimit:     limits.RequestsPerMonth,
		HourlyRemaining:  maxInt(hourlyRemaining, 0),
		DailyRemaining:   maxInt(dailyRemaining, 0),
		MonthlyRemaining: maxInt(monthlyRemaining, 0),
		HourlyResetAt:    hourlyResetAt,
		DailyResetAt:     dailyResetAt,
		MonthlyResetAt:   monthlyResetAt,
		Allowed:          allowed,
		RetryAfter:       retryAfter,
	}, nil
}

// IncrementUsage increments the usage counters for a user
func (rl *RateLimiter) IncrementUsage(ctx context.Context, userID int) error {
	now := time.Now()
	hourKey := fmt.Sprintf("rate_limit:user:%d:hour:%s", userID, GenerateHourBucket(now))
	dayKey := fmt.Sprintf("rate_limit:user:%d:day:%s", userID, GenerateDateBucket(now))
	monthKey := fmt.Sprintf("rate_limit:user:%d:month:%s", userID, now.Format("2006-01"))

	pipe := rl.client.Pipeline()

	// Increment counters
	pipe.Incr(ctx, hourKey)
	pipe.Incr(ctx, dayKey)
	pipe.Incr(ctx, monthKey)

	// Set expiry times
	pipe.Expire(ctx, hourKey, time.Hour+5*time.Minute) // Extra buffer
	pipe.Expire(ctx, dayKey, 25*time.Hour)             // Extra buffer
	pipe.Expire(ctx, monthKey, 32*24*time.Hour)        // Extra buffer

	_, err := pipe.Exec(ctx)
	return err
}

// RateLimitMiddleware creates a rate limiting middleware
func (rl *RateLimiter) RateLimitMiddleware(authService *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting for non-API endpoints
		if !shouldRateLimit(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Get user from context (set by API key middleware)
		userInterface, exists := c.Get("user")
		if !exists {
			c.Next() // Let auth middleware handle this
			return
		}

		user, ok := userInterface.(*User)
		if !ok {
			c.JSON(500, gin.H{
				"error": gin.H{
					"code":    "internal_error",
					"message": "Invalid user context",
				},
			})
			c.Abort()
			return
		}

		// Get plan limits
		limits, err := authService.GetPlanLimits(user.ID)
		if err != nil {
			c.JSON(500, gin.H{
				"error": gin.H{
					"code":    "rate_limit_check_failed",
					"message": "Failed to check rate limits",
				},
			})
			c.Abort()
			return
		}

		// Check rate limits
		rateLimitInfo, err := rl.CheckRateLimit(c.Request.Context(), user.ID, limits)
		if err != nil {
			c.JSON(500, gin.H{
				"error": gin.H{
					"code":    "rate_limit_check_failed",
					"message": "Failed to check rate limits",
				},
			})
			c.Abort()
			return
		}

		// Add rate limit headers
		c.Header("X-RateLimit-Limit-Hour", strconv.Itoa(rateLimitInfo.HourlyLimit))
		c.Header("X-RateLimit-Remaining-Hour", strconv.Itoa(rateLimitInfo.HourlyRemaining))
		c.Header("X-RateLimit-Reset-Hour", strconv.FormatInt(rateLimitInfo.HourlyResetAt.Unix(), 10))
		c.Header("X-RateLimit-Limit-Day", strconv.Itoa(rateLimitInfo.DailyLimit))
		c.Header("X-RateLimit-Remaining-Day", strconv.Itoa(rateLimitInfo.DailyRemaining))

		// Check if rate limit exceeded
		if !rateLimitInfo.Allowed {
			c.Header("Retry-After", strconv.FormatInt(int64(rateLimitInfo.RetryAfter.Seconds()), 10))
			
			var limitType string
			if rateLimitInfo.HourlyRemaining <= 0 {
				limitType = "hourly"
			} else if rateLimitInfo.DailyRemaining <= 0 {
				limitType = "daily"
			} else {
				limitType = "monthly"
			}

			c.JSON(429, gin.H{
				"error": gin.H{
					"code":    "rate_limit_exceeded",
					"message": fmt.Sprintf("%s rate limit exceeded", limitType),
					"details": gin.H{
						"limit_type": limitType,
						"retry_after_seconds": int64(rateLimitInfo.RetryAfter.Seconds()),
					},
				},
			})
			c.Abort()
			return
		}

		// Increment usage counter
		if err := rl.IncrementUsage(c.Request.Context(), user.ID); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to increment usage counter: %v\n", err)
		}

		c.Next()
	}
}

// shouldRateLimit determines if a path should be rate limited
func shouldRateLimit(path string) bool {
	rateLimitedPaths := []string{
		"/v1/recommend",
		"/v1/generate",
		"/v1/models",
	}

	for _, p := range rateLimitedPaths {
		if path == p {
			return true
		}
	}
	return false
}

// maxInt returns the maximum of two integers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Close closes the Redis connection
func (rl *RateLimiter) Close() error {
	return rl.client.Close()
}