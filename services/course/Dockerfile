FROM golang:1.25-alpine AS builder

WORKDIR /workspace

# go.work + ALL go.mod/go.sum first (layer cache + workspace resolution)
COPY go.work go.work.sum ./
COPY proto/go.mod proto/go.sum ./proto/
COPY gateway/go.mod gateway/go.sum ./gateway/
COPY mock-gateway/go.mod mock-gateway/go.sum ./mock-gateway/
COPY notification/go.mod notification/go.sum ./notification/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/
COPY services/course/go.mod services/course/go.sum ./services/course/
COPY services/assessment/go.mod services/assessment/go.sum ./services/assessment/

RUN go work sync

# Copy only what's needed to build course-svc
COPY proto/ ./proto/
COPY services/course/ ./services/course/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/course-svc ./services/course/cmd/course

# ── runtime ────────────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
USER app

COPY --from=builder /out/course-svc ./course-svc
COPY services/course/migrations /migrations

ENV COURSE_MIGRATIONS_DIR=/migrations

EXPOSE 50052
ENTRYPOINT ["./course-svc"]
