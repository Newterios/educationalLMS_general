# Screenshots Guide — Team Final

Save each screenshot to `sre/final_team/screenshots/new/` with the exact
filename listed below. Filenames matter — REPORT.md references them.

## Already prepared (from your previous End Term PDF)

These 8 PNG files were extracted from `source_endterm_report.pdf` and
live in `sre/final_team/screenshots/`:

| File                          | Shows                              |
|-------------------------------|------------------------------------|
| `kubectl-apply-output.png`    | `kubectl apply` + `get pods,svc`   |
| `terraform-init-plan.png`     | `terraform init` + `plan` output   |
| `terraform-main-tf-code.png`  | `main.tf` in VS Code               |
| `ansible-playbooks-output.png`| 3 ansible playbook runs (PLAY RECAP) |
| `prometheus-alerts-firing.png`| Prometheus alerts — PostgresDown FIRING |
| `grafana-dashboard-full.png`  | Grafana — Golden Signals dashboard |
| `cicd-workflow-yml.png`       | `.github/workflows/ci.yml` editor  |
| `grafana-slo-overview.png`    | Grafana — SLO compliance overview  |

You can reuse them as-is. The new screenshots below complement them.

## New screenshots needed (15 total)

Save into `sre/final_team/screenshots/new/`:

### A. Repository structure (folders)

**`01-repo-tree.png`** — terminal screenshot of:
```bash
cd /Users/aitbek/Desktop/final_apt_eduLms
tree -L 2 -I 'node_modules|.git|.next|vendor' | head -60
```

**`02-sre-folder-tree.png`** — screenshot of the `sre/` folder contents:
```bash
tree sre/ -L 3 -I '__pycache__|extracted'
```

**`03-k8s-folder-listing.png`** — screenshot of the K8s manifests folder:
```bash
ls -la sre/k8s/
```

**`04-terraform-folder.png`** — screenshot of:
```bash
ls -la sre/terraform/aws/
cat sre/terraform/aws/main.tf | head -40
```

**`05-ansible-folder.png`** — screenshot of:
```bash
tree sre/ansible/
```

### B. CI/CD (the new mandatory requirement)

**`06-cicd-yml-vscode.png`** — open `.github/workflows/ci-cd.yml` in VS Code
(or any editor) and screenshot the whole file. Highlight the 5 stages:
SSH → Pull → Ansible → Docker → Kubectl → Health check.

**`07-github-actions-runs.png`** — on GitHub, go to repo →
**Actions** tab → screenshot the list of workflow runs (green checkmarks).

**`08-github-actions-detail.png`** — click on one successful run, expand
the **Deploy to aitbek.tech** job, screenshot the green steps (SSH, pull,
ansible, kubectl, health check).

### C. Deployed system on aitbek.tech

**`09-website-live.png`** — browser screenshot of <https://aitbek.tech>
(main page of the LMS).

**`10-server-ssh-deploy.png`** — terminal screenshot showing manual
deploy:
```bash
./scripts/deploy-to-server.sh
```
(or just `make -C sre up` running on the server)

**`11-server-docker-ps.png`** — on the server (via SSH), screenshot:
```bash
ssh ubuntu@aitbek.tech
docker compose -p edulmsv2 ps
```

### D. Monitoring evidence (live system)

**`12-prometheus-targets-live.png`** — <https://aitbek.tech:9090/targets>
(or whatever URL your Prometheus is exposed at) — all targets UP.

**`13-grafana-live.png`** — your live Grafana dashboard with real data
from the deployed system.

**`14-alert-firing-live.png`** — repeat the incident on the live system
(stop one container), screenshot the alert in FIRING state.

### E. Team

**`15-team-photo.png`** — group photo of the 4 team members
(Aitbek, Syrym, Fariza, Mansur) — used on the title slide of the
presentation. Optional but recommended.

## How to take screenshots on macOS

* **Whole screen**:  `Cmd + Shift + 3`
* **Selected area**: `Cmd + Shift + 4`, then drag
* **Selected window**: `Cmd + Shift + 4`, then press `Space`, then click
  the window
* By default files land on `~/Desktop/`

## Quick helper to import screenshots

After taking all the new screenshots above (in the order they're listed),
run:

```bash
cd /Users/aitbek/Desktop/final_apt_eduLms/sre/final_team
./import-screenshots-team.sh
```

It will grab the 15 most recent `Screenshot*.png` / `Снимок экрана*.png`
files from `~/Desktop`, rename them 01..15 and copy them into
`screenshots/new/`.

After that — run `./build-pdf-team.sh` to produce the final PDF.
