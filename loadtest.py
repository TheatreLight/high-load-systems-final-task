"""
Locust load test script for High-Load IoT Service
Usage: locust -f loadtest.py --host=http://localhost:8080
"""

from locust import HttpUser, task, between
import random
from datetime import datetime


class MetricsUser(HttpUser):
    """
    Simulates IoT device sending metrics to the service.
    """
    wait_time = between(0.01, 0.05)  # Minimal delay for high RPS

    @task(3)
    def post_metrics(self):
        """POST /ingest - Send single metric (most common operation)"""
        self.client.post("/ingest", json={
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "cpu": random.uniform(20, 95),
            "rps": random.randint(500, 2000)
        })

    @task(1)
    def post_metrics_with_anomaly(self):
        """POST /ingest - Send metric with potential anomaly"""
        # Occasionally send anomalous values
        if random.random() < 0.1:  # 10% chance of anomaly
            cpu = random.choice([5, 99, random.uniform(0, 10), random.uniform(95, 100)])
            rps = random.choice([50, 5000, random.randint(0, 100), random.randint(4000, 6000)])
        else:
            cpu = random.uniform(40, 80)
            rps = random.randint(800, 1500)

        self.client.post("/ingest", json={
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "cpu": cpu,
            "rps": rps
        })

    @task(1)
    def get_analyze(self):
        """GET /analyze - Retrieve analytics results"""
        self.client.get("/analyze")

    @task(1)
    def get_anomalies(self):
        """GET /anomalies - Check anomaly statistics"""
        self.client.get("/anomalies")

    @task(1)
    def get_stats(self):
        """GET /stats - Get service statistics"""
        self.client.get("/stats")

    @task(2)
    def get_health(self):
        """GET /health - Health check (lightweight)"""
        self.client.get("/health")


class BatchMetricsUser(HttpUser):
    """
    Simulates batch metric ingestion (less frequent but larger payloads).
    """
    wait_time = between(0.5, 1.0)

    @task
    def post_batch_metrics(self):
        """POST /ingest/batch - Send batch of metrics"""
        batch = []
        for _ in range(random.randint(10, 50)):
            batch.append({
                "timestamp": datetime.utcnow().isoformat() + "Z",
                "cpu": random.uniform(20, 95),
                "rps": random.randint(500, 2000)
            })

        self.client.post("/ingest/batch", json=batch)


# For headless execution:
# locust -f loadtest.py --host=http://localhost:8080 --users 500 --spawn-rate 50 --run-time 5m --headless
