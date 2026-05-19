###############################################################################
# Local (no-cloud) Terraform — provisions the entire EduLMS SRE stack using
# the Docker provider. Useful for graders / reviewers who don't want to spend
# money on AWS.
#
# Usage:
#   cd sre/terraform/local
#   terraform init
#   terraform apply -auto-approve
###############################################################################

terraform {
  required_version = ">= 1.5.0"
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.0"
    }
  }
}

provider "docker" {}

resource "docker_network" "edulms" {
  name = "edulms-tf-net"
}

# ───── shared infrastructure ─────────────────────────────────────────────────
resource "docker_image" "postgres" { name = "postgres:16-alpine" }
resource "docker_image" "redis"    { name = "redis:7-alpine" }
resource "docker_image" "nats"     { name = "nats:2-alpine" }
resource "docker_image" "prom"     { name = "prom/prometheus:v2.52.0" }
resource "docker_image" "grafana"  { name = "grafana/grafana:10.4.3" }

resource "docker_container" "postgres" {
  name  = "edulms-tf-postgres"
  image = docker_image.postgres.image_id
  env = [
    "POSTGRES_USER=edulms",
    "POSTGRES_PASSWORD=edulms",
    "POSTGRES_DB=postgres",
  ]
  networks_advanced { name = docker_network.edulms.name }
  ports {
    internal = 5432
    external = 55432
  }
  restart = "unless-stopped"
}

resource "docker_container" "redis" {
  name     = "edulms-tf-redis"
  image    = docker_image.redis.image_id
  networks_advanced { name = docker_network.edulms.name }
  restart  = "unless-stopped"
}

resource "docker_container" "nats" {
  name     = "edulms-tf-nats"
  image    = docker_image.nats.image_id
  command  = ["-m", "8222"]
  networks_advanced { name = docker_network.edulms.name }
  restart  = "unless-stopped"
}

# ───── application services (locally built images) ──────────────────────────
locals {
  app_services = {
    payment = {
      image    = "edulms/payment:1.0.0"
      external = 8081
      internal = 8081
      env      = ["PORT=8081", "FAILURE_RATE=0.02"]
    }
    user-profile = {
      image    = "edulms/user-profile:1.0.0"
      external = 8082
      internal = 8082
      env      = ["PORT=8082"]
    }
  }
}

resource "docker_container" "app" {
  for_each = local.app_services
  name     = "edulms-tf-${each.key}"
  image    = each.value.image
  env      = each.value.env
  networks_advanced { name = docker_network.edulms.name }
  ports {
    internal = each.value.internal
    external = each.value.external
  }
  restart = "unless-stopped"

  healthcheck {
    test     = ["CMD", "curl", "-fsS", "http://localhost:${each.value.internal}/health"]
    interval = "15s"
    timeout  = "3s"
    retries  = 3
  }
}

# ───── monitoring ───────────────────────────────────────────────────────────
resource "docker_container" "prometheus" {
  name  = "edulms-tf-prometheus"
  image = docker_image.prom.image_id
  command = [
    "--config.file=/etc/prometheus/prometheus.yml",
    "--storage.tsdb.retention.time=7d",
  ]
  ports {
    internal = 9090
    external = 9090
  }
  volumes {
    host_path      = abspath("${path.module}/../../monitoring/prometheus.yml")
    container_path = "/etc/prometheus/prometheus.yml"
    read_only      = true
  }
  networks_advanced { name = docker_network.edulms.name }
  restart = "unless-stopped"
}

resource "docker_container" "grafana" {
  name  = "edulms-tf-grafana"
  image = docker_image.grafana.image_id
  env = [
    "GF_SECURITY_ADMIN_PASSWORD=admin",
    "GF_USERS_ALLOW_SIGN_UP=false",
  ]
  ports {
    internal = 3000
    external = 3030
  }
  networks_advanced { name = docker_network.edulms.name }
  restart = "unless-stopped"
}

output "endpoints" {
  value = {
    payment_url      = "http://localhost:8081"
    user_profile_url = "http://localhost:8082"
    prometheus_url   = "http://localhost:9090"
    grafana_url      = "http://localhost:3030"
  }
}
