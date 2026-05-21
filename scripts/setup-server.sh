#!/usr/bin/env bash
#
# One-time server bootstrap — run this ONCE before the first demo.
# Installs: Docker, Docker Compose, kubectl, Ansible, and clones the project.
#
# Usage:
#   chmod +x scripts/setup-server.sh
#   SSH_KEY=/Users/aitbek/Downloads/conection.pem bash scripts/setup-server.sh

set -euo pipefail

SSH_KEY="${SSH_KEY:-$HOME/Downloads/conection.pem}"
HOST="${EDULMS_HOST:-13.63.140.216}"
USER="${EDULMS_USER:-ubuntu}"
DEPLOY_PATH="${EDULMS_PATH:-/opt/edulms}"
REPO_URL="${EDULMS_REPO:-https://github.com/Newterios/lms-system-prob.git}"
BRANCH="${EDULMS_BRANCH:-main}"

SSH="ssh -i $SSH_KEY -o StrictHostKeyChecking=no $USER@$HOST"

step() { printf '\n\033[1;36m══ %s\033[0m\n' "$*"; }
ok()   { printf '\033[1;32m  ✓ %s\033[0m\n' "$*"; }

step "Testing SSH connection"
$SSH "echo connected as \$(whoami) on \$(hostname) [\$(uname -r)]"
ok "SSH works"

step "Installing Docker + Docker Compose"
$SSH bash -s << 'ENDSSH'
  set -euo pipefail
  if ! command -v docker &>/dev/null; then
    curl -fsSL https://get.docker.com | sh
    sudo usermod -aG docker ubuntu
    newgrp docker || true
  fi
  docker --version
  docker compose version
ENDSSH
ok "Docker ready"

step "Installing kubectl"
$SSH bash -s << 'ENDSSH'
  set -euo pipefail
  if ! command -v kubectl &>/dev/null; then
    curl -LO "https://dl.k8s.io/release/$(curl -sL https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    chmod +x kubectl
    sudo mv kubectl /usr/local/bin/kubectl
  fi
  kubectl version --client
ENDSSH
ok "kubectl ready"

step "Installing k3s (lightweight Kubernetes)"
$SSH bash -s << 'ENDSSH'
  set -euo pipefail
  if ! command -v k3s &>/dev/null; then
    curl -sfL https://get.k3s.io | sh -
    sleep 10
  fi
  # Make kubectl work without sudo
  mkdir -p ~/.kube
  sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
  sudo chown ubuntu:ubuntu ~/.kube/config
  kubectl get nodes
ENDSSH
ok "k3s ready"

step "Installing Ansible"
$SSH bash -s << 'ENDSSH'
  set -euo pipefail
  if ! command -v ansible &>/dev/null; then
    sudo apt-get update -q
    sudo apt-get install -y ansible
  fi
  ansible --version | head -1
ENDSSH
ok "Ansible ready"

step "Cloning / updating project at $DEPLOY_PATH"
$SSH bash -s <<ENDSSH
  set -euo pipefail
  if [ -d "$DEPLOY_PATH/.git" ]; then
    cd "$DEPLOY_PATH"
    git fetch --all
    git reset --hard origin/$BRANCH
  else
    sudo mkdir -p "$DEPLOY_PATH"
    sudo chown ubuntu:ubuntu "$DEPLOY_PATH"
    git clone "$REPO_URL" "$DEPLOY_PATH"
    cd "$DEPLOY_PATH"
    git checkout "$BRANCH"
  fi
  git log -1 --oneline
ENDSSH
ok "Project at $DEPLOY_PATH is up to date"

step "Copying SSH key for Ansible"
$SSH bash -s << 'ENDSSH'
  mkdir -p ~/.ssh
  # Ansible uses the same key we SSH in with; just make sure the known_hosts are set
  ssh-keyscan -H 13.63.140.216 >> ~/.ssh/known_hosts 2>/dev/null || true
  ssh-keyscan -H localhost     >> ~/.ssh/known_hosts 2>/dev/null || true
ENDSSH

# Copy the PEM key to the server for local Ansible runs
scp -i "$SSH_KEY" -o StrictHostKeyChecking=no \
    "$SSH_KEY" "$USER@$HOST:~/.ssh/edulms-sre-key.pem"
$SSH "chmod 600 ~/.ssh/edulms-sre-key.pem"
ok "SSH key copied"

printf '\n\033[1;32m'
echo '╔══════════════════════════════════════════════════╗'
echo '║  Server bootstrap COMPLETE                       ║'
echo '║                                                  ║'
echo '║  Next steps:                                     ║'
echo "║  1. ssh -i $SSH_KEY $USER@$HOST"
echo "║  2. cd $DEPLOY_PATH"
echo '║  3. make -C sre demo-destroy   # clean slate     ║'
echo '║  4. make -C sre demo-full      # full SRE demo   ║'
echo '╚══════════════════════════════════════════════════╝'
printf '\033[0m\n'
