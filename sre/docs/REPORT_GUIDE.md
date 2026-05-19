# Гайд: как собрать финальный отчёт с скриншотами

## 0. План действий (10 минут работы)

1. Запустить стек.
2. Сделать 6 скриншотов (команды и URL ниже).
3. Сохранить их в `sre/docs/screenshots/`.
4. Открыть `REPORT.md`, заменить плейсхолдеры (ФИО, ссылка на git, и т.д.).
5. Сконвертировать `REPORT.md` → `REPORT.pdf`.
6. Загрузить PDF.

---

## 1. Поднять стек (если ещё не запущен)

```bash
cd /Users/aitbek/Desktop/final_apt_eduLms
make -C sre build
make -C sre up
# Подождать ~30 сек, проверить:
docker compose -f docker-compose.dev.yml -f sre/docker-compose.sre.yml ps
```

Адреса для скриншотов:
- Prometheus → http://localhost:9090
- Grafana    → http://localhost:3002 (admin / admin)
- Payment    → http://localhost:8081
- Frontend   → http://localhost:8080

---

## 2. Какие скриншоты нужны (6 штук — обязательный минимум)

Сохраняйте все скриншоты в папку `sre/docs/screenshots/` с указанными именами.

### Скриншот 1 — `01-docker-ps.png`
**Что показать**: что все сервисы подняты в Docker.
**Как получить**:
```bash
docker compose -f docker-compose.dev.yml -f sre/docker-compose.sre.yml ps
```
Сделайте скриншот терминала — должен быть виден список с `payment`, `user-profile`, `prometheus`, `grafana`, `postgres`, `redis`, `nats` и т.д. (Status: Up / healthy).

### Скриншот 2 — `02-microservices-running.png`
**Что показать**: ответ от двух новых сервисов.
**Как получить** — в браузере или в терминале:
```bash
curl http://localhost:8081/
curl http://localhost:8082/
curl -X POST http://localhost:8081/pay \
  -H 'Content-Type: application/json' \
  -d '{"amount":1500,"currency":"USD","order_id":"o-42"}'
```
Скриншот терминала с ответами JSON.

### Скриншот 3 — `03-prometheus-targets.png`
**Что показать**: что Prometheus собирает метрики.
**Как получить**: открыть http://localhost:9090/targets — скриншот страницы. Должно быть видно, что таргеты в состоянии UP.

### Скриншот 4 — `04-prometheus-alerts.png`
**Что показать**: правила алертов.
**Как получить**: открыть http://localhost:9090/alerts — скриншот списка правил (PaymentErrorRateAboveSLO, PaymentLatencyP95AboveSLO, и т.д.).

### Скриншот 5 — `05-grafana-dashboard.png`
**Что показать**: SRE-дашборд в Grafana.
**Как получить**:
1. Открыть http://localhost:3002 (admin/admin).
2. Импортировать дашборд: Settings → Dashboards → Import → загрузить файл `sre/monitoring/dashboards/edulms-sre-overview.json`.
3. Сделать запрос несколько раз чтобы появились данные:
   ```bash
   for i in {1..50}; do
     curl -s -X POST http://localhost:8081/pay \
       -H 'Content-Type: application/json' \
       -d '{"amount":100,"order_id":"o-'$i'"}' > /dev/null
   done
   ```
4. Скриншот дашборда с графиками.

### Скриншот 6 — `06-incident-simulation.png`
**Что показать**: сработавший алерт после симуляции инцидента.
**Как получить**:
```bash
# Включить инцидент (100% ошибок)
make -C sre incident-on

# Сгенерировать трафик
for i in {1..100}; do
  curl -s -X POST http://localhost:8081/pay \
    -H 'Content-Type: application/json' \
    -d '{"amount":100,"order_id":"o-'$i'"}' > /dev/null
done

# Подождать 5 минут (нужно для `for: 5m` в алерте)
# Открыть http://localhost:9090/alerts — алерт PaymentErrorRateAboveSLO должен быть FIRING (красный).
# Сделать скриншот.

# Восстановить
make -C sre incident-off
```

### (Опционально) Скриншот 7 — `07-kubernetes-pods.png`
Только если у вас есть кластер K8s (minikube/kind):
```bash
kubectl apply -f sre/k8s/
kubectl get pods -n edulms
kubectl get hpa -n edulms
```

### (Опционально) Скриншот 8 — `08-architecture.png`
Можно либо нарисовать в excalidraw.com / draw.io, либо просто скопировать ASCII-диаграмму из `sre/docs/ARCHITECTURE.md`.

---

## 3. Заполнить REPORT.md

Откройте файл `sre/docs/REPORT.md` (создан рядом с этим гайдом) и замените плейсхолдеры:
- `<ВАШЕ_ФИО>`
- `<ГРУППА>`
- `<URL_РЕПОЗИТОРИЯ>` — например, `https://github.com/your-username/final_apt_eduLms`
- `<ДАТА>`

---

## 4. Конвертировать в PDF

Любой из вариантов:

### Вариант A — VS Code
1. Установить расширение **Markdown PDF** (yzane.markdown-pdf).
2. Открыть `sre/docs/REPORT.md`.
3. Ctrl+Shift+P → "Markdown PDF: Export (pdf)".

### Вариант B — Pandoc (через терминал)
```bash
brew install pandoc           # mac
# или: sudo apt install pandoc texlive-xetex   # linux
cd sre/docs
pandoc REPORT.md -o REPORT.pdf --pdf-engine=xelatex
```

### Вариант C — онлайн
Загрузить `REPORT.md` на https://md2pdf.netlify.app или https://www.markdowntopdf.com.

### Вариант D — самое простое
Открыть `REPORT.md` в любом markdown-вьювере (Typora, Obsidian, GitHub preview) → Print → Save as PDF.

---

## 5. Чек-лист перед сдачей

- [ ] PDF содержит ваше ФИО и группу
- [ ] PDF содержит **рабочую** ссылку на Git (кликабельную, не картинкой)
- [ ] Все 6 скриншотов вставлены и видны в PDF
- [ ] Архитектурная диаграмма есть
- [ ] Список 6+ микросервисов есть
- [ ] Краткое описание инцидента есть (или ссылка на POSTMORTEM.md в репо)
- [ ] Размер PDF разумный (< 10 МБ — если больше, сжать скриншоты)
- [ ] Репозиторий **публичный** (или с доступом для преподавателя)
- [ ] В репозиторий запушено всё: `services/`, `sre/`, `docker-compose.dev.yml`, `Makefile`
