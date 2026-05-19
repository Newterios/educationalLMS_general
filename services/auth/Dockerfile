# ── Stage 1: build ───────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /workspace

# go.work + all go.mod/go.sum files first (layer cache for dependency download)
COPY go.work go.work.sum ./
COPY proto/go.mod proto/go.sum ./proto/
COPY gateway/go.mod gateway/go.sum ./gateway/
COPY mock-gateway/go.mod mock-gateway/go.sum ./mock-gateway/
COPY notification/go.mod notification/go.sum ./notification/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/
COPY services/course/go.mod services/course/go.sum ./services/course/
COPY services/assessment/go.mod services/assessment/go.sum ./services/assessment/

RUN go work sync

# Copy source for this service
COPY proto/ ./proto/
COPY services/auth/ ./services/auth/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/auth ./services/auth/cmd/auth

# ── Stage 2: runtime ─────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/auth /auth
COPY services/auth/migrations/ /migrations/

EXPOSE 50051

ENV MIGRATIONS_DIR=/migrations

ENTRYPOINT ["/auth"]
