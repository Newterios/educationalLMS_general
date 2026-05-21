# EduLMS SRE — convenience Makefile.
# Run from the repository root via `make -C sre <target>`.
# Every target jumps to the repo root with `cd $(ROOT)` first, so paths
# in docker-compose files are resolved consistently.

SHELL  := /bin/bash
ROOT   := $(realpath $(dir $(lastword $(MAKEFILE_LIST)))/..)
PROJECT := edulmsv2
COMPOSE := docker compose -p $(PROJECT) -f docker-compose.dev.yml -f sre/docker-compose.sre.yml --profile web

.PHONY: help build up down logs ps k8s-apply k8s-delete swarm-up swarm-down \
        tf-aws-init tf-aws-plan tf-aws-apply tf-local-up tf-local-down \
        ansible incident-on incident-off \
        demo-destroy demo-pull demo-ansible demo-docker-build demo-k8s-up \
        demo-swarm-up demo-health demo-full

help:
	@echo "EduLMS SRE Make targets (run from repo root with 'make -C sre <target>'):"
	@echo ""
	@echo "  Regular targets:"
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
	@echo ""
	@echo "  DEMO targets (whiteboard steps):"
	@echo "  demo-destroy   - STEP 0: stop + remove ALL resources (clean slate)"
	@echo "  demo-pull      - STEP 1: git pull latest code"
	@echo "  demo-ansible   - STEP 2: run Ansible playbooks"
	@echo "  demo-docker-build - STEP 3: build Docker images"
	@echo "  demo-k8s-up    - STEP 4: kubectl apply (Kubernetes deploy)"
	@echo "  demo-swarm-up  - STEP 4b: docker stack deploy (Swarm alternative)"
	@echo "  demo-health    - STEP 5: health check all endpoints"
	@echo "  demo-full      - run ALL demo steps in sequence"

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

# ── DEMO targets (follow the whiteboard: branch→team1/2/3) ───────────────────
# Usage on server:
#   cd /opt/edulms
#   make -C sre demo-destroy   # start of demo — clean slate
#   make -C sre demo-pull      # step 1
#   make -C sre demo-ansible   # step 2
#   make -C sre demo-docker-build  # step 3
#   make -C sre demo-k8s-up    # step 4
#   make -C sre demo-health    # step 5

demo-destroy:
	@echo "════════════════════════════════════════"
	@echo "  DEMO STEP 0 — Destroying all resources"
	@echo "════════════════════════════════════════"
	cd $(ROOT) && kubectl delete -f sre/k8s/ --ignore-not-found 2>/dev/null || true
	docker stack rm edulms 2>/dev/null || true
	cd $(ROOT) && $(COMPOSE) --profile services --profile obs down -v --remove-orphans 2>/dev/null || true
	docker system prune -af --volumes 2>/dev/null || true
	@echo "✓ All resources removed — environment is clean"

demo-pull:
	@echo "════════════════════════════════════════"
	@echo "  DEMO STEP 1 — Pull latest code"
	@echo "════════════════════════════════════════"
	cd $(ROOT) && git fetch --all && git reset --hard origin/main
	@echo "✓ Code up to date: $$(git -C $(ROOT) log --oneline -1)"

demo-ansible:
	@echo "════════════════════════════════════════"
	@echo "  DEMO STEP 2 — Ansible configuration"
	@echo "════════════════════════════════════════"
	cd $(ROOT) && ansible-playbook -i sre/ansible/inventory.ini sre/ansible/site.yml --tags deploy
	@echo "✓ Ansible done"

demo-docker-build:
	@echo "════════════════════════════════════════"
	@echo "  DEMO STEP 3 — Build Docker images"
	@echo "════════════════════════════════════════"
	cd $(ROOT) && $(COMPOSE) --profile services --profile obs build
	@echo "✓ Docker images built"
	@docker images --filter=reference='*edulms*' --format "  {{.Repository}}:{{.Tag}}"

demo-k8s-up:
	@echo "════════════════════════════════════════"
	@echo "  DEMO STEP 4 — Kubernetes deploy"
	@echo "════════════════════════════════════════"
	cd $(ROOT) && kubectl apply -f sre/k8s/
	@echo "Waiting for pods..."
	kubectl -n edulms wait --for=condition=ready pod --all --timeout=180s 2>/dev/null || true
	kubectl -n edulms get pods
	@echo "✓ Kubernetes deploy done"

demo-swarm-up:
	@echo "════════════════════════════════════════"
	@echo "  DEMO STEP 4b — Docker Swarm deploy"
	@echo "════════════════════════════════════════"
	cd $(ROOT) && docker swarm init 2>/dev/null || true
	cd $(ROOT) && docker stack deploy -c sre/swarm/docker-stack.yml edulms
	@sleep 5
	docker stack ps edulms
	@echo "✓ Swarm deploy done"

demo-health:
	@echo "════════════════════════════════════════"
	@echo "  DEMO STEP 5 — Health checks"
	@echo "════════════════════════════════════════"
	@passed=0; failed=0; \
	for ep in \
	    "https://aitbek.tech/health" \
	    "https://aitbek.tech/api/payments/health" \
	    "https://aitbek.tech/api/profiles/health"; do \
	  printf "  %-45s" "$$ep"; \
	  ok=false; \
	  for i in 1 2 3 4 5; do \
	    if curl -fsS --max-time 8 "$$ep" >/dev/null 2>&1; then ok=true; break; fi; \
	    sleep 3; \
	  done; \
	  if $$ok; then echo "✓ OK"; passed=$$((passed+1)); \
	  else echo "✗ FAIL"; failed=$$((failed+1)); fi; \
	done; \
	echo ""; \
	echo "Results: $$passed passed, $$failed failed"; \
	[ $$failed -eq 0 ]

demo-full: demo-destroy demo-pull demo-ansible demo-docker-build demo-k8s-up demo-health
	@echo ""
	@echo "╔══════════════════════════════════════╗"
	@echo "║    EduLMS SRE Demo — COMPLETE ✓      ║"
	@echo "╚══════════════════════════════════════╝"
