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

	"notification-center/notifications-service/internal/config"
	"notification-center/notifications-service/internal/domain"
	"notification-center/notifications-service/internal/handler"
	"notification-center/notifications-service/internal/middleware"
	"notification-center/notifications-service/internal/repository"
	"notification-center/notifications-service/internal/service"
)

func main() {
	cfg := config.Load()

	// Database connection
	dbPool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Verify database connection
	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	// Initialize repositories
	notificationRepo := repository.NewNotificationRepository(dbPool)
	outboxRepo := repository.NewOutboxRepository(dbPool)
	historyRepo := repository.NewHistoryRepository(dbPool)
	analyticsRepo := repository.NewAnalyticsRepository(dbPool)

	// Initialize service
	notificationSvc := service.NewNotificationService(
		notificationRepo,
		outboxRepo,
		historyRepo,
		analyticsRepo,
		cfg.RecipientsURL,
		cfg.SourcesURL,
		cfg.UnsubscribeSecret,
	)

	// Initialize handler
	notificationHandler := handler.NewNotificationHandler(notificationSvc)

	// Setup Gin router
	router := gin.Default()
	router.Use(middleware.RequestID())
	router.Use(middleware.TraceID())

	// Public API routes
	api := router.Group("/api")
	notificationHandler.RegisterRoutes(api)

	// Internal API routes
	internal := router.Group("/internal")
	internal.POST("/notifications/:notification_id/delivery-status", handleDeliveryStatus(notificationSvc))
	internal.POST("/notifications/scheduler/run", handleSchedulerRun(notificationSvc))
	internal.GET("/notifications/:notification_id", handleInternalGet(notificationSvc))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "component": cfg.ServiceName})
	})
	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf("starting %s on port %s", cfg.ServiceName, cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}

type DeliveryStatusRequest struct {
	AttemptID       string `json:"attempt_id" binding:"required"`
	TransportStatus string `json:"transport_status" binding:"required"`
	ProviderCode    string `json:"provider_code" binding:"required"`
	ErrorCode       string `json:"error_code,omitempty"`
	OccurredAt      string `json:"occurred_at" binding:"required"`
}

func handleDeliveryStatus(svc *service.NotificationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		notificationID := c.Param("notification_id")

		var req DeliveryStatusRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		var status domain.NotificationStatus
		switch req.TransportStatus {
		case "accepted", "delivered":
			status = domain.StatusDelivered
		case "failed":
			status = domain.StatusFailed
		default:
			status = domain.StatusSent
		}

		err := svc.UpdateDeliveryStatus(c.Request.Context(), notificationID, status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func handleSchedulerRun(svc *service.NotificationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// This would trigger processing of scheduled notifications
		// For now, just return OK
		c.JSON(http.StatusOK, gin.H{"status": "scheduler triggered"})
	}
}

func handleInternalGet(svc *service.NotificationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		notificationID := c.Param("notification_id")

		notification, err := svc.GetByID(c.Request.Context(), notificationID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"notification": notification})
	}
}
