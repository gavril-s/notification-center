package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"notification-center/recipients/internal/config"
	"notification-center/recipients/internal/handler"
	"notification-center/recipients/internal/middleware"
	"notification-center/recipients/internal/repository"
	"notification-center/recipients/internal/service"
)

func main() {
	// Load configuration
	cfg := config.Load()

	ctx := context.Background()

	// Connect to database
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Run migrations if enabled
	if os.Getenv("RUN_MIGRATIONS") == "true" {
		if err := runMigrations(cfg); err != nil {
			log.Fatalf("failed to run migrations: %v", err)
		}
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(pool)
	contactRepo := repository.NewContactRepository(pool)
	prefRepo := repository.NewPreferenceRepository(pool)
	tokenRepo := repository.NewRefreshTokenRepository(pool)
	unsubscribeTokenRepo := repository.NewUnsubscribeTokenRepository(pool)
	unsubscribeRuleRepo := repository.NewUnsubscribeRuleRepository(pool)
	auditRepo := repository.NewAuditLogRepository(pool)

	// Initialize services
	authService := service.NewAuthService(userRepo, tokenRepo, auditRepo, cfg)
	contactService := service.NewContactService(contactRepo, auditRepo)
	prefService := service.NewPreferenceService(prefRepo, contactRepo, auditRepo)
	unsubscribeService := service.NewUnsubscribeService(unsubscribeTokenRepo, unsubscribeRuleRepo, contactRepo, auditRepo)
	internalResolveService := service.NewInternalResolveService(contactRepo, prefService, unsubscribeRuleRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	contactHandler := handler.NewContactHandler(contactService)
	prefHandler := handler.NewPreferenceHandler(prefService)
	unsubscribeHandler := handler.NewUnsubscribeHandler(unsubscribeService)
	internalHandler := handler.NewInternalHandler(internalResolveService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	// Setup router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Public API routes
	api := router.Group("/api/recipients")
	{
		api.POST("/register", authHandler.Register)
		api.POST("/login", authHandler.Login)
		api.POST("/refresh", authHandler.Refresh)
		api.POST("/unsubscribe", unsubscribeHandler.Unsubscribe)
	}

	// Protected API routes
	protected := router.Group("/api/recipients")
	protected.Use(authMiddleware.Authenticate())
	{
		protected.GET("/me", authHandler.Me)
		protected.GET("/contacts", contactHandler.List)
		protected.POST("/contacts", contactHandler.Create)
		protected.PUT("/contacts/:contact_id", contactHandler.Update)
		protected.DELETE("/contacts/:contact_id", contactHandler.Delete)
		protected.GET("/preferences", prefHandler.List)
		protected.PUT("/preferences", prefHandler.Update)
	}

	// Internal routes (for Track 4)
	internal := router.Group("/internal/recipients")
	{
		internal.POST("/resolve", internalHandler.Resolve)
		internal.GET("/users/:user_id/contacts", internalHandler.GetUserContacts)
	}

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf("Recipients service starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func runMigrations(cfg *config.Config) error {
	log.Println("Running migrations...")
	// Migrations are handled via init script in docker-compose
	// This is a placeholder for future implementation
	return nil
}
