package services

import (
	"log"
	"sync"
	"time"

	"high-load-service/analytics"
	"high-load-service/cache"
	"high-load-service/models"
)

const (
	WindowSize      = 50
	ZScoreThreshold = 2.0
	ChannelBuffer   = 1000
)

// MetricsService handles metrics processing with analytics
type MetricsService struct {
	redis *cache.RedisClient

	// Rolling averages for smoothing
	cpuRolling *analytics.RollingAverage
	rpsRolling *analytics.RollingAverage

	// Z-score detectors for anomaly detection
	cpuZScore *analytics.ZScoreDetector
	rpsZScore *analytics.ZScoreDetector

	// Channels for async processing
	metricsChan  chan models.Metric
	anomalyChan  chan models.AnomalyEvent
	stopChan     chan struct{}

	// Latest values
	latestMetric models.Metric
	latestMu     sync.RWMutex

	// Anomaly counters
	cpuAnomalyCount int64
	rpsAnomalyCount int64
	anomalyMu       sync.RWMutex

	// Total metrics counter
	totalMetrics int64
	totalMu      sync.RWMutex

	// Anomaly callback for Prometheus metrics
	onAnomaly func(metricType string)
}

// NewMetricsService creates a new metrics service
func NewMetricsService(redisClient *cache.RedisClient, onAnomaly func(string)) *MetricsService {
	ms := &MetricsService{
		redis:       redisClient,
		cpuRolling:  analytics.NewRollingAverage(WindowSize),
		rpsRolling:  analytics.NewRollingAverage(WindowSize),
		cpuZScore:   analytics.NewZScoreDetector(WindowSize, ZScoreThreshold),
		rpsZScore:   analytics.NewZScoreDetector(WindowSize, ZScoreThreshold),
		metricsChan: make(chan models.Metric, ChannelBuffer),
		anomalyChan: make(chan models.AnomalyEvent, ChannelBuffer),
		stopChan:    make(chan struct{}),
		onAnomaly:   onAnomaly,
	}

	// Start background workers
	go ms.processMetrics()
	go ms.processAnomalies()

	return ms
}

// ProcessMetric processes an incoming metric
func (ms *MetricsService) ProcessMetric(metric models.Metric) error {
	// Update latest metric
	ms.latestMu.Lock()
	ms.latestMetric = metric
	ms.latestMu.Unlock()

	// Increment total counter
	ms.totalMu.Lock()
	ms.totalMetrics++
	ms.totalMu.Unlock()

	// Send to channel for async processing
	select {
	case ms.metricsChan <- metric:
	default:
		// Channel full, process synchronously
		ms.processMetricSync(metric)
	}

	// Store in Redis if available
	if ms.redis != nil {
		if err := ms.redis.StoreMetric(metric); err != nil {
			log.Printf("Warning: failed to store metric in Redis: %v", err)
		}
	}

	return nil
}

// processMetrics runs the background metric processor
func (ms *MetricsService) processMetrics() {
	for {
		select {
		case metric := <-ms.metricsChan:
			ms.processMetricSync(metric)
		case <-ms.stopChan:
			return
		}
	}
}

// processMetricSync processes a metric synchronously
func (ms *MetricsService) processMetricSync(metric models.Metric) {
	// Update rolling averages
	ms.cpuRolling.Add(metric.CPU)
	ms.rpsRolling.Add(metric.RPS)

	// Check for anomalies
	cpuAnomaly, cpuZScore := ms.cpuZScore.Add(metric.CPU)
	rpsAnomaly, rpsZScore := ms.rpsZScore.Add(metric.RPS)

	if cpuAnomaly {
		ms.anomalyMu.Lock()
		ms.cpuAnomalyCount++
		ms.anomalyMu.Unlock()

		if ms.onAnomaly != nil {
			ms.onAnomaly("cpu")
		}

		mean, stddev := ms.cpuZScore.GetStats()
		event := models.AnomalyEvent{
			Timestamp:  metric.Timestamp,
			MetricType: "cpu",
			Value:      metric.CPU,
			ZScore:     cpuZScore,
			Mean:       mean,
			StdDev:     stddev,
		}
		select {
		case ms.anomalyChan <- event:
		default:
		}
	}

	if rpsAnomaly {
		ms.anomalyMu.Lock()
		ms.rpsAnomalyCount++
		ms.anomalyMu.Unlock()

		if ms.onAnomaly != nil {
			ms.onAnomaly("rps")
		}

		mean, stddev := ms.rpsZScore.GetStats()
		event := models.AnomalyEvent{
			Timestamp:  metric.Timestamp,
			MetricType: "rps",
			Value:      metric.RPS,
			ZScore:     rpsZScore,
			Mean:       mean,
			StdDev:     stddev,
		}
		select {
		case ms.anomalyChan <- event:
		default:
		}
	}
}

// processAnomalies handles detected anomalies
func (ms *MetricsService) processAnomalies() {
	for {
		select {
		case event := <-ms.anomalyChan:
			log.Printf("ANOMALY DETECTED: type=%s value=%.2f zscore=%.2f mean=%.2f stddev=%.2f",
				event.MetricType, event.Value, event.ZScore, event.Mean, event.StdDev)

			// Store in Redis if available
			if ms.redis != nil {
				ms.redis.IncrementAnomalyCount(event.MetricType)
			}
		case <-ms.stopChan:
			return
		}
	}
}

// GetAnalytics returns current analytics results
func (ms *MetricsService) GetAnalytics() models.AnalyticsResult {
	ms.latestMu.RLock()
	latest := ms.latestMetric
	ms.latestMu.RUnlock()

	ms.totalMu.RLock()
	total := ms.totalMetrics
	ms.totalMu.RUnlock()

	// Check if current values are anomalies
	cpuAnomaly, cpuZScore := ms.cpuZScore.IsAnomaly(latest.CPU)
	rpsAnomaly, rpsZScore := ms.rpsZScore.IsAnomaly(latest.RPS)

	return models.AnalyticsResult{
		CurrentCPU:   latest.CPU,
		CurrentRPS:   latest.RPS,
		AvgCPU:       ms.cpuRolling.GetAverage(),
		AvgRPS:       ms.rpsRolling.GetAverage(),
		PredictedCPU: ms.cpuRolling.GetPrediction(),
		PredictedRPS: ms.rpsRolling.GetPrediction(),
		CPUZScore:    cpuZScore,
		RPSZScore:    rpsZScore,
		CPUAnomaly:   cpuAnomaly,
		RPSAnomaly:   rpsAnomaly,
		TotalMetrics: int(total),
		WindowSize:   WindowSize,
		LastUpdated:  time.Now(),
	}
}

// GetAnomalyCounts returns anomaly counters
func (ms *MetricsService) GetAnomalyCounts() (cpu, rps int64) {
	ms.anomalyMu.RLock()
	defer ms.anomalyMu.RUnlock()
	return ms.cpuAnomalyCount, ms.rpsAnomalyCount
}

// GetTotalMetrics returns total metrics processed
func (ms *MetricsService) GetTotalMetrics() int64 {
	ms.totalMu.RLock()
	defer ms.totalMu.RUnlock()
	return ms.totalMetrics
}

// Stop gracefully stops the service
func (ms *MetricsService) Stop() {
	close(ms.stopChan)
}
