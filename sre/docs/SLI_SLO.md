# SLIs, SLOs & Error Budgets — EduLMS

Service Level Indicators (SLIs) tell us *how the service is doing*.
Service Level Objectives (SLOs) define *how good is good enough*.
Error budgets are the difference between *100 %* and the SLO — the
amount of "bad" we accept before we pause new feature work and pay
down reliability debt instead.

This document captures the SLIs / SLOs for every microservice in the
EduLMS SRE End Term Project. Targets are intentionally moderate so
they are achievable in a lab.

---

## 1. SLI catalogue

For every service we track the four Google "Golden Signals":

| Signal      | SLI                                                            | Source metric                                          |
|-------------|----------------------------------------------------------------|--------------------------------------------------------|
| Latency     | p50 / p95 / p99 of request duration                            | `*_request_latency_seconds` histogram                  |
| Traffic     | Requests per second                                            | `rate(*_requests_total[1m])`                           |
| Errors      | Fraction of HTTP 5xx (or non-OK gRPC) responses                | `*_requests_total{status=~"5.."}` over total           |
| Saturation  | CPU %, memory %, in-flight requests, worker pool utilisation   | `process_cpu_seconds_total`, `*_inflight_requests` etc |

Availability is a composite SLI: a request succeeds if it returns a
non-5xx response within the latency budget.

---

## 2. SLOs per service

### 2.1 Authentication Service

| SLO          | Target           | Measurement window |
|--------------|------------------|--------------------|
| Availability | ≥ 99.5 %         | 30 days, rolling   |
| Latency p95  | ≤ 150 ms         | 5 minutes          |
| Error rate   | ≤ 0.5 %          | 5 minutes          |

Reasoning: every other request flows through auth, so it has the
strictest budget.

### 2.2 Course Service

| SLO          | Target          | Window      |
|--------------|-----------------|-------------|
| Availability | ≥ 99 %          | 30 d        |
| Latency p95  | ≤ 200 ms        | 5 m         |
| Error rate   | ≤ 1 %           | 5 m         |

### 2.3 Assessment Service

| SLO          | Target          | Window      |
|--------------|-----------------|-------------|
| Availability | ≥ 99 %          | 30 d        |
| Latency p95  | ≤ 250 ms        | 5 m         |
| Error rate   | ≤ 1 %           | 5 m         |

Slightly higher latency budget because assessments do CPU-bound work.

### 2.4 Notification Service

| SLO              | Target                        | Window |
|------------------|-------------------------------|--------|
| Job throughput   | ≥ 30 jobs/min sustained       | 1 h    |
| DLQ depth        | ≤ 5 jobs                      | 5 m    |
| Job success rate | ≥ 99 %                        | 1 h    |

This service is asynchronous; latency at the API edge does not apply.

### 2.5 Payment Service (new)

| SLO          | Target          | Window      |
|--------------|-----------------|-------------|
| Availability | ≥ 99 %          | 30 d        |
| Latency p95  | ≤ 200 ms        | 5 m         |
| Error rate   | ≤ 1 %           | 5 m         |

The `FAILURE_RATE` env-var lets us deliberately burn error budget for
incident-simulation training (see `INCIDENT.md`).

### 2.6 User-Profile Service (new)

| SLO          | Target          | Window      |
|--------------|-----------------|-------------|
| Availability | ≥ 99 %          | 30 d        |
| Latency p95  | ≤ 150 ms        | 5 m         |
| Error rate   | ≤ 1 %           | 5 m         |

### 2.7 API Gateway (edge)

| SLO          | Target            | Window |
|--------------|-------------------|--------|
| Availability | ≥ 99.9 %          | 30 d   |
| Latency p95  | ≤ 50 ms (overhead)| 5 m    |

Edge has the highest budget because everything depends on it.

---

## 3. Error budget policy

* **At 100 % of budget remaining** — feature work is unrestricted.
* **At 50 % budget burned** — code reviewers require a reliability
  consideration in every PR; SREs prioritise hot-spot fixes.
* **At 0 % budget left** — feature deploys to that service are frozen.
  Only reliability work is merged until the next 30-day window starts.
* The budget resets after every full 30-day window.

Burn-rate alerts (fast & slow) translate this policy into Prometheus
rules — see `sre/monitoring/alerts/slo-alerts.yml`.

---

## 4. How the SLOs map to Prometheus

| SLO                    | PromQL                                                                                              |
|------------------------|-----------------------------------------------------------------------------------------------------|
| Payment error rate     | `sum(rate(payment_requests_total{status=~"5.."}[5m])) / sum(rate(payment_requests_total[5m]))`     |
| Payment p95 latency    | `histogram_quantile(0.95, sum by (le) (rate(payment_request_latency_seconds_bucket[5m])))`         |
| Availability (30 d)    | `1 - (sum_over_time(absent(up{job="payment"})[30d:1m]) / (30*24*60))`                              |

---

## 5. Reporting

A Grafana dashboard (`sre/monitoring/dashboards/edulms-sre-overview.json`)
visualises every SLI. The same metrics drive the alerts in
`slo-alerts.yml`. Weekly the on-call engineer publishes an SLO
compliance report; the template lives in `POSTMORTEM.md`.
