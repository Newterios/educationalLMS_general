# Incident Report — INC-2026-05-01

> Order/Payment Service degradation due to misconfigured database DSN

| Field          | Value                                  |
|----------------|----------------------------------------|
| Incident ID    | INC-2026-05-01                         |
| Severity       | SEV-2 (partial functionality lost)     |
| Detected at    | 2026-05-01 14:02 UTC                   |
| Mitigated at   | 2026-05-01 14:31 UTC                   |
| Resolved at    | 2026-05-01 14:48 UTC                   |
| Duration       | 46 minutes                             |
| Author         | EduLMS SRE on-call                     |
| Status         | Resolved                               |

## 1. Impact

* **Order creation** unavailable for the duration of the incident.
* **Payment service**: 100 % of `/pay` requests returned HTTP 500
  because it could not reach the database via the misconfigured DSN.
* User-profile service unaffected (no DB dependency).
* All other read paths (course, auth) continued to serve.

User-visible symptoms: checkout flow showed "Something went wrong"
modal; orders stuck in `PENDING_PAYMENT` state.

## 2. Trigger

A change to `sre/k8s/01-config-and-secrets.yaml` accidentally renamed
the `POSTGRES_HOST` key to `POSTGRES_HOSTNAME`. The Deployment
manifest still mounted the old key, so the env-var became empty in
the payment pods. On the next rollout the new pods came up and
immediately started failing.

## 3. Detection

* **14:02 UTC** — Prometheus alert `PaymentErrorRateAboveSLO` fired
  (5xx rate jumped to 100 %).
* **14:03 UTC** — Alertmanager paged the on-call engineer (PagerDuty).
* **14:04 UTC** — Synthetic checkout probe in `tests/` started
  failing.

## 4. Response timeline

| Time (UTC) | Action                                                         |
|------------|----------------------------------------------------------------|
| 14:02      | Alert fires                                                    |
| 14:05      | On-call acknowledges, opens incident channel                   |
| 14:08      | `kubectl logs -n edulms deploy/payment` — DSN errors found     |
| 14:14      | `kubectl describe pod payment-...` — env var missing           |
| 14:17      | Rolled back ConfigMap (`kubectl rollout undo`)                 |
| 14:22      | New pods schedule; error rate begins to drop                   |
| 14:31      | Error rate < 1 % (SLO restored) → declared mitigated           |
| 14:48      | Post-fix verification: 100 successful `/pay` requests          |

## 5. Root cause

A typo in a key name combined with no schema validation on the
ConfigMap caused the env var to be silently dropped. The
liveness/readiness probes were too lax (they hit `/health` which only
returns 200 — it did not actually exercise the DB), so Kubernetes did
not mark the pods unhealthy and continued to send traffic to them.

## 6. Contributing factors

1. The PR description didn't call out the env-var rename → reviewer
   missed it.
2. Readiness probe did not test the dependency path (DB connection).
3. No canary/blue-green stage in the rollout — bad config hit 100 %
   of replicas at once.

## 7. Resolution

The bad ConfigMap was rolled back via `kubectl rollout undo`. A
follow-up commit restored the original key name and added validation
in CI to reject ConfigMaps whose keys don't appear in the
corresponding Deployment.

## 8. Action items

| ID  | Action                                                                 | Owner   | Due       |
|-----|------------------------------------------------------------------------|---------|-----------|
| A-1 | Add DB-touching probe to `/ready` in payment service                   | backend | +3 days   |
| A-2 | CI lint to cross-check ConfigMap keys vs Deployment env consumers      | platform| +1 week   |
| A-3 | Switch payment deployment to canary (10 %) before full rollout         | sre     | +2 weeks  |
| A-4 | Add Slack `#sre-alerts` integration to Alertmanager                    | sre     | +5 days   |
| A-5 | Practise this scenario in a quarterly game-day                         | sre     | quarterly |

A full postmortem is captured in `POSTMORTEM.md`.
