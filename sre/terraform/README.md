# Terraform — Infrastructure as Code

Two configurations are provided.

## `aws/` — cloud provisioning (Assignment 5)

Creates a small reproducible cluster on AWS that Ansible later configures:

* VPC (`10.20.0.0/16`)
* 2 public subnets in 2 availability zones
* Internet gateway + public route table
* Security group with SSH/HTTP/app/monitoring ports
* `var.node_count` (default 2) EC2 nodes (Ubuntu 22.04, gp3 30 GiB)

### Run

```bash
cd aws
cp terraform.tfvars.example terraform.tfvars
# edit owner / ssh_key_name etc.

terraform init
terraform plan
terraform apply -auto-approve

terraform output node_public_ips
```

The `ansible_inventory_hint` output prints a ready-to-paste inventory
for the Ansible playbook in `../../ansible/`.

## `local/` — Docker provider (no cloud cost)

For graders without AWS credentials. Brings up Postgres, Redis, NATS,
the payment + user-profile services, Prometheus and Grafana on the
local Docker engine.

```bash
cd local

# Pre-build the application images first:
docker build -t edulms/payment:1.0.0      ../../services/payment
docker build -t edulms/user-profile:1.0.0 ../../services/user-profile

terraform init
terraform apply -auto-approve
terraform output endpoints
```

Tear down:

```bash
terraform destroy -auto-approve
```

## Why Terraform?

* **Declarative** — the desired state of the infrastructure lives in
  version control.
* **Idempotent** — re-running `apply` only changes what drifted.
* **Reproducible** — graders can stand up the same lab on their own
  AWS account in minutes.
* **Plan/Apply workflow** — every change is reviewed before it lands.
