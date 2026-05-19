# Ansible — Configuration Management

Roles:

| Role        | Purpose                                                           |
|-------------|-------------------------------------------------------------------|
| `common`    | Hostname, baseline packages, NTP, UFW firewall                    |
| `docker`    | Install Docker engine + compose plugin, add user to docker group  |
| `swarm`     | `docker swarm init` on the manager, join workers                  |
| `deploy`    | Copy the stack file and run `docker stack deploy edulms`          |
| `monitoring`| Wait for Prometheus/Grafana to be ready and hot-reload Prometheus |

## Run

```bash
cd sre/ansible
# Edit inventory.ini with public IPs from `terraform output`.

ansible-playbook site.yml                 # everything
ansible-playbook site.yml --tags docker   # only the docker role
ansible-playbook site.yml --limit managers
```

## Why Ansible alongside Terraform?

* **Terraform** is great at *creating* immutable resources (VMs, networks).
* **Ansible** is great at *configuring* what runs *on* those resources
  (packages, daemons, app deployment).
* Together they cover the full lifecycle.
