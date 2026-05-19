# EDULMS v2 — Architecture

> Design document. The "why" behind every choice in `PLAN.md`. Anything not justified here is fair game during defense, so if you change a decision update this file in the same PR.

---

## 1. System map

```
           ┌────────────────────────────────────────────────────┐
           │  Browser (Next.js 14 — /web, port 4000)            │
           └────────────────────────────────────────────────────┘
                                  │  HTTPS  (final.aitbek.tech)
                                  ▼
           ┌────────────────────────────────────────────────────┐
           │  nginx on the VPS                                  │
           │   final.aitbek.tech     → :4000  (Next.js)         │
           │   api.final.aitbek.tech → :9080  (API Gateway)     │
           │   /monitoring/grafana-v2/ → :3002 (Grafana)        │
           └────────────────────────────────────────────────────┘
                                  │  HTTP/JSON  (Bearer JWT)
                                  ▼
           ┌────────────────────────────────────────────────────┐
           │  API Gateway   :9080                               │
           │  three route groups, one per teammate              │
           │  HTTP → gRPC, forwards Authorization metadata      │
           └────────────────────────────────────────────────────┘
                  │  gRPC + auth metadata + trace ctx
       ┌──────────┼─────────────────────────────────────────┐
       ▼          ▼                                         ▼
 ┌──────────┐  ┌────────────┐                       ┌──────────────┐
 │ auth     │  │ course     │  ── gRPC client ─►   │ assessment   │
 │ :50051   │  │ :50052     │                       │ :50053       │
 └──────────┘  └────────────┘                       └──────────────┘
   │   │   │       │   │   │                          │   │   │   │
   │   │   ▼       │   │   ▼                          │   │   ▼   ▼
   │   │ Postgres  │   │ Postgres                     │   │ Postgres
   │   │ auth_v2   │   │ course_v2                    │   │ assessment_v2
   │   │           │   │                              │   │ (incl. outbox)
   │   ▼           │   ▼                              │   ▼
   │  Redis        │  Redis                           │  Redis
   ▼               ▼                                  ▼
                          NATS Core   :4222
                              │
                              ▼
                ┌────────────────────────────┐
                │ notification-svc-v2        │
                │  subscriber + worker pool  │
                │  + idempotency + dead-letter
                └────────────────────────────┘
                  │                       │
                  │ SMTP (Gmail App PW)   │ HTTP POST /notify
                  ▼                       ▼
              real emails           mock-notify-gateway :8090
                                    (20 % 503 to exercise retries)

All Go services export OTLP → otel-collector :4317
  ├─ traces → Tempo :3200
  ├─ metrics → Prometheus :9091
  └─ logs   → Loki :3100
                              ↓
                  Grafana :3002 (dashboards committed in repo)
```

---

## 2. Clean Architecture per service

The four reference assignments hammer the same layering. We follow it without deviation. Layers, top-down:

```
                    ┌────────────────────────┐
external clients ──►│ transport/grpc         │ ← parses proto, returns codes.*
                    │   server.go            │
                    │   mappers.go (proto↔domain)
                    │   interceptors/        │ ← auth, rate-limit, recovery, otel
                    └──────────┬─────────────┘
                               │ calls port.* interfaces (Go interfaces)
                               ▼
                    ┌────────────────────────┐
                    │ usecase                │ ← all business rules
                    │   register.go, login.go, ...
                    │   port/                │ ← UserRepo, Cache, Mailer, EventPublisher
                    └──────────┬─────────────┘
                               │ depends only on its own port/ interfaces
                               ▼
                    ┌────────────────────────┐
                    │ model                  │ ← pure domain (no pgx, no gin, no proto)
                    │   user.go, session.go
                    └────────────────────────┘
                               ▲
                               │  implementations are wired by app/ at startup
                               │
       ┌──────────────┬────────┴─────────────┬─────────────────┐
       ▼              ▼                      ▼                 ▼
 repository/postgres  cache/redis     event/nats         mailer/smtp
 (pgxpool)           (go-redis/v9)   (nats.go)          (net/smtp)
```

Hard rules — anything that breaks them is a Worst-Case bullet from the rubrics:

1. **`internal/model` imports nothing from `pgx`, `redis`, `nats.go`, `gin`, `grpc`, or `proto`.** It's pure Go structs and methods.
2. **`internal/usecase` imports `model` and its own `port/` interfaces.** It never imports `pgx`, `redis`, `nats.go`, or any generated proto.
3. **Mapping `proto ↔ model` happens in `transport/grpc/mappers.go` only.**
4. **All infrastructure types (`pgxpool.Pool`, `redis.Client`, `nats.Conn`, `smtp.Auth`) are constructed in `internal/app/` and injected into usecases through interfaces.** No globals.
5. **Repositories return `model.*` and domain-level errors (`errs.ErrNotFound`, `errs.ErrAlreadyExists`).** Translation to `codes.*` happens in a single helper in `transport/grpc/errors.go`.

Why so strict: it's literally the rubric. AP2 Clean-Arch Preservation is 25 %, AP3 keeps it at 15 %, AP4 keeps it at 15 %. Cumulative impact on the final grade dwarfs any time saved by skipping a layer.

---

## 3. gRPC contracts

### 3.1 Proto layout

```
proto/
├── auth/v1/auth.proto
├── course/v1/course.proto
├── assessment/v1/assessment.proto
└── common/v1/common.proto      ← Pagination, ErrorDetail, Timestamp shims
```

Each owner edits **only** their own proto file. Generated files (`*.pb.go`, `*_grpc.pb.go`) are committed to keep `go run .` working without extra setup (AP2 rubric explicitly demands this).

### 3.2 Stub generation

```makefile
proto:
\tprotoc \
\t  --proto_path=proto \
\t  --go_out=proto --go_opt=paths=source_relative \
\t  --go-grpc_out=proto --go-grpc_opt=paths=source_relative \
\t  $$(find proto -name '*.proto')
```

We deliberately do **not** use Buf or grpc-gateway codegen. Hand-written REST routes in the API Gateway are simpler to grade and let each member own their own slice of HTTP.

### 3.3 Error mapping

Single source of truth, in `internal/transport/grpc/errors.go`:

| Domain error | gRPC status |
|---|---|
| `errs.ErrInvalidInput`        | `codes.InvalidArgument` |
| `errs.ErrUnauthenticated`     | `codes.Unauthenticated` |
| `errs.ErrPermissionDenied`    | `codes.PermissionDenied` |
| `errs.ErrNotFound`            | `codes.NotFound` |
| `errs.ErrAlreadyExists`       | `codes.AlreadyExists` |
| `errs.ErrFailedPrecondition`  | `codes.FailedPrecondition` |
| `errs.ErrRateLimited`         | `codes.ResourceExhausted` |
| `errs.ErrRemoteUnavailable`   | `codes.Unavailable` |
| anything else                 | `codes.Internal` (log full error with trace ID) |

Matches the AP2 §8 table verbatim and extends it with AP4 codes.

### 3.4 Inter-service gRPC

Deliberately minimal — services trust JWT claims (`sub`, `role`, `email_verified`) so most reads don't need a cross-call. The **one** real cross-service gRPC dependency is:

- `assessment-svc-v2` → `course-svc-v2.ListEnrollments(course_id=X, student_id=Y)` before `StartAttempt` and `SubmitAttempt`. An empty result means "not enrolled" and Assessment returns `codes.FailedPrecondition`; a network/availability failure surfaces as `codes.Unavailable`.

We deliberately did **not** add a dedicated `CheckEnrollment` RPC — the existing `ListEnrollments` already covers it with two optional filters, which keeps the public surface of `course-svc-v2` at exactly 12 RPCs and avoids a single-use endpoint. The Course client is hidden behind a `port.EnrollmentChecker` interface; the gRPC stub implementation lives in `internal/client/course/` and just wraps the filtered `ListEnrollments` call.

We intentionally do **not** make Auth-call from other services. Auth is consulted once, at the gateway, by validating the JWT signature against the shared `JWT_ACCESS_SECRET`. The user id and role rides in gRPC metadata after that.

---

## 4. Database — one DB per service, golang-migrate, transactions

### 4.1 Schemas (initial migrations, abbreviated)

**`auth_v2`** — owned by Person A

```sql
CREATE TABLE users (
  id              UUID PRIMARY KEY,
  email           TEXT NOT NULL UNIQUE,
  password_hash   TEXT NOT NULL,
  full_name       TEXT NOT NULL,
  locale          TEXT NOT NULL DEFAULT 'en',
  role            TEXT NOT NULL DEFAULT 'student',
  email_verified  BOOLEAN NOT NULL DEFAULT false,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
  id              UUID PRIMARY KEY,
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  refresh_hash    TEXT NOT NULL,
  user_agent      TEXT,
  ip              INET,
  expires_at      TIMESTAMPTZ NOT NULL,
  revoked_at      TIMESTAMPTZ
);

CREATE TABLE verification_codes (
  id              UUID PRIMARY KEY,
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind            TEXT NOT NULL CHECK (kind IN ('email','password_reset')),
  code_hash       TEXT NOT NULL,
  expires_at      TIMESTAMPTZ NOT NULL,
  used_at         TIMESTAMPTZ
);
```

**`course_v2`** — owned by Person B

```sql
CREATE TABLE courses (
  id              UUID PRIMARY KEY,
  title           TEXT NOT NULL,
  description     TEXT NOT NULL DEFAULT '',
  teacher_id      UUID NOT NULL,
  language        TEXT NOT NULL DEFAULT 'en',
  deleted_at      TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sections (
  id              UUID PRIMARY KEY,
  course_id       UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  title           TEXT NOT NULL,
  position        INT NOT NULL
);

CREATE TABLE materials (
  id              UUID PRIMARY KEY,
  section_id      UUID NOT NULL REFERENCES sections(id) ON DELETE CASCADE,
  kind            TEXT NOT NULL,                 -- 'pdf','video','link'
  url             TEXT NOT NULL,
  title           TEXT NOT NULL
);

CREATE TABLE enrollments (
  id              UUID PRIMARY KEY,
  course_id       UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  student_id      UUID NOT NULL,
  enrolled_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (course_id, student_id)
);
```

**`assessment_v2`** — owned by Person C, includes the **outbox**

```sql
CREATE TABLE quizzes (
  id              UUID PRIMARY KEY,
  course_id       UUID NOT NULL,                 -- soft FK to course-svc
  title           TEXT NOT NULL,
  time_limit_sec  INT NOT NULL DEFAULT 0,
  shuffle         BOOLEAN NOT NULL DEFAULT true,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE questions (
  id              UUID PRIMARY KEY,
  quiz_id         UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
  body            TEXT NOT NULL,
  choices         JSONB NOT NULL,                -- [{"k":"a","v":"...","correct":true},...]
  points          INT NOT NULL DEFAULT 1
);

CREATE TABLE attempts (
  id              UUID PRIMARY KEY,
  quiz_id         UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
  student_id      UUID NOT NULL,
  started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  submitted_at    TIMESTAMPTZ,
  auto_score      NUMERIC(6,2),
  manual_score    NUMERIC(6,2),
  status          TEXT NOT NULL DEFAULT 'in_progress'
);

CREATE TABLE outbox (                              -- AP3 outbox pattern
  id              BIGSERIAL PRIMARY KEY,
  aggregate_id    UUID NOT NULL,
  event_type      TEXT NOT NULL,
  payload         JSONB NOT NULL,
  occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  published_at    TIMESTAMPTZ
);
CREATE INDEX outbox_unpublished_idx ON outbox (id) WHERE published_at IS NULL;
```

> **Deployment note:** In dev all three DBs live on one Postgres instance (port 5433), created by `deploy/postgres/init.sql` on first volume init. On the VPS, same — three logical databases on one host Postgres. Production-ready alternative: one Postgres cluster per service.

### 4.2 Migration discipline

- `migrations/000001_*.up.sql` and matching `.down.sql`, golang-migrate naming, one logical change per pair.
- `up` runs automatically on service startup, **before** the gRPC listener binds. If a migration fails the process exits non-zero — preferred over a half-initialised service.
- `down` must reverse exactly its `up`. CI verifies this by running `up` → `down` → `up` on every PR.
- No raw `CREATE TABLE` / `ALTER TABLE` inside Go code. Anywhere.

### 4.3 Transactions

The two RPCs that touch more than one table inside a single business operation must use a transaction:

- `assessment.SubmitAttempt`: update `attempts.submitted_at` + `auto_score` **and** insert into `outbox` in the same TX.
- `assessment.GradeSubmission`: update `attempts.manual_score` **and** insert into `outbox` in the same TX.

Pattern (pgx v5):

```go
func (r *AttemptRepo) SubmitAttempt(ctx context.Context, fn func(tx pgx.Tx) error) error {
    return pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, fn)
}
```

The usecase calls `repo.SubmitAttempt(ctx, func(tx) error { ... })` and runs everything inside `fn`. The repo never exposes raw `*pgxpool.Pool` outside its package.

The reason for read-committed (not serializable): we don't have anomalies that need stronger isolation here, and read-committed gives the rubric-required ACID guarantees without the contention penalty.

---

## 5. Messaging — NATS Core + Outbox in Assessment

### 5.1 Why NATS Core (not JetStream, not RabbitMQ)

The final's rubric says "preferably NATS". AP3 §2 explicitly tests Pub/Sub vs Point-to-Point understanding using core NATS. We use core NATS to match. The trade-off — no persistence, at-most-once — is exactly what AP3 §10 expects us to talk about during defense.

### 5.2 Topic naming — uniform `<service>.<entity>.<verb_past>`

| Topic | Publisher | Consumers | Payload |
|---|---|---|---|
| `auth.user.registered`              | auth       | notification | `id, email, full_name, locale, occurred_at` |
| `auth.user.verified`                | auth       | notification | `id, email, occurred_at` |
| `auth.user.updated`                 | auth       | course (cache invalidation) | `id, occurred_at` |
| `auth.session.revoked`              | auth       | notification | `session_id, user_id, occurred_at` |
| `auth.password.reset_requested`     | auth       | notification | `user_id, email, code, expires_at, occurred_at` |
| `auth.password.changed`             | auth       | notification | `user_id, email, occurred_at` |
| `course.course.created`             | course     | notification, assessment | `id, title, teacher_id, occurred_at` |
| `course.course.updated`             | course     | assessment (cache invalidation) | `id, occurred_at` |
| `course.course.deleted`             | course     | assessment | `id, occurred_at` |
| `course.section.created`            | course     | — | — (logged only) |
| `course.material.added`             | course     | notification | `id, course_id, title, occurred_at` |
| `course.enrollment.created`         | course     | notification, assessment | `course_id, student_id, occurred_at` |
| `course.enrollment.removed`         | course     | assessment | `course_id, student_id, occurred_at` |
| `assessment.quiz.created`           | assessment | notification | `id, course_id, title, occurred_at` |
| `assessment.attempt.started`        | assessment | — | — |
| `assessment.attempt.submitted`      | assessment | notification | `id, quiz_id, student_id, auto_score, occurred_at` |
| `assessment.submission.graded`      | assessment | notification (job queue trigger) | `id, quiz_id, student_id, manual_score, occurred_at` |

Every payload is JSON and includes `event_type` (the topic) and `occurred_at` (RFC3339) as mandated by AP3 §5.3.

> **Phase 2 TODO — Envelope wrapper**: Currently `port.EventPublisher.Publish` accepts raw `[]byte` and use-cases pass `nil`. Before wiring real NATS in Phase 2, introduce `internal/event/envelope.go` with `type Envelope struct { EventType, OccurredAt string; Data map[string]any }` and change the signature to `Publish(ctx, eventType string, data map[string]any) error`. The publisher marshals the envelope internally; use-cases stop touching JSON. Noop and real NATS share the same signature.

### 5.3 Outbox in Assessment

`assessment-svc-v2` writes business changes and a row in `outbox` inside the same DB transaction. A background goroutine polls every 200 ms (`SELECT … WHERE published_at IS NULL ORDER BY id LIMIT 100`), publishes to NATS, and `UPDATE outbox SET published_at = now() WHERE id = $1`. On crash between publish and update the event is republished — duplicates are absorbed by the Notification Service's idempotency key, so consumers see exactly-once.

`auth-svc-v2` and `course-svc-v2` do not use an outbox. Best-effort publish from inside the usecase; if the broker is down the RPC still succeeds and a warning is logged (AP3 §10 expected behaviour). Rationale to give the grader: **submission grades are the only events that have grade-altering consequences for the student; everything else is recoverable from the source-of-truth tables.** This is a defensible engineering judgement, not laziness.

### 5.4 Subscriber side — Notification Service

Layout (AP4 §10):

```
notification/
├── cmd/notification/main.go              ← graceful shutdown, wiring
└── internal/
    ├── subscriber/                       ← NATS Subscribe per topic, hands to logger + jobqueue
    ├── logger/                           ← AP3 §7.2 JSON line to stdout
    ├── jobqueue/                         ← AP4 worker pool, idempotency, retry, dead-letter
    └── mailer/                           ← Mailer interface, SMTP impl, mock impl
```

Three concerns, three packages, no cross-dependencies. Subscriber → calls `logger.Log(...)` first, then conditionally `jobqueue.Enqueue(...)`. The job queue only fires for `assessment.submission.graded`. The mailer is also used directly from `subscriber` for the simpler "send welcome email" / "send password reset" flows.

---

## 6. Caching — strategy per service

`go-redis/v9`. Single Redis instance, key namespace `service:entity:id` per AP4 §4.1.

| Service | RPC | Strategy | Key | TTL | Reasoning |
|---|---|---|---|---|---|
| auth      | `GetMe`            | Cache-Aside (Read-Through) | `auth:user:<id>`           | `CACHE_TTL_SECONDS` (60s) | hot read, stale-ok for 60s |
| auth      | `Login` / refresh  | Write-Through              | `auth:session:<id>`        | matches `JWT_REFRESH_TTL` | session lookup on every RPC, refresh writes both DB + cache atomically |
| auth      | `UpdateProfile`    | Invalidate                 | `auth:user:<id>`           | n/a | force re-read after write |
| course    | `GetCourse`        | Cache-Aside                | `course:course:<id>`       | 60s | high read fan-out |
| course    | `ListCourses`      | Cache-Aside                | `course:courses:list:<filter-hash>` | 60s | list keys are short and hashed |
| course    | `ListSections`     | Cache-Aside                | `course:sections:<course_id>` | 60s | per-course list, invalidated on `AddSection` |
| course    | `CreateCourse`     | Write-Around               | invalidate `course:courses:list:*` | immediate | new course must show up |
| course    | `EnrollStudent`    | Invalidate                 | `course:enrollments:<course_id>` | immediate | |
| assessment| `GetQuiz`          | Cache-Aside                | `assessment:quiz:<id>`     | 60s | rarely changes |
| assessment| `ListQuizzes`      | Cache-Aside                | `assessment:quizzes:<course_id>` | 60s | |
| assessment| `GradeSubmission`  | Write-Around               | invalidate `assessment:gradebook:<course_id>` | immediate | gradebook must reflect new grade |
| assessment| `GetGradebook`     | Cache-Aside                | `assessment:gradebook:<course_id>` | 60s | aggregate view |

Rules (AP4 §5.3):

- A cache miss never errors — fall through to Postgres silently.
- A cache write failure logs and continues — best-effort.
- All Redis calls live in `cache/redis/`; nothing outside that package imports `go-redis`.

### 6.1 Rate limiting

Same Redis. Sliding-window counter with `INCR` + `EXPIRE` on a key shaped `rl:<ip>:<minute>`. Implemented once in `pkg/interceptor/ratelimit.go`, registered as `grpc.UnaryServerInterceptor` for all three services.

```
default                          100 RPM per client IP
overrides via metadata:
  auth.Login                      10 RPM per client IP
  auth.Register                    5 RPH per client IP
  assessment.SubmitAttempt         5 RPM per (user_id, quiz_id)
```

Exceeded → `codes.ResourceExhausted` with a `retry-after` field in the error message. AP4 §4.2 verbatim.

---

## 7. Background jobs in Notification Service

AP4 §4.3 + §6 verbatim, wired this way:

```
NATS subject "assessment.submission.graded"
        │
        ▼
 subscriber.handle(msg)
        │
        ├─► logger.Log(...)                              ← AP3 §7.2 line
        └─► jobqueue.Enqueue(Job{
                idempotency_key = sha256(event_type|id|occurred_at),
                attempt_id, student_id, doctor_id (← teacher_id), occurred_at,
                channel = "email",
                recipient = student.email (resolved via gRPC GetMe? no — comes in the event),
                message  = "Your attempt <id> has been graded.",
            })

 jobqueue:
   buffered chan Job (size = 256)
   N worker goroutines (WORKER_POOL_SIZE, default 3)

 worker:
   for job := range ch {
       if alreadyDone(job.IdempotencyKey) { drop & log "duplicate"; continue }
       try POST /notify on mock-gateway:8090 (or real SMTP if MAILER=smtp)
         200 + status="accepted"  → mark Redis key "done" with TTL 24h; log success
         200 + status="duplicate" → log info; mark Redis key "done"
         503 / network error      → exp backoff 1s,2s,4s; max 3 attempts
         after 3 failures         → LPUSH dlq:notification {job}; log dead-letter to stderr
   }
```

Idempotency key store: `notif:idempotency:<sha>` in Redis, value `done`, TTL 24h. So even a service restart inside the 24-hour window doesn't redeliver.

Dead-letter list `dlq:notification` is a Redis `LPUSH`-ed JSON list. The defender can `LRANGE dlq:notification 0 -1` during the AP4 dead-letter checkpoint to show the entry is recoverable, not just printed.

Graceful shutdown: on SIGTERM the subscriber stops pulling new NATS messages, closes the input channel, the workers drain to empty, then `nc.Drain()` flushes outstanding subscriptions, then the process exits 0.

---

## 8. API Gateway

One Go binary, three route groups. Owner of each file = owner of that service.

```
gateway/
├── cmd/gateway/main.go         ← chi router, OTel middleware, JWT verification
└── internal/
    ├── routes/auth.go          ← Person A: POST /api/v1/auth/register → AuthService.Register
    ├── routes/course.go        ← Person B: GET  /api/v1/courses        → CourseService.ListCourses
    ├── routes/assessment.go    ← Person C: POST /api/v1/quizzes/:id/submit → AssessmentService.SubmitAttempt
    └── clients/                ← gRPC client pool with reconnect
```

- HTTP is **only** the transport. No business logic, ever. Each handler:
  1. read JSON,
  2. verify `Authorization: Bearer <jwt>` against `JWT_ACCESS_SECRET` (this is the *only* place JWT is checked — internal gRPC just trusts the metadata we attach),
  3. build the proto request,
  4. forward via the gRPC client with `metadata.AppendToOutgoingContext(ctx, "user_id", uid, "role", role)`,
  5. translate `codes.*` back to HTTP `4xx` / `5xx`.
- Auth endpoints that don't need a session (`Register`, `Login`, `RefreshToken`, `RequestPasswordReset`, `VerifyEmail`) bypass step 2.
- `ExportGrades` is a **unary** RPC returning `{ bytes csv = 1; string filename = 2; }`. The gateway serves it as `GET /api/v1/courses/:id/grades.csv` by setting `Content-Type: text/csv` + `Content-Disposition: attachment; filename="<filename>"` and writing the bytes in a single response. (We keep all 12 RPCs unary on purpose — homogeneous surface for the rubric. A server-streaming `WatchSubmissions` RPC is left as an optional bonus, see PLAN.md §3.3.)

Rationale for one gateway instead of three: simpler ingress, one ACME cert, easier rate-limit at edge if we ever add it. Ownership is preserved by file boundaries, not process boundaries; the defense answer to "who built the gateway" is "all three of us, one route group each".

---

## 9. Frontend (bonus)

The existing `web/` (Next.js 14 + Tailwind + Zustand + React Query) is repointed at the new gateway. Three small changes:

1. `NEXT_PUBLIC_API_URL=https://api.final.aitbek.tech`
2. Replace REST shapes that differ from the new v1 contract (most are identical because we deliberately kept field names).
3. Add a thin auth-token store that handles refresh on 401.

We do not write new UI — keeping the bonus achievable without sinking days into design work. If time permits, polish one screen (course detail + start quiz) so the demo flows nicely.

---

## 10. Observability (bonus)

Stack:

```
each Go service
  └─► OpenTelemetry SDK (otelgrpc auto-instrument + manual spans inside usecases)
       └─► OTel Collector :4317 (OTLP)
             ├─► Tempo  :3200   (traces)
             ├─► Loki   :3100   (logs, via OTel Logs SDK)
             └─► Prometheus remote-write :9091 (metrics)

Grafana :3002
  ├─ Datasources: Tempo, Loki, Prometheus (provisioned via observability/grafana/datasources.yaml)
  └─ Dashboards:
       1. "EDULMS v2 — service overview"  (RED metrics, top RPCs, p95 latency)
       2. "Cache + Rate-limit"            (hit ratio, RL rejects per route)
       3. "Notification job queue"        (in-flight, retries, dead-letter rate)
       4. "Trace view"                    (Tempo trace explorer with service map)
```

Trace context is propagated from the API Gateway through gRPC metadata into all downstream services and into the NATS message headers, so a `Register` call produces a single trace covering: gateway → auth → NATS publish → notification consumer → SMTP send.

Manual spans we *will* add (worth bonus marks during defense):

- `usecase.SubmitAttempt` (parent), child spans `repo.LoadQuiz`, `repo.PersistAnswers`, `outbox.Insert`.
- `jobqueue.Process` per worker iteration with attempt number as attribute.

---

## 11. Security checklist

These are easy to forget under deadline pressure; keep them visible.

- bcrypt cost ≥ 12 for password hashing.
- JWT signed with HS256; access TTL 15 min, refresh TTL 30 days; refresh tokens are stored hashed in `sessions.refresh_hash` (`sha256` is fine because the entropy is already 256-bit).
- All RPCs except the AuthService's pre-login set require a valid JWT in metadata.
- Postgres + Redis + NATS bind to the internal Docker network only; nginx is the only public surface.
- `.env` and any printed secrets are in `.gitignore`. CI fails if it sees `SMTP_PASSWORD=` in any committed file.
- gRPC server listens on a non-TLS port internally; TLS terminates at nginx. (Same posture as the existing project — keeps the diff small.)
- Rate-limit interceptor wraps the gRPC chain *before* the auth interceptor: we don't want unauthenticated traffic to chew through JWT verification on a flood.

---

## 12. Failure scenarios & how they're handled

A condensed map for defense, lifted from the four assignments and applied to v2:

| Failure | What v2 does | Defended by reading… |
|---|---|---|
| `course-svc` down while `assessment.SubmitAttempt` runs | Assessment returns `codes.Unavailable` with a "course service unreachable" message; trace shows the failed outbound call | AP2 §3 failure scenario |
| Postgres down on startup | Service exits non-zero with structured log message | AP3 §10 |
| Postgres goes down mid-RPC | Repo returns wrapped `pgx` error; transport translates to `codes.Internal`; OTel span records exception | AP3 §10 |
| NATS down at startup (auth/course/assessment) | Service starts, logs warning, RPCs still succeed, events are dropped (best-effort) | AP3 §10 |
| NATS down at startup (notification) | Retries with backoff 1s/2s/4s; after max retries exits non-zero | AP3 §7.1 |
| Outbox publish fails after DB commit (Assessment) | Row stays `published_at IS NULL`; relay retries on next tick — exactly-once preserved | AP3 §10 + custom |
| Redis down | Cache miss, fall through to DB, log warning. Rate limiter degrades to "allow everything" with a loud warning — documented trade-off | AP4 §11 |
| Mock gateway returns 503 | Worker backs off (1s, 2s, 4s), retries up to 3 times | AP4 §6.3 |
| Duplicate event redelivered | Idempotency key in Redis short-circuits, job dropped silently with info log | AP4 §6.3 |
| 3 retries exhausted | Dead-letter to stderr **and** `LPUSH dlq:notification`; worker continues | AP4 §6.3 |
| Gmail SMTP rate-limits us at defense | Set `MAILER=mock`; mock gateway covers the rubric | this doc §11 |

---

## 13. Open questions for the team

Put these in the team chat before Phase 1 so we don't discover them mid-implementation:

1. Do we keep `material.url` pointing at the existing MinIO bucket (v1) or stand up a fresh one for v2? *Recommendation:* reuse v1 MinIO read-only; uploads in v2 stay out of scope.
2. Multilingual error messages — do we localise gRPC error text per `Accept-Language`, or always English and let the frontend translate? *Recommendation:* English only on the wire; frontend handles i18n.
3. Do we expose the API Gateway as gRPC-Web in addition to REST? *Recommendation:* no, REST is simpler and the existing frontend already speaks it.
4. Will the defender allow `make demo` instead of typing commands by hand? Probably yes, but ask before Phase 5.

Update this section when each one is decided so the README in Phase 4 can reference the answers.
