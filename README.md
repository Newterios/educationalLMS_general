# EDULMS v2

**Microservice-based Learning Management System** — Final AP4 project.  
3 Go gRPC services + API Gateway + Notification Worker + NATS + PostgreSQL + Redis.

## Architecture

```
                           ┌──────────────────────────────────────────────┐
                           │            api-gateway  :9080                │
                           │   HTTP/JSON ──► gRPC (36 routes)             │
                           └──────┬──────────────┬──────────────┬─────────┘
                                  │              │              │
                     ┌────────────▼──┐ ┌─────────▼──┐ ┌───────▼──────────┐
                     │  auth-svc-v2  │ │course-svc-v2│ │assessment-svc-v2 │
                     │   :50051      │ │  :50052     │ │     :50053       │
                     │ 12 RPCs       │ │ 12 RPCs     │ │ 12 RPCs          │
                     │ JWT + bcrypt  │ │ enrollments │ │ outbox + relay   │
                     └──────┬────────┘ └──────┬──────┘ └───────┬──────────┘
                            │                 │                │
                     ┌──────▼─────────────────▼────────────────▼──────────┐
                     │                   NATS  :4222                       │
                     │       auth.> / course.> / assessment.>              │
                     └──────────────────────────┬─────────────────────────┘
                                                │
                               ┌────────────────▼────────────────┐
                               │    notification-svc-v2           │
                               │  worker pool (3 goroutines)      │
                               │  exp. backoff + dead-letter      │
                               └────────────────┬────────────────┘
                                                │ POST /notify
                               ┌────────────────▼────────────────┐
                               │       mock-gateway  :8090        │
                               │  20% random 503 · idempotency   │
                               └─────────────────────────────────┘

  Infrastructure:  PostgreSQL :5433 · Redis :6380 · NATS :4222
  Observability:   OTel Collector :4317 · Prometheus :9090 · Grafana :3000
```

## Services

| Service | Port | Description |
|---|---|---|
| `auth-svc-v2` | 50051 | Register, Login, JWT refresh, sessions, password reset, profile |
| `course-svc-v2` | 50052 | Courses, sections, materials, enrollments |
| `assessment-svc-v2` | 50053 | Quizzes, attempts, grading, gradebook, CSV export, outbox relay |
| `api-gateway` | 9080 | HTTP/JSON ↔ gRPC proxy, CORS, 36 routes |
| `notification-svc-v2` | — | NATS subscriber → worker pool → mock-gateway |
| `mock-gateway` | 8090 | AP4 §4.4 fake external API (idempotency + failure simulation) |

## Rate Limits (Redis sliding window, 1 min)

| Method | Limit |
|---|---|
| `auth.Login` | 10 RPM |
| `course.EnrollStudent` | 20 RPM |
| `assessment.StartAttempt` | 20 RPM |
| `assessment.SubmitAttempt` | 5 RPM |
| All other methods | 100 RPM |

## Caching (Redis, Cache-Aside)

| Pattern | TTL | Used by |
|---|---|---|
| Course entity | 60s | `GetCourse`, invalidated on Update/Delete |
| Course list | 60s | `ListCourses`, invalidated on Create/Delete |
| Gradebook | 60s | `GetGradebook`, invalidated on grade change |
| Quiz entity | 60s | `GetQuiz`, invalidated on Update/Delete |
| User session | — | `GetMe` (noop in Phase 2, Redis in Phase 3) |

## Prerequisites

```bash
brew install go protobuf grpcurl jq redis   # macOS
```

- Go 1.22+
- Docker + Docker Compose v2
- `grpcurl` — for smoke tests and defense demo

## Quickstart

```bash
# 1. Clone and set up env
cp .env.example .env        # fill JWT_ACCESS_SECRET, JWT_REFRESH_SECRET with:
                             #   openssl rand -hex 32

# 2. Install dev tools (one-off)
make tools

# 3. Start infra (postgres :5433, redis :6380, nats :4222)
make up

# 4. Run services locally (each in a separate terminal)
make dev-auth
make dev-course
make dev-assessment
make dev-gateway
make dev-notification        # optional — needs NATS running

# OR run everything in Docker
make up-all                  # starts infra + all 5 services + gateway
```

## Run Tests

```bash
make test                    # unit tests (all services)
make test-auth               # auth-svc-v2 unit tests + coverage
make test-course             # course-svc-v2 unit tests
make test-assessment         # assessment-svc-v2 (≥70% coverage required)

# Integration tests (require docker infra running)
make test-integration
make test-integration-course
make test-integration-assessment
```

## Smoke Tests

```bash
make smoke-auth              # 10 checks: register, login, sessions, logout…
make smoke-course            # CRUD, enroll, materials…
make smoke-assessment        # quiz lifecycle, attempt, grade, export CSV…
```

## AP4 Defense Demo

```bash
make up-all                  # start all services + mock-gateway
make demo                    # 5 AP4 checkpoints:
                             #   1. Cache hit/miss (Redis KEYS check)
                             #   2. Rate limiting (429 after 10 Login/min)
                             #   3. Notification delivery to mock-gateway
                             #   4. Idempotency deduplication
                             #   5. Dead-letter queue (Redis LLEN)
```

## Frontend (Next.js 14)

```bash
# Local dev — gateway must be running first (make up-all or make up + dev services)
cd web && npm install && npm run dev   # http://localhost:3000
                                       # uses NEXT_PUBLIC_API_URL from web/.env.local

# Docker (builds and serves on port 4000)
make up-web                            # starts infra + services + Next.js
bash scripts/smoke_web.sh              # HTTP 200 check
```

Key flows available: Login → Course list → (teacher) Create quiz → (student) Take quiz.
Gateway URL for dev: `http://localhost:9080`. Configured via `web/.env.local`.

## Observability

```bash
make up-obs                  # otel-collector :4317, prometheus :9090
                             # grafana :3000  (admin / admin)
```

Grafana dashboard: **EDULMS v2 — Service Overview**
- gRPC RPS by method
- p50 / p99 latency histograms
- Overall error rate
- NATS connection count
- Notification DLQ length

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `JWT_ACCESS_SECRET` | ✅ | — | 32-byte hex, sign access tokens |
| `JWT_REFRESH_SECRET` | ✅ | — | 32-byte hex, sign refresh tokens |
| `DB_URL` | ✅ | — | auth-svc Postgres DSN |
| `DATABASE_URL_COURSE` | ✅ | — | course-svc Postgres DSN |
| `DATABASE_URL_ASSESSMENT` | ✅ | — | assessment-svc Postgres DSN |
| `NATS_URL` | — | noop publisher | `nats://nats:4222` |
| `REDIS_URL` | — | noop cache | `redis://redis:6379/0` |
| `GATEWAY_URL` | — | `http://localhost:8090` | notification → mock-gateway |
| `WORKER_POOL_SIZE` | — | `3` | notification worker pool size |
| `JOB_RETRY_ATTEMPTS` | — | `3` | max retries before DLQ |
| `MOCK_GATEWAY_FAILURE_RATE` | — | `20` | % chance of 503 response |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | disabled | OTel Collector endpoint |

## Project Structure

```
.
├── gateway/            # HTTP/JSON ↔ gRPC API Gateway
├── mock-gateway/       # AP4 §4.4 fake external notification API
├── notification/       # NATS subscriber + worker pool + DLQ
├── observability/      # OTel Collector, Prometheus, Grafana configs
├── proto/              # Protobuf definitions + generated Go stubs
│   ├── auth/v1/
│   ├── course/v1/
│   ├── assessment/v1/
│   └── common/v1/
├── scripts/            # smoke_*.sh, demo.sh
├── services/
│   ├── auth/           # auth-svc-v2  (Clean Architecture)
│   ├── course/         # course-svc-v2
│   └── assessment/     # assessment-svc-v2 (outbox pattern)
├── docker-compose.dev.yml
├── go.work
├── Makefile
└── PLAN.md             # Full spec, rubric trace, API contracts
```

## Key Design Decisions

- **Clean Architecture** — `model → usecase/port ← repository + transport` (no framework leakage)
- **Outbox Pattern** — assessment writes events to `outbox` table in the same TX; relay goroutine publishes to NATS
- **Cache-Aside** — `Get → miss → DB → Set`; invalidation on write with `Delete`/`DeleteByPrefix`
- **Rate Limiting** — Redis `INCR + EXPIRE` sliding window; fail-open if Redis down
- **Idempotency** — mock-gateway deduplicates by SHA-256 key; notification worker generates key from `event_type + entity_id + occurred_at`
- **Dead-Letter Queue** — Redis `LPUSH dlq:notification` after N retries with exponential backoff (200ms base, 5s cap)

---

*See [PLAN.md](PLAN.md) for full scope, team split, rubric trace, and API contracts.*
