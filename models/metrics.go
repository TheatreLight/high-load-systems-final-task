package models

import (
	"errors"
	"time"
)

// Metric represents an IoT device metric data point
type Metric struct {
	Timestamp time.Time `json:"timestamp"`
	CPU       float64   `json:"cpu"`
	RPS       float64   `json:"rps"`
}

// MetricInput represents incoming metric data from API
type MetricInput struct {
	Timestamp string  `json:"timestamp"`
	CPU       float64 `json:"cpu"`
	RPS       float64 `json:"rps"`
}

// Validate checks if the metric data is valid
func (m *MetricInput) Validate() error {
	if m.Timestamp == "" {
		return errors.New("timestamp is required")
	}
	if m.CPU < 0 || m.CPU > 100 {
		return errors.New("cpu must be between 0 and 100")
	}
	if m.RPS < 0 {
		return errors.New("rps must be non-negative")
	}
	return nil
}

// ToMetric converts MetricInput to Metric with parsed timestamp
func (m *MetricInput) ToMetric() (Metric, error) {
	t, err := time.Parse(time.RFC3339, m.Timestamp)
	if err != nil {
		return Metric{}, errors.New("invalid timestamp format, use RFC3339")
	}
	return Metric{
		Timestamp: t,
		CPU:       m.CPU,
		RPS:       m.RPS,
	}, nil
}

// AnalyticsResult represents the result of analytics processing
type AnalyticsResult struct {
	CurrentCPU       float64   `json:"current_cpu"`
	CurrentRPS       float64   `json:"current_rps"`
	AvgCPU           float64   `json:"avg_cpu"`
	AvgRPS           float64   `json:"avg_rps"`
	PredictedCPU     float64   `json:"predicted_cpu"`
	PredictedRPS     float64   `json:"predicted_rps"`
	CPUZScore        float64   `json:"cpu_zscore"`
	RPSZScore        float64   `json:"rps_zscore"`
	CPUAnomaly       bool      `json:"cpu_anomaly"`
	RPSAnomaly       bool      `json:"rps_anomaly"`
	TotalMetrics     int       `json:"total_metrics"`
	WindowSize       int       `json:"window_size"`
	LastUpdated      time.Time `json:"last_updated"`
}

// AnomalyEvent represents a detected anomaly
type AnomalyEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	MetricType  string    `json:"metric_type"` // "cpu" or "rps"
	Value       float64   `json:"value"`
	ZScore      float64   `json:"zscore"`
	Mean        float64   `json:"mean"`
	StdDev      float64   `json:"stddev"`
}
