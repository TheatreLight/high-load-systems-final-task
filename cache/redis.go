package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"

	"high-load-service/models"
)

const (
	MetricsListKey    = "metrics:list"
	MetricsCounterKey = "metrics:counter"
	DefaultTTL        = 24 * time.Hour
	MaxMetricsStored  = 10000
)

// RedisClient wraps the Redis client for metrics caching
type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisClient creates a new Redis client
func NewRedisClient() (*RedisClient, error) {
	host := getEnv("REDIS_HOST", "localhost")
	port := getEnv("REDIS_PORT", "6379")
	password := getEnv("REDIS_PASSWORD", "")

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", host, port),
		Password:     password,
		DB:           0,
		PoolSize:     100,
		MinIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx := context.Background()

	// Test connection
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("Connected to Redis at %s:%s", host, port)

	return &RedisClient{
		client: client,
		ctx:    ctx,
	}, nil
}

// StoreMetric stores a metric in Redis
func (rc *RedisClient) StoreMetric(metric models.Metric) error {
	data, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	// Push to list (newest first)
	err = rc.client.LPush(rc.ctx, MetricsListKey, data).Err()
	if err != nil {
		return fmt.Errorf("failed to store metric: %w", err)
	}

	// Trim list to max size
	rc.client.LTrim(rc.ctx, MetricsListKey, 0, MaxMetricsStored-1)

	// Increment counter
	rc.client.Incr(rc.ctx, MetricsCounterKey)

	return nil
}

// GetRecentMetrics retrieves the most recent N metrics
func (rc *RedisClient) GetRecentMetrics(count int64) ([]models.Metric, error) {
	data, err := rc.client.LRange(rc.ctx, MetricsListKey, 0, count-1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	metrics := make([]models.Metric, 0, len(data))
	for _, d := range data {
		var metric models.Metric
		if err := json.Unmarshal([]byte(d), &metric); err != nil {
			continue // Skip invalid entries
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// GetMetricsCount returns the total number of metrics received
func (rc *RedisClient) GetMetricsCount() (int64, error) {
	count, err := rc.client.Get(rc.ctx, MetricsCounterKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get metrics count: %w", err)
	}
	return count, nil
}

// GetStoredMetricsCount returns the number of metrics in the list
func (rc *RedisClient) GetStoredMetricsCount() (int64, error) {
	count, err := rc.client.LLen(rc.ctx, MetricsListKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get stored metrics count: %w", err)
	}
	return count, nil
}

// StoreAnalyticsResult caches the latest analytics result
func (rc *RedisClient) StoreAnalyticsResult(result models.AnalyticsResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal analytics result: %w", err)
	}

	err = rc.client.Set(rc.ctx, "analytics:latest", data, DefaultTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to store analytics result: %w", err)
	}

	return nil
}

// GetLatestAnalyticsResult retrieves the cached analytics result
func (rc *RedisClient) GetLatestAnalyticsResult() (*models.AnalyticsResult, error) {
	data, err := rc.client.Get(rc.ctx, "analytics:latest").Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get analytics result: %w", err)
	}

	var result models.AnalyticsResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal analytics result: %w", err)
	}

	return &result, nil
}

// IncrementAnomalyCount increments the anomaly counter
func (rc *RedisClient) IncrementAnomalyCount(metricType string) error {
	key := fmt.Sprintf("anomaly:count:%s", metricType)
	return rc.client.Incr(rc.ctx, key).Err()
}

// GetAnomalyCount returns the anomaly count for a metric type
func (rc *RedisClient) GetAnomalyCount(metricType string) (int64, error) {
	key := fmt.Sprintf("anomaly:count:%s", metricType)
	count, err := rc.client.Get(rc.ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

// HealthCheck checks Redis connectivity
func (rc *RedisClient) HealthCheck() error {
	_, err := rc.client.Ping(rc.ctx).Result()
	return err
}

// Close closes the Redis connection
func (rc *RedisClient) Close() error {
	return rc.client.Close()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
