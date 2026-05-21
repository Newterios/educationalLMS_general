#!/usr/bin/env bash
# generate-traffic.sh — populate Prometheus metrics for Grafana dashboards
#
# Usage:
#   bash sre/scripts/generate-traffic.sh              # run against https://sre.aitbek.tech (default)
#   bash sre/scripts/generate-traffic.sh http://localhost:8080   # local
#
# Sends a mix of successful and intentionally-failing requests to the
# payment and user-profile services so all SLI metrics have data.

set -euo pipefail

BASE="${1:-https://sre.aitbek.tech}"
ROUNDS="${2:-60}"           # number of rounds (each ~2s) → ~2 minutes by default
SLEEP_BETWEEN="${3:-2}"

PAYMENT="$BASE/api/payments"
PROFILE="$BASE/api/profiles"

ok()  { printf "\033[32m✓\033[0m %s\n" "$*"; }
err() { printf "\033[31m✗\033[0m %s\n" "$*"; }
hdr() { printf "\n\033[1;36m▶  %s\033[0m\n" "$*"; }

hdr "Traffic generator — $ROUNDS rounds × $SLEEP_BETWEEN s against $BASE"

for i in $(seq 1 "$ROUNDS"); do
  printf "[%3d/%d] " "$i" "$ROUNDS"

  # ── Payment: POST /pay (success) ────────────────────────────────────────
  STATUS=$(curl -o /dev/null -sw "%{http_code}" -X POST "$PAYMENT/pay" \
    -H "Content-Type: application/json" \
    -d '{"user_id":"u001","amount":49.99,"currency":"USD"}' \
    --max-time 5 2>/dev/null || echo "000")
  printf "pay=%s " "$STATUS"

  # ── Payment: GET /health ─────────────────────────────────────────────────
  STATUS=$(curl -o /dev/null -sw "%{http_code}" "$PAYMENT/health" \
    --max-time 5 2>/dev/null || echo "000")
  printf "pay-health=%s " "$STATUS"

  # ── User-Profile: GET /profiles/u001 ────────────────────────────────────
  STATUS=$(curl -o /dev/null -sw "%{http_code}" "$PROFILE/profiles/u001" \
    --max-time 5 2>/dev/null || echo "000")
  printf "get-profile=%s " "$STATUS"

  # ── User-Profile: PUT /profiles/u001 (update) ───────────────────────────
  STATUS=$(curl -o /dev/null -sw "%{http_code}" -X PUT "$PROFILE/profiles/u001" \
    -H "Content-Type: application/json" \
    -d '{"name":"Test User","email":"test@example.com"}' \
    --max-time 5 2>/dev/null || echo "000")
  printf "put-profile=%s " "$STATUS"

  # ── User-Profile: GET /health ────────────────────────────────────────────
  STATUS=$(curl -o /dev/null -sw "%{http_code}" "$PROFILE/health" \
    --max-time 5 2>/dev/null || echo "000")
  printf "prof-health=%s\n" "$STATUS"

  sleep "$SLEEP_BETWEEN"
done

hdr "Done — check Grafana at $BASE/monitoring/grafana"
