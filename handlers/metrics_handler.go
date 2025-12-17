package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"high-load-service/models"
	"high-load-service/services"
	"high-load-service/utils"
)

// MetricsHandler handles HTTP requests for metrics operations
type MetricsHandler struct {
	service *services.MetricsService
}

// NewMetricsHandler creates a new MetricsHandler
func NewMetricsHandler(service *services.MetricsService) *MetricsHandler {
	return &MetricsHandler{service: service}
}

// IngestMetric handles POST /metrics - accepts incoming metric data
func (h *MetricsHandler) IngestMetric(w http.ResponseWriter, r *http.Request) {
	var input models.MetricInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if err := input.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Convert to Metric
	metric, err := input.ToMetric()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Process the metric
	if err := h.service.ProcessMetric(metric); err != nil {
		go utils.HandleError(err, "IngestMetric: processing metric")
		http.Error(w, "Failed to process metric", http.StatusInternalServerError)
		return
	}

	// Return success with basic info
	response := map[string]interface{}{
		"status":    "accepted",
		"timestamp": metric.Timestamp,
		"processed": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// IngestMetricBatch handles POST /metrics/batch - accepts multiple metrics
func (h *MetricsHandler) IngestMetricBatch(w http.ResponseWriter, r *http.Request) {
	var inputs []models.MetricInput

	if err := json.NewDecoder(r.Body).Decode(&inputs); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	processed := 0
	failed := 0

	for _, input := range inputs {
		if err := input.Validate(); err != nil {
			failed++
			continue
		}

		metric, err := input.ToMetric()
		if err != nil {
			failed++
			continue
		}

		if err := h.service.ProcessMetric(metric); err != nil {
			failed++
			continue
		}
		processed++
	}

	response := map[string]interface{}{
		"status":    "completed",
		"processed": processed,
		"failed":    failed,
		"total":     len(inputs),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAnalytics handles GET /analyze - returns analytics results
func (h *MetricsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	result := h.service.GetAnalytics()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		go utils.HandleError(err, "GetAnalytics: encoding response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// GetAnomalies handles GET /anomalies - returns anomaly statistics
func (h *MetricsHandler) GetAnomalies(w http.ResponseWriter, r *http.Request) {
	cpuCount, rpsCount := h.service.GetAnomalyCounts()

	response := map[string]interface{}{
		"cpu_anomalies": cpuCount,
		"rps_anomalies": rpsCount,
		"total":         cpuCount + rpsCount,
		"threshold":     services.ZScoreThreshold,
		"window_size":   services.WindowSize,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetStats handles GET /stats - returns service statistics
func (h *MetricsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	analytics := h.service.GetAnalytics()
	cpuCount, rpsCount := h.service.GetAnomalyCounts()

	response := map[string]interface{}{
		"total_metrics":   h.service.GetTotalMetrics(),
		"window_size":     services.WindowSize,
		"zscore_threshold": services.ZScoreThreshold,
		"current": map[string]float64{
			"cpu": analytics.CurrentCPU,
			"rps": analytics.CurrentRPS,
		},
		"averages": map[string]float64{
			"cpu": analytics.AvgCPU,
			"rps": analytics.AvgRPS,
		},
		"predictions": map[string]float64{
			"cpu": analytics.PredictedCPU,
			"rps": analytics.PredictedRPS,
		},
		"anomalies": map[string]int64{
			"cpu":   cpuCount,
			"rps":   rpsCount,
			"total": cpuCount + rpsCount,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
