# FlashFlow

> A high-concurrency distributed reservation and order processing platform built with Go.

FlashFlow is a backend-focused engineering project designed to demonstrate production-oriented backend development skills rather than CRUD-heavy application development.

The project simulates high-demand scenarios such as concert ticket sales, limited-product drops, event reservations, and flash sales, where a large number of users compete for limited inventory within a short period of time.

The primary engineering goals are:

- Prevent inventory overselling under concurrent requests
- Design reliable asynchronous workflows
- Handle duplicate messages and partial failures
- Improve system performance through caching and database optimization
- Build observable services with metrics, logs, and distributed tracing
- Containerize and deploy services through a complete CI/CD pipeline
- Demonstrate practical understanding of distributed systems and cloud-native backend development

The project intentionally follows a **70% backend / 30% frontend** development model. The frontend exists mainly to demonstrate and interact with backend capabilities.

---

## 1. Project Overview

FlashFlow models a typical high-demand reservation workflow:

```text
10,000 users
      │
      ▼
 Request Tickets
      │
      ▼
 Inventory Reservation
      │
      ├── Prevent Overselling
      │
      ├── Rate Limiting
      │
      └── Idempotency
      │
      ▼
    Kafka
      │
      ▼
 Order Processing
      │
      ▼
 PostgreSQL
```

Instead of focusing on the number of application features, FlashFlow focuses on several backend engineering problems commonly encountered in production systems:

1. High concurrency
2. Cache consistency
3. Database transactions
4. Distributed communication
5. Event-driven architecture
6. Idempotent processing
7. Reliable event publishing
8. Horizontal scaling
9. Observability
10. Automated testing and deployment

---

# 2. Tech Stack

## Backend

- **Go**
- **Gin / Chi**
- **pgx**
- **sqlc**
- **gRPC**
- **Protocol Buffers**

## Data Layer

- **PostgreSQL**
- **Redis**

## Messaging

- **Apache Kafka**

## Observability

- **OpenTelemetry**
- **Prometheus**
- **Grafana**
- **Jaeger / Grafana Tempo**
- Structured logging with `slog`, `zap`, or `zerolog`

## Infrastructure

- **Docker**
- **Docker Compose**
- **Kubernetes**
- **GitHub Actions**
- **GitHub Container Registry**

## Testing

- Go `testing`
- `testcontainers-go`
- Integration tests
- Race detector
- k6 / Vegeta load testing

## Frontend

- **Next.js**
- **TypeScript**
- **Tailwind CSS**

---

# 3. Architecture

The project starts as a modular backend and gradually evolves into several independently scalable services.

```text
                              ┌────────────────────┐
                              │      Browser       │
                              └──────────┬─────────┘
                                         │
                                      HTTPS
                                         │
                              ┌──────────▼─────────┐
                              │      Next.js       │
                              │      Frontend      │
                              └──────────┬─────────┘
                                         │
                                        REST
                                         │
                         ┌───────────────▼───────────────┐
                         │          API Gateway           │
                         │               Go               │
                         │                                │
                         │ Auth / Events / Rate Limiting │
                         └──────────────┬─────────────────┘
                                        │
                                      gRPC
                                        │
                         ┌──────────────▼──────────────┐
                         │      Inventory Service      │
                         │              Go             │
                         └───────────┬─────────┬───────┘
                                     │         │
                                     │         │
                                     ▼         ▼
                                PostgreSQL   Redis
                                     │
                                     │
                                     ▼
                                  Kafka
                                     │
                          ┌──────────▼───────────┐
                          │    Order Service     │
                          │         Go           │
                          └──────────┬───────────┘
                                     │
                                     ▼
                                PostgreSQL
                                     │
                                     ▼
                                Outbox Table
                                     │
                                     ▼
                                Outbox Worker
                                     │
                                     ▼
                                   Kafka
                                     │
                         ┌───────────▼───────────┐
                         │ Notification Worker  │
                         └───────────────────────┘
```

Observability is applied across all services:

```text
Services
   │
   ├── Metrics ──────► Prometheus ─────► Grafana
   │
   ├── Traces ───────► OpenTelemetry ──► Jaeger / Tempo
   │
   └── Logs ─────────► Structured Logs
```

---

# 4. Core Domain

The project intentionally keeps the business domain relatively small.

Core entities:

```text
User
Event
Inventory
Reservation
Order
Payment
OutboxEvent
```

Example relationship:

```text
User
 │
 └──── Reservation
          │
          ├──── Event
          │
          └──── Order
                  │
                  └──── Payment
```

---

# 5. Core API

The REST API remains intentionally compact.

## Authentication

```http
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
```

## Events

```http
GET /api/v1/events
GET /api/v1/events/:id
```

## Reservations

```http
POST /api/v1/events/:id/reservations
GET  /api/v1/reservations/:id
```

## Orders

```http
GET /api/v1/orders
GET /api/v1/orders/:id
```

## Payment

```http
POST /api/v1/payments/:orderId
POST /api/v1/payments/webhook
```

## Internal APIs

Internal service-to-service communication uses gRPC.

```text
InventoryService
OrderService
```

Example:

```protobuf
service InventoryService {
  rpc ReserveInventory(ReserveInventoryRequest)
      returns (ReserveInventoryResponse);

  rpc ReleaseInventory(ReleaseInventoryRequest)
      returns (ReleaseInventoryResponse);
}
```

---

# 6. High-Concurrency Inventory Reservation

One of the primary engineering challenges of FlashFlow is preventing overselling.

Consider the following scenario:

```text
Available Tickets: 100

Concurrent Requests: 10,000
```

A naïve implementation such as:

```go
if stock > 0 {
    stock--
}
```

cannot safely handle concurrent requests.

FlashFlow explores several approaches.

---

## 6.1 PostgreSQL Transaction Locking

The initial implementation uses PostgreSQL transactions and row-level locks.

```sql
BEGIN;

SELECT available_stock
FROM inventories
WHERE event_id = $1
FOR UPDATE;

UPDATE inventories
SET available_stock = available_stock - 1
WHERE event_id = $1;

COMMIT;
```

Advantages:

- Strong consistency
- Simple architecture
- PostgreSQL remains the source of truth

Limitations:

- Hot rows become contention points
- Throughput decreases during extreme flash-sale workloads

This implementation provides a baseline for later performance comparisons.

---

# 7. Redis Atomic Reservation

For high-demand inventory, Redis can be placed in the critical reservation path.

Instead of:

```text
Request
   │
   ▼
PostgreSQL transaction
```

the flow becomes:

```text
Request
   │
   ▼
Redis atomic reservation
   │
   ▼
Kafka
   │
   ▼
Asynchronous persistence
```

Inventory updates are performed atomically through Redis Lua scripts.

Example:

```lua
local stock = tonumber(redis.call("GET", KEYS[1]))

if not stock then
    return -2
end

if stock <= 0 then
    return -1
end

redis.call("DECR", KEYS[1])

return stock - 1
```

This prevents race conditions between:

```text
GET inventory
```

and:

```text
DECR inventory
```

by executing the entire operation atomically.

---

# 8. Cache Strategy

Redis is also used as a cache for frequently accessed event information.

FlashFlow uses the **Cache-Aside Pattern**.

```text
Client
   │
   ▼
Application
   │
   ▼
Redis
   │
   ├── HIT ───────────► Return response
   │
   └── MISS
         │
         ▼
    PostgreSQL
         │
         ▼
    Populate Redis
         │
         ▼
      Response
```

Example:

```text
GET /api/v1/events/{event_id}
```

The implementation handles:

- Cache TTL
- Cache invalidation
- Cache penetration
- Cache stampede
- Hot keys
- Cache hit ratio

---

# 9. Cache Stampede Protection

A popular event may receive thousands of requests immediately after its cached value expires.

Without protection:

```text
             Redis cache expires
                     │
         ┌───────────┴───────────┐
         ▼           ▼           ▼
      Request     Request      Request
         │           │           │
         └───────────┼───────────┘
                     ▼
                 PostgreSQL
```

Thousands of requests may hit PostgreSQL simultaneously.

FlashFlow uses techniques such as:

```text
singleflight
randomized TTL
background cache refresh
```

Example Go structure:

```go
var group singleflight.Group

value, err, _ := group.Do(cacheKey, func() (interface{}, error) {
    return loadFromDatabase(ctx)
})
```

Only one database query is executed for the same missing key while other requests wait for the result.

---

# 10. Event-Driven Order Processing

Inventory reservation and order creation do not need to happen synchronously.

Instead:

```text
POST /reservations
       │
       ▼
Reserve inventory
       │
       ▼
Publish Event
       │
       ▼
202 Accepted
```

The API may return:

```json
{
  "reservationId": "4b59531c-...",
  "status": "PROCESSING"
}
```

The reservation event is then processed asynchronously.

```text
Reservation Service
       │
       ▼
reservation.created
       │
       ▼
     Kafka
       │
       ▼
 Order Service
       │
       ▼
 PostgreSQL
```

This architecture allows:

- Traffic buffering
- Independent scaling
- Asynchronous processing
- Better fault isolation
- Lower request latency

---

# 11. Kafka

Kafka is used only where asynchronous communication provides a clear architectural benefit.

Example topics:

```text
reservation.created
reservation.expired
order.created
order.paid
order.cancelled
notification.requested
```

Example event:

```json
{
  "eventId": "event-123",
  "reservationId": "reservation-456",
  "userId": "user-789",
  "quantity": 2,
  "timestamp": "2026-08-20T17:00:00Z"
}
```

Kafka-related concepts demonstrated in the project include:

- Producers
- Consumers
- Consumer groups
- Partitioning
- Offset management
- Retry
- Dead-letter topics
- Consumer lag
- At-least-once delivery

---

# 12. Idempotency

Kafka typically provides at-least-once processing semantics.

The same event may therefore be delivered multiple times.

Without idempotency:

```text
reservation.created
        │
        ├────► Order #1
        │
        └────► Order #2
```

FlashFlow protects against duplicate processing.

Possible database constraint:

```sql
CREATE UNIQUE INDEX idx_orders_reservation_id
ON orders(reservation_id);
```

Consumers also store processed event IDs where appropriate.

```text
event_id
processed_at
consumer
```

The expected behavior becomes:

```text
Message received
      │
      ▼
Already processed?
   │         │
 YES        NO
   │         │
Ignore    Process
```

---

# 13. API Idempotency

Idempotency is also supported for selected HTTP operations.

Example:

```http
POST /api/v1/events/123/reservations
Idempotency-Key: 54e53dd2-...
```

If the client retries the request because of a timeout:

```text
Request
   │
   ▼
Network timeout
   │
   ▼
Retry
```

the server does not accidentally create multiple reservations.

---

# 14. Transactional Outbox Pattern

A classic distributed system problem occurs when a database operation succeeds but Kafka publishing fails.

Consider:

```text
BEGIN

INSERT order

COMMIT

publish Kafka event
```

Possible failure:

```text
Database COMMIT
       │
       ▼
    SUCCESS

Kafka publish
       │
       ▼
    FAILURE
```

Now the order exists, but downstream systems never receive the corresponding event.

FlashFlow solves this using the **Transactional Outbox Pattern**.

```text
BEGIN

INSERT INTO orders ...

INSERT INTO outbox_events ...

COMMIT
```

Both writes occur inside the same PostgreSQL transaction.

The outbox worker then publishes events asynchronously.

```text
PostgreSQL
    │
    ▼
outbox_events
    │
    ▼
Outbox Worker
    │
    ▼
Kafka
```

After successful publication:

```text
processed_at = NOW()
```

This ensures reliable event delivery without requiring a distributed transaction between PostgreSQL and Kafka.

---

# 15. Retry and Dead-Letter Queue

Temporary failures are retried.

```text
Kafka Event
    │
    ▼
Consumer
    │
    ▼
Processing failed
    │
    ▼
Retry
    │
    ├── success ──► complete
    │
    └── failed
          │
          ▼
        DLQ
```

A typical retry policy may use exponential backoff:

```text
1s
2s
4s
8s
...
```

Messages exceeding the retry limit are published to a dead-letter topic for further inspection.

Example:

```text
order.processing.dlq
```

---

# 16. Reservation Expiration

Reservations should not permanently lock inventory.

Example:

```text
User reserves ticket
       │
       ▼
10-minute reservation window
       │
       ├──── Payment success ───► Order confirmed
       │
       └──── Timeout ───────────► Inventory released
```

Possible implementation approaches:

- Redis TTL
- Delayed message
- Scheduled worker
- Expiration queue

Expired reservations generate:

```text
reservation.expired
```

which triggers inventory release.

---

# 17. Database Design

PostgreSQL is the primary persistent datastore.

Important topics demonstrated include:

- Schema design
- Primary keys
- Foreign keys
- Unique constraints
- Transactions
- Index design
- Query optimization
- Isolation levels
- Row locking
- Optimistic locking
- Pessimistic locking
- Connection pooling
- Pagination

---

# 18. SQL over Heavy ORM Abstraction

The backend uses:

```text
pgx
+
sqlc
```

instead of relying completely on a traditional ORM.

This allows important database behavior to remain visible and explicit.

Example:

```sql
-- name: GetUserOrders :many

SELECT
    id,
    user_id,
    status,
    total_amount,
    created_at
FROM orders
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;
```

`sqlc` generates type-safe Go code from SQL queries.

---

# 19. Database Index Optimization

Example query:

```sql
SELECT *
FROM orders
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 20;
```

Corresponding index:

```sql
CREATE INDEX idx_orders_user_created
ON orders(user_id, created_at DESC);
```

Performance is measured using:

```sql
EXPLAIN ANALYZE
```

Benchmark results should be documented rather than guessed.

Example format:

```text
Before optimization

Execution Time: TBD ms

After optimization

Execution Time: TBD ms
```

---

# 20. Pagination

Large datasets use cursor-based pagination where appropriate.

Instead of:

```sql
OFFSET 100000
LIMIT 20
```

FlashFlow can use:

```sql
WHERE created_at < $1
ORDER BY created_at DESC
LIMIT 20
```

This avoids increasingly expensive large-offset scans.

---

# 21. Go Concurrency

The project uses Go concurrency where it provides practical value.

Examples include:

- Kafka consumers
- Worker pools
- Notification processing
- Outbox processing
- Concurrent I/O
- Graceful shutdown

Example worker pool:

```go
jobs := make(chan Job, 100)

for i := 0; i < workerCount; i++ {
    go worker(ctx, jobs)
}
```

The implementation demonstrates:

```text
goroutines
channels
sync.WaitGroup
context.Context
errgroup
worker pools
```

---

# 22. Context Propagation

`context.Context` is propagated across application layers.

```text
HTTP request
     │
     ▼
Handler
     │
     ▼
Service
     │
     ▼
Repository
     │
     ▼
PostgreSQL / Redis / gRPC
```

This provides:

- Cancellation
- Request deadlines
- Trace propagation
- Graceful resource cleanup

---

# 23. Graceful Shutdown

Services handle `SIGTERM` and `SIGINT`.

```text
SIGTERM
   │
   ▼
Stop accepting requests
   │
   ▼
Wait for inflight requests
   │
   ▼
Stop Kafka consumers
   │
   ▼
Stop workers
   │
   ▼
Close DB / Redis
   │
   ▼
Exit
```

This is particularly important when running inside Kubernetes.

---

# 24. Rate Limiting

High-demand APIs require request throttling.

FlashFlow implements rate limiting using Redis.

Example policy:

```text
100 requests / minute / user
```

Possible algorithms include:

- Token bucket
- Sliding window
- Fixed window

Rate limiting may be applied by:

```text
IP
User ID
API key
Endpoint
```

---

# 25. Authentication and Authorization

Authentication uses:

```text
JWT Access Token
+
Refresh Token
```

Security features include:

- Password hashing
- JWT validation
- Refresh token rotation
- Role-based access control
- Input validation
- Rate limiting
- Secure secret management
- CORS configuration
- Parameterized SQL

Example roles:

```text
USER
ADMIN
```

Admin-only endpoints can manage:

```text
events
inventory
system status
```

---

# 26. Service Communication

External APIs use REST.

```text
Browser
   │
 REST
   ▼
Gateway
```

Internal synchronous communication uses gRPC.

```text
Gateway
   │
 gRPC
   ▼
Inventory Service
```

Asynchronous domain events use Kafka.

```text
Inventory Service
       │
     Kafka
       │
       ▼
 Order Service
```

Each communication mechanism therefore serves a distinct purpose.

---

# 27. Modular Monolith First

The project does **not** begin as a large collection of microservices.

The initial architecture is:

```text
Application
├── auth
├── events
├── inventory
├── reservations
└── orders
```

This keeps the initial system simple.

Services are split only when there is a concrete reason for independent scaling or failure isolation.

For example:

```text
Inventory
```

and:

```text
Order Processing
```

can eventually become separate services because they have different workload characteristics.

This demonstrates the trade-off between:

```text
development simplicity
```

and:

```text
independent deployment / scaling
```

rather than using microservices solely for architectural complexity.

---

# 28. Observability

FlashFlow implements the three major pillars of observability:

```text
Metrics
Logs
Traces
```

---

# 29. Metrics

Prometheus collects application and infrastructure metrics.

Example application metrics:

```text
http_requests_total

http_request_duration_seconds

reservation_requests_total

reservation_success_total

reservation_failure_total

orders_created_total

cache_hits_total

cache_misses_total

kafka_messages_processed_total

kafka_consumer_errors_total
```

Infrastructure metrics may include:

```text
CPU usage
memory usage
goroutines
database connections
Redis connections
Kafka consumer lag
```

---

# 30. Grafana Dashboard

Grafana provides dashboards for:

```text
Requests per second

P50 latency

P95 latency

P99 latency

Error rate

Reservation success rate

Redis cache hit ratio

Database latency

Kafka consumer lag

CPU usage

Memory usage

Pod count
```

Example:

```text
┌─────────────────────────────────────────┐
│ FlashFlow Production Dashboard          │
├────────────────┬────────────────────────┤
│ Requests/sec   │  P95 Latency           │
├────────────────┼────────────────────────┤
│ Error Rate     │  Reservation Success   │
├────────────────┼────────────────────────┤
│ Cache Hit Rate │  Kafka Consumer Lag    │
├────────────────┼────────────────────────┤
│ CPU            │  Memory                │
└────────────────┴────────────────────────┘
```

---

# 31. Structured Logging

Services emit structured logs.

Example:

```json
{
  "timestamp": "2026-08-20T17:30:00Z",
  "level": "INFO",
  "service": "order-service",
  "trace_id": "47d...",
  "reservation_id": "84d...",
  "order_id": "20f...",
  "message": "order created",
  "latency_ms": 23
}
```

Important identifiers such as:

```text
trace_id
request_id
user_id
reservation_id
order_id
```

make debugging distributed requests significantly easier.

---

# 32. Distributed Tracing

OpenTelemetry propagates traces across services.

Example:

```text
POST /reservations
        │
        ▼
API Gateway
        │
        ▼
Inventory Service
        │
        ├── Redis
        │
        └── Kafka Produce
               │
               ▼
         Order Consumer
               │
               ▼
          PostgreSQL
```

A trace may look like:

```text
POST /reservations               15ms
├── auth.validate                 1ms
├── inventory.reserve             3ms
│   └── redis.eval                2ms
└── kafka.publish                 5ms

order.consume                    21ms
├── postgres.insert_order         8ms
└── postgres.insert_outbox        4ms
```

Tracing is visualized using:

```text
Jaeger
```

or:

```text
Grafana Tempo
```

---

# 33. Health Checks

Services expose health endpoints.

```http
GET /health
GET /ready
```

Example:

```json
{
  "status": "ok"
}
```

Kubernetes uses these endpoints for:

```text
livenessProbe
readinessProbe
```

---

# 34. Docker

Every backend service is containerized.

Example build:

```bash
docker build -t flashflow-api .
```

A multi-stage Dockerfile keeps production images small.

```dockerfile
FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /flashflow ./cmd/api


FROM gcr.io/distroless/static-debian12

COPY --from=builder /flashflow /flashflow

ENTRYPOINT ["/flashflow"]
```

---

# 35. Local Development

Local infrastructure runs using Docker Compose.

Services include:

```text
PostgreSQL
Redis
Kafka
Prometheus
Grafana
Jaeger
```

Example:

```bash
docker compose up -d
```

Run the application:

```bash
make run
```

Or:

```bash
go run ./cmd/api
```

---

# 36. Kubernetes

Production-like deployment uses Kubernetes.

Resources include:

```text
Deployment
Service
Ingress
ConfigMap
Secret
HorizontalPodAutoscaler
```

Example architecture:

```text
                    Ingress
                       │
                       ▼
                 API Service
                       │
              ┌────────┴────────┐
              ▼                 ▼
           API Pod           API Pod
              │                 │
              └────────┬────────┘
                       ▼
                  Redis / Kafka
```

---

# 37. Horizontal Pod Autoscaling

API and worker services can scale independently.

Example:

```text
Minimum replicas: 2

Maximum replicas: 10

Target CPU: 70%
```

During load testing:

```text
Traffic increases
       │
       ▼
CPU increases
       │
       ▼
HPA detects load
       │
       ▼
Additional Pods
       │
       ▼
Higher capacity
```

Actual scaling and throughput numbers should be recorded after testing.

---

# 38. CI/CD

GitHub Actions provides automated CI/CD.

Pull request pipeline:

```text
Pull Request
     │
     ▼
gofmt
     │
     ▼
golangci-lint
     │
     ▼
go vet
     │
     ▼
unit tests
     │
     ▼
race detector
     │
     ▼
integration tests
```

Main branch pipeline:

```text
Merge to main
      │
      ▼
Build application
      │
      ▼
Build Docker image
      │
      ▼
Push image to GHCR
      │
      ▼
Deploy to Kubernetes
      │
      ▼
Health check
```

---

# 39. GitHub Actions Example

```yaml
name: CI

on:
  pull_request:
  push:
    branches:
      - main

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"

      - run: go mod download

      - run: go vet ./...

      - run: go test -race ./...
```

The production workflow can additionally:

```text
build Docker images
push images
deploy Kubernetes manifests
```

---

# 40. Automated Testing

The project contains several testing layers.

```text
Unit Tests
    │
Integration Tests
    │
API Tests
    │
Concurrency Tests
    │
Load Tests
```

---

# 41. Unit Testing

Unit tests cover:

```text
business logic
validation
reservation rules
idempotency
retry policies
```

Run:

```bash
go test ./...
```

---

# 42. Race Detection

Go's race detector is used to detect unsafe concurrent access.

```bash
go test -race ./...
```

This is particularly useful for:

```text
worker pools
shared state
concurrent consumers
```

---

# 43. Integration Testing

Integration tests use real infrastructure where appropriate.

For example:

```text
Go Test
   │
   ▼
Testcontainers
   │
   ├── PostgreSQL
   ├── Redis
   └── Kafka
```

This provides higher confidence than mocking every infrastructure dependency.

---

# 44. Load Testing

Performance tests use:

```text
k6
```

or:

```text
Vegeta
```

Example k6 test:

```javascript
import http from "k6/http";
import { check } from "k6";

export const options = {
  vus: 1000,
  duration: "30s",
};

export default function () {
  const response = http.post(
    "http://localhost:8080/api/v1/events/123/reservations",
  );

  check(response, {
    "status accepted": (r) => r.status === 200 || r.status === 202,
  });
}
```

---

# 45. Performance Metrics

Load tests record:

```text
Requests / second

P50 latency

P95 latency

P99 latency

Error rate

CPU usage

Memory usage

DB connection usage

Redis cache hit ratio

Kafka consumer lag
```

---

# 46. Benchmark Results

All performance numbers in this section must come from real measurements.

Do not replace these values until the tests have actually been executed.

## Baseline

Environment:

```text
CPU: TBD
Memory: TBD
OS: TBD
Go version: TBD
PostgreSQL version: TBD
Redis version: TBD
Kafka version: TBD
```

Results:

```text
Concurrent Clients: TBD

Throughput:
TBD req/s

Latency:
P50: TBD ms
P95: TBD ms
P99: TBD ms

Error Rate:
TBD %
```

---

## PostgreSQL Reservation

```text
Throughput: TBD

P95: TBD

P99: TBD
```

## Redis Reservation

```text
Throughput: TBD

P95: TBD

P99: TBD
```

## Cache Disabled

```text
Throughput: TBD

P95: TBD

Database Queries/sec: TBD
```

## Cache Enabled

```text
Throughput: TBD

P95: TBD

Cache Hit Rate: TBD
```

---

# 47. Failure Testing

FlashFlow intentionally tests system behavior under failure.

Scenarios include:

```text
Order service crash

Consumer crash

Duplicate Kafka delivery

Redis unavailable

PostgreSQL unavailable

Kafka unavailable

Slow PostgreSQL query

Network timeout

Pod termination

Outbox worker restart
```

---

# 48. Example Failure Scenario

Consider:

```text
Kafka event received
       │
       ▼
Order inserted
       │
       ▼
Consumer crashes
       │
       ▼
Offset not committed
       │
       ▼
Kafka redelivers event
```

Without protection:

```text
Duplicate Order
```

With idempotency:

```text
Kafka redelivery
       │
       ▼
INSERT order
       │
       ▼
UNIQUE reservation_id
       │
       ▼
Duplicate detected
       │
       ▼
No duplicate order
```

This behavior can be verified through integration tests.

---

# 49. Resilience Principles

The system demonstrates several reliability techniques:

```text
Timeouts

Retries

Exponential backoff

Idempotency

Dead-letter queues

Graceful shutdown

Health checks

Transactional Outbox

Circuit-breaking where justified
```

Not every pattern is introduced by default.

Patterns are added only where they solve a concrete failure mode.

---

# 50. Frontend

The frontend intentionally remains small.

The main pages are:

```text
/login

/events

/events/:id

/orders

/admin
```

The frontend demonstrates:

- Authentication
- Event browsing
- Reservation
- Order tracking
- Real-time status updates
- System monitoring

It is not intended to become a large design-focused web application.

---

# 51. Admin Dashboard

The admin page displays operational information such as:

```text
Requests / second

P95 latency

Active reservations

Successful orders

Available inventory

Kafka consumer lag

Cache hit ratio

Error rate
```

Example:

```text
┌──────────────────────────────────────┐
│ FlashFlow Admin                      │
├─────────────────┬────────────────────┤
│ RPS             │ 8,420              │
│ P95             │ 61ms               │
│ Orders/min      │ 1,200              │
│ Cache hit rate  │ 93%                │
│ Kafka lag       │ 18                 │
│ Inventory       │ 1,382              │
└─────────────────┴────────────────────┘
```

Numbers above are illustrative until replaced with actual system measurements.

---

# 52. Repository Structure

```text
flashflow/
│
├── cmd/
│   ├── api/
│   │   └── main.go
│   ├── order-worker/
│   │   └── main.go
│   └── outbox-worker/
│       └── main.go
│
├── internal/
│   ├── auth/
│   ├── event/
│   ├── inventory/
│   ├── reservation/
│   ├── order/
│   ├── payment/
│   ├── messaging/
│   ├── observability/
│   └── platform/
│
├── pkg/
│
├── migrations/
│
├── queries/
│
├── proto/
│
├── deployments/
│   ├── docker/
│   └── kubernetes/
│
├── observability/
│   ├── prometheus/
│   ├── grafana/
│   └── otel/
│
├── loadtest/
│   ├── reservation.js
│   ├── event-read.js
│   └── spike-test.js
│
├── scripts/
│
├── web/
│
├── docs/
│   ├── architecture.md
│   ├── database.md
│   ├── reliability.md
│   ├── observability.md
│   └── benchmarks.md
│
├── .github/
│   └── workflows/
│       ├── ci.yml
│       └── deploy.yml
│
├── docker-compose.yml
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

---

# 53. Development Roadmap

The project is developed incrementally instead of introducing all infrastructure on day one.

## Phase 1 — Backend Foundation

Goals:

- [ ] Initialize Go project
- [ ] Define project architecture
- [ ] Set up PostgreSQL
- [ ] Add database migrations
- [ ] Implement authentication
- [ ] Implement event APIs
- [ ] Implement inventory
- [ ] Implement reservation
- [ ] Implement order APIs
- [ ] Add validation
- [ ] Add structured logging
- [ ] Add unit tests
- [ ] Add integration tests
- [ ] Dockerize application

Architecture:

```text
Go API
  │
  ▼
PostgreSQL
```

---

## Phase 2 — Performance

Goals:

- [ ] Add Redis
- [ ] Implement Cache-Aside
- [ ] Add cache TTL
- [ ] Protect against cache stampede
- [ ] Implement rate limiting
- [ ] Benchmark database queries
- [ ] Add PostgreSQL indexes
- [ ] Add connection pool tuning
- [ ] Build k6 load tests
- [ ] Measure P50 / P95 / P99

Architecture:

```text
      Go API
      /    \
     ▼      ▼
 Redis   PostgreSQL
```

---

## Phase 3 — High-Concurrency Reservation

Goals:

- [ ] Implement PostgreSQL locking version
- [ ] Implement Redis atomic reservation
- [ ] Implement Redis Lua script
- [ ] Test overselling
- [ ] Compare approaches
- [ ] Add idempotency keys
- [ ] Add reservation expiration

Benchmark:

```text
PostgreSQL Locking
vs
Redis Atomic Reservation
```

---

## Phase 4 — Distributed Architecture

Goals:

- [ ] Introduce Kafka
- [ ] Publish `reservation.created`
- [ ] Create Order Worker
- [ ] Implement consumer groups
- [ ] Implement idempotent consumers
- [ ] Implement retry
- [ ] Implement DLQ
- [ ] Implement Transactional Outbox
- [ ] Add gRPC
- [ ] Separate inventory service
- [ ] Separate order processing

Architecture:

```text
Gateway
   │
 gRPC
   ▼
Inventory
   │
   ▼
 Kafka
   │
   ▼
Orders
```

---

## Phase 5 — Observability

Goals:

- [ ] Add Prometheus
- [ ] Add HTTP metrics
- [ ] Add business metrics
- [ ] Add Kafka metrics
- [ ] Add Redis metrics
- [ ] Add PostgreSQL metrics
- [ ] Add Grafana
- [ ] Build dashboards
- [ ] Add OpenTelemetry
- [ ] Add distributed tracing
- [ ] Add trace IDs to logs

---

## Phase 6 — Cloud Native

Goals:

- [ ] Write Kubernetes manifests
- [ ] Configure Deployments
- [ ] Configure Services
- [ ] Configure Ingress
- [ ] Configure ConfigMaps
- [ ] Configure Secrets
- [ ] Add readiness probes
- [ ] Add liveness probes
- [ ] Add resource limits
- [ ] Configure HPA

---

## Phase 7 — CI/CD

Goals:

- [ ] Configure GitHub Actions
- [ ] Run formatting checks
- [ ] Run lint
- [ ] Run tests
- [ ] Run race detector
- [ ] Run integration tests
- [ ] Build Docker images
- [ ] Push to GHCR
- [ ] Deploy automatically
- [ ] Verify deployment health

---

## Phase 8 — Reliability Testing

Goals:

- [ ] Kill consumer during processing
- [ ] Test duplicate Kafka events
- [ ] Restart outbox worker
- [ ] Kill API pods
- [ ] Simulate Redis failure
- [ ] Simulate Kafka failure
- [ ] Simulate PostgreSQL latency
- [ ] Verify graceful shutdown
- [ ] Verify Kubernetes recovery

---

## Phase 9 — Frontend and Demo

Goals:

- [ ] Build login page
- [ ] Build event page
- [ ] Build reservation UI
- [ ] Build orders page
- [ ] Build admin dashboard
- [ ] Add real-time status
- [ ] Prepare demonstration workflow

---

# 54. Architecture Decisions

Important architectural decisions should be documented explicitly.

Examples:

```text
Why Go?

Why PostgreSQL instead of MongoDB?

Why Redis?

Why Kafka?

Why REST externally?

Why gRPC internally?

Why begin as a modular monolith?

Why split Inventory before Auth?

Why use sqlc instead of a heavy ORM?

Why Transactional Outbox?

Why at-least-once processing?

Why Kubernetes?
```

Each decision should explain:

```text
Problem
↓
Options
↓
Trade-offs
↓
Decision
↓
Consequences
```

---

# 55. Key Engineering Challenges

FlashFlow focuses specifically on the following engineering challenges.

### Preventing Overselling

Multiple concurrent clients must never successfully reserve more inventory than exists.

### Maintaining Idempotency

Retries and duplicate events must not create duplicate reservations or orders.

### Reliable Event Publishing

Database changes and domain events must not silently diverge.

### Handling Traffic Spikes

The system should remain responsive during sudden high-demand events.

### Database Performance

Indexes and queries should be measured and optimized based on evidence.

### Cache Consistency

Redis should improve performance without introducing uncontrolled stale data.

### Service Reliability

Individual service crashes should not permanently corrupt system state.

### Observability

Failures should be discoverable through metrics, logs, and traces.

### Horizontal Scaling

Stateless services should be able to scale across multiple replicas.

---

# 56. What This Project Is Not

FlashFlow intentionally avoids becoming:

```text
A giant CRUD application

A 30-page frontend

A microservice demo with 15 unnecessary services

A collection of technologies with no architectural reason

A benchmark containing invented numbers
```

The goal is not to maximize the number of technologies.

The goal is to demonstrate an understanding of:

```text
why
```

and:

```text
when
```

each technology should be used.

---

# 57. Engineering Principles

The project follows several principles.

## Start Simple

Do not introduce distributed infrastructure before it solves a real problem.

## Measure Before Optimizing

Performance improvements must be supported by benchmarks.

## Prefer Explicit Behavior

Critical database and consistency behavior should be visible in the code.

## Design for Failure

Network calls and asynchronous processing are assumed to fail eventually.

## Keep Services Independently Operable

Services should expose:

```text
health checks
metrics
logs
traces
```

## Avoid Resume-Driven Architecture

Technologies are introduced because they solve problems, not because they look impressive in a technology list.

---

# 58. Expected Learning Outcomes

After completing FlashFlow, the project should demonstrate practical understanding of:

## Go

```text
goroutines
channels
context
interfaces
error handling
worker pools
graceful shutdown
race detection
```

## HTTP Backend Development

```text
REST
middleware
authentication
validation
rate limiting
API versioning
idempotency
```

## PostgreSQL

```text
transactions
locking
indexes
constraints
query planning
connection pooling
pagination
```

## Redis

```text
cache-aside
TTL
atomic operations
Lua scripts
distributed rate limiting
cache stampede protection
```

## Kafka

```text
producer
consumer
consumer groups
offsets
partitions
retry
DLQ
consumer lag
at-least-once delivery
```

## Distributed Systems

```text
idempotency
eventual consistency
partial failure
retry
timeouts
distributed communication
transactional outbox
```

## Cloud Native

```text
Docker
Kubernetes
horizontal scaling
health checks
configuration
secrets
```

## DevOps

```text
CI/CD
automated testing
container registry
deployment automation
```

## Observability

```text
metrics
logging
distributed tracing
Prometheus
Grafana
OpenTelemetry
```

---

# 59. Interview Topics

This project is intentionally designed to support discussion around common backend interview questions.

Examples:

### Why did you use Redis?

To reduce database load for read-heavy workloads and support atomic high-concurrency inventory operations.

### How do you prevent overselling?

Compare PostgreSQL row locking and Redis atomic reservation strategies.

### What happens if Kafka delivers the same event twice?

Consumers are idempotent and database constraints prevent duplicate state changes.

### What happens if PostgreSQL commits but Kafka fails?

The Transactional Outbox Pattern stores domain events inside the same database transaction.

### What happens if a worker crashes?

Kafka can redeliver messages while idempotency protects against duplicate processing.

### Why not start with microservices?

A modular monolith minimizes early complexity. Services are extracted only when independent scaling or failure isolation becomes useful.

### Why Kafka instead of synchronous HTTP?

Kafka decouples workloads, buffers traffic spikes, and supports asynchronous event-driven workflows.

### Why gRPC?

It provides strongly typed contracts and efficient communication for synchronous internal service calls.

### How do you know Redis improves performance?

k6 benchmarks compare throughput and P95/P99 latency before and after caching.

### How do you debug a slow request?

Use trace IDs, distributed traces, service metrics, structured logs, and database query analysis.

---

# 60. Resume Description

After the project has been completed and benchmarked, it can be summarized on a resume as:

**FlashFlow — Distributed High-Concurrency Reservation Platform**

`Go · PostgreSQL · Redis · Kafka · gRPC · Docker · Kubernetes · OpenTelemetry · Prometheus · Grafana`

- Designed and implemented a Go-based distributed reservation platform supporting atomic inventory allocation under concurrent traffic using PostgreSQL transactions and Redis Lua scripts to prevent overselling.
- Built asynchronous order processing with Kafka, including idempotent consumers, retry and dead-letter mechanisms, and the Transactional Outbox Pattern for reliable event delivery.
- Optimized PostgreSQL queries and Redis caching strategies using k6 load testing, measuring throughput, P95/P99 latency, database load, and cache hit ratios.
- Instrumented backend services with OpenTelemetry, Prometheus, Grafana, and structured logging for metrics, distributed tracing, and production debugging.
- Containerized services with Docker and deployed them to Kubernetes with health checks, horizontal autoscaling, and GitHub Actions CI/CD.

Once real benchmarks are available, these descriptions should be updated with measured numbers.

For example:

```text
Improved P95 latency from X ms to Y ms.

Increased throughput from X req/s to Y req/s.

Reduced PostgreSQL query load by X%.

Maintained zero overselling during X concurrent reservation attempts.

Recovered successfully from duplicate delivery and consumer crashes.
```

Do **not** include numbers until they have actually been measured.

---

# 61. Running the Project

Clone:

```bash
git clone https://github.com/<your-username>/flashflow.git

cd flashflow
```

Start infrastructure:

```bash
docker compose up -d
```

Run migrations:

```bash
make migrate-up
```

Start backend:

```bash
make run
```

Run tests:

```bash
make test
```

Run race detector:

```bash
make test-race
```

Run integration tests:

```bash
make test-integration
```

Run load tests:

```bash
make load-test
```

Stop environment:

```bash
docker compose down
```

---

# 62. Makefile Commands

Planned commands:

```text
make dev

make run

make build

make test

make test-race

make test-integration

make lint

make migrate-up

make migrate-down

make docker-build

make docker-up

make docker-down

make load-test

make proto

make deploy
```

---

# 63. Environment Variables

Example:

```bash
APP_ENV=development

HTTP_PORT=8080

DATABASE_URL=postgres://flashflow:flashflow@localhost:5432/flashflow

REDIS_ADDR=localhost:6379

KAFKA_BROKERS=localhost:9092

JWT_SECRET=development-only-secret

OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

Production secrets must not be committed to Git.

---

# 64. Documentation

Additional engineering documentation lives under:

```text
docs/
```

Planned documents:

```text
docs/architecture.md

docs/database.md

docs/cache.md

docs/messaging.md

docs/reliability.md

docs/observability.md

docs/benchmarks.md

docs/failure-testing.md
```

The README provides the overview, while these files explain individual engineering decisions in greater depth.

---

# 65. Future Improvements

Possible future extensions include:

```text
Distributed tracing across Kafka headers

Schema Registry

Kafka partition strategy experiments

Redis Cluster

PostgreSQL read replicas

CDC

Debezium

Service mesh

Argo CD

Helm

Chaos engineering

Multi-region deployment
```

These are intentionally considered **future work** rather than requirements for the first version.

They should only be added when they provide additional engineering value.

---

# 66. Project Philosophy

FlashFlow is not designed to prove that the developer knows how to import many frameworks.

It is designed to prove that the developer can take a backend service through its complete lifecycle:

```text
Requirements
      │
      ▼
API Design
      │
      ▼
Database Design
      │
      ▼
Implementation
      │
      ▼
Testing
      │
      ▼
Performance Optimization
      │
      ▼
Distributed Architecture
      │
      ▼
Reliability Engineering
      │
      ▼
Observability
      │
      ▼
Containerization
      │
      ▼
Deployment
      │
      ▼
Operations
```

The project's central question is therefore not:

> How many technologies are used?

It is:

> Can every engineering decision be explained, measured, tested, and justified?

---
