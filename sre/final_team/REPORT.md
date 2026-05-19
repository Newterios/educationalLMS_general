## Repository

**GitHub:** <https://github.com/Newterios/educationalLMS>
**Live:** <https://aitbek.tech>

---

**Team:**

| Role                                  | Member                |
|---------------------------------------|-----------------------|
| Team Lead / Backend & SRE             | Aitbek Nugmanov       |
| DevOps / CI-CD                        | Syrym Shadiyarbek     |
| Frontend / Monitoring dashboards      | Fariza Arstanbek      |
| Backend / Infrastructure              | Mansur Ryskali        |

**Course:** Site Reliability Engineering
**Date:** 2026-05-19

---

# Final Project — Comprehensive SRE Implementation

Distributed microservices system **deployed on `aitbek.tech`** with a full
CI/CD pipeline (GitHub Actions), Docker Compose, Docker Swarm, Kubernetes,
Terraform (DigitalOcean), Ansible, Prometheus and Grafana.

---

## 1. Repository Structure

![repo tree](screenshots/new/01-repo-tree.png)

![sre folder](screenshots/new/02-sre-folder-tree.png)

---

## 2. Kubernetes

![kubectl apply](screenshots/kubectl-apply-output.png)

![k8s folder](screenshots/new/03-k8s-folder-listing.png)

```
kubectl apply -f sre/k8s/
kubectl get pods,svc,ing -n edulms
```

---

## 3. Terraform (DigitalOcean)

![terraform main.tf](screenshots/terraform-main-tf-code.png)

![terraform plan](screenshots/terraform-init-plan.png)

![terraform folder](screenshots/new/04-terraform-folder.png)

```
terraform init && terraform plan && terraform apply
```

---

## 4. Ansible

![ansible playbooks](screenshots/ansible-playbooks-output.png)

![ansible folder](screenshots/new/05-ansible-folder.png)

```
ansible-playbook -i inventory.yml playbooks/01_prepare.yml
ansible-playbook -i inventory.yml playbooks/02_deploy.yml
ansible-playbook -i inventory.yml playbooks/03_monitoring.yml
```

---

## 5. CI/CD Pipeline (new for the team final)

Flow: **branch → push/PR → CI (lint/test/build) → CD (SSH deploy)**

5 stages of CD job:

1. SSH to `aitbek.tech`
2. `git pull` on the server
3. Run Ansible playbook
4. Pull / build Docker images
5. `kubectl apply` + rolling restart + **health check**

![cicd workflow yml](screenshots/cicd-workflow-yml.png)

![cicd new yml](screenshots/new/06-cicd-yml-vscode.png)

![github actions runs](screenshots/new/07-github-actions-runs.png)

![github actions detail](screenshots/new/08-github-actions-detail.png)

---

## 6. Deployed System — aitbek.tech

![website live](screenshots/new/09-website-live.png)

![server deploy](screenshots/new/10-server-ssh-deploy.png)

![server docker ps](screenshots/new/11-server-docker-ps.png)

---

## 7. Monitoring & Alerting

![grafana dashboard](screenshots/grafana-dashboard-full.png)

![grafana slo](screenshots/grafana-slo-overview.png)

![prometheus targets live](screenshots/new/12-prometheus-targets-live.png)

![grafana live](screenshots/new/13-grafana-live.png)

---

## 8. Incident Simulation — PostgresDown Alert FIRING

We stopped the `postgres-exporter` container to demonstrate the full
alerting pipeline. PostgreSQL itself kept serving traffic.

```
docker stop diplom-postgres-exporter-1
```

Alert transition: **inactive → pending → FIRING (~65s)**.

![alerts firing](screenshots/prometheus-alerts-firing.png)

![alert live](screenshots/new/14-alert-firing-live.png)

---

## 9. SLIs & SLOs

| Service       | Availability | p95 latency | Error rate |
|---------------|--------------|-------------|------------|
| Auth          | ≥ 99.5 %     | ≤ 150 ms    | ≤ 0.5 %    |
| Course        | ≥ 99 %       | ≤ 200 ms    | ≤ 1 %      |
| Assessment    | ≥ 99 %       | ≤ 250 ms    | ≤ 1 %      |
| Payment       | ≥ 99 %       | ≤ 200 ms    | ≤ 1 %      |
| User Profile  | ≥ 99 %       | ≤ 150 ms    | ≤ 1 %      |
| Gateway       | ≥ 99.9 %     | ≤ 50 ms     | ≤ 0.1 %    |

Full SLI/SLO design + error budget policy: `sre/docs/SLI_SLO.md`.

---

## 10. Capacity Planning (summary)

- Assessment + payment services have the highest burst CPU.
- PostgreSQL is the primary bottleneck under concurrent writes.
- HPA in K8s set to scale on CPU 70 %.

---

## 11. Deliverables Checklist

- [x] 6+ microservices
- [x] Docker Compose / Swarm
- [x] Kubernetes manifests
- [x] Terraform (DigitalOcean)
- [x] Ansible playbooks
- [x] **CI/CD GitHub Actions pipeline** (new)
- [x] Prometheus + Grafana + alerts
- [x] Incident + postmortem
- [x] Deployed at <https://aitbek.tech>
- [x] Git link: <https://github.com/Newterios/educationalLMS>

---

**Repository:** <https://github.com/Newterios/educationalLMS>
**Live system:** <https://aitbek.tech>
