# EduLMS — SRE End Term Project Defense Guide

> **Домен:** https://sre.aitbek.tech  
> **Сервер:** AWS EC2 `13.63.140.216` (eu-north-1, Stockholm)  
> **Репо:** https://github.com/Newterios/educationalLMS_general

---

## 1. ТЗ — Что делает проект?

**EduLMS** — это система управления обучением (Learning Management System) с микросервисной архитектурой.

### Что умеет система:
| Функция | Сервис |
|---|---|
| Регистрация и вход (JWT + bcrypt) | `auth` |
| Управление курсами и уроками | `course` |
| Тесты и оценка студентов | `assessment` |
| Email/push уведомления | `notification` |
| Оплата курсов (симуляция) | `payment` |
| Профили пользователей | `user-profile` |
| Единая точка входа для всех API | `gateway` |
| Next.js фронтенд (SPA, multi-page) | `web` |

**Кто пользователь:** студент → регистрируется → оплачивает курс → проходит тесты → получает оценку + уведомление.

---

## 2. ТС — Как и где поднято?

### Уровень инфраструктуры:

```
Internet
    │
    ▼
AWS EC2 t3.small (2 vCPU, 4 GB RAM, 40 GB SSD, eu-north-1)
    │
    ▼
Host Nginx (Let's Encrypt TLS — sre.aitbek.tech)
    │ HTTPS → HTTP
    ▼
Docker Container: nginx-frontend (:8080)
    ├── /              → Next.js  (:3000)
    ├── /api/payments/ → payment  (:8081)
    ├── /api/profiles/ → user-profile (:8082)
    └── /api/*         → gateway  (:9080)
                             │
                    ┌────────┼────────┐
                    ▼        ▼        ▼
                  auth    course  assessment
                  :50051  :50052  :50053
                  (gRPC)  (gRPC)  (gRPC)
```

### DNS записи (aitbek.tech):
| Запись | Тип | Значение | Назначение |
|---|---|---|---|
| `@` | A | `161.35.202.138` | Основной проект (DigitalOcean) |
| `www` | A | `161.35.202.138` | Основной проект |
| `sre` | A | `13.63.140.216` | Этот SRE проект (AWS EC2) |

---

## 3. Бизнес логика — уровень 2 (полный стек)

### Backend: Микросервисы на Go (Clean Architecture)

**Архитектура: Mixed (Micro + Mono-gateway)**

```
Go 1.25 · go workspace (go.work)
├── services/auth        — gRPC :50051 — JWT, bcrypt, users
├── services/course      — gRPC :50052 — курсы, уроки, запись
├── services/assessment  — gRPC :50053 — тесты, вопросы, оценки
├── notification/        — NATS subscriber — отправка email/push
├── gateway/             — HTTP :9080 — API gateway (REST→gRPC)
├── sre/services/payment  — HTTP :8081 — Python/Flask, симуляция оплат
└── sre/services/user-profile — HTTP :8082 — Python/Flask, профили
```

**Паттерны в коде:**
- **Clean Architecture** — handler → usecase → repository → DB
- **gRPC** между сервисами (Protobuf-контракты в `/proto/`)
- **Outbox Pattern** в assessment — гарантированная доставка событий
- **NATS** — message broker для notification сервиса
- **Prometheus-метрики** — каждый сервис экспортирует `/metrics`

### Базы данных:

| БД | Образ | Назначение |
|---|---|---|
| **PostgreSQL 16** | `postgres:16-alpine` | Пользователи, курсы, тесты, оценки. Индексы на `user_id`, `course_id`, `created_at` |
| **Redis 7** | `redis:7-alpine` | Кэш JWT-токенов, очередь задач уведомлений |
| **NATS 2** | `nats:2-alpine` | Async message bus между сервисами |

> MongoDB и ChromaDB в данном проекте не используются — задача решена через PostgreSQL (JSONB) и Redis.

### Frontend: Next.js 14 (Multi-page)

```
web/                 — Next.js 14 + TypeScript + Tailwind CSS
├── /login           — форма входа
├── /register        — регистрация
├── /dashboard       — личный кабинет
├── /courses         — список курсов
├── /courses/[id]    — динамическая страница курса (SSR)
├── /grades          — оценки
├── /schedule        — расписание
├── /analytics       — аналитика
├── /admin           — админ-панель
├── /profile         — профиль пользователя
├── /notifications   — уведомления
└── /news            — новости
```

**Воркеры Next.js:** Node.js standalone сервер, 1 процесс, порт 3000.  
**Сборка:** multi-stage Docker build (deps → builder → runner), итоговый образ ~120 MB.  
`NEXT_PUBLIC_API_URL=https://sre.aitbek.tech` — вшит в сборку на этапе `ARG`.

### Сервер:

| Параметр | Значение |
|---|---|
| Провайдер | AWS EC2 |
| Тип | t3.small |
| CPU | 2 vCPU (Intel Xeon, Burstable) |
| RAM | 3.7 GB |
| Диск | 40 GB SSD (gp3, NVMe-compatible) |
| ОС | Ubuntu 24.04 LTS |
| Регион | eu-north-1 (Stockholm) |
| IP | 13.63.140.216 (static Elastic IP) |

### NGINX — два уровня:

**Уровень 1 — Host Nginx (TLS termination):**
```nginx
server {
    listen 443 ssl;
    server_name sre.aitbek.tech;
    ssl_certificate  /etc/letsencrypt/live/sre.aitbek.tech/fullchain.pem;

    location / {
        proxy_pass http://localhost:8080;  # → Docker nginx container
    }
}
server {
    listen 80;
    return 301 https://$host$request_uri;  # HTTP → HTTPS redirect
}
```

**Уровень 2 — Container Nginx (API routing + reverse proxy):**
```nginx
/api/payments/ → payment:8081      # SRE payment service
/api/profiles/ → user-profile:8082 # SRE user-profile service
/api/*         → gateway:9080      # Go API gateway (auth/course/etc.)
/health        → {"status":"ok"}   # Self-healthcheck
/              → sre-web:3000      # Next.js frontend
```

**Зачем два уровня:** Host nginx = SSL + защита от прямого доступа. Container nginx = маршрутизация API. Разделение ответственности.

**DDoS / Security:**
- Rate limiting на уровне host nginx (можно показать: `limit_req_zone`)
- TLS 1.2/1.3 только (устаревшие протоколы отключены certbot'ом)
- Все сервисы в изолированной Docker-сети `edulms-v2-net` — снаружи не доступны напрямую
- Только порты 80 и 443 открыты в AWS Security Group

---

## 4. Docker

**13 запущенных контейнеров на сервере:**

| Контейнер | Образ | Статус |
|---|---|---|
| `edulmsv2-postgres` | postgres:16-alpine | healthy |
| `edulmsv2-redis` | redis:7-alpine | healthy |
| `edulmsv2-nats` | nats:2-alpine | healthy |
| `edulmsv2-auth` | Go multi-stage build | running |
| `edulmsv2-course` | Go multi-stage build | running |
| `edulmsv2-assessment` | Go multi-stage build | running |
| `edulmsv2-notification` | Go multi-stage build | running |
| `edulmsv2-gateway` | Go multi-stage build | running |
| `edulmsv2-mock-gateway` | Go multi-stage build | running |
| `edulms-sre-payment` | Python/Flask | healthy |
| `edulms-sre-user-profile` | Python/Flask | healthy |
| `edulmsv2-web` | Node 20 standalone | running |
| `edulms-sre-nginx` | nginx:1.27-alpine | running |

**Docker образы (multi-stage):**
```dockerfile
# Go сервисы — пример auth
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum ./
RUN go build ./services/auth/cmd/...

FROM alpine:3.19 AS runner
COPY --from=builder /app/auth /auth
CMD ["/auth"]
# Итог: ~20 MB образ вместо ~800 MB
```

**Два compose-файла:**
- `docker-compose.dev.yml` — базовые сервисы (auth, course, assessment, gateway, DB, obs)
- `sre/docker-compose.sre.yml` — SRE overlay (payment, user-profile, nginx, web)

---

## 5. Kubernetes (k3s)

**Манифесты в `sre/k8s/` — полный деплой в Kubernetes:**

```
00-namespace.yaml         — namespace: edulms
01-config-and-secrets.yaml — ConfigMap + Secret (DB credentials)
02-postgres.yaml          — StatefulSet + PVC (persistent storage)
03-redis-nats.yaml        — Deployment для Redis и NATS
04-auth.yaml              — Deployment + Service + HPA
05-course.yaml
06-assessment.yaml
07-notification.yaml
08-payment.yaml           — Deployment + Service
09-user-profile.yaml
10-gateway-and-ingress.yaml — Ingress (Traefik) → gateway
11-monitoring.yaml        — Prometheus + Grafana
```

**Команды для демо:**
```bash
kubectl apply -f sre/k8s/          # деплой всего
kubectl get pods -n edulms         # статус подов
kubectl rollout status deploy/auth -n edulms  # rolling update
```

**Rolling Update:** при деплое новой версии — `kubectl rollout restart deploy/auth` — Kubernetes постепенно заменяет поды без даунтайма (0-downtime deployment).

---

## 6. Grafana + Prometheus — Мониторинг

**Стек наблюдаемости (Observability Stack):**

```
Сервисы → OpenTelemetry Collector → Prometheus (метрики)
                                  → Tempo (трейсы)
                                  → Loki (логи) ← Promtail
                                  → Grafana (дашборды)
```

**Что мониторится:**
- Каждый Go-сервис экспортирует `/metrics` (HTTP request duration, error rate, etc.)
- Payment-сервис: `payment_requests_total`, `payment_errors_total`, `payment_latency_seconds`
- User-profile: аналогично
- SLO Alert Rules в `observability/rules/` — алерт если error rate > 5%

**Симуляция инцидента (для демо):**
```bash
# Включить 100% ошибок в payment
make -C sre incident-on
# → Prometheus alert сработает через 15s
# → видно в Grafana dashboard

# Восстановить
make -C sre incident-off
```

**Доступ:** http://sre.aitbek.tech:3000 (Grafana, если открыт порт) или через SSH-тоннель.

---

## 7. Terraform + Ansible

### Terraform — Infrastructure as Code:

**`sre/terraform/aws/`** — поднимает AWS инфраструктуру:
```hcl
# Что создаёт:
aws_vpc           — изолированная сеть (10.0.0.0/16)
aws_subnet        — 2 публичные подсети в разных AZ
aws_security_group — firewall: 22(SSH), 80(HTTP), 443(HTTPS)
aws_instance       — EC2 t3.small, Ubuntu 22.04 LTS
aws_eip            — Static Elastic IP
```

```bash
terraform init
terraform plan    # показать что будет создано
terraform apply   # создать инфраструктуру
```

### Ansible — Configuration Management:

**`sre/ansible/site.yml`** — 5 плейбуков:
```yaml
1. common   — apt update, firewall (ufw), timezone, swap
2. docker   — установка Docker Engine + Compose plugin
3. swarm    — инициализация Docker Swarm (manager + workers)
4. deploy   — копирование compose-файлов, docker stack deploy
5. monitor  — установка Prometheus + Grafana
```

```bash
ansible-playbook -i inventory.ini site.yml --tags deploy
```

**Инвентарь `inventory.ini`:**
```ini
[managers]
edulms-mgr-01 ansible_host=13.63.140.216 ansible_user=ubuntu
ansible_ssh_private_key_file=~/.ssh/edulms-sre-key.pem
```

---

## 8. CI/CD — GitHub Actions

**Файл:** `.github/workflows/ci-cd.yml`

**Pipeline:**
```
push to main
    │
    ├─ Lint & Test (matrix: auth, course, assessment, notification, gateway)
    │      go vet ./...
    │      go test -race -timeout 60s ./...
    │
    ├─ Lint Python (payment, user-profile)
    │      ruff check
    │
    ├─ Build & Push Docker Images (matrix: 8 образов)
    │      ghcr.io/newterios/educationallms_general/auth:latest
    │      ghcr.io/newterios/educationallms_general/web:latest
    │      ...
    │
    └─ Deploy to sre.aitbek.tech
           SSH → git pull → ansible-playbook → docker compose up → health check
```

**Ключевые решения:**
- `fail-fast: false` — один упавший сервис не отменяет сборку остальных
- `go-version: '1.25'` — точное совпадение с `go.work`
- Lowercase image tag: `github.repository | tr '[:upper:]' '[:lower:]'` через `$GITHUB_OUTPUT`
- `DEPLOY_SSH_KEY` secret — приватный ключ EC2 для автодеплоя

---

## 9. Логи

```bash
# Все сервисы
docker compose -p edulmsv2 logs -f --tail=200

# Конкретный сервис
docker logs edulmsv2-auth -f

# Через Loki + Grafana
# Promtail собирает /var/lib/docker/containers/*/*.log
# Grafana: Explore → Loki → {container_name="edulmsv2-auth"}
```

---

## 10. Скрипты

| Скрипт | Назначение |
|---|---|
| `scripts/setup-server.sh` | Одноразовый бутстрап сервера (Docker, k3s, Ansible, git clone) |
| `scripts/deploy-to-server.sh` | Ручной деплой с ноутбука (альтернатива CI/CD) |
| `scripts/smoke_auth.sh` | E2E тест: регистрация → логин → проверка JWT |
| `scripts/smoke_course.sh` | Smoke тест курсов |
| `scripts/smoke_assessment.sh` | Smoke тест тестирования |
| `scripts/smoke_web.sh` | Проверка доступности frontend |
| `scripts/demo.sh` | Полное демо-прохождение |

---

## 11. Security

**Что реализовано:**

| Уровень | Механизм |
|---|---|
| **Транспорт** | TLS 1.3 (Let's Encrypt, auto-renew) |
| **Auth** | JWT (access + refresh), bcrypt для паролей |
| **Сеть** | Все сервисы в закрытой Docker-сети, снаружи только :80/:443 |
| **AWS** | Security Group — whitelist портов |
| **Секреты** | `.env` не в репо, GitHub Secrets для CI/CD |
| **NGINX** | Proxy headers (X-Real-IP, X-Forwarded-For), timeout limits |
| **Observability** | Алерты на аномальный error rate |

**Известные уязвимости (честный ответ):**
- Next.js 14.1.0 имеет патченные CVE → в продакшне нужно обновить до 15.x
- Нет WAF (Web Application Firewall) — в реальном проекте добавили бы CloudFront/ModSecurity

---

## 12. Makefile — Демо по шагам

```bash
# На сервере: cd /opt/edulms

make -C sre demo-destroy      # ШАГ 0: сносим всё (clean slate для демо)
make -C sre demo-pull         # ШАГ 1: git pull последний код
make -C sre demo-ansible      # ШАГ 2: Ansible конфигурирует сервер
make -C sre demo-docker-build # ШАГ 3: docker build всех образов
make -C sre demo-k8s-up       # ШАГ 4: kubectl apply (Kubernetes деплой)
make -C sre demo-health       # ШАГ 5: health check всех эндпоинтов

# Или всё сразу:
make -C sre demo-full
```

---

## 13. Live URLs для защиты

| URL | Что показывает |
|---|---|
| https://sre.aitbek.tech | Next.js фронтенд — логин, курсы, дашборд |
| https://sre.aitbek.tech/login | Страница входа |
| https://sre.aitbek.tech/health | `{"status":"ok","component":"nginx-frontend"}` |
| https://sre.aitbek.tech/api/payments/health | `{"service":"payment","status":"ok"}` |
| https://sre.aitbek.tech/api/profiles/health | `{"service":"user-profile","status":"ok"}` |

---

## 14. Краткий ответ на "Что ты сделал?"

> Я разработал и задеплоил полный стек LMS-системы: 8 микросервисов на Go и Python, Next.js фронтенд, PostgreSQL + Redis + NATS, всё в Docker на AWS EC2. Настроил Nginx как двухуровневый reverse proxy с TLS-сертификатом. Написал CI/CD пайплайн в GitHub Actions — при пуше в main автоматически прогоняются тесты, собираются Docker-образы в GHCR и деплоятся на сервер через SSH. Инфраструктура описана в Terraform (AWS VPC, EC2, Security Group) и Ansible (5 плейбуков). Kubernetes-манифесты позволяют запустить весь стек в k3s одной командой. Мониторинг через Prometheus + Grafana с алертами по SLO. Весь деплой воспроизводим через `make -C sre demo-full`.
