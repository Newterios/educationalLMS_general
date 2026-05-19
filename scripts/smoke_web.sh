#!/usr/bin/env bash
# Smoke test: verify the Next.js web server is alive.
set -euo pipefail

WEB_URL="${WEB_URL:-http://localhost:4000}"
MAX_WAIT=60   # seconds

echo "Waiting for web at ${WEB_URL} (up to ${MAX_WAIT}s)..."
for i in $(seq 1 $MAX_WAIT); do
  if curl -sf -o /dev/null -w "" "${WEB_URL}/" 2>/dev/null; then
    break
  fi
  sleep 1
done

STATUS=$(curl -s -o /dev/null -w "%{http_code}" "${WEB_URL}/")
BODY=$(curl -s "${WEB_URL}/" | wc -c | tr -d ' ')

if [ "$STATUS" = "200" ] && [ "$BODY" -gt 100 ]; then
  echo "✓ PASS — web server alive: HTTP ${STATUS}, body ${BODY} bytes"
  exit 0
else
  echo "✗ FAIL — HTTP ${STATUS}, body ${BODY} bytes"
  exit 1
fi
