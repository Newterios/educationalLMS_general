#!/usr/bin/env bash
# Smoke test for auth-svc-v2 (Phase 1D).
# Prerequisites: grpcurl, docker, go 1.25+
# Usage:
#   make smoke-auth
#   # or directly:
#   DB_URL=postgres://... JWT_ACCESS_SECRET=... JWT_REFRESH_SECRET=... bash scripts/smoke_auth.sh

set -euo pipefail

HOST="127.0.0.1:50051"
COMPOSE="docker compose -p edulmsv2 -f docker-compose.dev.yml"

# ── env defaults for local dev ────────────────────────────────────────────────
export DB_URL="${DB_URL:-postgres://edulms:edulms@127.0.0.1:5433/auth_v2?sslmode=disable}"
export JWT_ACCESS_SECRET="${JWT_ACCESS_SECRET:-smoke-access-secret-dev}"
export JWT_REFRESH_SECRET="${JWT_REFRESH_SECRET:-smoke-refresh-secret-dev}"
export GRPC_PORT="${GRPC_PORT:-50051}"
export MIGRATIONS_DIR="${MIGRATIONS_DIR:-services/auth/migrations}"

# ── guards ────────────────────────────────────────────────────────────────────
for cmd in grpcurl docker go; do
  command -v "$cmd" >/dev/null || { echo "ERROR: $cmd not found in PATH"; exit 1; }
done

# Unique email per run so re-runs don't collide on ErrAlreadyExists
SMOKE_EMAIL="smoke.$(date +%s)@example.com"

AUTH_PID=""

# Pre-flight: release the port if a leftover process is holding it
lsof -ti :"$GRPC_PORT" 2>/dev/null | xargs kill -9 2>/dev/null || true

cleanup() {
  echo ""
  echo "── cleanup ────────────────────────────────────────────────────────────"
  if [[ -n "$AUTH_PID" ]] && kill -0 "$AUTH_PID" 2>/dev/null; then
    kill "$AUTH_PID" 2>/dev/null
    wait "$AUTH_PID" 2>/dev/null || true
    echo "auth-svc stopped (pid $AUTH_PID)"
  fi
  $COMPOSE stop postgres 2>/dev/null || true
  echo "done."
}
trap cleanup EXIT

# ── 1. infra ──────────────────────────────────────────────────────────────────
$COMPOSE down -v >/dev/null 2>&1 || true
echo "── starting postgres ──────────────────────────────────────────────────"
$COMPOSE up -d postgres
echo "waiting for postgres to be healthy..."
for i in $(seq 1 30); do
  $COMPOSE exec -T postgres pg_isready -U edulms >/dev/null 2>&1 && break
  sleep 1
done

# ── 2. auth binary ────────────────────────────────────────────────────────────
echo "── starting auth-svc-v2 ───────────────────────────────────────────────"
go run ./services/auth/cmd/auth &
AUTH_PID=$!

echo "waiting for health check (pid $AUTH_PID)..."
for i in $(seq 1 30); do
  if grpcurl -plaintext "$HOST" grpc.health.v1.Health/Check >/dev/null 2>&1; then
    echo "health check passed"
    break
  fi
  if ! kill -0 "$AUTH_PID" 2>/dev/null; then
    echo "ERROR: auth-svc-v2 exited unexpectedly"
    exit 1
  fi
  sleep 1
done

grpcurl -plaintext "$HOST" grpc.health.v1.Health/Check | grep -q "SERVING" \
  || { echo "ERROR: health check not SERVING"; exit 1; }

# ── helpers ───────────────────────────────────────────────────────────────────
# grpcurl emits camelCase JSON (protobuf JSON mapping).
grpc() {
  local method="$1"; shift
  grpcurl -plaintext "$@" "$HOST" "auth.v1.AuthService/$method"
}

# extract VALUE from  "camelKey": "VALUE"
jval() { echo "$1" | grep -o "\"$2\": *\"[^\"]*\"" | head -1 | sed 's/.*": *"\(.*\)"/\1/'; }

pass() { echo "  ✓ $1"; }
fail() { echo "  ✗ $1"; exit 1; }

# ── 3. smoke sequence ─────────────────────────────────────────────────────────
echo ""
echo "── smoke sequence ─────────────────────────────────────────────────────"

# Step 1: Register
echo "1. Register"
REGISTER=$(grpc Register -d "{\"email\":\"$SMOKE_EMAIL\",\"password\":\"smoke1234\",\"full_name\":\"Smoke\",\"locale\":\"en\"}")
jval "$REGISTER" "userId" | grep -q "." || fail "Register: no userId in response"
pass "Register returned userId"

# Step 2: Login
echo "2. Login"
LOGIN=$(grpc Login -d "{\"email\":\"$SMOKE_EMAIL\",\"password\":\"smoke1234\"}")
ACCESS=$(jval "$LOGIN" "accessToken")
REFRESH=$(jval "$LOGIN" "refreshToken")
[[ -n "$ACCESS" ]]  || fail "Login: no accessToken"
[[ -n "$REFRESH" ]] || fail "Login: no refreshToken"
pass "Login returned tokens"

AUTH_HEADER="authorization: Bearer $ACCESS"

# Step 3: GetMe (authenticated)
echo "3. GetMe"
GETME=$(grpc GetMe -H "$AUTH_HEADER" -d '{}')
echo "$GETME" | grep -q "$SMOKE_EMAIL" || fail "GetMe: email not found"
pass "GetMe returned correct email"

# Step 4: ListSessions
echo "4. ListSessions"
SESSIONS=$(grpc ListSessions -H "$AUTH_HEADER" -d '{}')
# Response is {"sessions":[...]} — non-empty list has at least one id
echo "$SESSIONS" | grep -qE '"sessions"|"id"' || fail "ListSessions: unexpected response"
pass "ListSessions returned sessions"

# Step 5: RefreshToken
echo "5. RefreshToken"
REFRESH_RESP=$(grpc RefreshToken -d "{\"refreshToken\":\"$REFRESH\"}")
NEW_ACCESS=$(jval "$REFRESH_RESP" "accessToken")
NEW_REFRESH=$(jval "$REFRESH_RESP" "refreshToken")
[[ -n "$NEW_ACCESS" ]] || fail "RefreshToken: no new accessToken"
pass "RefreshToken returned new tokens"

# Step 6: Logout (revoke the original refresh token — idempotent)
echo "6. Logout"
grpc Logout -d "{\"refreshToken\":\"$REFRESH\"}" >/dev/null
pass "Logout succeeded"

# Step 7: GetMe with new access token (issued in step 5) — must still work
echo "7. GetMe with new access token (post-logout)"
NEW_AUTH="authorization: Bearer $NEW_ACCESS"
GETME2=$(grpc GetMe -H "$NEW_AUTH" -d '{}')
echo "$GETME2" | grep -q "$SMOKE_EMAIL" || fail "GetMe post-refresh: email not found"
pass "GetMe with new access token works"

echo ""
echo "══════════════════════════════════════════════════════════════════════"
echo "  ALL SMOKE CHECKS PASSED"
echo "══════════════════════════════════════════════════════════════════════"
