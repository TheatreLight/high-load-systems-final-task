# High-Load IoT Metrics Service

A high-performance Go service for processing streaming IoT metrics data with statistical analytics, anomaly detection, and Kubernetes deployment.

## Features

- **High-Performance Metrics Ingestion**: Handle 1000+ RPS with low latency
- **Rolling Average Analytics**: Smoothing with configurable window size (default: 50 events)
- **Z-Score Anomaly Detection**: Real-time anomaly detection (threshold: 2σ)
- **Redis Caching**: Fast data access and persistence
- **Prometheus Metrics**: Full observability with custom metrics
- **Kubernetes Ready**: HPA, Ingress, and full manifest set
- **Goroutines & Channels**: Concurrent processing for high throughput

## Technology Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.22+ |
| HTTP Router | gorilla/mux |
| Cache | Redis 7 |
| Metrics | Prometheus |
| Visualization | Grafana |
| Container | Docker |
| Orchestration | Kubernetes |

## Project Structure

```
high-load-service/
├── main.go                 # Application entry point
├── analytics/
│   ├── rolling.go          # Rolling average implementation
│   └── zscore.go           # Z-score anomaly detection
├── cache/
│   └── redis.go            # Redis client wrapper
├── handlers/
│   └── metrics_handler.go  # HTTP handlers
├── metrics/
│   └── prometheus.go       # Prometheus metrics
├── models/
│   └── metrics.go          # Data models
├── services/
│   └── metrics_service.go  # Business logic
├── utils/
│   ├── logger.go           # Logging utilities
│   └── rate_limiter.go     # Rate limiting
├── k8s/                    # Kubernetes manifests
│   ├── namespace.yaml
│   ├── configmap.yaml
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── hpa.yaml
│   ├── ingress.yaml
│   ├── prometheus-rules.yaml
│   ├── servicemonitor.yaml  # For Prometheus Operator
│   └── kustomization.yaml
├── grafana/
│   └── dashboards/
│       └── iot-metrics-dashboard.json  # Pre-made Grafana dashboard
├── Dockerfile
├── docker-compose.yml
└── prometheus.yml
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/ingest` | Ingest single metric |
| POST | `/ingest/batch` | Ingest batch of metrics |
| GET | `/analyze` | Get analytics results |
| GET | `/anomalies` | Get anomaly statistics |
| GET | `/stats` | Get service statistics |
| GET | `/health` | Health check |
| GET | `/metrics` | Prometheus metrics |

## Metric Data Format

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "cpu": 75.5,
  "rps": 1250
}
```

## Quick Start

### Local Development

```bash
# Install dependencies
go mod download

# Run locally (requires Redis)
go run .

# Or use Docker Compose
docker-compose up -d
```

### Docker Compose

```bash
# Start all services (app, redis, prometheus, grafana)
docker-compose up -d

# View logs
docker-compose logs -f app

# Stop services
docker-compose down
```

### Kubernetes Deployment

```bash
# Start Minikube
minikube start --cpus=2 --memory=4g

# Enable Ingress
minikube addons enable ingress

# Build and load image
docker build -t hls-iot-service:latest .
minikube image load hls-iot-service:latest

# Deploy with Kustomize
kubectl apply -k k8s/

# Or apply manifests individually
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/hpa.yaml
kubectl apply -f k8s/ingress.yaml

# Check status
kubectl get pods -n hls-iot
kubectl get hpa -n hls-iot

# Port forward for local access
kubectl port-forward svc/hls-iot-service 8080:80 -n hls-iot
```

## Testing

### Manual Testing

```bash
# Health check
curl http://localhost:8080/health

# Ingest metric
curl -X POST http://localhost:8080/ingest \
  -H "Content-Type: application/json" \
  -d '{"timestamp":"2024-01-15T10:30:00Z","cpu":75.5,"rps":1250}'

# Get analytics
curl http://localhost:8080/analyze

# Get anomaly stats
curl http://localhost:8080/anomalies

# Get service stats
curl http://localhost:8080/stats

# Prometheus metrics
curl http://localhost:8080/metrics
```

### Load Testing

```bash
# Using Apache Bench
ab -n 10000 -c 100 -p payload.json -T application/json http://localhost:8080/ingest

# Using wrk
wrk -t12 -c500 -d60s -s post.lua http://localhost:8080/ingest

# Using k6
k6 run loadtest.js
```

## Monitoring

### Prometheus Metrics

- `http_requests_total` - Total HTTP requests
- `http_request_duration_seconds` - Request latency histogram
- `anomaly_detected_total` - Detected anomalies counter
- `metrics_processed_total` - Processed metrics counter
- `iot_cpu_current` / `iot_rps_current` - Current metric values
- `iot_cpu_avg` / `iot_rps_avg` - Rolling averages
- `iot_cpu_zscore` / `iot_rps_zscore` - Z-scores

### Grafana Dashboards

Access Grafana at http://localhost:3000 (admin/admin)

**Import pre-made dashboard:**
1. Go to Dashboards → Import
2. Upload `grafana/dashboards/iot-metrics-dashboard.json`
3. Select Prometheus as data source
4. Click Import

The dashboard includes 10 panels:
- HTTP Request Rate (RPS)
- Request Latency (p50, p95, p99)
- Z-Score Anomaly Detection
- Anomalies Detected
- CPU/RPS Metrics (Current vs Rolling Average)
- Stat panels for key metrics

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | HTTP server port |
| REDIS_HOST | localhost | Redis host |
| REDIS_PORT | 6379 | Redis port |
| REDIS_PASSWORD | | Redis password |

## Analytics

### Rolling Average
- Window size: 50 events
- Returns smoothed prediction based on recent values

### Z-Score Anomaly Detection
- Threshold: 2σ (2 standard deviations)
- Window size: 50 events
- Flags values deviating significantly from mean

## License

MIT
