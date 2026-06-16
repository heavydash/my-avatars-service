# Gophprofile — Микросервис управления аватарками пользователей

> Сервис для загрузки, хранения и управления аватарками пользователей.  
> Реализован на Go с использованием Clean Architecture, развёртывается в Kubernetes через Helm.

## Архитектурная схема

    Client[Клиент<br/>Web / Mobile] --> Ingress[Ingress NGINX]

    Ingress --> Service[Service:80]

    Service --> Server[Server Pod:8085]

    Server -->|SQL| Postgres[(PostgreSQL)]
    Server -->|S3 API| MinIO[(MinIO)]
    Server -->|AMQP| RabbitMQ[(RabbitMQ)]
    Server -->|OTLP| Jaeger[Jaeger<br/>Tracing]

    Worker[Worker Pod] -->|AMQP| RabbitMQ
    Worker -->|S3 API| MinIO
    Worker -->|SQL| Postgres


## Мониторинг и метрики

Сервис экспортирует метрики в формате Prometheus по адресу `GET /metrics`.

### Доступные метрики

| Метрика | Тип | Уровень | Описание                                           |
|---------|-----|---------|----------------------------------------------------|
| `http_requests_total` | Counter | RED | Общее количество HTTP-запросов                     |
| `http_request_duration_seconds` | Histogram | RED | Время обработки HTTP-запросов (сек)                |
| `http_errors_total` | Counter | RED | Количество HTTP-ошибок                             |
| `avatar_uploads_total` | Counter | Business | Количество загруженных аватарок                    |
| `avatar_upload_duration_seconds` | Histogram | Business | Время загрузки аватарки в хранилище                |
| `avatar_deletes_total` | Counter | Business | Количество удалённых аватарок                      |
| `avatar_storage_bytes` | Gauge | Business | Объём занятого места в хранилище (по пользователям) |
| `avatar_storage_objects_total` | Gauge | Business | Общее количество объектов аватарок в хранилище     |

### Как посмотреть метрики

```bash
kubectl port-forward -n gophprofile svc/gophprofile 8085:80


### Как поднять приложение

Запуск Minicube
# Запуск с достаточными ресурсами
minikube start --driver=docker --cpus=3 --memory=6144m --addons=ingress

# Включение ingress (если не включился)
minikube addons enable ingress


# Установка / обновление чарта (из корневой директории)
helm upgrade --install gophprofile ./helm/gophprofile \
  --namespace gophprofile \
  --create-namespace

# Проверка запуска
# Статус подов
kubectl get pods -n gophprofile -w

# Все ресурсы
kubectl get all -n gophprofile


# Доступ к приложению
Bash# Проброс порта
kubectl port-forward -n gophprofile svc/gophprofile 8085:80


Интерфейсы:
Swagger UI: http://localhost:8085/swagger/index.html
Health check: http://localhost:8085/health
Метрики: http://localhost:8085/metrics

# Структура проекта

gophprofile/
├── cmd/server/main.go              # Точка входа HTTP-сервера
├── internal/
│   ├── api/                        # Роутеры, handlers, middleware
│   ├── config/                     # Конфигурация приложения
│   ├── domain/                     # Доменные сущности и ошибки
│   ├── repository/                 # Работа с БД
│   ├── service/                    # Бизнес-логика
│   ├── storage/                    # Работа с MinIO
│   └── pkg/                        # Вспомогательные пакеты (logger, metrics, tracing)
├── migrations/                     # SQL-миграции
├── web/                            # Статические файлы (front-end)
├── helm/gophprofile/               # Helm Chart для Kubernetes
├── k8s/                            # Raw Kubernetes манифесты (deployment, service, ingress и т.д.)
├── Dockerfile
├── .env
├── go.mod
└── README.md

# Технологический стек

Backend: Go 1.25 + Gin
База данных: PostgreSQL
Хранилище: MinIO (S3-совместимое)
Очередь: RabbitMQ
Трейсинг: OpenTelemetry + Jaeger
Мониторинг: Prometheus метрики
Документация: Swagger (OpenAPI)
Деплой: Kubernetes + Helm
Безопасность: non-root, SecurityContext, NetworkPolicy