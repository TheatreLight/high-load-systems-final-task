package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/time/rate"

	"high-load-service/cache"
	"high-load-service/handlers"
	"high-load-service/metrics"
	"high-load-service/services"
	"high-load-service/utils"
)

func main() {
	log.Println("Starting High-Load IoT Metrics Service...")

	// Initialize Redis client
	var redisClient *cache.RedisClient
	var err error

	redisClient, err = cache.NewRedisClient()
	if err != nil {
		log.Printf("Warning: Redis not available, running without cache: %v", err)
		redisClient = nil
	}

	// Anomaly callback for Prometheus metrics
	onAnomaly := func(metricType string) {
		metrics.RecordAnomaly(metricType)
	}

	// Initialize services
	metricsService := services.NewMetricsService(redisClient, onAnomaly)

	// Initialize handlers
	metricsHandler := handlers.NewMetricsHandler(metricsService)

	// Create router
	r := mux.NewRouter()

	// API routes - metrics ingestion
	r.HandleFunc("/ingest", metricsHandler.IngestMetric).Methods("POST")
	r.HandleFunc("/ingest/batch", metricsHandler.IngestMetricBatch).Methods("POST")

	// Analytics endpoints
	r.HandleFunc("/analyze", metricsHandler.GetAnalytics).Methods("GET")
	r.HandleFunc("/anomalies", metricsHandler.GetAnomalies).Methods("GET")
	r.HandleFunc("/stats", metricsHandler.GetStats).Methods("GET")

	// Health check
	r.HandleFunc("/health", healthCheck(redisClient)).Methods("GET")

	// Prometheus metrics endpoint
	r.Handle("/metrics", metrics.MetricsHandler()).Methods("GET")

	// Apply middlewares
	// Rate limiter: 2000 req/s with burst 50000 for stable work under high load
	rateLimiter := utils.NewRateLimiter(rate.Limit(2000), 50000)
	rateLimitMiddleware := utils.RateLimitMiddleware(rateLimiter)

	// Wrap handler with middlewares (order: rate limit first, then metrics)
	var handler http.Handler = r
	handler = rateLimitMiddleware(handler)
	handler = metrics.MetricsMiddleware(handler)

	// Configure HTTP server for high performance
	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:           ":" + port,
		Handler:        handler,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Log startup info
	log.Printf("Server starting on port %s", port)
	log.Printf("Endpoints:")
	log.Printf("  - POST   /ingest           (ingest single metric)")
	log.Printf("  - POST   /ingest/batch     (ingest batch of metrics)")
	log.Printf("  - GET    /analyze          (get analytics results)")
	log.Printf("  - GET    /anomalies        (get anomaly statistics)")
	log.Printf("  - GET    /stats            (get service statistics)")
	log.Printf("  - GET    /health           (health check)")
	log.Printf("  - GET    /metrics          (Prometheus metrics)")

	// Graceful shutdown handling
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down gracefully...")
		metricsService.Stop()
		if redisClient != nil {
			redisClient.Close()
		}
		os.Exit(0)
	}()

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// healthCheck returns a health check handler
func healthCheck(redis *cache.RedisClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		redisStatus := "not configured"

		if redis != nil {
			if err := redis.HealthCheck(); err != nil {
				redisStatus = "unhealthy"
				status = "degraded"
			} else {
				redisStatus = "healthy"
			}
		}

		response := map[string]interface{}{
			"service": "high-load-iot-service",
			"status":  status,
			"redis":   redisStatus,
			"version": "1.0.0",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// getEnv returns environment variable value or default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
