package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// TotalRequests counts total HTTP requests
	TotalRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// RequestDuration measures request latency
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "endpoint"},
	)

	// ActiveRequests tracks number of active HTTP requests
	ActiveRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_requests",
			Help: "Number of active HTTP requests",
		},
	)

	// ErrorsTotal counts total errors
	ErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Total number of HTTP errors",
		},
		[]string{"method", "endpoint", "error_type"},
	)

	// AnomalyDetectedTotal counts detected anomalies
	AnomalyDetectedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "anomaly_detected_total",
			Help: "Total number of detected anomalies",
		},
		[]string{"metric_type"},
	)

	// AnomalyRate tracks the rate of anomalies
	AnomalyRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "anomaly_rate",
			Help: "Current anomaly rate per metric type",
		},
		[]string{"metric_type"},
	)

	// MetricsProcessed counts processed metrics
	MetricsProcessed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "metrics_processed_total",
			Help: "Total number of metrics processed",
		},
	)

	// CurrentCPU tracks current CPU metric value
	CurrentCPU = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "iot_cpu_current",
			Help: "Current CPU metric value from IoT devices",
		},
	)

	// CurrentRPS tracks current RPS metric value
	CurrentRPS = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "iot_rps_current",
			Help: "Current RPS metric value from IoT devices",
		},
	)

	// AvgCPU tracks rolling average CPU
	AvgCPU = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "iot_cpu_avg",
			Help: "Rolling average CPU metric",
		},
	)

	// AvgRPS tracks rolling average RPS
	AvgRPS = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "iot_rps_avg",
			Help: "Rolling average RPS metric",
		},
	)

	// ZScoreCPU tracks CPU z-score
	ZScoreCPU = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "iot_cpu_zscore",
			Help: "Z-score of current CPU metric",
		},
	)

	// ZScoreRPS tracks RPS z-score
	ZScoreRPS = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "iot_rps_zscore",
			Help: "Z-score of current RPS metric",
		},
	)
)

func init() {
	// HTTP metrics
	prometheus.MustRegister(TotalRequests)
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(ActiveRequests)
	prometheus.MustRegister(ErrorsTotal)

	// Anomaly metrics
	prometheus.MustRegister(AnomalyDetectedTotal)
	prometheus.MustRegister(AnomalyRate)

	// IoT metrics
	prometheus.MustRegister(MetricsProcessed)
	prometheus.MustRegister(CurrentCPU)
	prometheus.MustRegister(CurrentRPS)
	prometheus.MustRegister(AvgCPU)
	prometheus.MustRegister(AvgRPS)
	prometheus.MustRegister(ZScoreCPU)
	prometheus.MustRegister(ZScoreRPS)
}

// RecordAnomaly increments the anomaly counter for a metric type
func RecordAnomaly(metricType string) {
	AnomalyDetectedTotal.WithLabelValues(metricType).Inc()
}

// UpdateMetricValues updates the current metric gauges
func UpdateMetricValues(cpu, rps, avgCPU, avgRPS, zscoreCPU, zscoreRPS float64) {
	CurrentCPU.Set(cpu)
	CurrentRPS.Set(rps)
	AvgCPU.Set(avgCPU)
	AvgRPS.Set(avgRPS)
	ZScoreCPU.Set(zscoreCPU)
	ZScoreRPS.Set(zscoreRPS)
}

// IncrementMetricsProcessed increments the processed metrics counter
func IncrementMetricsProcessed() {
	MetricsProcessed.Inc()
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

// MetricsMiddleware records metrics for each request
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip metrics endpoint itself
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		ActiveRequests.Inc()
		defer ActiveRequests.Dec()

		wrapped := newResponseWriter(w)
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(wrapped.statusCode)

		// Normalize endpoint for metrics (avoid high cardinality)
		endpoint := normalizeEndpoint(r.URL.Path)

		TotalRequests.WithLabelValues(r.Method, endpoint, statusCode).Inc()
		RequestDuration.WithLabelValues(r.Method, endpoint).Observe(duration)

		if wrapped.statusCode >= 400 {
			ErrorsTotal.WithLabelValues(r.Method, endpoint, statusCode).Inc()
		}
	})
}

// normalizeEndpoint reduces cardinality by grouping similar endpoints
func normalizeEndpoint(path string) string {
	switch path {
	case "/metrics", "/health", "/analyze", "/anomalies", "/stats":
		return path
	default:
		if len(path) > 0 && path[0] == '/' {
			// Return first path segment
			for i := 1; i < len(path); i++ {
				if path[i] == '/' {
					return path[:i]
				}
			}
		}
		return path
	}
}

// MetricsHandler returns the Prometheus metrics handler
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
