# Kubernetes manifests

These manifests deploy the full EduLMS SRE stack into any Kubernetes
cluster (Minikube, kind, k3s, EKS, GKE, AKS).

## Quick start

```bash
# 1. Create namespaces + secrets + config.
kubectl apply -f 00-namespace.yaml
kubectl apply -f 01-config-and-secrets.yaml

# 2. Stateful infra.
kubectl apply -f 02-postgres.yaml
kubectl apply -f 03-redis-nats.yaml

# 3. Six microservices.
kubectl apply -f 04-auth.yaml
kubectl apply -f 05-course.yaml
kubectl apply -f 06-assessment.yaml
kubectl apply -f 07-notification.yaml
kubectl apply -f 08-payment.yaml
kubectl apply -f 09-user-profile.yaml

# 4. Gateway + ingress.
kubectl apply -f 10-gateway-and-ingress.yaml

# 5. Monitoring stack.
kubectl apply -f 11-monitoring.yaml

# Inspect.
kubectl get pods -n edulms
kubectl get hpa  -n edulms
kubectl get pods -n monitoring
```

## Build & push the application images first

The `image:` lines refer to `edulms/<svc>:1.0.0`. Build them locally and
either push to your registry or load them into the cluster:

```bash
docker build -t edulms/payment:1.0.0       sre/services/payment
docker build -t edulms/user-profile:1.0.0  sre/services/user-profile
docker build -t edulms/auth:1.0.0          -f services/auth/Dockerfile .
docker build -t edulms/course:1.0.0        -f services/course/Dockerfile .
docker build -t edulms/assessment:1.0.0    -f services/assessment/Dockerfile .
docker build -t edulms/notification:1.0.0  -f notification/Dockerfile .
docker build -t edulms/gateway:1.0.0       -f gateway/Dockerfile .

# kind:
kind load docker-image edulms/payment:1.0.0
# minikube:
minikube image load edulms/payment:1.0.0
```

## SRE features demonstrated

| Feature              | Where                                |
|----------------------|--------------------------------------|
| Self-healing         | livenessProbe + restartPolicy        |
| Horizontal scaling   | HorizontalPodAutoscaler for 6 apps   |
| Rolling deployments  | strategy.RollingUpdate               |
| Resource quotas      | requests/limits on every container   |
| Service discovery    | ClusterIP services                   |
| External access      | Ingress (`edulms.local`)             |
| Observability        | Prometheus pod-discovery + Grafana   |
| Alerting             | PrometheusRule (HighErrorRate, p95)  |
