# RabbitMQ implementation with DLX, retry queue, and Postgres

Order-processing demo: Fiber API publishes to RabbitMQ (direct exchanges), workers consume with DLX/retry patterns, optional Postgres persistence.

## Features

- RabbitMQ direct exchanges + routing keys
- Dead-letter / retry oriented worker layout
- Fiber HTTP API (`POST /order`)
- **Prometheus metrics** (`/metrics`)
- **Docker Compose** with RabbitMQ, Postgres, Prometheus, Grafana

## Quick start

```bash
docker compose up -d --build
```

| Service | URL |
|---------|-----|
| API | http://localhost:8001 |
| Metrics | http://localhost:8001/metrics |
| RabbitMQ UI | http://localhost:15672 (guest/guest) |
| Prometheus | http://localhost:9091 |
| Grafana | http://localhost:3001 (admin/admin) |

## Metrics

- `orders_submitted_total`
- `rabbitmq_publish_total{exchange,status}`
- `http_requests_total{method,path,status}`

## Layout

```
api/           # Fiber entrypoint + metrics
rabbitmq/      # connection helpers
workers/       # payment-worker, user-order-worker
entity/        # domain types
repository/    # Postgres access
deploy/        # Prometheus + Grafana provisioning
```

## License

MIT

## Free ports before docker compose

```bash
# Stop previous stack
docker compose down

# Free host ports used by this project (Linux)
sudo fuser -k 8001/tcp 2>/dev/null || true
sudo fuser -k 5672/tcp 2>/dev/null || true
sudo fuser -k 15672/tcp 2>/dev/null || true
sudo fuser -k 9091/tcp 2>/dev/null || true
sudo fuser -k 3001/tcp 2>/dev/null || true

docker compose up -d --build
```

| Service | Host port |
|---------|-----------|
| API | 8001 |
| RabbitMQ AMQP | 5672 |
| RabbitMQ UI | 15672 |
| Prometheus | 9091 |
| Grafana | 3001 |
