package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"github.com/Askeban/llm-router-go/internal/auth"
	httpHandlers "github.com/Askeban/llm-router-go/internal/http"
	"github.com/Askeban/llm-router-go/internal/services"
)

var (
	db            *sql.DB
	routerService *services.EnhancedRouterService
	authHandlers  *auth.Handlers
)

func main() {
	log.Println("[ROUTER] Starting RouteLLM Production Server v1.0")

	// Initialize database connection
	if err := initDatabase(); err != nil {
		log.Fatalf("[ROUTER] Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize enhanced router service
	if err := initRouterService(); err != nil {
		log.Fatalf("[ROUTER] Failed to initialize router service: %v", err)
	}

	// Initialize auth handlers
	if err := initAuthHandlers(); err != nil {
		log.Fatalf("[ROUTER] Failed to initialize auth handlers: %v", err)
	}

	// Setup Gin router
	r := setupRouter()

	// Start server with graceful shutdown
	startServer(r)
}

func initDatabase() error {
	// Get database connection string from environment
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	instanceConnectionName := os.Getenv("INSTANCE_CONNECTION_NAME")

	// Default values
	if dbUser == "" {
		dbUser = "postgres"
	}
	if dbName == "" {
		dbName = "routellm"
	}

	var dsn string
	if instanceConnectionName != "" {
		// Cloud SQL connection via Unix socket
		dsn = fmt.Sprintf("host=/cloudsql/%s user=%s password=%s dbname=%s sslmode=disable",
			instanceConnectionName, dbUser, dbPassword, dbName)
	} else if dbHost != "" {
		// Direct connection
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=require",
			dbHost, dbUser, dbPassword, dbName)
	} else {
		return fmt.Errorf("no database configuration found")
	}

	log.Printf("[DATABASE] Connecting to PostgreSQL database: %s", dbName)

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("[DATABASE] Successfully connected to PostgreSQL")

	// Apply schema if needed
	if err := applySchema(); err != nil {
		return fmt.Errorf("failed to apply schema: %w", err)
	}

	return nil
}

func applySchema() error {
	// Check if schema is already applied
	var exists bool
	err := db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users')").Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check schema: %w", err)
	}

	if exists {
		log.Println("[DATABASE] Schema already exists")
		return nil
	}

	log.Println("[DATABASE] Applying database schema...")

	// Read and execute schema file
	schemaPath := os.Getenv("SCHEMA_PATH")
	if schemaPath == "" {
		schemaPath = "./database/schema_postgres.sql"
	}

	schemaSQL, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	if _, err := db.Exec(string(schemaSQL)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	log.Println("[DATABASE] Schema applied successfully")
	return nil
}

func initRouterService() error {
	modelPath := os.Getenv("MODEL_PATH")
	if modelPath == "" {
		modelPath = "./configs/model_1.json"
	}

	log.Printf("[ROUTER] Initializing model service with path: %s", modelPath)

	var err error
	routerService, err = services.NewEnhancedRouterService(modelPath)
	if err != nil {
		return fmt.Errorf("failed to initialize router service: %w", err)
	}

	stats := routerService.GetStats()
	log.Printf("[ROUTER] Service initialized:")
	log.Printf("  - Total models: %v", stats["total_models"])
	log.Printf("  - Data sources: %v", stats["data_sources"])

	return nil
}

func initAuthHandlers() error {
	log.Println("[AUTH] Initializing authentication handlers...")

	// Get JWT secret from environment or use default
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "kIQuPaMIDulFsCJmB6iolLF0yhE5pCnN" // Default from GCloud secret
	}

	// Create JWT manager
	jwtManager := auth.NewJWTManager(jwtSecret, 24*time.Hour)

	// Create auth service
	authService := auth.NewService(db)

	// Create auth handlers
	authHandlers = auth.NewHandlers(authService, jwtManager)

	log.Println("[AUTH] Authentication handlers initialized")
	return nil
}

func setupRouter() *gin.Engine {
	// Set Gin mode
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	// Health check endpoint
	r.GET("/health", healthCheck)
	r.GET("/healthz", healthCheck)

	// Root endpoint
	r.GET("/", rootHandler)

	// Setup enhanced handlers (model recommendations)
	enhancedHandlers := httpHandlers.NewEnhancedHandlers(routerService)
	enhancedHandlers.SetupEnhancedRoutes(r)

	// Setup authentication handlers
	setupAuthRoutes(r)

	return r
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func healthCheck(c *gin.Context) {
	// Check database connection
	dbStatus := "healthy"
	if err := db.Ping(); err != nil {
		dbStatus = "unhealthy: " + err.Error()
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "healthy",
		"service":    "llm-router-go",
		"version":    "4.0.0",
		"timestamp":  time.Now().Format(time.RFC3339),
		"database":   dbStatus,
		"models":     routerService.GetStats()["total_models"],
		"domain":     "routellm.dev",
		"categories": []string{"text", "image-generation", "video-generation", "voice-generation", "multimodal"},
		"sources":    []string{"HuggingFace", "Commercial APIs", "Latest 2025 Models", "Analytics"},
		"analytics":  "enabled",
		"multimodal": "enabled",
	})
}

func rootHandler(c *gin.Context) {
	stats := routerService.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"service":     "RouteLLM - AI Model Router",
		"version":     "1.0",
		"description": "Production-ready LLM routing with authentication",
		"features": []string{
			"Smart model recommendations",
			"User authentication & API keys",
			"Rate limiting & usage tracking",
			"Multi-modal support",
			"Analytics integration",
			"GitHub OAuth",
		},
		"stats": stats,
		"endpoints": gin.H{
			"auth_signup":           "POST /api/v1/auth/signup",
			"auth_login":            "POST /api/v1/auth/login",
			"auth_me":               "GET /api/v1/auth/me",
			"waitlist":              "POST /api/v1/auth/waitlist",
			"smart_recommendations": "POST /api/v2/recommend/smart",
			"direct_recommendations":"POST /api/v2/recommend/direct",
			"models":                "GET /api/v2/models",
			"health":                "GET /health",
		},
	})
}

func setupAuthRoutes(r *gin.Engine) {
	authGroup := r.Group("/api/v1/auth")
	{
		// Public endpoints
		authGroup.POST("/signup", authHandlers.Register)
		authGroup.POST("/login", authHandlers.Login)
		authGroup.POST("/waitlist", authHandlers.Waitlist)
		authGroup.POST("/oauth/github", authHandlers.GitHubOAuth)
		authGroup.POST("/refresh", authHandlers.RefreshToken)

		// Protected endpoints (require JWT)
		protected := authGroup.Group("")
		protected.Use(authHandlers.AuthMiddleware())
		{
			protected.GET("/me", authHandlers.GetProfile)
			protected.POST("/logout", authHandlers.Logout)
			protected.GET("/usage", authHandlers.GetUsage)
			protected.GET("/api-keys", authHandlers.ListAPIKeys)
			protected.POST("/api-keys", authHandlers.CreateAPIKey)
		}
	}
}

func startServer(r *gin.Engine) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("[SERVER] Starting on port %s", port)
		log.Println("[SERVER] Endpoints:")
		log.Println("  Auth:   POST /api/v1/auth/signup, /login, /waitlist")
		log.Println("  Router: POST /api/v2/recommend/smart")
		log.Println("  Models: GET /api/v2/models")
		log.Println("  Health: GET /health")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[SERVER] Failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[SERVER] Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[SERVER] Forced shutdown: %v", err)
	}

	log.Println("[SERVER] Exited gracefully")
}
