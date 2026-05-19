## Repository

**GitHub:** <https://github.com/Newterios/educationalLMS_general>

---

**Student:** Kaiyrbekuly Aitbek
**Group:** SE-2424
**Date:** 2026-05-19

---

# End Term Project — SRE Implementation

EduLMS distributed microservices system with Docker Compose, Docker
Swarm, Kubernetes, Terraform, Ansible, Prometheus and Grafana.

## Microservices (6)

| # | Service       | Stack             |
|---|---------------|-------------------|
| 1 | Auth          | Go, gRPC          |
| 2 | Course        | Go, gRPC          |
| 3 | Assessment    | Go, gRPC          |
| 4 | Notification  | Go, NATS consumer |
| 5 | Payment       | Python / Flask    |
| 6 | User Profile  | Python / Flask    |

## SLOs

| Service       | Availability | p95 latency | Error rate |
|---------------|--------------|-------------|------------|
| Auth          | ≥ 99.5 %     | ≤ 150 ms    | ≤ 0.5 %    |
| Course        | ≥ 99 %       | ≤ 200 ms    | ≤ 1 %      |
| Assessment    | ≥ 99 %       | ≤ 250 ms    | ≤ 1 %      |
| Payment       | ≥ 99 %       | ≤ 200 ms    | ≤ 1 %      |
| User Profile  | ≥ 99 %       | ≤ 150 ms    | ≤ 1 %      |
| Gateway       | ≥ 99.9 %     | ≤ 50 ms     | ≤ 0.1 %    |

## Screenshots

**1. Containers running**

![docker ps](screenshots/01-docker-ps.png)

**2. Microservices responding**

![microservices](screenshots/02-microservices-running.png)

**3. Prometheus targets — all UP**

![targets](screenshots/03-prometheus-targets.png)

**4. Prometheus alert rules**

![alerts](screenshots/04-prometheus-alerts.png)

**5. Grafana SRE dashboard**

![grafana](screenshots/05-grafana-dashboard.png)

**6. Incident simulation — alert in PENDING**

![incident](screenshots/06-incident-simulation.png)

## Deliverables

- [x] 6 microservices
- [x] Docker Compose / Swarm
- [x] Kubernetes manifests
- [x] Terraform (AWS + local)
- [x] Ansible playbooks
- [x] Prometheus + Grafana + alerts
- [x] Incident report + postmortem
- [x] Git link: <https://github.com/Newterios/educationalLMS_general>
