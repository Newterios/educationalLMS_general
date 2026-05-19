# Postmortem — INC-2026-05-01

> Payment service down for 46 minutes after a ConfigMap rename

This postmortem follows the Google SRE blameless-postmortem template.
Names are anonymised; the goal is to learn, not to blame.

## Summary

On 2026-05-01 the EduLMS payment service was unable to process any
`/pay` requests for 46 minutes. The root cause was a renamed key in
the Kubernetes ConfigMap that fed environment variables into the
payment Deployment. The new pods started but could not connect to
PostgreSQL because the `POSTGRES_HOST` env var resolved to an empty
string.

The incident was detected within 60 seconds by the
`PaymentErrorRateAboveSLO` alert, mitigated 29 minutes later via a
ConfigMap rollback, and fully verified within 46 minutes. Roughly
**0.4 %** of the monthly error budget for the payment service was
consumed.

## Impact

* **Customer impact**: ~120 abandoned checkouts in the affected
  window. No payments were double-charged (the failures happened
  before the upstream processor was contacted).
* **Internal impact**: order service queued ~430 retries in NATS;
  caught up automatically within 12 minutes of mitigation.
* **SLO impact**:
  * Availability (30 d): dropped from 99.97 % → 99.85 % (still above
    the 99 % target).
  * Error budget burned: 4.2 % of the monthly budget in a single
    event.

## Trigger

PR #142 ("rename env vars to be more descriptive") renamed the key
`POSTGRES_HOST → POSTGRES_HOSTNAME` in the ConfigMap but **not** in
the Deployment that consumed it. The change passed review because
both files were not opened side-by-side in the PR diff view.

## Root cause

Kubernetes silently substitutes empty strings for missing ConfigMap
keys when the consumer uses `valueFrom.configMapKeyRef` without
`optional: false`. The payment app then constructed
`postgres://...:5432/...` with no host, threw a DNS error, and
returned 500 to every caller.

The liveness probe checked `/health`, which only reports process
uptime. It did **not** detect that the database dependency was
broken, so Kubernetes happily routed traffic to broken pods.

## Detection

The Prometheus rule below fired in 60 seconds:

```yaml
- alert: PaymentErrorRateAboveSLO
  expr: |
    sum(rate(payment_requests_total{status=~"5.."}[5m]))
      / clamp_min(sum(rate(payment_requests_total[5m])), 0.001) > 0.01
  for: 5m
```

Detection time **MTTD = 0:01:23**.
Acknowledgement time **MTTA = 0:03:11**.
Mitigation time **MTTM = 0:29:04**.

## Response

The on-call engineer used the runbook in this document to:

1. `kubectl logs -n edulms deploy/payment --tail=200` — found
   `dial tcp :5432: connect: connection refused`.
2. `kubectl get deploy payment -n edulms -o yaml | yq .spec.template.spec.containers[0].env`
   — saw `POSTGRES_HOST: ""`.
3. `kubectl rollout history configmap/edulms-config -n edulms`
   (via GitOps log).
4. `kubectl rollout undo -n edulms deploy/payment` after restoring
   the ConfigMap key in Git and re-applying it.

The longest single step was the rollback: the team initially tried
to hot-patch the ConfigMap with `kubectl edit`, which works but does
not trigger pod restarts. The proper sequence was: edit ConfigMap →
`kubectl rollout restart deploy/payment`.

## What went well

* Alert fired within the agreed MTTD of 5 minutes.
* The on-call had clear access to logs and the GitOps history.
* The blast radius was limited to one service because each
  microservice has its own ConfigMap.

## What went wrong

* Readiness probe did not validate the DB dependency.
* No canary stage — bad config went to 100 % of replicas in one step.
* The PR review did not catch the env-var rename mismatch.
* Runbook was missing from the alert annotation
  (`runbook_url: TODO`).

## Where we got lucky

* The failure was a clean 5xx, not silent data corruption.
* PostgreSQL was unaffected, so the new replicas with the fixed config
  could start serving traffic instantly.
* Customer impact was bounded because the order service marks orders
  `PENDING_PAYMENT` rather than `FAILED` when payment errors out, so
  no orders had to be manually refunded.

## Action items

| ID  | Action                                                       | Type        | Owner    | Due       | Status |
|-----|--------------------------------------------------------------|-------------|----------|-----------|--------|
| A-1 | Make `/ready` actively call `SELECT 1` against the DB        | Prevent     | backend  | +3 days   | open   |
| A-2 | CI check: every ConfigMap key referenced by a Deployment     | Prevent     | platform | +1 week   | open   |
| A-3 | Argo Rollouts canary for payment + course                    | Mitigate    | sre      | +2 weeks  | open   |
| A-4 | Add runbook URL to every alert annotation                    | Process     | sre      | +2 days   | done   |
| A-5 | Quarterly game-day: replay this incident in staging          | Process     | sre      | quarterly | open   |
| A-6 | Add `optional: false` to all `configMapKeyRef`s              | Prevent     | platform | +1 week   | open   |
| A-7 | Update onboarding doc: "ConfigMap edits require restart"     | Process     | sre      | +5 days   | open   |

## Timeline

```
14:02:00  Alert PaymentErrorRateAboveSLO fires
14:03:11  On-call acknowledges
14:05:00  Incident channel opened, PR #142 identified as suspect
14:08:30  Logs show empty POSTGRES_HOST
14:14:00  ConfigMap diff confirmed as the trigger
14:17:00  Revert PR opened and merged
14:19:30  CI/CD applies revert; new pods start rolling
14:22:00  Error rate begins to drop
14:31:04  Error rate <1 % SLO threshold → MITIGATED
14:48:00  100 successful /pay calls observed → RESOLVED
14:55:00  Postmortem template opened
```

## Glossary

* **SLI** — Service Level Indicator, the metric.
* **SLO** — Service Level Objective, the threshold on the metric.
* **MTTD** — Mean Time To Detect.
* **MTTA** — Mean Time To Acknowledge.
* **MTTM** — Mean Time To Mitigate.
* **MTTR** — Mean Time To Recovery (≈ resolved).
