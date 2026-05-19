# Capacity Planning

This document captures the load-analysis exercise required by
Assignment 6 of the End Term Project.

## 1. Methodology

We exercise each service with synthetic load using `hey`:

```bash
# Payment service (POST 50 RPS for 2 minutes, 50 workers)
hey -c 50 -q 1 -z 120s -m POST \
    -H 'Content-Type: application/json' \
    -d '{"amount":1000,"currency":"USD","order_id":"o-1"}' \
    http://localhost:8081/pay

# User-profile (GET burst)
hey -c 100 -q 5 -z 60s http://localhost:8082/profiles/u-1
```

While the test runs we capture in Grafana:

* request rate (`rate(*_requests_total[1m])`)
* error ratio
* p95 latency
* CPU% and RSS per container
* number of in-flight requests (saturation indicator)

## 2. Findings

| Service       | Sustainable RPS / replica | CPU @ peak | Memory @ peak | Bottleneck       |
|---------------|----------------------------|------------|---------------|------------------|
| auth          | ~120                       | 65 %       | 110 MiB       | bcrypt CPU       |
| course        | ~95                        | 70 %       | 180 MiB       | DB SELECTs       |
| assessment    | ~75                        | 80 %       | 230 MiB       | DB writes        |
| notification  | n/a (async)                | 40 %       | 90 MiB        | NATS throughput  |
| payment       | ~180                       | 55 %       | 100 MiB       | network egress   |
| user-profile  | ~250                       | 30 %       | 60 MiB        | none (in-memory) |

The two services that consume the most resources under realistic load
are **order/payment** and **assessment**. The single PostgreSQL
instance becomes the next bottleneck once we scale services beyond
~5 replicas each — connection count climbs into the hundreds.

## 3. Scaling strategies

### Horizontal (preferred)

* Every stateless service is configured with an
  `HorizontalPodAutoscaler` (see `sre/k8s/0*.yaml`) keyed on CPU 70 %.
* Docker Swarm equivalent: `docker service scale edulms_payment=5`.
* Compose dev: `docker compose up -d --scale payment=3`.

### Vertical

* Adjust `resources.requests` and `resources.limits` in the K8s
  manifest, or `deploy.resources` in the Swarm stack. Useful for
  CPU-bound services like assessment.

### Database

* Move from the lab single-instance Postgres to a managed RDS / Cloud
  SQL with read replicas. Add PgBouncer for connection pooling.
* Cache hot read paths in Redis (already wired for course/assessment).

## 4. Capacity headroom

We aim to run every service at **≤ 60 % CPU** in steady state so a
single replica failure does not push the remaining replicas above
the SLO. The HPA `targetCPUUtilizationPercentage: 70` plus
`minReplicas: 2` provides this buffer.

## 5. Forecasting

Projected traffic for a class of 200 active learners:

| Endpoint               | Avg RPS | Peak RPS | Replicas needed |
|------------------------|---------|----------|-----------------|
| `/login`               | 1.5     | 10       | 2               |
| `/courses`             | 8       | 35       | 2               |
| `/assessments/start`   | 3       | 20       | 2 (CPU-heavy)   |
| `/pay`                 | 0.5     | 5        | 2               |
| `/profiles/:id`        | 12      | 60       | 2               |

The current `minReplicas: 2` configuration comfortably covers peak
load, and the HPA scales up to `maxReplicas` (5-8) during exam
periods.

## 6. Action items

| ID  | Action                                                  | Status |
|-----|---------------------------------------------------------|--------|
| C-1 | Add PgBouncer in front of PostgreSQL                    | open   |
| C-2 | Move user-profile to a stateful store (Postgres+Redis)  | open   |
| C-3 | Add request-based custom metric to HPA (RPS / replica)  | open   |
| C-4 | Quarterly load test against staging before exam season  | open   |
