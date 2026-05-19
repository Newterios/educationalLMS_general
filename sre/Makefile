# EduLMS SRE — convenience Makefile.
# Run from the repository root via `make -C sre <target>`.
# Every target jumps to the repo root with `cd $(ROOT)` first, so paths
# in docker-compose files are resolved consistently.

SHELL  := /bin/bash
ROOT   := $(realpath $(dir $(lastword $(MAKEFILE_LIST)))/..)
PROJECT := edulmsv2
COMPOSE := docker compose -p $(PROJECT) -f docker-compose.dev.yml -f sre/docker-compose.sre.yml

.PHONY: help build up down logs ps k8s-apply k8s-delete swarm-up swarm-down \
        tf-aws-init tf-aws-plan tf-aws-apply tf-local-up tf-local-down \
        ansible incident-on incident-off

help:
	@echo "EduLMS SRE Make targets (run from repo root with 'make -C sre <target>'):"
	@echo "  build          - build payment + user-profile images"
	@echo "  up             - docker compose up (base + SRE services + obs)"
	@echo "  down           - tear everything down"
	@echo "  logs           - tail logs"
	@echo "  ps             - list services"
	@echo "  swarm-up       - docker stack deploy"
	@echo "  swarm-down     - docker stack rm"
	@echo "  k8s-apply      - kubectl apply -f sre/k8s/"
	@echo "  k8s-delete     - kubectl delete -f sre/k8s/"
	@echo "  tf-aws-*       - Terraform AWS"
	@echo "  tf-local-*     - Terraform local (docker provider)"
	@echo "  ansible        - run site.yml"
	@echo "  incident-on    - inject 100% failure rate into payment"
	@echo "  incident-off   - restore payment to 2% failure rate"

build:
	cd $(ROOT) && docker build -t edulms/payment:1.0.0      sre/services/payment
	cd $(ROOT) && docker build -t edulms/user-profile:1.0.0 sre/services/user-profile

up: build
	cd $(ROOT) && $(COMPOSE) --profile services --profile obs up -d --build

down:
	cd $(ROOT) && $(COMPOSE) --profile services --profile obs down -v

logs:
	cd $(ROOT) && $(COMPOSE) logs -f --tail=200

ps:
	cd $(ROOT) && $(COMPOSE) ps

swarm-up:
	cd $(ROOT) && docker swarm init 2>/dev/null || true
	cd $(ROOT) && docker stack deploy -c sre/swarm/docker-stack.yml edulms

swarm-down:
	docker stack rm edulms

k8s-apply:
	cd $(ROOT) && kubectl apply -f sre/k8s/

k8s-delete:
	cd $(ROOT) && kubectl delete -f sre/k8s/ --ignore-not-found

tf-aws-init:
	cd $(ROOT) && terraform -chdir=sre/terraform/aws init

tf-aws-plan:
	cd $(ROOT) && terraform -chdir=sre/terraform/aws plan

tf-aws-apply:
	cd $(ROOT) && terraform -chdir=sre/terraform/aws apply -auto-approve

tf-local-up:
	cd $(ROOT) && terraform -chdir=sre/terraform/local init
	cd $(ROOT) && terraform -chdir=sre/terraform/local apply -auto-approve

tf-local-down:
	cd $(ROOT) && terraform -chdir=sre/terraform/local destroy -auto-approve

ansible:
	cd $(ROOT) && ansible-playbook -i sre/ansible/inventory.ini sre/ansible/site.yml

incident-on:
	cd $(ROOT) && PAYMENT_FAILURE_RATE=1.0 $(COMPOSE) up -d --no-deps payment
	@echo "Payment now returns 100% errors — watch Prometheus alerts fire."

incident-off:
	cd $(ROOT) && PAYMENT_FAILURE_RATE=0.02 $(COMPOSE) up -d --no-deps payment
	@echo "Payment failure rate restored to baseline (2%)."
