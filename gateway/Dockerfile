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

# Copy only what's needed to build gateway
COPY proto/ ./proto/
COPY gateway/ ./gateway/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/gateway ./gateway/cmd/gateway

# ── runtime ────────────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
USER app

COPY --from=builder /out/gateway ./gateway

EXPOSE 9080
ENTRYPOINT ["./gateway"]
