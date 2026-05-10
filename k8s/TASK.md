# Задание 1.1 — Локальный кластер Kubernetes с Cilium CNI

## Зачем вообще Kubernetes?

Сейчас сервисы проекта запускаются через `docker-compose` — каждый контейнер поднимается вручную на одной машине. Kubernetes (k8s) делает то же самое, но с супервозможностями:

- автоматически перезапускает упавшие поды
- масштабирует поды под нагрузкой
- балансирует трафик между репликами
- управляет конфигурациями, секретами, хранилищем

Если docker-compose — это «запусти контейнеры», то Kubernetes — «следи за ними и держи живыми».

---

## Что такое k3d?

**k3s** — облегчённый дистрибутив Kubernetes от Rancher (весит ~70 МБ вместо стандартных ~500 МБ). Всё то же k8s API, но без редко используемых компонентов.

**k3d** — утилита, которая запускает k3s *внутри Docker-контейнеров*. То есть «ноды кластера» — это просто контейнеры на вашей машине:

```
Docker Desktop (Windows)
└── контейнер: k3d-autoresponse-server-0   ← control plane
└── контейнер: k3d-autoresponse-agent-0    ← worker node
└── контейнер: k3d-autoresponse-agent-1    ← worker node
```

Альтернативы для локального кластера: **kind** (похож), **minikube** (VM), **Talos Linux** (production-grade, сложнее).  
Выбран k3d: быстрее всего стартует, хорошо работает на Windows через Docker Desktop.

---

## Что такое CNI?

**CNI (Container Network Interface)** — плагин, который отвечает за сетевое общение между подами.

Когда под A хочет поговорить с подом B на другой ноде — кто строит этот маршрут? CNI-плагин. Без него поды изолированы и не видят друг друга.

Стандартный CNI в k3s — **Flannel** (простой, VXLAN-туннели). Мы его отключаем и ставим Cilium.

---

## Что такое Cilium?

**Cilium** — CNI-плагин нового поколения. Главное отличие от Flannel/Calico/Weave: он работает через **eBPF**, а не через iptables.

### Что даёт Cilium:

| Возможность | Flannel | Cilium |
|---|---|---|
| Сетевое взаимодействие подов | ✅ | ✅ |
| NetworkPolicy (firewall между подами) | ❌ | ✅ eBPF |
| Observability (кто с кем говорит) | ❌ | ✅ Hubble |
| Балансировка без kube-proxy | ❌ | ✅ |
| Производительность | средняя | высокая |

---

## Что такое eBPF?

**eBPF (extended Berkeley Packet Filter)** — технология Linux-ядра, позволяющая запускать произвольный код *прямо в ядре* без его пересборки.

Раньше чтобы перехватить сетевой пакет нужно было:
```
пакет → ядро → копия в user space → обработка → обратно в ядро
```

С eBPF:
```
пакет → ядро → eBPF-программа прямо здесь → готово
```

Cilium использует это чтобы:
- **фильтровать трафик** (NetworkPolicy) прямо в ядре, без iptables
- **собирать метрики** о каждом соединении (Hubble)
- **балансировать нагрузку** без kube-proxy

---

## Hubble — eBPF observability

Hubble — встроенная в Cilium система наблюдаемости. Через неё видно:

- какой под обращается к какому (граф сервисов)
- какие DNS-запросы идут
- какие соединения дропаются и почему

```
kubectl port-forward -n kube-system svc/hubble-ui 12000:80
# Открыть: http://localhost:12000
```

---

## Архитектура получившегося кластера

```
Windows 11 / Docker Desktop
│
├── k3d-autoresponse-server-0 (control plane)
│     ├── kube-apiserver  :6443
│     ├── etcd
│     ├── cilium-operator
│     └── hubble-relay, hubble-ui
│
├── k3d-autoresponse-agent-0 (worker)
│     ├── cilium (eBPF CNI)
│     └── metrics-server
│
└── k3d-autoresponse-agent-1 (worker)
      └── cilium (eBPF CNI)

CNI: Cilium (flannel отключён)
Observability: Hubble UI → http://localhost:12000
API: kubectl → localhost:6443
```

---

## Как запустить

```powershell
# Установить зависимости (один раз):
winget install k3d.k3d Kubernetes.kubectl Helm.Helm

# Поднять кластер:
.\k8s\setup.ps1

# Проверить:
kubectl get nodes
kubectl get pods -n kube-system
```

---

---

# Задание 2.1 — Terraform: Infrastructure as Code

## Что такое IaC?

**IaC (Infrastructure as Code)** — подход, при котором инфраструктура описывается в коде, а не настраивается вручную.

Без IaC: зашёл на сервер, ввёл команды, потом сам не помнишь что делал.  
С IaC: написал `.tf`-файл → `terraform apply` → инфраструктура создана, воспроизводима, версионирована в git.

## Что такое Terraform?

**Terraform** — самый популярный IaC-инструмент. Ты пишешь конфиг на HCL (HashiCorp Configuration Language), Terraform сам вычисляет что создать/изменить/удалить и делает это.

```
terraform plan   # показать что изменится (dry-run)
terraform apply  # применить
terraform destroy # удалить всё
```

Terraform хранит текущее состояние в `terraform.tfstate`. Если состояние расходится с кластером — `plan` покажет разницу.

## Что мы описали в Terraform

```
k8s/terraform/
├── main.tf              # провайдер kubernetes + backend (local state)
├── variables.tf         # входные параметры (пароли, имена, окружение)
├── namespaces.tf        # 4 неймспейса: autoresponse, kafka, monitoring, argocd
├── service-accounts.tf  # SA для каждого сервиса + Role + RoleBinding
├── secrets.tf           # postgres-credentials, jwt-signing-key, kafka-config
├── outputs.tf           # вывод созданных ресурсов
└── terraform.tfvars.example  # шаблон (скопировать в terraform.tfvars)
```

### Неймспейсы

Каждый логический домен живёт в своём неймспейсе — изоляция, квоты, политики:

| Неймспейс | Что живёт |
|---|---|
| `autoresponse` | Go/Java/Python сервисы проекта |
| `kafka` | Strimzi оператор + Kafka |
| `monitoring` | Prometheus + Grafana |
| `argocd` | ArgoCD сам |

### Service Accounts

По умолчанию под запускается от `default` SA, у которого нет никаких прав. Мы создаём отдельный SA для каждого сервиса с минимальными правами (принцип наименьших привилегий).

### Секреты

`kubernetes_secret` хранит base64-закодированные данные в etcd. В поде они монтируются как переменные окружения:

```yaml
envFrom:
  - secretRef:
      name: postgres-credentials
```

> Пароли передаются через `terraform.tfvars` (в `.gitignore`), не через код.

## Как запустить

```powershell
# Установить Terraform (один раз):
winget install Hashicorp.Terraform

# Перейти в директорию:
cd k8s/terraform

# Скопировать шаблон переменных и заполнить:
Copy-Item terraform.tfvars.example terraform.tfvars
# Открыть terraform.tfvars и заменить CHANGE_ME

# Инициализировать провайдеры:
terraform init

# Посмотреть что будет создано:
terraform plan

# Применить:
terraform apply
```

---

---

# Задание 2.2 — ArgoCD: GitOps

## Что такое GitOps?

**GitOps** — подход к деплою, при котором единственный источник правды — Git.

Без GitOps: `kubectl apply -f ...` вручную, непонятно кто что деплоил.  
С GitOps: всё в git → ArgoCD смотрит в репозиторий → автоматически приводит кластер в соответствие.

```
Разработчик → git push → ArgoCD замечает изменение → kubectl apply автоматически
```

Если кто-то вручную изменил ресурс в кластере — ArgoCD через ~3 минуты откатит обратно (`selfHeal: true`).

## Что такое ArgoCD?

**ArgoCD** — GitOps-контроллер для Kubernetes. Работает внутри кластера:

1. Следит за Git-репозиторием (polling или webhook)
2. Сравнивает манифесты в git с реальным состоянием кластера
3. При расхождении — синхронизирует (применяет манифесты)

У него есть Web UI где видно статус каждого приложения (Synced / OutOfSync / Degraded).

## App of Apps паттерн

Проблема: как задеплоить 10 приложений? Не хочется применять 10 YAML-ов вручную.

Решение — App of Apps:

```
app-of-apps (один объект, применяется вручную один раз)
    │
    └── смотрит в директорию k8s/argocd/apps/
            │
            ├── kafka.yaml       → ArgoCD Application для Kafka
            ├── monitoring.yaml  → ArgoCD Application для Prometheus/Grafana
            └── autoresponse.yaml → ArgoCD Application для сервисов проекта
```

Добавить новый сервис = добавить файл в `k8s/argocd/apps/` и сделать git push. ArgoCD сам его задеплоит.

## Структура файлов

```
k8s/argocd/
├── install/
│   └── values.yaml        # Helm values для самого ArgoCD
├── app-of-apps.yaml        # корневое приложение (применить один раз)
├── apps/
│   ├── kafka.yaml          # sub-app: Strimzi Kafka
│   ├── monitoring.yaml     # sub-app: kube-prometheus-stack
│   └── autoresponse.yaml   # sub-app: сервисы проекта
└── setup.ps1               # скрипт установки
```

## Как запустить

```powershell
# Установить ArgoCD:
.\k8s\argocd\setup.ps1

# Заменить REPO_URL в app-of-apps.yaml на адрес своего репо:
#   git remote get-url origin

# Зарегистрировать корневое приложение:
kubectl apply -f k8s/argocd/app-of-apps.yaml

# Открыть UI:
kubectl port-forward svc/argocd-server -n argocd 8090:80
# http://localhost:8090
```

---

---

# Задание 2.3 — Ansible + Strimzi: Kafka в Kubernetes

## Что такое Ansible?

**Ansible** — инструмент автоматизации. Ты описываешь желаемое состояние в YAML (плейбуках), Ansible выполняет нужные шаги чтобы его достичь.

В отличие от Terraform (инфраструктура), Ansible — для **конфигурации и деплоя**: установить пакеты, скопировать файлы, выполнить команды, применить манифесты.

```
ansible-playbook playbook.yml
```

Ansible подключается к хостам (или к localhost) и выполняет задачи последовательно.

## Что такое Ansible Role?

**Role** — способ организовать плейбук в переиспользуемый модуль. Стандартная структура:

```
roles/kafka_strimzi/
├── tasks/main.yml       # список задач (что делать)
├── defaults/main.yml    # переменные по умолчанию
├── templates/           # Jinja2-шаблоны манифестов
└── meta/main.yml        # метаданные роли
```

В `playbook.yml` просто указываешь `role: kafka_strimzi` — и роль выполняется.

## Что такое Strimzi?

**Strimzi** — Kubernetes оператор для Apache Kafka.

**Оператор** — программа, которая работает внутри k8s и управляет сложным stateful-приложением (Kafka, PostgreSQL, Redis, Elasticsearch...). Ты создаёшь Custom Resource (CR), оператор читает его и создаёт нужные поды, сервисы, конфиги.

```
Ты создаёшь:              Strimzi создаёт автоматически:
KafkaTopic CR    →        реальный топик в Kafka
Kafka CR         →        StatefulSet брокеров
                          StatefulSet ZooKeeper
                          Services (bootstrap, per-broker)
                          ConfigMap с server.properties
```

Без Strimzi: нужно вручную писать StatefulSet, ConfigMap, Headless Service, init-контейнеры, liveness-пробы, обновления rolling...  
Со Strimzi: одна строчка `kind: Kafka` и оператор делает всё сам.

## Что делает роль `kafka_strimzi`

| Шаг | Что происходит |
|---|---|
| 1 | Создаёт неймспейс `kafka` |
| 2 | Добавляет Helm репозиторий Strimzi |
| 3 | Устанавливает Strimzi Cluster Operator через Helm |
| 4 | Ждёт пока оператор готов |
| 5 | Применяет `Kafka` CR (из Jinja2 шаблона) |
| 6 | Ждёт пока кластер Kafka `Ready=True` |
| 7 | Создаёт все 5 топиков через `KafkaTopic` CR |
| 8 | Выводит bootstrap-адрес |

## Топики создаются по списку из `defaults/main.yml`

```yaml
kafka_topics:
  - { name: vacancies.parsed,  partitions: 3, replicas: 1 }
  - { name: parse.requested,   partitions: 1, replicas: 1 }
  - { name: token.updated,     partitions: 1, replicas: 1 }
  - { name: vacancy_input,     partitions: 3, replicas: 1 }
  - { name: vacancy_output,    partitions: 3, replicas: 1 }
```

## Как запустить

```powershell
# Установить Ansible (WSL или Git Bash):
pip install ansible

# Установить коллекцию kubernetes.core:
ansible-galaxy collection install -r k8s/ansible/requirements.yml

# Запустить деплой Kafka:
ansible-playbook k8s/ansible/playbook.yml

# Переопределить переменные (например 3 реплики для prod):
ansible-playbook k8s/ansible/playbook.yml -e kafka_replicas=3
```

После запуска bootstrap-адрес Kafka внутри кластера:
```
autoresponse-kafka-kafka-bootstrap.kafka.svc.cluster.local:9092
```

---

---

# Задание 3.1 — Istio: Service Mesh

## Что такое Service Mesh?

В микросервисной архитектуре сервисы постоянно вызывают друг друга. Возникают вопросы:
- Что если сервис А упал — как сервис Б об этом узнает?
- Как зашифровать трафик между подами?
- Как посмотреть кто с кем говорит?

**Service Mesh** — инфраструктурный слой, который решает эти задачи. Вместо того чтобы встраивать retry/circuit breaker/mTLS в каждый сервис — они выносятся в отдельный прокси.

## Что такое Istio?

**Istio** — самый популярный service mesh. Принцип работы: к каждому поду **автоматически** добавляется sidecar-контейнер `envoy`. Весь входящий и исходящий трафик пода идёт через этот Envoy.

```
Pod A                           Pod B
┌────────────────┐              ┌────────────────┐
│  app container │              │  app container │
│  Envoy sidecar │ ──── mTLS ──→│  Envoy sidecar │
└────────────────┘              └────────────────┘
        ↑                               ↑
        └─── управляет istiod ──────────┘
                (control plane)
```

Приложение **не знает** про mTLS, retry, circuit breaker — всё делает Envoy.

## Что такое Circuit Breaker?

Паттерн "Автоматический выключатель". Когда сервис начинает давать ошибки — лучше сразу вернуть ошибку клиенту, чем ждать таймаут.

```
Состояния:
  CLOSED   → всё нормально, запросы проходят
  OPEN     → сервис нездоров, запросы отклоняются сразу (fast fail)
  HALF-OPEN → пробуем несколько запросов, если ок → CLOSED
```

В Istio это реализовано через **Outlier Detection** в `DestinationRule`:
- `consecutiveGatewayErrors: 5` — 5 ошибок подряд → хост исключается из пула
- `baseEjectionTime: 30s` — минимальное время "карантина" (удваивается при повторах)
- `maxEjectionPercent: 50` — не более половины хостов может быть исключено одновременно

## Retry Policy

В `VirtualService` задаётся сколько раз и при каких условиях повторить запрос:

```yaml
retries:
  attempts: 3          # максимум 3 попытки
  perTryTimeout: 5s    # каждая попытка не дольше 5с
  retryOn: gateway-error,connect-failure,reset
```

`retryOn: reset` — повторить если сервер закрыл соединение. Это идемпотентные операции (GET, HEAD). Для POST (откликнуться на вакансию) retry опасен — указываем только `connect-failure`.

## Что мы настроили

| Файл | Содержимое |
|---|---|
| `istiod-values.yaml` | control plane, включены метрики CB и retry |
| `gateway-values.yaml` | IngressGateway NodePort :30080 |
| `mesh-config/destination-rules.yaml` | Circuit Breaker для 4 сервисов проекта |
| `mesh-config/virtual-services.yaml` | Retry-политики (разные для Go/Java/Python) |
| `mesh-config/peer-authentication.yaml` | mTLS STRICT для namespace autoresponse |

## Как запустить

```powershell
.\k8s\istio\setup.ps1

# Проверить mesh:
kubectl get destinationrule -n autoresponse
kubectl get peerauthentication -n autoresponse
```

---

---

# Задание 3.2 — HAProxy + Keepalived: отказоустойчивая точка входа

## Зачем HAProxy если есть Istio IngressGateway?

| | Istio IngressGateway | HAProxy Ingress |
|---|---|---|
| Протокол | HTTP/2, gRPC | HTTP/1.1, HTTP/2, TCP |
| Конфиг | Istio CRD (Gateway/VS) | Kubernetes Ingress аннотации |
| Алгоритмы LB | round-robin, random | leastconn, source, uri, rdp-cookie |
| Stick Tables | ❌ | ✅ (session affinity по cookie/IP) |
| ACL | через VirtualService | гибкие HAProxy ACL |

HAProxy исторически используется как внешний L7-балансировщик перед кластером. В нашей архитектуре:
```
Client → HAProxy Ingress Controller (L7 routing) → Services
                     ↓
           Istio mesh (mTLS, circuit breaker, retry)
```

## Что такое Keepalived?

**Keepalived** — демон, который реализует **VRRP** (Virtual Router Redundancy Protocol).

Когда несколько серверов держат один **Virtual IP (VIP)** — это HA. Keepalived выбирает MASTER, который держит VIP. Если MASTER падает — BACKUP перехватывает VIP за миллисекунды.

```
                VIP: 192.168.1.200
                        │
            ┌───────────┴───────────┐
      MASTER (priority 110)    BACKUP (priority 100)
      держит VIP               ждёт VRRP heartbeat
      
  → MASTER упал → BACKUP поднимает VIP → никакого простоя
```

В Kubernetes Keepalived запускается как **DaemonSet** на ingress-нодах.

## Структура файлов

```
k8s/ingress/
├── haproxy-values.yaml   # Helm values для HAProxy Ingress Controller
├── keepalived.yaml       # DaemonSet + ConfigMap с VRRP конфигурацией
├── ingress-routes.yaml   # Istio Gateway+VS + Kubernetes Ingress для HAProxy
└── setup.ps1
```

## Как запустить

```powershell
.\k8s\ingress\setup.ps1

# HAProxy NodePort:
# http://localhost:31080
# Istio NodePort:
# http://localhost:30080
```

---

---

# Задание 3.3 — Rate Limiting: Envoy + Valkey

## Зачем Rate Limiting в k8s?

В docker-compose проекта ограничение было в nginx.conf: 5 req/min на `/auth`, 60 req/min на `/api`. В Kubernetes это нужно сделать на уровне Istio/Envoy — иначе нет единой точки контроля когда сервисов много.

## Компоненты

```
Клиент → Envoy (IngressGateway)
              │
              │ gRPC: "разрешить этот запрос?"
              ▼
    Envoy Rate Limit Service
              │
              │ GET/INCR счётчик
              ▼
           Valkey (Redis-совместимый)
```

**Valkey** — open-source форк Redis (BSD лицензия, возник после смены лицензии Redis). Полностью совместим по протоколу. Хранит счётчики вида `autoresponse-ratelimit_remote_address_1.2.3.4 = 42`.

**Envoy Rate Limit Service** (envoyproxy/ratelimit) — Go-сервис, принимает gRPC запросы, проверяет и инкрементирует счётчики в Redis/Valkey, возвращает `OK` или `OVER_LIMIT`.

**EnvoyFilter** — Istio CRD, позволяет патчить конфиг Envoy напрямую. Мы добавляем HTTP-фильтр `envoy.filters.http.ratelimit` к IngressGateway и прикрепляем `rate_limit actions` к маршрутам.

## Как работает EnvoyFilter

Istio управляет Envoy через xDS (API для динамической конфигурации). `EnvoyFilter` позволяет сделать хирургический патч поверх того что Istio сгенерировал:

```yaml
applyTo: HTTP_FILTER          # что патчим
match:
  context: GATEWAY            # где патчим (IngressGateway)
patch:
  operation: INSERT_BEFORE    # как патчим
  value: { ... }              # что вставляем
```

## Правила из конфига

| Путь | Лимит | Назначение |
|---|---|---|
| `/auth/*` | 5 req/min | Защита от брутфорса паролей |
| `/api/*` | 60 req/min | Стандартный API лимит |
| Любой путь | 120 req/min per IP | Общий лимит per-IP |

## Как запустить

```powershell
# Сначала должен быть установлен Istio:
.\k8s\istio\setup.ps1

# Затем:
.\k8s\ratelimit\setup.ps1

# Проверить конфиг:
kubectl port-forward svc/ratelimit -n ratelimit 6070:8080
curl http://localhost:6070/rlconfig
```

---

---

# Блок 4 — Observability

## Выбор стека: сравнение вариантов

### Логи

| | **Loki** | **VictoriaLogs** | **ELK** | **SigNoz** |
|---|---|---|---|---|
| Модель хранения | Чанки по лейблам | Лейблы + полнотекст | Инвертированный индекс | ClickHouse |
| Минимум RAM | ~100 МБ | ~50 МБ | **~2 ГБ** | ~1 ГБ |
| Язык запросов | LogQL | LogQL-совместим | KQL / Lucene | SQL-подобный |
| Интеграция с k8s | ★★★★★ | ★★★★ | ★★★ | ★★★★ |
| Grafana native | ✅ | через Loki API | через плагин | ✅ |
| Полнотекстовый поиск | ❌ | ✅ | ✅ | ✅ |

**Выбран Loki**: нативная интеграция с Grafana, тот же label-подход что и у Prometheus, минимальные ресурсы. VictoriaLogs интересен как альтернатива (меньше RAM + полнотекст), но меньше community. ELK слишком тяжёлый для учебного проекта.

---

### Метрики

| | **Prometheus** | **VictoriaMetrics** | **InfluxDB** |
|---|---|---|---|
| Модель | Pull | Pull + Push | Push |
| Минимум RAM | ~500 МБ | **~50 МБ** | ~200 МБ |
| Сжатие | TSDB | 5–10× лучше TSDB | TSM |
| PromQL | ✅ native | ✅ совместим | ❌ InfluxQL |
| Долгосрочное хранение | Ограничено | ✅ | ✅ |
| Миграция с Prometheus | — | **drop-in замена** | Ломает всё |

**Выбран VictoriaMetrics**: drop-in замена Prometheus — одна строчка в конфиге. Существующий `prometheus.yml` работает без изменений. В 5–10 раз меньше памяти. В проекте уже есть scrape config и Grafana dashboard — просто меняем backend.

---

### Трейсы

| | **Jaeger** | **Tempo** | **Uptrace** |
|---|---|---|---|
| Хранилище | ES / Cassandra / Badger | Объектное / локальный диск | ClickHouse |
| Минимум RAM | ~200 МБ | **~50 МБ** | ~500 МБ |
| Grafana native | через плагин | ✅ native | через плагин |
| OTLP | ✅ | ✅ | ✅ |
| Trace → Logs | ручная настройка | **автоматически** через Grafana | ручная настройка |
| Граф сервисов | ✅ | ✅ через Grafana | ✅ |

**Выбран Tempo**: самый легковесный, Grafana-native, генерирует RED-метрики (Rate/Errors/Duration) из трейсов и пишет их в VictoriaMetrics. Главное преимущество — автоматическая корреляция trace ↔ logs ↔ metrics в одном Grafana-интерфейсе.

---

## Итоговый выбранный стек

```
┌─────────────────────────────────────────────────────────┐
│                        GRAFANA                          │  ← единый UI
│              (метрики + логи + трейсы)                  │
└──────────┬──────────────┬────────────────┬──────────────┘
           │              │                │
    VictoriaMetrics     Loki            Tempo
    (метрики, PromQL)  (логи, LogQL)   (трейсы, OTLP)
           ▲              ▲                ▲
           └──────────────┴────────────────┘
                          │
              OpenTelemetry Collector
              (агент: принимает всё,
               маршрутизирует к бэкендам)
                          ▲
              ┌───────────┴───────────┐
         Promtail               Сервисы (OTLP SDK)
         (k8s pod logs)
```

Это упрощённый **Grafana LGTM-стек** (Loki + Grafana + Tempo + VictoriaMetrics вместо Mimir).

---

## Три сигнала Observability

**Метрики** — *что происходит* (числа во времени): RPS, latency p99, error rate, CPU.  
**Логи** — *почему* (текст событий): stack trace, запрос с параметрами, ошибка.  
**Трейсы** — *как* (цепочка вызовов): запрос прошёл через auth → autoapply → kafka → generate, каждый шаг сколько занял.

Настоящая observability — это переходы между ними:
- Смотришь метрику → видишь аномалию → кликаешь → открываются логи за этот период
- Видишь медленный запрос в логе → кликаешь trace_id → открывается Tempo с цепочкой вызовов
- Видишь span с высокой latency → кликаешь → открываются RED-метрики сервиса

Grafana настроена именно для таких переходов через `derivedFields` и `tracesToLogsV2`.

---

## Что такое OpenTelemetry Collector?

**OpenTelemetry** — открытый стандарт (CNCF) для инструментации и сбора телеметрии.

Без OTel каждый backend (Jaeger, Prometheus, Loki) требует свой агент и свой SDK. Это vendor lock-in.

С OTel: один SDK в приложении (`go.opentelemetry.io/otel`) → отправляет по OTLP протоколу в Collector → Collector маршрутизирует куда нужно.

```
Код приложения
  └── OTel SDK (один)
        └── OTLP → OTel Collector
                      ├── метрики → VictoriaMetrics
                      ├── логи    → Loki
                      └── трейсы  → Tempo
```

Если через год решишь заменить Jaeger на Tempo — меняешь только конфиг Collector, код не трогаешь.

---

## AI-мониторинг: vmanomaly

**vmanomaly** — ML-сервис от VictoriaMetrics для автоматического обнаружения аномалий.

Принцип: читает метрики из VictoriaMetrics → обучает модель (Z-score, Prophet, MAD) → пишет обратно `anomaly_score` → VMAlert может поднять алерт если score > порога.

```yaml
# Что мониторить:
queries:
  autoapply_rps:
    expr: 'rate(http_requests_total{job="autoapply-service"}[1m])'

# Модель Z-score: аномалия если значение > 2.5σ от среднего
models:
  zscore:
    class: "model.zscore.ZscoreModel"
    z_threshold: 2.5
```

В Grafana появится метрика `anomaly_score` — можно добавить на dashboard как дополнительный ряд или алерт.

Альтернативы для AI мониторинга:
- **Grafana ML plugin** — anomaly detection прямо в Grafana, без отдельного сервиса
- **Robusta** — AIOps для k8s: автоматически создаёт GitHub Issues из алертов, обогащает их контекстом

---

## Структура файлов

```
k8s/observability/
├── victoriametrics/values.yaml  # scrape config + ресурсы
├── loki/values.yaml             # SingleBinary, filesystem backend
├── tempo/values.yaml            # монолит + metrics generator → VM
├── promtail/values.yaml         # DaemonSet, парсинг Go/Java логов
├── otel-collector/values.yaml   # receivers/processors/exporters
├── grafana/values.yaml          # datasources + корреляции + dashboards
└── setup.ps1                    # всё одной командой (+ опциональный vmanomaly)
```

## Как запустить

```powershell
# Полный стек (~5 мин):
.\k8s\observability\setup.ps1

# Без AI мониторинга:
.\k8s\observability\setup.ps1 -SkipVMAnomaly

# Открыть Grafana:
kubectl port-forward svc/grafana -n monitoring 3000:80
# http://localhost:3000  (admin/admin)
```

---

# Блок 5 — CI/CD и окружение разработки

## Задание 5.1 — GitHub Actions Self-Hosted (ARC)

### Зачем Self-Hosted Runner?

GitHub Actions по умолчанию запускает джобы на облачных машинах GitHub (ubuntu-latest и т.д.). Для нашего кластера это не подходит: образы должны пушиться в **локальный** registry внутри k3d, а до него с облачного runner'а не добраться.

Решение — **Self-Hosted Runner**: pod внутри кластера, который GitHub использует как исполнителя джобов.

### ARC — Actions Runner Controller

**ARC (Actions Runner Controller)** — Kubernetes-оператор, который управляет runner-подами через Custom Resources:

```
GitHub   →  webhook  →  ARC Controller  →  создаёт Pod с runner'ом
Actions                  (в k8s)              ↓
                                         Runner Pod  →  выполняет джоб  →  done
                                         (автоматически удаляется)
```

Ключевые понятия:
- **`gha-runner-scale-set-controller`** — сам оператор (управляет масштабированием)
- **`gha-runner-scale-set`** — набор runner'ов для конкретного репозитория
- **minRunners / maxRunners** — автомасштабирование: 0 runner'ов в тишине, N под нагрузкой
- **`containerMode: kubernetes`** — каждый runner — отдельный Pod (изоляция)

### Почему не GitLab Runner?

GitLab Runner — аналог для GitLab CI. Репозиторий на GitHub → используем ARC. Если бы репозиторий был на GitLab — аналогично `gitlab-runner register` с executor `kubernetes`.

---

## Задание 5.2 — Пайплайн: Kaniko → Registry → Helm → ArgoCD

### Что такое Kaniko?

Docker требует Docker daemon с root-правами — это **небезопасно** в Kubernetes pod'е. **Kaniko** строит образы прямо из Dockerfile, без демона, работая полностью в пространстве пользователя:

```
Kaniko Pod
├── читает Dockerfile и контекст сборки
├── распаковывает base image слой за слоем в /kaniko/
├── выполняет каждый RUN-шаг в изолированной ФС
└── пушит финальный образ в registry (без docker push)
```

Никаких привилегированных контейнеров. Никакого Docker socket. Безопасно.

**Кеш**: `--cache=true` сохраняет промежуточные слои в registry как теги — повторная сборка только пересобирает изменившиеся слои.

### Локальный Docker Registry

Для хранения образов разворачиваем **Docker Registry v2** прямо в k3d:

```
NodePort 30050  ←  docker push localhost:30050/auth-service:abc123
                   ↑
             registry pod в namespace cicd
             (HTTP, без TLS — только внутри кластера)
```

Адреса:
- Снаружи кластера: `localhost:30050`
- Внутри кластера: `registry.cicd.svc.cluster.local:5000`

### GitOps Image Update Pattern

Проблема: ArgoCD следит за Git. Но как сообщить ArgoCD, что собран новый образ?

Решение — **GitOps image update pattern**:

```
1. Kaniko собирает образ → пушит localhost:30050/auth-service:abc1234
2. CI пайплайн делает: sed -i 's|tag:.*|tag: "abc1234"|' k8s/charts/auth-service/values.yaml
3. git commit && git push  ← фиксируем новый тег в репозитории
4. ArgoCD обнаруживает изменение values.yaml → синхронизирует
5. Kubernetes обновляет Deployment с новым тегом образа
```

Таким образом **Git — единственный источник истины**: в values.yaml всегда видно, какая версия образа работает.

### Полный пайплайн (`.github/workflows/build-and-deploy.yml`)

```
push to main
     │
     ▼
detect-changes          # dorny/paths-filter проверяет, что изменилось
     │
     ├──→ auth-service?    ──→ build-auth     # Kaniko → localhost:30050
     ├──→ autoapply?        ──→ build-autoapply
     ├──→ aggregate?        ──→ build-aggregate
     └──→ generate?         ──→ build-generate
                                    │
                                    ▼
                           update-chart      # обновляет image.tag в values.yaml
                                    │
                                    ▼
                           argocd-sync       # argocd app sync + wait
```

Параллельная сборка: все затронутые сервисы собираются одновременно. Если изменился только auth-service — три другие джобы пропускаются.

---

## Задание 5.3 — Helm-чарты для микросервисов

### Что такое Helm и зачем чарты?

**Helm** — пакетный менеджер для Kubernetes. Вместо 5-10 разрозненных YAML-файлов для одного сервиса — один **чарт** с шаблонами и одним файлом значений.

```
k8s/charts/auth-service/
├── Chart.yaml          # метаданные (имя, версия)
├── values.yaml         # все настройки, которые можно переопределить
└── templates/
    ├── _helpers.tpl    # переиспользуемые фрагменты (метки, имена)
    ├── deployment.yaml # шаблон Deployment
    ├── service.yaml    # шаблон Service
    └── hpa.yaml        # шаблон HPA (условно)
```

**Helm template** — это Go-шаблон. `{{ .Values.image.tag }}` подставляет значение из values.yaml. `{{- if .Values.hpa.enabled }}` условно включает блок.

### Как секреты попадают в поды

Terraform создал секреты в Kubernetes (задание 2.1). Helm-чарт ссылается на них:

```yaml
# values.yaml
externalSecrets:
  postgres: postgres-credentials   # имя секрета из Terraform
  jwt:      jwt-signing-key
  kafka:    kafka-config
```

```yaml
# templates/deployment.yaml  →  envFrom монтирует ВСЕ ключи секрета как env
envFrom:
  - secretRef:
      name: postgres-credentials   # POSTGRES_HOST, POSTGRES_PASSWORD, ...
  - secretRef:
      name: kafka-config           # KAFKA_BOOTSTRAP_SERVERS
```

Pod получает переменные окружения автоматически. Значения секретов **не хранятся** в Git.

### Специфика каждого сервиса

| Сервис | Секреты | Дополнительная конфигурация |
|--------|---------|----------------------------|
| auth-service | postgres + jwt | SERVER_PORT, LOG_LEVEL |
| autoapply-service | postgres + jwt + kafka | + Redis (Valkey), MinIO, 5 Kafka топиков |
| aggregate-service | postgres + jwt + kafka | + JAVA_TOOL_OPTIONS (JVM), Spring Actuator, планировщик |
| generate-service | kafka | + Ollama URL/model/timeout (LLM), нет HTTP-порта |

**generate-service** — особый случай: это Kafka consumer без HTTP-эндпоинта.
- `service.enabled: false` → Service не создаётся (нет смысла)
- `hpa.enabled: false` → масштабирование через **KEDA** по Kafka lag, а не по CPU

### KEDA vs HPA для Kafka consumer

**HPA** масштабирует по CPU/RAM. Для Kafka consumer CPU почти нулевой между сообщениями — HPA будет думать, что реплики не нужны.

**KEDA (Kubernetes Event-Driven Autoscaler)** масштабирует по **бизнес-метрикам**: количество непрочитанных сообщений в топике (`lag`). Если в `vacancy_input` накапливается 100 сообщений — KEDA добавляет реплики generate-service. Очередь опустела — реплики убираются вплоть до 0.

### HPA v2 (autoscaling/v2)

В чартах используется `autoscaling/v2` (не устаревший `v1`):

```yaml
apiVersion: autoscaling/v2
metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

v2 позволяет задавать несколько метрик, custom metrics, поведение при scale-down (stabilizationWindowSeconds).

---

## Структура файлов Блока 5

```
k8s/
├── cicd/
│   ├── arc-values.yaml      # ARC runner scale set
│   ├── registry-values.yaml # Docker Registry v2 (NodePort 30050)
│   ├── kaniko-rbac.yaml     # SA + ClusterRole + Secret для Kaniko
│   └── setup.ps1            # разворачивает всё одной командой
├── charts/
│   ├── auth-service/        # Go :8082, postgres + jwt
│   ├── autoapply-service/   # Go :8081, postgres + jwt + kafka + redis + minio
│   ├── aggregate-service/   # Java Spring Boot :8080, postgres + jwt + kafka
│   └── generate-service/    # Python Kafka consumer, нет HTTP
└── ...
.github/
└── workflows/
    └── build-and-deploy.yml # GitHub Actions пайплайн
.gitlab-ci.yml               # GitLab CI альтернатива
```

## Как запустить

```powershell
# Установить ARC + registry + Kaniko RBAC:
.\k8s\cicd\setup.ps1 -GithubOwner "ваш_логин" -GithubPAT "ghp_..."

# Вручную задеплоить один чарт:
helm upgrade --install auth-service .\k8s\charts\auth-service `
    --namespace autoresponse --create-namespace

# Проверить пайплайн:
# Push в main → открыть GitHub → Actions → Build and Deploy
```

---

# Блок 6 — Тестирование и валидация платформы

## Задание 6.1 — Нагрузочное тестирование с Locust

### Что такое Locust?

**Locust** — Python-фреймворк для нагрузочного тестирования. Тесты описываются как обычный Python-код, что позволяет реализовать сложное поведение: авторизацию, хранение состояния между запросами, вероятностные задачи.

```
Locust Master  ←  Web UI (http://localhost:8089)
     │               задаёшь Users + SpawnRate
     │
     ├── Worker Pod 1 ──→  /auth/login → JWT
     ├── Worker Pod 2 ──→  /api/autoapply POST → Kafka events
     └── Worker Pod 3 ──→  /api/autoapply/{id} GET → Redis hit
```

### Два типа пользователей в тесте

**BrowseUser** (вес 3 — большинство трафика):
- `GET /api/vacancies` (5x) — читает список вакансий
- `GET /api/hh-token` (2x) — проверяет статус HH-токена
- `GET /api/scheduler` (1x) — статус планировщика
- Генерирует HTTP-нагрузку, не вызывает Kafka

**ApplyUser** (вес 1 — запускает EDA-флоу):
- `POST /api/autoapply` (2x) → **весь Kafka-пайплайн**:
  - Go публикует `parse.requested`
  - Java парсит вакансии → `vacancies.parsed`
  - Go публикует `vacancy_input` для каждой вакансии
  - Python генерирует письмо → `vacancy_output`
- `GET /api/autoapply/{id}` (4x) — периодический опрос (нагружает Redis)
- `POST /api/archive/run` (1x) — тяжёлый запрос (MinIO)

### Зачем разные типы пользователей?

Реальный трафик неоднороден. 10 обычных пользователей читают вакансии на каждого, кто отправляет отклик. Если запускать только тяжёлые `POST /api/autoapply`, тест не отражает реальность и нагружает Kafka неправильно.

### Что наблюдать во время теста

В Locust Web UI:
- **RPS** — запросов в секунду (ожидаем ~20–50 для 20 users)
- **Failures %** — должно быть 0% (кроме 429 — это нормально)
- **P95 latency** — ожидаем <200ms для GET, <2000ms для POST autoapply
- **Charts** — видна нагрузка по времени

В Grafana (Platform Validation dashboard):
- Рост `HTTP Request Rate` в реальном времени
- Kafka Consumer Lag dashboard — рост lag при старте, снижение по мере обработки

### Как запустить

```powershell
# Локально (нужен pip install locust):
.\tests\load\run.ps1 -Mode local -Host http://localhost:30080 -Users 20 -SpawnRate 2

# В кластере (Locust pods):
.\tests\load\run.ps1 -Mode k8s

# Без UI (CI/CD):
.\tests\load\run.ps1 -Mode headless -RunTime 5m
```

---

## Задание 6.2 — Проверка Circuit Breaker (Istio Outlier Detection)

### Как работает Circuit Breaker в Istio

В Istio CB реализован через **Outlier Detection** в `DestinationRule`. Это не классический CB (half-open/closed/open), а **пассивное обнаружение деградирующих эндпоинтов** в балансировщике нагрузки Envoy:

```
Incoming request
       │
   Envoy sidecar
       │
       ├── Эндпоинт A (healthy)  ← трафик идёт сюда
       ├── Эндпоинт B (ejected!) ← исключён на baseEjectionTime=30s
       └── Эндпоинт C (healthy)  ← трафик идёт сюда
```

Параметры в `DestinationRule` (настроены в Блоке 3):
- `consecutiveGatewayErrors: 5` — 5 подряд 5xx → эндпоинт выбрасывается
- `interval: 10s` — интервал анализа
- `baseEjectionTime: 30s` — минимум времени в выброшенном состоянии
- `maxEjectionPercent: 50` — не выбрасывать больше 50% эндпоинтов

### Fault Injection — как создать искусственный сбой

Istio позволяет **внедрять сбои** без изменения кода через `VirtualService`:

```yaml
fault:
  abort:
    httpStatus: 503
    percentage:
      value: 50.0   # 50% запросов → немедленный 503
```

Это делает Envoy sidecar `autoapply-service` — он возвращает 503 ещё до того, как запрос достигает пода. С точки зрения вызывающего сервиса — это реальный сбой.

### Сценарий проверки

```
1. kubectl apply -f tests/chaos/fault-injection.yaml
   → 50% запросов на autoapply-service → 503

2. Нагружаем ~10 RPS (Locust или curl-цикл в circuit-breaker-test.ps1)

3. Через 5 ошибок подряд: Envoy выбрасывает unhealthy endpoint
   envoy_cluster_outlier_detection_ejections_active > 0  ← CB открылся

4. В Grafana: Panel "CB Active Ejections" становится красной

5. kubectl delete -f tests/chaos/fault-injection.yaml
   → через baseEjectionTime (30s) CB закрывается автоматически

6. Grafana: Panel "CB Active Ejections" → 0, Error Rate → 0%
```

### Разница между задержкой и абортом

| Тип fault | Эффект | Что тестирует |
|-----------|--------|---------------|
| `abort: 503` | Немедленный ответ 503 | Outlier Detection, ретраи |
| `delay: 3s` | Замедляет ответ | Timeout policy, P99 latency |
| Оба вместе | Реалистичный деградирующий сервис | Комплексная устойчивость |

`aggregate-fault-injection` в fault-injection.yaml применяет оба типа — 30% медленных + 20% ошибок.

### Как запустить

```powershell
# Полный сценарий (fault + нагрузка + наблюдение + восстановление):
.\tests\chaos\circuit-breaker-test.ps1

# Только наблюдение (Locust уже запущен):
.\tests\chaos\circuit-breaker-test.ps1 -SkipLoad
```

---

## Задание 6.3 — Валидация дашбордов Grafana

### Три дашборда для валидации

#### 1. Kafka Consumer Lag (`kafka-lag.json`)

**Ключевые панели:**

| Панель | PromQL | Что показывает |
|--------|--------|----------------|
| Consumer Group Lag | `kafka_consumergroup_lag` | Отставание consumers от producers |
| Messages/sec | `rate(kafka_topic_partition_current_offset[2m])` | Пропускная способность |
| Lag rate of change | `rate(kafka_consumergroup_lag[2m])` | Догоняет или отстаёт consumer |
| generate-consumer lag | `kafka_consumergroup_lag{consumergroup="generate-consumer"}` | Нагрузка на Python/Ollama |

**Что увидеть во время нагрузочного теста:**
1. Старт теста → `vacancy_input` lag начинает расти
2. Python generate_service обрабатывает → lag снижается
3. При высоком RPS → lag растёт значительно (Ollama не успевает) → здесь KEDA добавит реплики

#### 2. Platform Validation (`platform-validation.json`)

**Ключевые панели:**

| Панель | PromQL | Что показывает |
|--------|--------|----------------|
| HTTP Request Rate | `rate(istio_requests_total[1m])` | RPS по статус-кодам |
| Latency P50/P95/P99 | `histogram_quantile(0.99, ...)` | Хвосты латентности |
| CB Ejections/s | `rate(envoy_cluster_outlier_detection_ejections_enforced_total[1m])` | Circuit Breaker активность |
| Rate Limiter hits | `rate(ratelimit_service_rate_limit_over_limit[1m])` | Срабатывания rate limit |
| Error % | `istio_requests_total{response_code=~"5.."}` | Доля ошибок |

**Что увидеть:**
- До fault injection: все панели зелёные
- После `kubectl apply -f fault-injection.yaml`: "Error Rate %" → красный, "CB Active Ejections" → 1+
- После `kubectl delete -f fault-injection.yaml`: всё возвращается в зелёный через ~30s

#### 3. Community dashboard — Kafka (Grafana.com ID: 7589)

Strimzi Kafka Operator dashboard — подробная информация о брокерах, репликации, ISR.

### Откуда берутся метрики

```
kafka_consumergroup_lag     ← Strimzi Kafka Exporter (kafkaExporter: {} в Kafka CR)
                                сервис: kafka-autoresponse-kafka-exporter.kafka:9404

istio_requests_total        ← Envoy sidecar каждого пода
envoy_cluster_outlier_*     ← Envoy sidecar (outlier detection stats)

ratelimit_service_*         ← Envoy Rate Limit Service (ratelimit.ratelimit:8080)

Все scraped → VictoriaMetrics → отображаются в Grafana
```

### Как применить дашборды

```powershell
# Создать ConfigMap с JSON дашбордами:
kubectl create configmap grafana-dashboards-autoresponse `
  --from-file=kafka-lag.json=k8s/observability/grafana/dashboards/kafka-lag.json `
  --from-file=platform-validation.json=k8s/observability/grafana/dashboards/platform-validation.json `
  -n monitoring --dry-run=client -o yaml | kubectl apply -f -

# Перезапустить Grafana (подхватит новый ConfigMap):
kubectl rollout restart deployment/grafana -n monitoring

# Открыть Grafana:
kubectl port-forward svc/grafana -n monitoring 3000:80
# http://localhost:3000 → папка "AutoResponse Platform"
```

---

## Структура файлов Блока 6

```
tests/
├── load/
│   ├── locustfile.py        # два типа пользователей (BrowseUser + ApplyUser)
│   ├── locust.yaml          # Kubernetes: master + 3 workers + Service NodePort 31089
│   └── run.ps1              # local | k8s | headless режимы
└── chaos/
    ├── fault-injection.yaml # Istio VS: 50% abort 503 на autoapply + delay+abort на aggregate
    └── circuit-breaker-test.ps1  # полный CB validation workflow

k8s/observability/
├── grafana/
│   ├── dashboards/
│   │   ├── kafka-lag.json           # consumer lag, message rate, EDA topics
│   │   └── platform-validation.json # CB ejections, rate limits, error rate, latency
│   ├── dashboards-cm.yaml           # ConfigMap → монтируется в Grafana
│   └── values.yaml                  # + dashboardProviders.custom + dashboardsConfigMaps
├── victoriametrics/
│   └── values.yaml          # + scrape jobs: kafka-exporter, istiod, ratelimit
└── ...

k8s/ansible/roles/kafka_strimzi/templates/
└── kafka_cluster.yaml.j2    # + kafkaExporter (consumer lag metrics)
```

## Полный сценарий валидации (15 минут)

```
1. Запустить нагрузочный тест:
   .\tests\load\run.ps1 -Mode local -Users 20

2. Открыть Grafana → "AutoResponse Platform" → "Platform Validation":
   • Видим: RPS ~20, Error % = 0, CB Ejections = 0

3. Открыть Grafana → "AutoResponse Platform" → "Kafka Consumer Lag":
   • Видим: lag по vacancy_input начинает расти

4. Применить fault injection (в отдельном терминале):
   kubectl apply -f tests/chaos/fault-injection.yaml

5. В Grafana → Platform Validation:
   • HTTP 503 rate растёт
   • Через ~10с: CB Active Ejections > 0 (красный)
   • Error Rate % > 0

6. Убрать fault injection:
   kubectl delete -f tests/chaos/fault-injection.yaml

7. Наблюдаем восстановление (~30с):
   • CB Ejections → 0
   • Error Rate → 0%
   • P99 latency возвращается к норме

8. Остановить Locust: Ctrl+C
```
