# EduLMS — End Term Project: Comprehensive SRE Implementation

End-to-end implementation of Site Reliability Engineering practices on
top of the EduLMS distributed microservices system, using Docker
Compose, **Docker Swarm**, **Kubernetes**, **Terraform**, **Ansible**,
**Prometheus** and **Grafana**.

> The base application (Go gRPC microservices + Next.js front-end)
> lives in the repository root. Everything that is specifically about
> SRE — extra services, orchestration manifests, IaC, runbooks,
> dashboards — lives in this `sre/` folder.

---

## 1. Mapping to the project specification

| Spec section                              | Where it is implemented                                                |
|-------------------------------------------|-------------------------------------------------------------------------|
| ≥ 6 microservices                         | `services/{auth,course,assessment}` + `notification/` + `sre/services/{payment,user-profile}` |
| Frontend (Nginx)                          | `sre/nginx/default.conf` (+ existing `web/`)                            |
| Database (PostgreSQL)                     | `docker-compose.dev.yml`, `sre/k8s/02-postgres.yaml`                    |
| Message broker (NATS) + cache (Redis)     | Compose, Swarm, K8s                                                     |
| Assignment 1 — Docker environment + Compose | `docker-compose.dev.yml`, `sre/docker-compose.sre.yml`                |
| Assignment 2 — SLI / SLO design           | `sre/docs/SLI_SLO.md`                                                  |
| Assignment 3 — Prometheus + Grafana       | `sre/monitoring/`, `observability/`                                    |
| Midterm — initial microservices           | `services/`, `notification/`, `gateway/`                                |
| Assignment 4 — Incident response          | `sre/docs/INCIDENT.md` + `sre/docs/POSTMORTEM.md`                       |
| Assignment 5 — Terraform IaC              | `sre/terraform/{aws,local}/`                                            |
| Assignment 6 — Automation + capacity      | `sre/ansible/`, `sre/k8s/*-hpa`, `sre/docs/CAPACITY_PLANNING.md`        |
| 6.1 Docker Swarm                          | `sre/swarm/docker-stack.yml`                                            |
| 6.2 Kubernetes                            | `sre/k8s/`                                                              |
| Observability                             | `observability/`, `sre/monitoring/`                                     |

---

## 2. Repository layout (SRE bits only)

```
sre/
├── README.md                ← this file
├── docker-compose.sre.yml   ← adds payment + user-profile + nginx
├── nginx/
│   └── default.conf
├── services/                ← new Python microservices
│   ├── payment/
│   └── user-profile/
├── swarm/                   ← Docker Swarm stack
│   ├── docker-stack.yml
│   └── README.md
├── k8s/                     ← Kubernetes manifests (11 files)
│   ├── 00-namespace.yaml
│   ├── 01-config-and-secrets.yaml
│   ├── 02-postgres.yaml
│   ├── 03-redis-nats.yaml
│   ├── 04-auth.yaml
│   ├── 05-course.yaml
│   ├── 06-assessment.yaml
│   ├── 07-notification.yaml
│   ├── 08-payment.yaml
│   ├── 09-user-profile.yaml
│   ├── 10-gateway-and-ingress.yaml
│   ├── 11-monitoring.yaml
│   └── README.md
├── terraform/
│   ├── aws/                 ← VPC + EC2 + SG  (Assignment 5)
│   ├── local/               ← Docker provider (no cloud needed)
│   └── README.md
├── ansible/
│   ├── ansible.cfg
│   ├── inventory.ini
│   ├── site.yml
│   ├── roles/{common,docker,swarm,deploy,monitoring}/
│   └── README.md
├── monitoring/
│   ├── prometheus.yml
│   ├── alerts/slo-alerts.yml
│   └── dashboards/edulms-sre-overview.json
└── docs/
    ├── ARCHITECTURE.md
    ├── SLI_SLO.md
    ├── INCIDENT.md
    ├── POSTMORTEM.md
    └── CAPACITY_PLANNING.md
```

---

## 3. Quick start (local Docker)

```bash
# 1. Bring up base infra + the new SRE services.
docker compose -f docker-compose.dev.yml -f sre/docker-compose.sre.yml \
    --profile services --profile obs up -d --build

# 2. Probe the new services:
curl http://localhost:8081/                       # payment index
curl -X POST http://localhost:8081/pay \
     -H 'Content-Type: application/json' \
     -d '{"amount":1500,"currency":"USD","order_id":"o-42"}'
curl -X PUT http://localhost:8082/profiles/u-1 \
     -H 'Content-Type: application/json' \
     -d '{"display_name":"Rent","preferences":{"theme":"dark"}}'

# 3. Visit:
#    - Prometheus  http://localhost:9090
#    - Grafana     http://localhost:3002  (admin / admin)
#    - Frontend    http://localhost:8080
```

## 4. Docker Swarm

```bash
docker swarm init
docker stack deploy -c sre/swarm/docker-stack.yml edulms
docker stack services edulms
```

## 5. Kubernetes

```bash
# Apply manifests in order.
kubectl apply -f sre/k8s/
kubectl get pods -n edulms
kubectl get hpa  -n edulms
```

Full instructions in `sre/k8s/README.md`.

## 6. Terraform

```bash
# AWS:
cd sre/terraform/aws
cp terraform.tfvars.example terraform.tfvars
terraform init && terraform apply

# Local (no cloud cost):
cd sre/terraform/local
terraform init && terraform apply
```

## 7. Ansible

```bash
cd sre/ansible
# edit inventory.ini with the IPs printed by Terraform
ansible-playbook site.yml
```

---

## 8. SRE artefacts

* **SLIs / SLOs** — `sre/docs/SLI_SLO.md`
* **Alerts**     — `sre/monitoring/alerts/slo-alerts.yml`
* **Dashboard**  — `sre/monitoring/dashboards/edulms-sre-overview.json`
* **Incident**   — `sre/docs/INCIDENT.md`
* **Postmortem** — `sre/docs/POSTMORTEM.md` (Google blameless format)
* **Capacity**   — `sre/docs/CAPACITY_PLANNING.md`
* **Arch diagram** — `sre/docs/ARCHITECTURE.md`

## 9. Reproducing the simulated incident

```bash
# Crank the payment failure rate to 100% to reproduce the SEV-2.
docker compose -f sre/docker-compose.sre.yml \
    up -d --build --no-deps -e FAILURE_RATE=1.0 payment

# Generate traffic:
hey -c 10 -q 5 -z 30s -m POST \
    -H 'Content-Type: application/json' \
    -d '{"amount":100,"order_id":"o-x"}' \
    http://localhost:8081/pay

# Watch Prometheus fire PaymentErrorRateAboveSLO.

# Mitigate (the runbook from POSTMORTEM.md):
docker compose -f sre/docker-compose.sre.yml \
    up -d --build --no-deps -e FAILURE_RATE=0.02 payment
```

## 10. Deliverables checklist (per the project PDF)

- [x] Microservices source code (6+)
- [x] Docker Compose / Swarm configuration
- [x] Kubernetes manifests
- [x] Terraform files
- [x] Ansible playbooks
- [x] Monitoring setup (Prometheus + Grafana + alerts + dashboard)
- [x] Incident report and postmortem
- [x] Screenshots / demo evidence — see `sre/docs/` and Grafana once deployed
- [ ] Final PDF with Git link — submit `End_Term_Submission.pdf`
      (template under `sre/docs/`), with the link to **your own** Git repo.
