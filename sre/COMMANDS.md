# EduLMS SRE — Commands Reference

> All commands run from the **repository root** (`/opt/edulms` on server, `~/Desktop/final_apt_eduLms` locally).

---

## Makefile

**Files:** `Makefile` (root), `sre/Makefile` (SRE targets)
**Run:** `make -C sre <target>`

| Command | What it does |
|---------|-------------|
| `make -C sre help` | List all targets |
| `make -C sre up` | Build images + start all services (compose) |
| `make -C sre down` | Stop and remove all containers + volumes |
| `make -C sre build` | Build `payment` and `user-profile` Docker images |
| `make -C sre logs` | Tail logs of all running containers |
| `make -C sre ps` | List running services |
| `make -C sre ansible` | Run full Ansible site.yml playbook |
| `make -C sre k8s-apply` | Deploy all manifests to Kubernetes |
| `make -C sre k8s-delete` | Remove all K8s resources |
| `make -C sre swarm-up` | Init Swarm + deploy docker stack |
| `make -C sre swarm-down` | Remove docker stack |
| `make -C sre tf-aws-init` | Terraform: download AWS provider |
| `make -C sre tf-aws-plan` | Terraform: preview what will be created |
| `make -C sre tf-aws-apply` | Terraform: create AWS infrastructure |
| `make -C sre tf-local-up` | Terraform: spin up local docker environment |
| `make -C sre tf-local-down` | Terraform: destroy local docker environment |
| `make -C sre incident-on` | Inject 100% failure rate into payment service |
| `make -C sre incident-off` | Restore payment to 2% failure rate |

### Demo targets (whiteboard flow)

| Command | Step | What it does |
|---------|------|-------------|
| `make -C sre demo-destroy` | 0 | Stop everything, wipe all containers + volumes |
| `make -C sre demo-pull` | 1 | `git fetch && git reset --hard origin/main` |
| `make -C sre demo-ansible` | 2 | Run Ansible deploy tag |
| `make -C sre demo-docker-build` | 3 | Build all Docker images |
| `make -C sre demo-k8s-up` | 4 | `kubectl apply -f sre/k8s/` + wait for pods |
| `make -C sre demo-swarm-up` | 4b | `docker stack deploy` (Swarm alternative) |
| `make -C sre demo-health` | 5 | Curl all endpoints, report pass/fail |
| `make -C sre demo-full` | 0–5 | Run all demo steps in sequence |

---

## Docker Compose

**Files:** `docker-compose.dev.yml` + `sre/docker-compose.sre.yml`

```bash
# Start everything (services + observability + SRE microservices)
docker compose -p edulmsv2 \
  -f docker-compose.dev.yml \
  -f sre/docker-compose.sre.yml \
  --profile services --profile obs \
  up -d --build

# Stop everything
docker compose -p edulmsv2 \
  -f docker-compose.dev.yml \
  -f sre/docker-compose.sre.yml \
  down -v --remove-orphans

# Tail logs of one service
docker compose -p edulmsv2 -f docker-compose.dev.yml logs -f gateway

# Rebuild and restart one service only
docker compose -p edulmsv2 -f docker-compose.dev.yml \
  -f sre/docker-compose.sre.yml \
  --profile services --profile obs \
  up -d --no-deps --build payment

# Restart nginx after config change
docker restart edulms-sre-nginx
```

**Profiles:**

| Profile | Services started |
|---------|----------------|
| `services` | auth, course, assessment, notification, gateway, postgres, redis, nats |
| `obs` | prometheus, grafana, loki, tempo, otel-collector |
| *(no profile)* | payment, user-profile, sre-web, nginx-frontend (always on in SRE compose) |

---

## Ansible

**Files:**
```
sre/ansible/
  site.yml               # master playbook
  inventory.ini          # target hosts
  ansible.cfg            # config (stdout_callback, remote_user)
  roles/
    common/tasks/main.yml   # packages, timezone, firewall
    docker/tasks/main.yml   # install Docker + Compose plugin
    swarm/tasks/main.yml    # docker swarm init / join
    deploy/tasks/main.yml   # git pull + docker compose up
    monitoring/tasks/main.yml # Prometheus + Grafana setup
```

```bash
# Run full playbook
ansible-playbook -i sre/ansible/inventory.ini sre/ansible/site.yml

# Run only deploy role (used by CI/CD)
ansible-playbook -i sre/ansible/inventory.ini sre/ansible/site.yml --tags deploy

# Run only docker install
ansible-playbook -i sre/ansible/inventory.ini sre/ansible/site.yml --tags docker

# Dry run (check mode — no changes)
ansible-playbook -i sre/ansible/inventory.ini sre/ansible/site.yml --check

# Run against one host only
ansible-playbook -i sre/ansible/inventory.ini sre/ansible/site.yml --limit managers

# Test connectivity
ansible -i sre/ansible/inventory.ini all -m ping
```

---

## Kubernetes

**Files:** `sre/k8s/`

| File | What it creates |
|------|----------------|
| `00-namespace.yaml` | `edulms` namespace |
| `01-config-and-secrets.yaml` | ConfigMap + Secrets (DB creds, JWT key) |
| `02-postgres.yaml` | StatefulSet: PostgreSQL |
| `03-redis-nats.yaml` | Deployments: Redis + NATS |
| `04-auth.yaml` | Deployment + Service: auth gRPC :50051 |
| `05-course.yaml` | Deployment + Service: course gRPC :50052 |
| `06-assessment.yaml` | Deployment + Service: assessment gRPC :50053 |
| `07-notification.yaml` | Deployment + Service: notification |
| `08-payment.yaml` | Deployment + Service: payment :8081 |
| `09-user-profile.yaml` | Deployment + Service: user-profile :8082 |
| `10-gateway-and-ingress.yaml` | Deployment + Service + Ingress: gateway :9080 |
| `11-monitoring.yaml` | Deployment: Prometheus + Grafana |

```bash
# Apply all manifests
kubectl apply -f sre/k8s/

# Apply one file
kubectl apply -f sre/k8s/04-auth.yaml

# Watch pods come up
kubectl -n edulms get pods -w

# Get all resources in namespace
kubectl -n edulms get all

# Logs of a pod
kubectl -n edulms logs -f deployment/auth

# Describe pod (events, errors)
kubectl -n edulms describe pod <pod-name>

# Shell into a pod
kubectl -n edulms exec -it deployment/gateway -- sh

# Scale a deployment
kubectl -n edulms scale deployment/payment --replicas=3

# Rolling restart
kubectl -n edulms rollout restart deployment/payment

# Delete all resources
kubectl delete -f sre/k8s/ --ignore-not-found
```

---

## Terraform

**Files:**
```
sre/terraform/
  aws/
    providers.tf   # AWS provider config
    variables.tf   # region, instance_type, node_count, etc.
    main.tf        # VPC, subnets, security group, EC2 instances
    outputs.tf     # public IPs, Ansible inventory hint
  local/
    main.tf        # docker provider — local containers via Terraform
```

**AWS infrastructure provisions:**
- VPC `10.20.0.0/16` + Internet Gateway
- 2 public subnets across 2 AZs
- Security group (ports: 22, 80, 8081-8082, 9080, 3000-9090)
- N × EC2 `t3.medium` Ubuntu instances (default: 2)

```bash
# Init (download provider plugins)
terraform -chdir=sre/terraform/aws init

# Preview changes
terraform -chdir=sre/terraform/aws plan

# Apply (create resources on AWS)
terraform -chdir=sre/terraform/aws apply

# Apply without interactive prompt
terraform -chdir=sre/terraform/aws apply -auto-approve

# Override a variable
terraform -chdir=sre/terraform/aws apply -var="node_count=3" -var="instance_type=t3.large"

# Show current state
terraform -chdir=sre/terraform/aws show

# Get outputs (EC2 IPs)
terraform -chdir=sre/terraform/aws output node_public_ips

# Destroy everything
terraform -chdir=sre/terraform/aws destroy -auto-approve

# Local docker provider
terraform -chdir=sre/terraform/local init
terraform -chdir=sre/terraform/local apply -auto-approve
terraform -chdir=sre/terraform/local destroy -auto-approve
```

---

## Scripts

**Files:** `scripts/`, `sre/scripts/`

### `scripts/setup-server.sh` — one-time server bootstrap
```bash
# Installs: Docker, Docker Compose, kubectl, Ansible
# Clones the repo to /opt/edulms on the remote server
SSH_KEY=~/Downloads/conection.pem bash scripts/setup-server.sh
```

### `scripts/deploy-to-server.sh` — manual deploy (no CI/CD needed)
```bash
# Mirrors the GitHub Actions pipeline:
# SSH → git pull → ansible-playbook → docker compose up → health check
SSH_KEY=~/Downloads/conection.pem bash scripts/deploy-to-server.sh

# Override host/branch
EDULMS_HOST=1.2.3.4 EDULMS_BRANCH=develop bash scripts/deploy-to-server.sh
```

### `sre/scripts/generate-traffic.sh` — populate Prometheus metrics
```bash
# Send requests to live server (default: 60 rounds × 2s)
bash sre/scripts/generate-traffic.sh

# Custom target / rounds / interval
bash sre/scripts/generate-traffic.sh https://sre.aitbek.tech 30 1

# Local
bash sre/scripts/generate-traffic.sh http://localhost:8080 20 2
```

---

## CI/CD Pipeline

**File:** `.github/workflows/ci-cd.yml`
**Triggers:** push to `main` or `develop`, PR to `main`

```
lint-and-test  →  build-images  →  deploy  →  health-check
(5 services       (8 images →       (SSH →      (curl 3
 in parallel)      ghcr.io)          ansible +    endpoints)
                                     compose up)
```

```bash
# Trigger manually via GitHub CLI
gh workflow run ci-cd.yml

# View latest run
gh run list --workflow=ci-cd.yml --limit=5

# Watch a run live
gh run watch
```

---

## Live URLs

| URL | Service |
|-----|---------|
| `https://sre.aitbek.tech` | Next.js frontend |
| `https://sre.aitbek.tech/health` | Nginx health check |
| `https://sre.aitbek.tech/api/payments/health` | Payment service |
| `https://sre.aitbek.tech/api/profiles/health` | User Profile service |
| `https://sre.aitbek.tech/api/v1/auth/register` | Auth (via gateway) |
| `https://sre.aitbek.tech/monitoring/grafana` | Grafana (admin / EduLMS@SRE2026) |
| `https://sre.aitbek.tech/monitoring/prometheus` | Prometheus |

---

## Quick SSH to server

```bash
ssh -i ~/Downloads/conection.pem ubuntu@sre.aitbek.tech

# Check running containers
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# Check logs on server
docker logs edulms-sre-payment --tail=50 -f
docker logs edulmsv2-grafana --tail=50 -f

# Manual compose up on server
cd /opt/edulms
docker compose -p edulmsv2 \
  -f docker-compose.dev.yml \
  -f sre/docker-compose.sre.yml \
  --profile services --profile obs \
  up -d --build --remove-orphans
```
