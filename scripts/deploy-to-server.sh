#!/usr/bin/env bash
#
# Manual deploy from a developer laptop. Mirrors the CI/CD pipeline in
# .github/workflows/ci-cd.yml so anyone on the team can ship a hotfix
# even if GitHub Actions is unavailable.
#
# Implements the whiteboard flow:
#   1. ssh to server
#   2. pull project
#   3. cd project
#   4. run ansible / sh / makefile
#   5. docker build images
#   6. kubectl apply
#   7. health check

set -euo pipefail

HOST="${EDULMS_HOST:-aitbek.tech}"
USER="${EDULMS_USER:-ubuntu}"
PATH_REMOTE="${EDULMS_PATH:-/opt/edulms}"
BRANCH="${EDULMS_BRANCH:-main}"

step() { printf '\n\033[1;36m==> %s\033[0m\n' "$*"; }

step "1/5  SSH to $USER@$HOST"
ssh -o StrictHostKeyChecking=no "$USER@$HOST" "echo connected as \$(whoami) on \$(hostname)"

step "2/5  Pull latest code (branch=$BRANCH)"
ssh "$USER@$HOST" bash -se <<EOF
  set -euo pipefail
  cd $PATH_REMOTE
  git fetch --all
  git checkout $BRANCH
  git reset --hard origin/$BRANCH
  git log -1 --oneline
EOF

step "3/5  Run Ansible deploy + monitoring playbooks"
ssh "$USER@$HOST" bash -se <<EOF
  set -euo pipefail
  cd $PATH_REMOTE/sre/ansible
  ansible-playbook -i inventory.ini site.yml --tags deploy,monitor
EOF

step "4/5  Docker compose up / kubectl apply"
ssh "$USER@$HOST" bash -se <<EOF
  set -euo pipefail
  cd $PATH_REMOTE
  if command -v kubectl >/dev/null 2>&1; then
    kubectl apply -f sre/k8s/
    for d in auth course assessment notification payment user-profile gateway; do
      kubectl rollout restart deploy/\$d -n edulms || true
    done
  else
    docker compose -p edulmsv2 \
      -f docker-compose.dev.yml -f sre/docker-compose.sre.yml \
      --profile services --profile obs up -d --build
  fi
EOF

step "5/5  Health check"
sleep 10
for url in \
    "https://$HOST/health" \
    "https://$HOST/api/payments/" \
    "https://$HOST/api/profiles/"; do
  printf '  GET %-45s ' "$url"
  if curl -fsS --max-time 8 "$url" >/dev/null; then
    echo "✓"
  else
    echo "✗"; exit 1
  fi
done

printf '\n\033[1;32m✓ Deployment to %s complete.\033[0m\n' "$HOST"
