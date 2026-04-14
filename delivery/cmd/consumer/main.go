package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"

	"notification-center/delivery/internal/config"
	"notification-center/delivery/internal/repository"
	"notification-center/delivery/internal/service"
)

type response struct {
	Status    string `json:"status"`
	Component string `json:"component"`
	Message   string `json:"message,omitempty"`
}

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
	deliveryRepo := repository.NewDeliveryRepository(dbPool)
	deadLetterRepo := repository.NewDeadLetterRepository(dbPool)

	// Initialize services
	provider := service.NewMockProvider()
	consumerSvc := service.NewConsumerService(deliveryRepo, deadLetterRepo, provider)

	// Connect to RabbitMQ
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open channel: %v", err)
	}
	defer ch.Close()

	// Declare queue
	q, err := ch.QueueDeclare(
		"notification.dispatch.v1",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("failed to declare queue: %v", err)
	}

	// Set QoS
	err = ch.Qos(1, 0, false)
	if err != nil {
		log.Fatalf("failed to set QoS: %v", err)
	}

	// Start consuming
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("failed to register consumer: %v", err)
	}

	// HTTP server for health checks
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, http.StatusOK, response{Status: "ok", Component: cfg.ServiceName})
	})
	mux.HandleFunc("/ready", func(writer http.ResponseWriter, request *http.Request) {
		writeJSON(writer, http.StatusOK, response{Status: "ready", Component: cfg.ServiceName})
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("starting %s HTTP server on port %s", cfg.ServiceName, cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Start worker goroutines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Process messages
	log.Printf("starting delivery consumer, waiting for messages...")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				if err := consumerSvc.Consume(ctx, msg); err != nil {
					log.Printf("error processing message: %v", err)
				}
			}
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down consumer...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("failed to shutdown server: %v", err)
	}

	log.Println("consumer exited")
}

func writeJSON(writer http.ResponseWriter, statusCode int, payload response) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	if err := json.NewEncoder(writer).Encode(payload); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}
