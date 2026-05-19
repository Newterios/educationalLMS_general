# Docker Swarm deployment

This folder contains a Swarm stack that mirrors the Compose dev stack
but adds replica counts, update strategies, and rolling-update /
rollback policies — the features that turn Compose into a basic
orchestrator.

## Bootstrap

```bash
# 1. Initialise the swarm (run on the manager).
docker swarm init

# 2. (Optional) join workers using the token printed by the init.
# docker swarm join --token <token> <manager-ip>:2377

# 3. Build the local images (Swarm does not build for you).
make -C ../.. build  # or use `docker build` per service

# 4. Deploy.
docker stack deploy -c docker-stack.yml edulms

# 5. Inspect.
docker stack services edulms
docker service ps    edulms_payment
docker service logs  edulms_payment

# 6. Scale a service.
docker service scale edulms_payment=5

# 7. Rolling update (after rebuilding the image).
docker service update --image edulms/payment:1.1.0 edulms_payment

# 8. Tear down.
docker stack rm edulms
```

## Compared with Kubernetes

| Capability         | Docker Swarm                       | Kubernetes              |
|--------------------|------------------------------------|-------------------------|
| Service replicas   | `deploy.replicas`                  | `Deployment.replicas`   |
| Autoscaling        | none built-in (manual scale)       | HorizontalPodAutoscaler |
| Self-healing       | yes (restart_policy)               | yes (probes + ctrl)     |
| Rolling updates    | `update_config`                    | `RollingUpdate` strategy|
| Service discovery  | overlay network DNS                | ClusterIP DNS           |
| Secrets / configs  | `docker config / secret`           | Secret / ConfigMap      |
| Storage primitives | local volumes / plugins            | PV / PVC / CSI          |
| Complexity         | low                                | high                    |

The lab demonstrates **both** so the same workload runs on either
orchestrator with no code change.
