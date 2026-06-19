# FizzBuzz API

## The Exercise

> Write a REST API that exposes a single endpoint `/fizzbuzz` accepting five parameters: `int1`, `int2`, `limit`, `str1`, `str2`.
>
> It returns a JSON list of strings where:
> - multiples of `int1` are replaced by `str1`
> - multiples of `int2` are replaced by `str2`
> - multiples of both are replaced by `str1str2`
>
> Add a `/stats` endpoint that returns the parameters corresponding to the most-used request (i.e., the one that has been called the most), along with the number of hits.

---

## Why not just write to the database on each request?

The naive implementation writes straight to Postgres on every `/fizzbuzz` call:

```
Request → Handler → DB (INSERT/UPDATE hits)
```

That works. But it couples the request path directly to the database. Every request now carries the latency and availability risk of a synchronous DB write. Under load, the database becomes a bottleneck, and a DB hiccup translates directly into API errors.

I wanted to show a design that can survive that pressure and scale horizontally.

---

## What I built instead

```
Request → Handler → Kafka (publish event)
                          ↓
                    Stats Worker (consume) → Postgres
```

The HTTP handler publishes a lightweight event to Kafka and returns immediately. A separate worker process consumes those events and writes to the database at its own pace. The API latency is now bounded by Kafka (microseconds), not the DB.

**Why this matters beyond the exercise:**

- **Decoupling** — the API server and the stats aggregator are independent deployables. You can scale, redeploy, or crash either one without affecting the other.
- **Back-pressure** — if the DB is slow or down, events accumulate in Kafka instead of failing requests. The worker catches up when it recovers.
- **Other consumers** — because the event is on a Kafka topic, any future service can subscribe to it without touching the API. An analytics pipeline, a fraud detector, a recommendation engine — they all just consume the same topic.
- **At-least-once with idempotency** — Kafka can redeliver messages. Rather than accepting duplicate hits, each event carries a `request_id` UUID. The DB upsert only increments `hits` when `last_request_id` changes, so redeliveries are no-ops.

```sql
ON CONFLICT (idempotent_id) DO UPDATE
    SET hits            = fizz_requests.hits + 1,
        last_request_id = EXCLUDED.last_request_id
WHERE fizz_requests.last_request_id != EXCLUDED.last_request_id
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│  Client                                                  │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTP
┌──────────────────────▼──────────────────────────────────┐
│  API Server  (:8081)                                     │
│  GET /v1/fizzbuzz  →  generate sequence + publish event  │
│  GET /v1/stats     →  read from Postgres (direct)        │
│  GET /v1/health                                          │
│                                                          │
│  Middleware: API key · Request ID · Prometheus metrics   │
└────────────┬─────────────────────────┬───────────────────┘
             │ produce                 │ query
┌────────────▼───────────┐  ┌─────────▼────────────────────┐
│  Kafka                  │  │  Postgres                     │
│  topic: fizzbuzz-events │  │  table: fizz_requests         │
└────────────┬────────────┘  └──────────────────────────────┘
             │ consume                  ▲
┌────────────▼────────────┐             │ upsert (idempotent)
│  Stats Worker            │─────────────┘
│  consumer group:         │
│  fizzbuzz-stats          │
└──────────────────────────┘

┌──────────────────────────────────────────────────────────┐
│  Observability                                           │
│  Prometheus (:9091)  ·  Grafana (:3000)                 │
│  kminion — Kafka consumer group lag                      │
└──────────────────────────────────────────────────────────┘
```

---

## Tech stack

| Layer | Choice |
|---|---|
| Language | Go 1.25 |
| HTTP | Gin + oapi-codegen (spec-first) |
| Messaging | Confluent Kafka (confluent-kafka-go v2) |
| Database | PostgreSQL 16 via pgx/v5 |
| Migrations | golang-migrate |
| Observability | Prometheus + Grafana + kminion |
| Container | Docker / Docker Compose |
| Kubernetes | Helm + HPA + KEDA |
| CI | GitHub Actions |

---

## API

### `GET /v1/fizzbuzz`

| Parameter | Type | Required | Constraint |
|---|---|---|---|
| `int1` | integer | yes | |
| `int2` | integer | yes | |
| `limit` | integer | yes | 1 – 1000 |
| `str1` | string | yes | |
| `str2` | string | yes | |

```bash
curl "http://localhost:8081/v1/fizzbuzz?int1=3&int2=5&limit=15&str1=fizz&str2=buzz" \
  -H "X-API-Key: $API_KEY"
```

```json
{ "result": ["1","2","fizz","4","buzz","fizz","7","8","fizz","buzz","11","fizz","13","14","fizzbuzz"] }
```

### `GET /v1/stats`

Returns the parameter set that has been called the most and its hit count.

```json
{ "request": { "int1": 3, "int2": 5, "limit": 15, "str1": "fizz", "str2": "buzz" }, "hits": 42 }
```

### `GET /v1/health`

```json
{ "status": "ok" }
```

---

## Running locally

```bash
cp .env.example .env

# Start Postgres, Kafka, Prometheus, Grafana
docker compose up -d

# Run the API server
make run

# Run the stats worker
make run-worker
```

Grafana: [http://localhost:3000](http://localhost:3000) (admin / admin)
Kafka UI: [http://localhost:7777](http://localhost:7777)

---

## Configuration

| Variable | Default | Description |
|---|---|---|
| `ADDR` | `:8081` | API listen address |
| `ENV` | `dev` | `dev` skips API key check and auto-generates request IDs |
| `API_KEY` | | Required in `prod` — checked via `X-API-Key` header |
| `METRICS_ADDR` | `:9091` | Prometheus metrics endpoint |
| `DATABASE_URL` | | PostgreSQL connection string |
| `KAFKA_BROKER` | | Kafka broker address |
| `LOG_FORMAT` | `json` | `json` or `text` |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

---

## Development

```bash
make test          # unit tests
make lint          # go vet
make gen           # regenerate OpenAPI stubs (requires oapi-codegen)
make mock          # regenerate mocks (requires mockery)
```

---

## Performance testing

Load tests are written in [k6](https://k6.io/docs/get-started/installation/).

```bash
# against local stack (default)
make load-test

# against a remote environment
make load-test HOST=https://api.example.com API_KEY=secret
```

The smoke scenario (`k6/smoke.js`) ramps to **20 virtual users** over 15 seconds, holds for 30 seconds, then ramps down. It rotates across four different parameter sets so the stats table gets realistic variety.

**Thresholds** (the run fails if any are breached):

| Metric | Threshold |
|---|---|
| Error rate | < 1% |
| p95 overall | < 300 ms |
| p95 `/v1/fizzbuzz` | < 300 ms |
| p95 `/v1/stats` | < 100 ms |

At the end of the run a summary is printed:

```
=== FizzBuzz Load Test Summary ===
  Total requests : 1 247
  Error rate     : 0.00%
  p95 overall    : 18.43ms
  p95 /fizzbuzz  : 17.91ms
  p95 /stats     : 6.22ms
```

---

## Deployment

The Helm chart lives in `helm/fizzbuzz/`. It packages both components with all their Kubernetes resources and exposes every tunable through `values.yaml`.

```
helm/fizzbuzz/
├── Chart.yaml
├── values.yaml            # production defaults
├── values.local.yaml      # local minikube overrides
└── templates/
    ├── api.yaml           # Deployment + Service + HPA
    └── worker.yaml        # Deployment + KEDA ScaledObject
```

### Running on minikube (local)

The chart is designed to run the app inside minikube while reusing the docker-compose stack (Postgres, Kafka, Prometheus, Grafana) running on the host. Pods reach host services via `host.docker.internal`.

**Prerequisites**

```bash
# 1. start dependencies on the host
docker compose up -d

# 2. install KEDA (once)
helm repo add kedacore https://kedacore.github.io/charts && helm repo update
helm install keda kedacore/keda --namespace keda --create-namespace
```

**Deploy**

```bash
minikube start
make helm-local        # builds images + loads into minikube + helm upgrade --install
```

`values.local.yaml` sets `pullPolicy: Never` (uses the locally built image), `ENV=dev` (no API key enforcement), and points all connection strings at `host.docker.internal`.

**Test**

```bash
# forward API + metrics ports
kubectl port-forward service/fizzbuzz-api 8081:80 9091:9091

# hit the API
make curl-health
make curl-fizzbuzz
```

**Iterate** (rebuild after a code change):

```bash
make helm-local
kubectl rollout restart deployment/fizzbuzz-api deployment/fizzbuzz-worker
```

### Production — Kubernetes + HPA + KEDA

The architecture is designed to scale each component independently.

**API server** — stateless, scales on CPU and memory via a standard Kubernetes HPA:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: fizzbuzz-api
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: fizzbuzz-api
  minReplicas: 2
  maxReplicas: 20
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

**Stats worker** — consumer-based, scales on two signals via [KEDA](https://keda.sh):

- **Kafka consumer group lag** — reactive: scales up when messages pile up in the `fizzbuzz-requests` topic
- **HTTP request rate** from Prometheus — proactive: scales up before lag even builds, removing the delay between a traffic spike and worker response

```yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: fizzbuzz-worker
spec:
  scaleTargetRef:
    name: fizzbuzz-worker
  minReplicaCount: 1
  maxReplicaCount: 10
  triggers:
    - type: kafka
      metadata:
        bootstrapServers: kafka:9092
        consumerGroup: fizzbuzz-stats
        topic: fizzbuzz-requests
        lagThreshold: "50"
        offsetResetPolicy: earliest

    - type: prometheus
      metadata:
        serverAddress: http://prometheus:9090
        metricName: fizzbuzz_http_requests_total
        query: sum(rate(fizzbuzz_http_requests_total[1m]))
        threshold: "100"
```

> **Partition count in production** — worker replicas are capped by partition count. A consumer group can never have more active consumers than partitions; extras sit idle. Create the `fizzbuzz-requests` topic with at least as many partitions as `maxReplicaCount` (e.g. 10 partitions for `maxReplicaCount: 10`).

**Why KEDA on top of HPA?**

Standard HPA only reacts to CPU and memory. A Kafka consumer can be completely idle (low CPU) while sitting on thousands of unprocessed messages — HPA would never scale it up. KEDA lets the lag itself drive the replica count. The Prometheus trigger adds a proactive dimension so the worker scales ahead of the queue rather than behind it.

**Install on a real cluster**

```bash
helm upgrade --install fizzbuzz ./helm/fizzbuzz \
  --set secrets.apiKey=$API_KEY \
  --set secrets.databaseURL=$DATABASE_URL \
  --set secrets.kafkaBroker=$KAFKA_BROKER \
  --set image.tag=$IMAGE_TAG
```
