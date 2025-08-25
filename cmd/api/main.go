package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dustin/articles-backend/config"
	"github.com/dustin/articles-backend/internal/adapter"
	"github.com/dustin/articles-backend/internal/article"
	"github.com/dustin/articles-backend/internal/classifier"
	"github.com/dustin/articles-backend/internal/embedding"
	"github.com/dustin/articles-backend/internal/rating"
	"github.com/dustin/articles-backend/internal/recommendation"
	"github.com/dustin/articles-backend/internal/repository"
	"github.com/dustin/articles-backend/internal/user"
	"github.com/dustin/articles-backend/internal/worker"
	"github.com/dustin/articles-backend/pkg/database"
	"github.com/dustin/articles-backend/pkg/logger"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func main() {
	// Load configuration from environment variables
	cfg := config.Load()

	// Initialize logger with validation and defaults
	appLogger, err := logger.NewLogger(&cfg.Logging)
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	appLogger.Info("Starting articles backend service")

	// Connect to database with validation and defaults
	db, err := database.NewConnection(&cfg.Database)
	if err != nil {
		appLogger.Fatal("Failed to connect to database: " + err.Error())
	}

	appLogger.Info("Database connection established")

	// Run database migrations for all feature models
	if err := db.AutoMigrate(&user.User{}, &article.Article{}, &rating.Rating{}); err != nil {
		appLogger.Fatal("Failed to migrate database: " + err.Error())
	}

	appLogger.Info("Database migration completed")

	// Initialize GORM-based repositories
	userRepo := repository.NewGORMUserRepository(db, appLogger)
	articleRepo := repository.NewGORMArticleRepository(db, appLogger)
	ratingRepo := repository.NewGORMRatingRepository(db, appLogger)

	// Initialize recommendation-specific repositories
	recArticleRepo := repository.NewGORMRecommendationArticleRepository(db, appLogger)
	recRatingRepo := repository.NewGORMRecommendationRatingRepository(db, appLogger)

	// Initialize embedding client
	embeddingServiceURL := os.Getenv("EMBEDDING_SERVICE_URL")
	if embeddingServiceURL == "" {
		embeddingServiceURL = "http://localhost:8001"
	}
	embeddingClient := embedding.NewClient(embeddingServiceURL)
	appLogger.Info("Embedding client initialized with URL: " + embeddingServiceURL)

	// Initialize content classifier with validation and defaults
	metadataClassifier, err := classifier.NewReadabilityClassifier(&cfg.Classifier, embeddingClient, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to initialize classifier: " + err.Error())
	}

	// Create adapter to bridge interface compatibility
	metadataExtractor := adapter.NewClassifierToMetadataExtractor(metadataClassifier)

	// Initialize business services with dependency injection
	userService, err := user.NewService(&cfg.JWT, userRepo, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to initialize user service: " + err.Error())
	}
	articleService := article.NewService(articleRepo, metadataExtractor, appLogger)

	// Create service adapter for rating dependencies
	ratingArticleService := adapter.NewArticleServiceToRatingArticleService(articleService)
	ratingService := rating.NewService(ratingRepo, ratingArticleService, appLogger)
	recommendationService := recommendation.NewService(recArticleRepo, recRatingRepo, embeddingClient, appLogger)

	// Initialize HTTP handlers
	userHandler := user.NewHandler(userService)
	articleHandler := article.NewHandler(articleService)
	ratingHandler := rating.NewHandler(ratingService)
	recommendationHandler := recommendation.NewHandler(recommendationService)

	// Initialize background worker for metadata retries
	metadataRetryWorker, err := worker.NewRetryWorker(
		&cfg.Worker,
		"metadata-retry",
		articleService.RetryFailedMetadata,
		appLogger,
	)
	if err != nil {
		appLogger.Fatal("Failed to initialize retry worker: " + err.Error())
	}

	// Start background processing
	if err := metadataRetryWorker.Start(); err != nil {
		appLogger.Error("Failed to start metadata retry worker: " + err.Error())
	}

	// Setup HTTP router with middleware
	router := gin.New()

	// Configure standard middleware stack
	router.Use(requestid.New())
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"*"},
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders: []string{"X-Request-ID"},
	}))

	// Health check endpoints
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now(),
			"service":   "articles-backend",
		})
	})

	router.GET("/health/detailed", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":       "healthy",
			"timestamp":    time.Now(),
			"service":      "articles-backend",
			"retry_worker": metadataRetryWorker.IsRunning(),
			"database":     "connected",
			"classifier":   metadataClassifier.IsHealthy(),
		})
	})

	// Create simple JWT validation middleware
	jwtSecret := cfg.JWT.Secret
	if jwtSecret == "" {
		jwtSecret = "change-me-in-production" // default
	}
	authMiddleware := createJWTMiddleware(jwtSecret)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Register feature routes - each feature manages its own routes
		userHandler.RegisterRoutes(v1, authMiddleware)
		articleHandler.RegisterRoutes(v1, authMiddleware)
		ratingHandler.RegisterRoutes(v1, authMiddleware)
		recommendationHandler.RegisterRoutes(v1, authMiddleware)
	}

	// Legacy compatibility routes (can be removed later)
	legacyRoutes := router.Group("/")
	{
		// Auth routes (public)
		legacyRoutes.POST("/signup", userHandler.SignUp)
		legacyRoutes.POST("/login", userHandler.Login)

		// Protected routes with auth middleware
		protected := legacyRoutes.Group("/")
		protected.Use(authMiddleware)
		{
			protected.GET("/me", userHandler.GetMe)

			// Articles
			protected.POST("/articles", articleHandler.CreateArticle)
			protected.GET("/articles", articleHandler.GetArticles)
			protected.DELETE("/articles/:id", articleHandler.DeleteArticle)

			// Ratings - using simplified path as per requirements
			protected.POST("/articles/:id/rate", ratingHandler.RateArticle)
			protected.GET("/articles/:id/rate", ratingHandler.GetRating)
			protected.DELETE("/articles/:id/rate", ratingHandler.DeleteRating)

			// Recommendations
			protected.GET("/recommendations", recommendationHandler.GetRecommendations)
		}
	}

	// Parse server configuration with defaults
	serverPort := cfg.Server.Port
	if serverPort == "" {
		serverPort = "8080" // default
	}

	serverReadTimeout := 30 * time.Second // default
	if cfg.Server.ReadTimeout != "" {
		if duration, err := time.ParseDuration(cfg.Server.ReadTimeout); err == nil {
			serverReadTimeout = duration
		}
	}

	serverWriteTimeout := 30 * time.Second // default
	if cfg.Server.WriteTimeout != "" {
		if duration, err := time.ParseDuration(cfg.Server.WriteTimeout); err == nil {
			serverWriteTimeout = duration
		}
	}

	serverEnvironment := cfg.Server.Environment
	if serverEnvironment == "" {
		serverEnvironment = "development" // default
	}

	// Start HTTP server
	srv := &http.Server{
		Addr:         ":" + serverPort,
		Handler:      router,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
	}

	// Start server in goroutine for graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("Failed to start server: " + err.Error())
		}
	}()

	appLogger.Info("Server started successfully on port " + serverPort + " (" + serverEnvironment + " environment)")

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	// Stop retry worker first
	if err := metadataRetryWorker.Stop(); err != nil {
		appLogger.Error("Error stopping retry worker: " + err.Error())
	}

	// Shutdown server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Fatal("Server forced to shutdown: " + err.Error())
	}

	appLogger.Info("Server shutdown complete")
}

// loadConfig is no longer used - configuration is now loaded directly as raw strings
// and each package handles its own defaults and validation

// createJWTMiddleware creates a simple JWT validation middleware
func createJWTMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Next()
	}
}
