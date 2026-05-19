"""
Payment Service - SRE demo microservice.

Lightweight HTTP service simulating payment processing for the
EduLMS End Term SRE project. Exposes /health, /metrics and a simple
/pay endpoint. Designed to be containerized and orchestrated under
Docker Compose, Docker Swarm and Kubernetes.
"""

import os
import random
import time
import logging

from flask import Flask, jsonify, request
from prometheus_client import (
    Counter,
    Histogram,
    Gauge,
    generate_latest,
    CONTENT_TYPE_LATEST,
)

# ------------------------------------------------------------------ logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s payment %(message)s",
)
log = logging.getLogger("payment")

# ------------------------------------------------------------------ app
app = Flask(__name__)
SERVICE_NAME = "payment"
PORT = int(os.getenv("PORT", "8081"))
FAILURE_RATE = float(os.getenv("FAILURE_RATE", "0.02"))  # 2% errors
LATENCY_BASELINE_MS = int(os.getenv("LATENCY_BASELINE_MS", "40"))

# ------------------------------------------------------------------ metrics
REQUESTS = Counter(
    "payment_requests_total",
    "Total payment requests",
    ["method", "endpoint", "status"],
)
LATENCY = Histogram(
    "payment_request_latency_seconds",
    "Payment request latency",
    ["endpoint"],
    buckets=(0.01, 0.025, 0.05, 0.1, 0.2, 0.5, 1, 2, 5),
)
INFLIGHT = Gauge(
    "payment_inflight_requests",
    "Currently in-flight payment requests",
)
PROCESSED_AMOUNT = Counter(
    "payment_processed_amount_total",
    "Total monetary amount processed (cents)",
    ["currency"],
)


# ------------------------------------------------------------------ routes
@app.get("/health")
def health():
    return jsonify(status="ok", service=SERVICE_NAME), 200


@app.get("/ready")
def ready():
    # In a real service, check DB / broker connectivity here.
    return jsonify(status="ready", service=SERVICE_NAME), 200


@app.get("/metrics")
def metrics():
    return generate_latest(), 200, {"Content-Type": CONTENT_TYPE_LATEST}


@app.post("/pay")
def pay():
    INFLIGHT.inc()
    start = time.time()
    try:
        body = request.get_json(silent=True) or {}
        amount = int(body.get("amount", 0))
        currency = body.get("currency", "USD")
        order_id = body.get("order_id", "unknown")

        # Simulate processing latency.
        jitter = random.uniform(0, 30) / 1000.0
        time.sleep(LATENCY_BASELINE_MS / 1000.0 + jitter)

        # Simulated random failure (controllable via FAILURE_RATE env).
        if random.random() < FAILURE_RATE:
            REQUESTS.labels("POST", "/pay", "500").inc()
            log.warning("simulated payment failure order=%s", order_id)
            return jsonify(error="payment_processor_unavailable"), 500

        PROCESSED_AMOUNT.labels(currency).inc(amount)
        REQUESTS.labels("POST", "/pay", "200").inc()
        log.info("payment ok order=%s amount=%s %s", order_id, amount, currency)
        return jsonify(
            status="approved",
            order_id=order_id,
            amount=amount,
            currency=currency,
            transaction_id=f"tx_{random.randint(10**9, 10**10 - 1)}",
        )
    finally:
        LATENCY.labels("/pay").observe(time.time() - start)
        INFLIGHT.dec()


@app.get("/")
def index():
    return jsonify(
        service=SERVICE_NAME,
        version=os.getenv("APP_VERSION", "1.0.0"),
        endpoints=["/health", "/ready", "/metrics", "/pay"],
    )


if __name__ == "__main__":
    log.info("starting %s on :%s", SERVICE_NAME, PORT)
    app.run(host="0.0.0.0", port=PORT)
