# Architecture — EduLMS SRE End Term Project

The system is a small but realistic LMS with six independent
microservices, supporting infrastructure (DB, cache, broker), an edge
gateway, and a full observability + automation stack.

## High-level diagram

```
                ┌─────────────────────┐
                │  Learners / Admins  │
                └──────────┬──────────┘
                           │
              ┌────────────▼────────────┐
              │  Nginx (TLS / routing)  │  ← sre/nginx/
              └────────────┬────────────┘
                           │
              ┌────────────▼────────────┐
              │       API Gateway       │  ← gateway/   (Go)
              └────────────┬────────────┘
                           │
   ┌───────────┬───────────┼───────────┬───────────┬──────────────┐
   │           │           │           │           │              │
   ▼           ▼           ▼           ▼           ▼              ▼
┌──────┐  ┌────────┐  ┌──────────┐  ┌─────────┐  ┌─────────┐  ┌──────────────┐
│ Auth │  │ Course │  │Assessmnt │  │ Payment │  │ Profile │  │ Notification │
└──┬───┘  └───┬────┘  └────┬─────┘  └────┬────┘  └────┬────┘  └──────┬───────┘
   │          │            │             │            │              │
   ├──────────┴────────────┴─────────────┴────────────┴──────────────┤
   ▼                                                                 ▼
PostgreSQL                                                       NATS / Redis
                                                                       ▲
                                                                       │
                       Observability                                   │
   Prometheus  ──(scrape)──►   apps   ──(events)──►  NATS  ────────────┘
       │
       ▼
    Grafana ── alerts ──► On-call

Infrastructure: Terraform (sre/terraform/)
Configuration : Ansible   (sre/ansible/)
Orchestration : Docker Swarm  +  Kubernetes  (sre/swarm, sre/k8s)
```

## Microservices (≥ 6)

| #  | Service        | Stack             | Responsibility                     | Source                       |
|----|----------------|-------------------|------------------------------------|------------------------------|
| 1  | Auth           | Go, gRPC          | Login, JWT, RBAC                   | `services/auth/`             |
| 2  | Course         | Go, gRPC          | Course catalogue                   | `services/course/`           |
| 3  | Assessment     | Go, gRPC          | Quizzes / grading                  | `services/assessment/`       |
| 4  | Notification   | Go, NATS consumer | Async email / push                 | `notification/`              |
| 5  | Payment (new)  | Python / Flask    | Payment processing simulation      | `sre/services/payment/`      |
| 6  | User Profile   | Python / Flask    | Profile + preferences              | `sre/services/user-profile/` |

Plus supporting components: PostgreSQL, Redis, NATS, the Nginx edge,
Prometheus, Grafana and the Next.js web client under `web/`.

## Cross-cutting concerns

* **Observability** — every service exposes `/metrics` for Prometheus
  and emits structured logs to stdout (collected by Loki/Promtail).
* **Health** — each service has `/health` (liveness) and `/ready`
  (readiness) — see Kubernetes probes in `sre/k8s/`.
* **Configuration** — environment variables only. K8s ConfigMaps and
  Secrets, Swarm configs, or `.env` files for Compose.
* **Failure isolation** — each service has its own database schema
  (`auth_v2`, `course_v2`, `assessment_v2`), its own NATS subjects,
  and its own Prometheus job.
* **Backwards compatibility** — gRPC contracts in `proto/` ensure no
  breaking changes go un-noticed.

## Deployment topology

* **Local dev**     → `docker-compose.dev.yml`
* **Single-node**   → Docker Swarm (`sre/swarm/docker-stack.yml`)
* **Multi-node**    → Kubernetes (`sre/k8s/`)
* **Cloud infra**   → Terraform on AWS (`sre/terraform/aws/`)
* **OS / engine setup** → Ansible roles (`sre/ansible/`)
