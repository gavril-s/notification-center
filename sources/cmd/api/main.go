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
	"notification-center/sources/internal/config"
	"notification-center/sources/internal/handler"
	"notification-center/sources/internal/repository"
	"notification-center/sources/internal/service"
)

func main() {
	cfg := config.Load()

	// Connect to database
	pool, err := connectDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Printf("warning: failed to connect to database: %v (running in stub mode)", err)
	}

	// Initialize layers
	var h *handler.Handler
	if pool != nil {
		repo := repository.New(pool)
		svc := service.New(repo)
		h = handler.New(svc)
	} else {
		// Create minimal handler when no database is available
		h = &handler.Handler{}
	}

	// Setup router
	r := gin.Default()
	h.RegisterRoutes(r)

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("starting sources service on :%s", cfg.Port)
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
		log.Printf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}

func connectDatabase(databaseURL string) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, nil
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}

	log.Println("connected to database")
	return pool, nil
}
