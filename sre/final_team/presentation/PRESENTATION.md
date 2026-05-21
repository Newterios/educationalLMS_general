---
marp: true
theme: default
paginate: true
size: 16:9
header: 'EduLMS — Comprehensive SRE Implementation'
footer: 'aitbek.tech · github.com/Newterios/educationalLMS'
style: |
  section {
    font-size: 26px;
    padding: 50px;
  }
  h1 { color: #1e40af; }
  h2 { color: #1e3a8a; }
  table { font-size: 22px; }
  code { background: #f1f5f9; padding: 2px 6px; border-radius: 4px; }
  pre { font-size: 18px; }
  .speaker {
    background: #dbeafe;
    color: #1e3a8a;
    padding: 4px 12px;
    border-radius: 4px;
    font-size: 18px;
    font-weight: bold;
  }
  .live-badge {
    background: #16a34a;
    color: white;
    padding: 6px 14px;
    border-radius: 6px;
    font-weight: bold;
  }
---

<!-- _class: lead -->
<!-- _paginate: false -->

# Comprehensive SRE Implementation
## EduLMS Distributed Microservices System

**Team:** Aitbek · Syrym · Fariza · Mansur

<span class="live-badge">LIVE @ aitbek.tech</span>
**Repo:** github.com/Newterios/educationalLMS

Astana IT University · SRE · 2026

---

## Team & Speaking Order

| Member                | Role                         | Slides       |
|-----------------------|------------------------------|--------------|
| **Aitbek Nugmanov**   | Team Lead / Backend & SRE    | 1, 4, 9–11   |
| **Syrym Shadiyarbek** | DevOps / CI-CD               | 7, 8         |
| **Fariza Arstanbek**  | Frontend / Monitoring        | 12, 13       |
| **Mansur Ryskali**    | Backend / Infrastructure     | 3, 5, 6      |

<span class="speaker">Speaker: Aitbek</span>

---

## Goal

- 6+ microservices, **deployed and running** at <https://aitbek.tech>
- Define & meet SLOs (99 % availability, 200 ms p95, ≤ 1 % errors)
- **Multi-orchestration**: Docker Compose, Swarm, Kubernetes
- **IaC + ConfigMgmt**: Terraform + Ansible
- **CI/CD** — every push to `main` auto-deploys to the server
- Detect, alert and recover from incidents with a postmortem

<span class="speaker">Speaker: Aitbek</span>

---

## Repository Layout

![h:430 center](../screenshots/new/01-repo-tree.png)

<span class="speaker">Speaker: Aitbek</span>

---

## SRE Folder

![h:430 center](../screenshots/new/02-sre-folder-tree.png)

<span class="speaker">Speaker: Mansur</span>

---

## Kubernetes — 13 manifests for 6 services + infra

![h:330 center](../screenshots/new/03-k8s-folder-listing.png)

`kubectl apply -f sre/k8s/` — Deployments, Services, HPAs, Ingress.

<span class="speaker">Speaker: Aitbek</span>

---

## Terraform — DigitalOcean Infrastructure

![h:430 center](../screenshots/new/04-terraform-folder.png)

VPC + firewall + Ubuntu droplet → the host that runs **aitbek.tech**.

<span class="speaker">Speaker: Mansur</span>

---

## Ansible — Configuration Management

![h:430 center](../screenshots/new/05-ansible-folder.png)

5 roles: common · docker · swarm · deploy · monitoring.

<span class="speaker">Speaker: Mansur</span>

---

## CI/CD Pipeline (new for the team final)

![h:430 center](../screenshots/new/06-cicd-yml-vscode.png)

**Triggers**: push to `main` / PR
**Flow**: lint → test → build images → SSH → ansible → kubectl → health-check

<span class="speaker">Speaker: Syrym</span>

---

## CI/CD — Live runs on GitHub Actions

![h:380 center](../screenshots/new/08-github-actions-detail.png)

Matrix strategy: every microservice gets its own Lint & Test job.

<span class="speaker">Speaker: Syrym</span>

---

## SLIs & SLOs

| Service       | Availability | p95 latency | Error rate |
|---------------|--------------|-------------|------------|
| Auth          | ≥ 99.5 %     | ≤ 150 ms    | ≤ 0.5 %    |
| Course        | ≥ 99 %       | ≤ 200 ms    | ≤ 1 %      |
| Payment       | ≥ 99 %       | ≤ 200 ms    | ≤ 1 %      |
| User Profile  | ≥ 99 %       | ≤ 150 ms    | ≤ 1 %      |
| Gateway       | ≥ 99.9 %     | ≤ 50 ms     | ≤ 0.1 %    |

Error budget policy: feature freeze when monthly budget hits 0 %.

<span class="speaker">Speaker: Aitbek</span>

---

## Monitoring — Prometheus targets all UP

![h:300 center](../screenshots/new/12-prometheus-targets.png)

4/4 green: `prometheus`, `otel-collector`, `payment`, `user-profile`.

<span class="speaker">Speaker: Fariza</span>

---

## Monitoring — Grafana SRE Dashboard

![h:380 center](../screenshots/new/13-grafana-dashboard.png)

Golden Signals (rate, errors, latency p95/p99, saturation) for `payment`.

<span class="speaker">Speaker: Fariza</span>

---

## Incident — Alert FIRING

```bash
make -C sre incident-on    # 100 % failure rate injected
```

Alert path: **inactive → pending → FIRING** in ~5 minutes.

![h:320 center](../screenshots/new/14-alert-firing.png)

<span class="speaker">Speaker: Aitbek</span>

---

## Capacity Planning

**Findings**
- Assessment & payment = peak CPU
- PostgreSQL = primary bottleneck

**Strategy**
- Horizontal: HPA on CPU 70 %, max 6 replicas/service
- Vertical: `requests` / `limits` tuned
- DB: index hot paths, PgBouncer planned

<span class="speaker">Speaker: Aitbek</span>

---

## Live Demo Plan

1. Open <https://aitbek.tech> — deployed LMS
2. Show GitHub Actions tab — last green pipeline
3. Open Grafana dashboard (local) — golden signals
4. Run `make -C sre incident-on` → alert FIRING
5. `make -C sre incident-off` → alert clears

<span class="live-badge">Server is up & serving traffic right now</span>

<span class="speaker">Demo: Aitbek + Syrym</span>

---

## Deliverables ✓

- 6+ microservices · Compose · Swarm · K8s
- Terraform · Ansible · **CI/CD pipeline (new)**
- Prometheus + Grafana + SLO alerts
- Incident report + blameless postmortem
- **Deployed and running at <https://aitbek.tech>**
- Repo: <https://github.com/Newterios/educationalLMS>

---

<!-- _class: lead -->

# Questions?

**Live:** aitbek.tech
**Repo:** github.com/Newterios/educationalLMS

Aitbek · Syrym · Fariza · Mansur
