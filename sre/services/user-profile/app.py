"""
User Profile Service - SRE demo microservice.

Lightweight HTTP service managing user profile metadata (display name,
avatar, preferences). Designed for the EduLMS End Term SRE project to
demonstrate orchestration, monitoring, and capacity planning across
Docker Compose, Docker Swarm and Kubernetes.
"""

import os
import time
import logging
import threading
from typing import Dict

from flask import Flask, jsonify, request, abort
from prometheus_client import (
    Counter,
    Histogram,
    Gauge,
    generate_latest,
    CONTENT_TYPE_LATEST,
)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s user-profile %(message)s",
)
log = logging.getLogger("user-profile")

app = Flask(__name__)
SERVICE_NAME = "user-profile"
PORT = int(os.getenv("PORT", "8082"))

# In-memory store (single replica). For multi-replica deployment this would
# be replaced with PostgreSQL — but that is intentionally simple here.
_lock = threading.Lock()
_profiles: Dict[str, dict] = {}

REQUESTS = Counter(
    "user_profile_requests_total",
    "Total user-profile requests",
    ["method", "endpoint", "status"],
)
LATENCY = Histogram(
    "user_profile_request_latency_seconds",
    "User-profile request latency",
    ["endpoint"],
    buckets=(0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.5, 1, 2),
)
PROFILES_GAUGE = Gauge(
    "user_profile_count",
    "Number of profiles currently stored in memory",
)


def _observe(endpoint: str, status: int, started_at: float) -> None:
    LATENCY.labels(endpoint).observe(time.time() - started_at)
    REQUESTS.labels(request.method, endpoint, str(status)).inc()


@app.get("/health")
def health():
    return jsonify(status="ok", service=SERVICE_NAME), 200


@app.get("/ready")
def ready():
    return jsonify(status="ready", service=SERVICE_NAME), 200


@app.get("/metrics")
def metrics():
    PROFILES_GAUGE.set(len(_profiles))
    return generate_latest(), 200, {"Content-Type": CONTENT_TYPE_LATEST}


@app.get("/profiles/<user_id>")
def get_profile(user_id: str):
    started = time.time()
    with _lock:
        profile = _profiles.get(user_id)
    if profile is None:
        _observe("/profiles/:id", 404, started)
        return jsonify(error="not_found"), 404
    _observe("/profiles/:id", 200, started)
    return jsonify(profile), 200


@app.put("/profiles/<user_id>")
def upsert_profile(user_id: str):
    started = time.time()
    body = request.get_json(silent=True) or {}
    if not isinstance(body, dict):
        _observe("/profiles/:id", 400, started)
        abort(400, description="payload must be a JSON object")

    profile = {
        "user_id": user_id,
        "display_name": body.get("display_name", ""),
        "avatar_url": body.get("avatar_url", ""),
        "preferences": body.get("preferences", {}),
        "updated_at": int(time.time()),
    }
    with _lock:
        _profiles[user_id] = profile
        PROFILES_GAUGE.set(len(_profiles))
    log.info("upserted profile user=%s", user_id)
    _observe("/profiles/:id", 200, started)
    return jsonify(profile), 200


@app.delete("/profiles/<user_id>")
def delete_profile(user_id: str):
    started = time.time()
    with _lock:
        existed = _profiles.pop(user_id, None) is not None
        PROFILES_GAUGE.set(len(_profiles))
    status = 204 if existed else 404
    _observe("/profiles/:id", status, started)
    return ("", status)


@app.get("/")
def index():
    return jsonify(
        service=SERVICE_NAME,
        version=os.getenv("APP_VERSION", "1.0.0"),
        endpoints=[
            "/health",
            "/ready",
            "/metrics",
            "/profiles/:user_id (GET/PUT/DELETE)",
        ],
    )


if __name__ == "__main__":
    log.info("starting %s on :%s", SERVICE_NAME, PORT)
    app.run(host="0.0.0.0", port=PORT)
