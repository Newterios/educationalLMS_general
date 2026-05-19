# EduLMS SRE — convenience Makefile.
# All targets assume you run them from the repository root via:
#   make -f sre/Makefile <target>
# or                       make -C sre <target>

SHELL  := /bin/bash
COMPOSE := docker compose -f ../docker-compose.dev.yml -f docker-compose.sre.yml

.PHONY: help build up down logs ps k8s-apply k8s-delete swarm-up swarm-down \
        tf-aws-init tf-aws-plan tf-aws-apply tf-local-up tf-local-down \
        ansible incident-on incident-off

help:
	@echo "EduLMS SRE Make targets:"
	@echo "  build          - build payment + user-profile images"
	@echo "  up             - docker compose up (base + SRE services + obs)"
	@echo "  down           - tear everything down"
	@echo "  logs           - tail logs"
	@echo "  ps             - list services"
	@echo "  swarm-up       - docker stack deploy"
	@echo "  swarm-down     - docker stack rm"
	@echo "  k8s-apply      - kubectl apply -f k8s/"
	@echo "  k8s-delete     - kubectl delete -f k8s/"
	@echo "  tf-aws-*       - Terraform AWS"
	@echo "  tf-local-*     - Terraform local (docker provider)"
	@echo "  ansible        - run site.yml"
	@echo "  incident-on    - inject 100% failure rate into payment"
	@echo "  incident-off   - restore payment to 2% failure rate"

build:
	docker build -t edulms/payment:1.0.0      services/payment
	docker build -t edulms/user-profile:1.0.0 services/user-profile

up: build
	$(COMPOSE) --profile services --profile obs up -d --build

down:
	$(COMPOSE) down -v

logs:
	$(COMPOSE) logs -f --tail=200

ps:
	$(COMPOSE) ps

swarm-up:
	docker swarm init 2>/dev/null || true
	docker stack deploy -c swarm/docker-stack.yml edulms

swarm-down:
	docker stack rm edulms

k8s-apply:
	kubectl apply -f k8s/

k8s-delete:
	kubectl delete -f k8s/ --ignore-not-found

tf-aws-init:
	terraform -chdir=terraform/aws init

tf-aws-plan:
	terraform -chdir=terraform/aws plan

tf-aws-apply:
	terraform -chdir=terraform/aws apply -auto-approve

tf-local-up:
	terraform -chdir=terraform/local init
	terraform -chdir=terraform/local apply -auto-approve

tf-local-down:
	terraform -chdir=terraform/local destroy -auto-approve

ansible:
	ansible-playbook -i ansible/inventory.ini ansible/site.yml

incident-on:
	FAILURE_RATE=1.0 $(COMPOSE) up -d --no-deps payment
	@echo "Payment now returns 100% errors — watch Prometheus alerts fire."

incident-off:
	FAILURE_RATE=0.02 $(COMPOSE) up -d --no-deps payment
	@echo "Payment failure rate restored to baseline (2%)."
