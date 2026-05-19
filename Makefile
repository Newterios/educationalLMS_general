# EDULMS v2 — top-level Makefile.

.PHONY: help tools proto lint test test-auth test-course test-assessment test-integration \
        test-integration-course test-integration-assessment up up-all up-obs down dev-reset \
        migrate-up migrate-down dev-auth dev-course dev-assessment dev-gateway dev-notification \
        smoke-auth smoke-course smoke-assessment demo build

COMPOSE := docker compose -p edulmsv2 -f docker-compose.dev.yml
PROTOC  ?= protoc
PROTO_FILES := $(shell find proto -name '*.proto')

help:
	@echo "EDULMS v2 — available targets:"
	@echo "  tools                   install dev tooling (protoc plugins, golangci-lint)"
	@echo "  proto                   regenerate gRPC stubs"
	@echo "  lint                    golangci-lint across all modules"
	@echo "  build                   go build all binaries (compile check)"
	@echo "  test                    unit tests across all modules"
	@echo "  test-auth               unit tests for auth service"
	@echo "  test-course             unit tests for course service"
	@echo "  test-assessment         unit tests for assessment service (≥70% coverage)"
	@echo "  test-integration        auth integration tests (requires docker)"
	@echo "  test-integration-course course integration tests (requires docker)"
	@echo "  up-web                  start infra + services + Next.js frontend (profile web)"
	@echo "  up                      start infra containers (postgres, redis, nats)"
	@echo "  up-obs                  start infra + observability (otel, prometheus, grafana)"
	@echo "  down                    stop and remove containers"
	@echo "  dev-reset               wipe all infra volumes"
	@echo "  migrate-up              apply pending migrations for all services"
	@echo "  migrate-down            roll back 1 migration for all services"
	@echo "  dev-auth                run auth-svc-v2 locally"
	@echo "  dev-course              run course-svc-v2 locally"
	@echo "  dev-assessment          run assessment-svc-v2 locally"
	@echo "  dev-gateway             run api-gateway locally"
	@echo "  dev-notification        run notification-svc-v2 locally"
	@echo "  smoke-auth              end-to-end smoke test for auth-svc-v2"
	@echo "  smoke-course            end-to-end smoke test for course-svc-v2"
	@echo "  smoke-assessment        end-to-end smoke test for assessment-svc-v2"
	@echo "  demo                    AP4 full defense rehearsal flow"

# ── tooling ───────────────────────────────────────────────────────────────────

tools:
	@echo ">> installing dev tools"
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

proto:
	@command -v $(PROTOC) > /dev/null || { echo "ERROR: protoc not found (brew install protobuf)"; exit 1; }
	$(PROTOC) \
	  --proto_path=proto \
	  --go_out=proto --go_opt=paths=source_relative \
	  --go-grpc_out=proto --go-grpc_opt=paths=source_relative \
	  $(PROTO_FILES)

# ── build ─────────────────────────────────────────────────────────────────────

build:
	go build ./services/auth/... \
	         ./services/course/... \
	         ./services/assessment/... \
	         ./gateway/... \
	         ./notification/... \
	         ./mock-gateway/...

# ── testing ───────────────────────────────────────────────────────────────────

test:
	go test ./services/auth/... ./services/course/... ./services/assessment/...

test-auth:
	go test -cover ./services/auth/...

test-course:
	go test -cover ./services/course/...

test-assessment:
	go test -cover ./services/assessment/...

test-integration:
	go test -tags integration -timeout 120s ./services/auth/internal/repository/postgres/...

test-integration-course:
	go test -tags integration -timeout 120s ./services/course/internal/repository/postgres/...

test-integration-assessment:
	go test -tags integration -timeout 120s ./services/assessment/internal/repository/postgres/...

# ── infra ─────────────────────────────────────────────────────────────────────

up:
	$(COMPOSE) up -d postgres redis nats

up-web:
	$(COMPOSE) --profile web --profile services up -d
	@echo "Web: http://localhost:4000  (Next.js 14)"

up-all: kill-ports
	$(COMPOSE) --profile services up -d

up-obs:
	$(COMPOSE) --profile obs up -d
	@echo "Grafana:    http://localhost:3002  (admin/admin)"
	@echo "Prometheus: http://localhost:9090"
	@echo "Tempo:      http://localhost:3200"
	@echo "Loki:       http://localhost:3100"

## kill any local processes holding gRPC/gateway ports (left by smoke tests or dev servers)
kill-ports:
	@for port in 50051 50052 50053 9080; do \
	  pids=$$(lsof -nP -iTCP:$$port -sTCP:LISTEN 2>/dev/null | awk 'NR>1{print $$2}'); \
	  if [ -n "$$pids" ]; then \
	    echo "killing PID $$pids on :$$port"; \
	    kill -9 $$pids 2>/dev/null || true; \
	  fi; \
	done
	@sleep 0.5

down:
	$(COMPOSE) down

dev-reset:
	$(COMPOSE) down -v
	@echo "infra volumes wiped — next 'make up' will run init.sql"

migrate-up:
	@migrate -path services/auth/migrations       -database "$$DB_URL" up
	@migrate -path services/course/migrations     -database "$$DATABASE_URL_COURSE" up
	@migrate -path services/assessment/migrations -database "$$DATABASE_URL_ASSESSMENT" up

migrate-down:
	@migrate -path services/auth/migrations       -database "$$DB_URL" down 1
	@migrate -path services/course/migrations     -database "$$DATABASE_URL_COURSE" down 1
	@migrate -path services/assessment/migrations -database "$$DATABASE_URL_ASSESSMENT" down 1

# ── dev run ───────────────────────────────────────────────────────────────────

dev-auth:
	@echo ">> starting auth-svc-v2 (set DB_URL, JWT_ACCESS_SECRET, JWT_REFRESH_SECRET in env)"
	go run ./services/auth/cmd/auth

dev-course:
	@echo ">> starting course-svc-v2 (set DATABASE_URL_COURSE, JWT_ACCESS_SECRET in env)"
	go run ./services/course/cmd/course

dev-assessment:
	@echo ">> starting assessment-svc-v2 (set DATABASE_URL_ASSESSMENT, JWT_ACCESS_SECRET in env)"
	go run ./services/assessment/cmd/assessment

dev-gateway:
	@echo ">> starting api-gateway on :9080"
	go run ./gateway/cmd/gateway

dev-notification:
	@echo ">> starting notification-svc-v2 (set NATS_URL in env)"
	go run ./notification/cmd/notification

# ── smoke tests ───────────────────────────────────────────────────────────────

smoke-auth:
	@bash scripts/smoke_auth.sh

smoke-course:
	@bash scripts/smoke_course.sh

smoke-assessment:
	@bash scripts/smoke_assessment.sh

# ── lint ──────────────────────────────────────────────────────────────────────

lint:
	@for svc in services/auth services/course services/assessment gateway notification mock-gateway; do \
		echo ">> lint $$svc"; \
		(cd $$svc && golangci-lint run ./... --timeout 3m) || exit 1; \
	done

# ── defense demo ──────────────────────────────────────────────────────────────

demo:
	@bash scripts/demo.sh
