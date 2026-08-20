可以。下面这份是完整中文版，我把英文版里的技术设计、开发路线、架构权衡、测试、可观测性、CI/CD、Kubernetes、简历描述等都保留下来了，同时把表达改成更适合作为 GitHub 中文 README 的形式。

# FlashFlow

> 基于 Go 构建的高并发分布式预约与订单处理平台。

FlashFlow 是一个以后端工程能力为核心的项目，目标不是开发大量 CRUD 页面，而是完整展示一个后端系统从设计、实现、测试、性能优化，到分布式改造、部署和运维的全过程。

项目模拟演唱会抢票、限量商品抢购、活动预约、秒杀等典型高并发场景：

大量用户会在极短时间内同时请求有限库存，因此系统需要解决库存超卖、并发竞争、缓存一致性、消息重复、部分失败、流量突增、服务扩缩容和故障排查等问题。

项目整体按照约：

```text
70% Backend
30% Frontend
```

进行设计。

前端主要负责展示业务流程和后端能力，不追求大量页面或复杂 UI。

---

# 1. 项目目标

FlashFlow 重点解决以下后端工程问题：

- 高并发库存竞争
- 库存超卖
- 数据库事务与锁
- Redis 缓存设计
- 缓存击穿与缓存雪崩
- 异步订单处理
- Kafka 消息系统
- 消息重复与幂等
- 分布式事务一致性
- Transactional Outbox
- 服务拆分与微服务
- REST 与 gRPC
- 水平扩容
- Docker 容器化
- Kubernetes 部署
- CI/CD
- Metrics / Logs / Traces
- 自动化测试
- 压力测试
- 故障恢复
- 自动化运维

项目核心并不是：

> “用了多少技术。”

而是：

> “为什么使用这些技术，它们解决了什么问题，以及这些方案是否经过实际测试和测量。”

---

# 2. 业务场景

假设某个活动只有：

```text
100 张票
```

但开放预约的一瞬间产生：

```text
10,000 个并发请求
```

系统需要保证：

```text
成功预约数量 <= 实际库存
```

同时还需要：

```text
快速响应用户

避免数据库被瞬时流量打爆

异步完成后续订单处理

服务异常后能够恢复

消息重复不会创建重复订单

系统状态可以被监控和追踪
```

基础业务流程：

```text
10,000 Users
      │
      ▼
预约请求
      │
      ▼
库存服务
      │
      ├── 并发控制
      ├── 限流
      ├── 幂等
      └── Redis
      │
      ▼
    Kafka
      │
      ▼
订单消费者
      │
      ▼
 PostgreSQL
```

---

# 3. 技术栈

## 后端

```text
Go
Gin / Chi
pgx
sqlc
gRPC
Protocol Buffers
```

## 数据层

```text
PostgreSQL
Redis
```

## 消息系统

```text
Apache Kafka
```

## 可观测性

```text
OpenTelemetry
Prometheus
Grafana
Jaeger / Grafana Tempo
Structured Logging
```

日志库可以选择：

```text
slog
zap
zerolog
```

## 基础设施

```text
Docker
Docker Compose
Kubernetes
GitHub Actions
GitHub Container Registry
```

## 测试

```text
Go testing
testcontainers-go
Integration Test
Race Detector
k6 / Vegeta
```

## 前端

```text
Next.js
TypeScript
Tailwind CSS
```

---

# 4. 系统架构

目标架构：

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
                                     ▼         ▼
                                PostgreSQL   Redis
                                     │
                                     ▼
                                   Kafka
                                     │
                         ┌───────────▼────────────┐
                         │      Order Service      │
                         │            Go           │
                         └───────────┬────────────┘
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

整个系统统一接入：

```text
Services
   │
   ├── Metrics ─────► Prometheus ─────► Grafana
   │
   ├── Traces ──────► OpenTelemetry ──► Jaeger / Tempo
   │
   └── Logs ────────► Structured Logs
```

---

# 5. 核心领域模型

项目刻意控制业务复杂度。

核心实体包括：

```text
User
Event
Inventory
Reservation
Order
Payment
OutboxEvent
```

关系大致为：

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

项目重点不在于复杂业务表关系，而在于围绕这些实体构建完整的后端工程能力。

---

# 6. API 设计

API 数量保持克制。

## Authentication

```http
POST /api/v1/auth/register

POST /api/v1/auth/login

POST /api/v1/auth/refresh
```

---

## Events

```http
GET /api/v1/events

GET /api/v1/events/:id
```

---

## Reservations

```http
POST /api/v1/events/:id/reservations

GET /api/v1/reservations/:id
```

---

## Orders

```http
GET /api/v1/orders

GET /api/v1/orders/:id
```

---

## Payment

```http
POST /api/v1/payments/:orderId

POST /api/v1/payments/webhook
```

---

# 7. 内部服务通信

浏览器和外部客户端通过 REST 调用：

```text
Browser
   │
 REST
   ▼
API Gateway
```

内部同步调用通过 gRPC：

```text
API Gateway
     │
    gRPC
     │
     ▼
Inventory Service
```

异步业务流程通过 Kafka：

```text
Inventory Service
       │
       ▼
     Kafka
       │
       ▼
Order Service
```

因此三种通信方式各自承担明确职责：

```text
REST
→ 外部 API

gRPC
→ 内部同步服务通信

Kafka
→ 异步业务事件
```

---

# 8. gRPC

内部服务使用 Protocol Buffers 定义接口。

例如：

```protobuf
service InventoryService {
  rpc ReserveInventory(ReserveInventoryRequest)
      returns (ReserveInventoryResponse);

  rpc ReleaseInventory(ReleaseInventoryRequest)
      returns (ReleaseInventoryResponse);
}
```

通过 gRPC 可以展示：

```text
Protocol Buffers

IDL

Strongly Typed Contract

Service-to-Service Communication

Timeout

Context Propagation

Tracing
```

---

# 9. 高并发库存问题

这是整个项目最核心的技术问题之一。

假设：

```text
库存：100

并发请求：10,000
```

一个错误的实现可能是：

```go
if stock > 0 {
    stock--
}
```

多个请求同时读取：

```text
stock = 1
```

然后多个请求同时成功扣减。

最终产生：

```text
overselling
```

也就是库存超卖。

项目会分别实现多种方案并进行性能比较。

---

# 10. PostgreSQL 行锁方案

第一版采用 PostgreSQL Transaction + Row Lock。

例如：

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

`FOR UPDATE` 会锁住对应库存行。

优点：

```text
强一致

实现简单

数据库仍然是唯一事实来源
```

缺点：

```text
热点库存行产生锁竞争

高并发情况下吞吐量受到限制

数据库容易成为瓶颈
```

这个版本可以作为后续性能优化的 baseline。

---

# 11. Redis 原子库存扣减

为了处理更高并发，可以将 Redis 放入库存预约关键路径。

架构变为：

```text
Request
   │
   ▼
Redis Atomic Reservation
   │
   ▼
Kafka
   │
   ▼
Async Persistence
```

库存通过 Redis Lua Script 原子执行。

例如：

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

Lua Script 在 Redis 内部一次执行完成：

```text
读取库存

判断库存

扣减库存
```

避免：

```text
GET
↓
业务判断
↓
DECR
```

之间出现并发竞态。

---

# 12. PostgreSQL vs Redis 库存方案

项目最终应实际比较：

```text
PostgreSQL Row Lock
```

和：

```text
Redis Atomic Reservation
```

记录：

```text
RPS

P50

P95

P99

CPU

数据库连接数

错误率
```

最终 README 中应该出现真实 benchmark，而不是理论结论。

---

# 13. Redis Cache

Redis 另一个重要用途是缓存。

例如：

```http
GET /api/v1/events/:id
```

采用：

```text
Cache-Aside Pattern
```

调用流程：

```text
Client
   │
   ▼
Application
   │
   ▼
Redis
   │
   ├── HIT
   │     │
   │     ▼
   │   Return
   │
   └── MISS
         │
         ▼
    PostgreSQL
         │
         ▼
      Redis
         │
         ▼
      Response
```

---

# 14. Cache Aside

伪代码：

```go
func GetEvent(ctx context.Context, id string) (*Event, error) {
    event, err := cache.Get(ctx, id)

    if err == nil {
        return event, nil
    }

    event, err = repository.GetEvent(ctx, id)

    if err != nil {
        return nil, err
    }

    cache.Set(ctx, id, event)

    return event, nil
}
```

项目中需要明确考虑：

```text
TTL

缓存失效

缓存更新

缓存穿透

缓存击穿

缓存雪崩

Hot Key
```

---

# 15. 缓存击穿

假设某个热门活动：

```text
event:123
```

刚好缓存过期。

此时：

```text
10,000 requests
```

同时进入。

没有保护时：

```text
             Redis MISS
                 │
        ┌────────┼────────┐
        ▼        ▼        ▼
     Request  Request   Request
        │        │        │
        └────────┼────────┘
                 ▼
             PostgreSQL
```

数据库可能瞬间收到大量相同查询。

---

# 16. Singleflight

Go 可以使用：

```text
golang.org/x/sync/singleflight
```

保护相同缓存 Key。

例如：

```go
var group singleflight.Group

value, err, _ := group.Do(cacheKey, func() (interface{}, error) {
    return loadFromDatabase(ctx)
})
```

此时同一个 Key 的多个请求：

```text
1000 Cache Misses
        │
        ▼
Singleflight
        │
        ▼
1 PostgreSQL Query
```

其他请求共享查询结果。

---

# 17. 缓存过期优化

还可以进一步实现：

```text
Randomized TTL

Background Refresh

Logical Expiration
```

例如随机 TTL：

```text
TTL = baseTTL + randomOffset
```

避免大量缓存同时过期。

---

# 18. 异步订单处理

用户成功预约库存后，不需要同步完成整个订单流程。

同步模式：

```text
Reservation
     │
     ▼
Inventory
     │
     ▼
Create Order
     │
     ▼
Payment
     │
     ▼
Response
```

请求链过长。

FlashFlow 改为：

```text
POST /reservations
       │
       ▼
Reserve Inventory
       │
       ▼
Publish Event
       │
       ▼
202 Accepted
```

然后后台异步处理。

---

# 19. 预约请求响应

API 可以立即返回：

```json
{
  "reservationId": "4b59531c-...",
  "status": "PROCESSING"
}
```

随后：

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
   Order Worker
        │
        ▼
    PostgreSQL
```

---

# 20. Kafka

Kafka 用来解决：

```text
流量削峰

异步处理

服务解耦

事件驱动

不同服务独立扩缩容
```

而不是单纯为了使用消息队列。

---

# 21. Kafka Topics

可以设计：

```text
reservation.created

reservation.expired

order.created

order.paid

order.cancelled

notification.requested
```

例如：

```json
{
  "eventId": "event-123",
  "reservationId": "reservation-456",
  "userId": "user-789",
  "quantity": 2,
  "timestamp": "2026-08-20T17:00:00Z"
}
```

---

# 22. Kafka 学习目标

项目应该真正涉及：

```text
Producer

Consumer

Topic

Partition

Consumer Group

Offset

Retry

Dead Letter Queue

Consumer Lag

At-Least-Once Delivery
```

---

# 23. Kafka Partition Strategy

可以考虑：

```text
event_id
```

作为 partition key。

例如：

```text
reservation.created
```

按照：

```text
event_id
```

进行分区。

这样同一个活动的事件可以保持一定顺序。

同时也需要理解：

```text
Partition 数量影响并行度

Partition Key 影响数据分布

热点 Event 可能产生热点 Partition
```

---

# 24. 消息重复问题

Kafka 通常采用：

```text
At-Least-Once Delivery
```

因此同一条消息可能被处理多次。

例如：

```text
reservation.created
       │
       ├────► Order #1
       │
       └────► Order #2
```

如果 consumer 不支持幂等，就会创建重复订单。

---

# 25. Consumer 幂等

数据库可以通过：

```sql
CREATE UNIQUE INDEX idx_orders_reservation_id
ON orders(reservation_id);
```

保证：

```text
一个 Reservation 只能对应一个 Order
```

也可以记录：

```text
processed_events
```

例如：

```text
event_id

consumer

processed_at
```

处理流程：

```text
Message Received
       │
       ▼
Processed Before?
    │       │
   YES      NO
    │       │
 Ignore   Process
```

---

# 26. HTTP 幂等

不仅 Kafka Consumer 需要幂等。

用户调用：

```http
POST /api/v1/events/123/reservations
```

时，也可能因为网络超时重复请求。

例如：

```text
Client
  │
  ▼
POST Reservation
  │
  ▼
Server success
  │
  ▼
Response lost
  │
  ▼
Client retries
```

如果没有幂等：

```text
Reservation #1

Reservation #2
```

因此 API 可以支持：

```http
Idempotency-Key: 54e53dd2-...
```

服务端记录请求执行结果。

重复请求返回相同结果。

---

# 27. 分布式事务问题

假设 Order Service 执行：

```text
BEGIN

INSERT order

COMMIT

publish Kafka message
```

可能出现：

```text
PostgreSQL Commit
       │
       ▼
    Success

Kafka Publish
       │
       ▼
    Failure
```

此时：

```text
数据库中已有订单

但其他服务不知道订单已创建
```

这就是典型的双写一致性问题。

---

# 28. Transactional Outbox Pattern

FlashFlow 使用：

```text
Transactional Outbox Pattern
```

解决数据库和 Kafka 双写问题。

在同一个事务中：

```text
BEGIN

INSERT INTO orders ...

INSERT INTO outbox_events ...

COMMIT
```

这样：

```text
Order

Outbox Event
```

一定同时成功或者同时失败。

---

# 29. Outbox Worker

独立 Worker：

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

发送成功后：

```text
processed_at = NOW()
```

因此即使 Kafka 暂时不可用：

```text
事件仍然保存在 PostgreSQL
```

服务恢复后可以继续发送。

---

# 30. Retry

面对临时错误，不应该立刻永久失败。

可以采用：

```text
Exponential Backoff
```

例如：

```text
1s

2s

4s

8s

16s
```

---

# 31. Dead Letter Queue

如果多次重试仍然失败：

```text
Kafka Event
    │
    ▼
Consumer
    │
    ▼
Retry
    │
    ▼
Retry Failed
    │
    ▼
DLQ
```

例如：

```text
order.processing.dlq
```

后续可以：

```text
人工检查

自动告警

重新投递
```

---

# 32. Reservation Expiration

用户预约库存后不应该永久占用。

例如：

```text
Reservation Created
       │
       ▼
10 Minutes
       │
       ├──── Payment Success
       │         │
       │         ▼
       │      Confirm
       │
       └──── Timeout
                 │
                 ▼
           Release Inventory
```

实现方式可以比较：

```text
Redis TTL

Scheduled Worker

Delayed Message

Expiration Queue
```

过期后发布：

```text
reservation.expired
```

释放库存。

---

# 33. PostgreSQL

PostgreSQL 是主要持久化数据库。

项目应覆盖：

```text
Schema Design

Primary Key

Foreign Key

Unique Constraint

Transaction

Isolation Level

Indexes

Query Plan

Connection Pool

Row Lock

Optimistic Lock

Pessimistic Lock

Pagination
```

---

# 34. pgx + sqlc

项目推荐：

```text
pgx

+

sqlc
```

而不是所有数据库操作都依赖 ORM。

原因是希望显式展示：

```text
SQL

Transaction

Index

Lock

Query Optimization
```

---

# 35. sqlc 示例

SQL：

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

sqlc 根据 SQL 自动生成：

```text
Type-Safe Go Code
```

兼顾：

```text
原生 SQL 的透明性

Go 类型安全
```

---

# 36. 数据库索引优化

例如订单查询：

```sql
SELECT *
FROM orders
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 20;
```

可以建立：

```sql
CREATE INDEX idx_orders_user_created
ON orders(user_id, created_at DESC);
```

然后使用：

```sql
EXPLAIN ANALYZE
```

分析。

记录真实数据：

```text
Before Index

Execution Time: TBD

After Index

Execution Time: TBD
```

---

# 37. 不伪造性能数字

所有 benchmark：

```text
RPS

P95

P99

Cache Hit Ratio

Database Latency

CPU

Memory
```

必须来自真实测试。

README 中未测试的数据统一写：

```text
TBD
```

而不是填写想象中的数字。

---

# 38. Pagination

对于大数据分页，不一直使用：

```sql
OFFSET 100000
LIMIT 20;
```

可以采用 Cursor Pagination：

```sql
WHERE created_at < $1
ORDER BY created_at DESC
LIMIT 20;
```

用于展示：

```text
Offset Pagination

vs

Cursor Pagination
```

之间的性能和一致性区别。

---

# 39. Connection Pool

数据库连接使用 pgxpool。

需要理解：

```text
Max Connections

Min Connections

Connection Lifetime

Idle Timeout
```

并观察：

```text
高并发下 DB Connection Usage
```

避免：

```text
大量请求
    │
    ▼
无限创建 DB Connection
```

---

# 40. Go 并发模型

项目中应该自然使用：

```text
Goroutine

Channel

WaitGroup

Context

errgroup

Worker Pool
```

而不是人为写一些没有意义的并发 Demo。

---

# 41. Worker Pool

例如 Notification Worker：

```go
jobs := make(chan Job, 100)

for i := 0; i < workerCount; i++ {
    go worker(ctx, jobs)
}
```

可以控制：

```text
最大并发 Worker 数量
```

防止：

```text
一条消息创建一个 Goroutine

无限并发
```

---

# 42. Context

`context.Context` 应该贯穿：

```text
HTTP
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
PostgreSQL
```

也贯穿：

```text
HTTP
 │
 ▼
gRPC
 │
 ▼
Downstream Service
```

用于：

```text
Cancellation

Deadline

Timeout

Trace Propagation
```

---

# 43. Graceful Shutdown

Go 服务需要处理：

```text
SIGTERM

SIGINT
```

关闭流程：

```text
SIGTERM
   │
   ▼
停止接受新请求
   │
   ▼
等待正在执行的请求
   │
   ▼
停止 Kafka Consumer
   │
   ▼
停止 Worker
   │
   ▼
关闭 PostgreSQL
   │
   ▼
关闭 Redis
   │
   ▼
Exit
```

这对于 Kubernetes Deployment 非常重要。

---

# 44. Rate Limiting

高并发服务需要限流。

可以使用 Redis 实现：

```text
100 Requests / Minute / User
```

限流维度：

```text
IP

User ID

API Key

Endpoint
```

---

# 45. Rate Limiting Algorithm

可以实现或比较：

```text
Fixed Window

Sliding Window

Token Bucket
```

例如预约接口：

```text
POST /reservations
```

比：

```text
GET /events
```

拥有更严格的限流策略。

---

# 46. Authentication

认证采用：

```text
JWT Access Token

+

Refresh Token
```

需要实现：

```text
Password Hashing

JWT Validation

Refresh Token

Token Rotation

Logout
```

---

# 47. Authorization

使用 RBAC：

```text
USER

ADMIN
```

例如：

```text
USER

浏览活动

预约

查看订单
```

而：

```text
ADMIN

创建活动

调整库存

查看系统指标
```

---

# 48. Security

项目至少覆盖：

```text
Password Hashing

Input Validation

SQL Injection Prevention

Parameterized SQL

JWT Validation

CORS

Rate Limiting

Secret Management

Authorization

Webhook Validation
```

---

# 49. 模块化单体优先

项目不应该一开始就创建：

```text
10 个微服务
```

第一阶段采用：

```text
Modular Monolith
```

例如：

```text
Application

├── auth
├── event
├── inventory
├── reservation
├── order
└── payment
```

---

# 50. 为什么不直接微服务

因为一开始：

```text
业务规模小

一个开发者

部署规模小

服务边界仍然可能变化
```

此时直接拆成大量服务会增加：

```text
部署复杂度

网络调用

分布式事务

日志追踪

测试成本

基础设施成本
```

---

# 51. 服务什么时候拆

例如：

```text
Inventory
```

具有：

```text
高并发

低延迟

独立扩容
```

需求。

而：

```text
Order Worker
```

具有：

```text
异步消费

Kafka Consumer

后台处理
```

需求。

两者 workload 明显不同。

因此可以拆成：

```text
Inventory Service

Order Service
```

这是合理的服务拆分理由。

---

# 52. 微服务最终形态

后期可以形成：

```text
API Gateway

Identity Module / Service

Inventory Service

Order Service

Notification Worker
```

但不需要为了项目规模继续拆：

```text
User Service

Email Service

Profile Service

Event Service

Admin Service
```

等大量无意义服务。

---

# 53. Observability

项目需要完整展示：

```text
Metrics

Logs

Traces
```

这三类可观测能力。

---

# 54. Prometheus Metrics

应用指标例如：

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

---

# 55. Infrastructure Metrics

还应该观察：

```text
CPU

Memory

Goroutines

Database Connections

Redis Connections

Kafka Consumer Lag
```

---

# 56. Business Metrics

除了基础设施指标，也需要业务指标：

```text
Reservation Success Rate

Reservation Failure Rate

Orders Created / Minute

Inventory Remaining

Payment Success Rate
```

这样 Grafana 不只是：

```text
CPU Dashboard
```

而是：

```text
真正理解系统业务状态的 Dashboard
```

---

# 57. Grafana

Dashboard 可以包含：

```text
Requests / Second

P50 Latency

P95 Latency

P99 Latency

Error Rate

Reservation Success Rate

Cache Hit Ratio

DB Query Latency

Kafka Consumer Lag

CPU

Memory

Pod Count
```

例如：

```text
┌──────────────────────────────────────────┐
│ FlashFlow Backend Dashboard              │
├───────────────────┬──────────────────────┤
│ Requests / sec    │ P95 Latency          │
├───────────────────┼──────────────────────┤
│ Error Rate        │ Reservation Success  │
├───────────────────┼──────────────────────┤
│ Cache Hit Ratio   │ Kafka Consumer Lag   │
├───────────────────┼──────────────────────┤
│ CPU               │ Memory               │
└───────────────────┴──────────────────────┘
```

---

# 58. Structured Logging

日志使用 JSON。

例如：

```json
{
  "timestamp": "2026-08-20T17:30:00Z",
  "level": "INFO",
  "service": "order-service",
  "trace_id": "47d...",
  "request_id": "52a...",
  "reservation_id": "84d...",
  "order_id": "20f...",
  "message": "order created",
  "latency_ms": 23
}
```

重要字段：

```text
trace_id

request_id

user_id

reservation_id

order_id

service
```

---

# 59. Distributed Tracing

OpenTelemetry 追踪：

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
        └── Kafka Producer
                 │
                 ▼
           Order Consumer
                 │
                 ▼
            PostgreSQL
```

例如 Trace：

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

使用：

```text
Jaeger
```

或者：

```text
Grafana Tempo
```

展示。

---

# 60. Kafka Trace Propagation

异步链路也应该传递 Trace Context：

```text
HTTP Request
     │
     ▼
Kafka Producer
     │
     ▼
Kafka Headers
     │
     ▼
Kafka Consumer
```

这样可以让：

```text
HTTP Span
```

和：

```text
Kafka Consumer Span
```

关联起来。

---

# 61. Health Check

服务暴露：

```http
GET /health

GET /ready
```

例如：

```json
{
  "status": "ok"
}
```

---

# 62. Kubernetes Probe

Kubernetes 使用：

```text
livenessProbe

readinessProbe
```

其中：

```text
livenessProbe
```

判断服务是否需要重启。

```text
readinessProbe
```

判断服务是否可以接受流量。

---

# 63. Docker

所有服务均容器化。

例如：

```bash
docker build -t flashflow-api .
```

推荐 Multi-Stage Build。

例如：

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

# 64. Docker Compose

本地开发使用：

```text
Docker Compose
```

运行：

```text
PostgreSQL

Redis

Kafka

Prometheus

Grafana

Jaeger
```

启动：

```bash
docker compose up -d
```

---

# 65. Kubernetes

生产模拟环境使用 Kubernetes。

主要资源：

```text
Deployment

Service

Ingress

ConfigMap

Secret

HorizontalPodAutoscaler
```

---

# 66. Kubernetes Architecture

例如：

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
               Redis / Kafka / DB
```

---

# 67. Resource Requests / Limits

Pod 应设置：

```text
CPU Request

CPU Limit

Memory Request

Memory Limit
```

例如：

```yaml
resources:
  requests:
    cpu: "250m"
    memory: "256Mi"

  limits:
    cpu: "1000m"
    memory: "512Mi"
```

实际值应该根据 benchmark 调整。

---

# 68. Horizontal Pod Autoscaler

例如：

```text
Min Replicas: 2

Max Replicas: 10

Target CPU: 70%
```

流程：

```text
流量增加
   │
   ▼
CPU 增加
   │
   ▼
HPA 检测
   │
   ▼
增加 Pod
   │
   ▼
容量提升
```

---

# 69. CI/CD

使用：

```text
GitHub Actions
```

Pull Request：

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
unit test
     │
     ▼
race detector
     │
     ▼
integration test
```

---

# 70. Main Branch Pipeline

代码合并后：

```text
Merge Main
    │
    ▼
Build Go Binary
    │
    ▼
Build Docker Image
    │
    ▼
Push GHCR
    │
    ▼
Deploy Kubernetes
    │
    ▼
Health Check
```

---

# 71. GitHub Actions 示例

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

后续 Deployment Workflow 可以加入：

```text
Docker Build

Push GHCR

kubectl apply

Health Check
```

---

# 72. 自动化测试体系

项目测试层级：

```text
Unit Test
    │
    ▼
Integration Test
    │
    ▼
API Test
    │
    ▼
Concurrency Test
    │
    ▼
Load Test
```

---

# 73. Unit Test

单元测试主要覆盖：

```text
Business Logic

Validation

Reservation Rule

Idempotency

Retry Policy
```

执行：

```bash
go test ./...
```

---

# 74. Race Detector

Go 自带：

```bash
go test -race ./...
```

检测：

```text
Data Race
```

尤其用于：

```text
Worker Pool

Shared State

Concurrent Consumer
```

---

# 75. Integration Test

Integration Test 尽量使用真实基础设施。

可以通过：

```text
testcontainers-go
```

动态启动：

```text
PostgreSQL

Redis

Kafka
```

结构：

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

避免所有基础设施都通过 mock 测试。

---

# 76. 并发测试

特别需要测试：

```text
100 Inventory

10,000 Requests
```

最终必须满足：

```text
Successful Reservations <= 100
```

并且：

```text
Database Orders <= Successful Reservations
```

确保真正没有超卖。

---

# 77. Load Testing

使用：

```text
k6
```

或者：

```text
Vegeta
```

例如：

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
    "reservation accepted": (r) => r.status === 200 || r.status === 202,
  });
}
```

---

# 78. Load Test Scenario

至少测试：

```text
100 Users

1,000 Users

5,000 Users

10,000 Users
```

还可以增加：

```text
Ramp Test

Spike Test

Stress Test

Soak Test
```

---

# 79. 需要记录的性能数据

每次性能测试应该记录：

```text
Throughput

Requests / Second

P50

P95

P99

Error Rate

CPU

Memory

Database Connections

Redis Cache Hit Ratio

Kafka Consumer Lag
```

---

# 80. Benchmark 环境

性能数据必须同时写明测试环境。

例如：

```text
CPU: TBD

Memory: TBD

OS: TBD

Go Version: TBD

PostgreSQL Version: TBD

Redis Version: TBD

Kafka Version: TBD
```

否则：

```text
10,000 RPS
```

这个数字没有意义。

---

# 81. Benchmark Results

在实际测试完成前：

```text
TBD
```

例如：

## PostgreSQL Reservation

```text
Concurrent Clients: TBD

Throughput: TBD req/s

P50: TBD ms

P95: TBD ms

P99: TBD ms

Error Rate: TBD %
```

---

## Redis Reservation

```text
Concurrent Clients: TBD

Throughput: TBD req/s

P50: TBD ms

P95: TBD ms

P99: TBD ms

Error Rate: TBD %
```

---

## Cache Disabled

```text
Throughput: TBD

P95: TBD

DB Queries / sec: TBD
```

---

## Cache Enabled

```text
Throughput: TBD

P95: TBD

Cache Hit Ratio: TBD
```

---

# 82. Failure Testing

项目主动制造故障。

例如：

```text
Kill Order Consumer

Kill API Pod

Restart Outbox Worker

Duplicate Kafka Event

Redis Down

PostgreSQL Down

Kafka Down

Slow PostgreSQL

Network Timeout

SIGTERM Pod
```

重点不是让系统永远不失败。

而是：

```text
知道失败以后会发生什么。
```

---

# 83. Consumer Crash 测试

假设：

```text
Kafka Event Received
       │
       ▼
Insert Order
       │
       ▼
Consumer Crash
       │
       ▼
Offset Not Committed
       │
       ▼
Kafka Redelivery
```

如果没有幂等：

```text
Duplicate Order
```

有幂等：

```text
Kafka Redelivery
       │
       ▼
INSERT order
       │
       ▼
UNIQUE reservation_id
       │
       ▼
Duplicate Detected
       │
       ▼
Ignore
```

---

# 84. Outbox Worker Crash

场景：

```text
Outbox Worker
     │
     ▼
Publish Kafka
     │
     ▼
Crash Before Marking Processed
```

Worker 重启后可能再次发送。

因此：

```text
Consumer Idempotency
```

依然非常重要。

Transactional Outbox 并不自动提供：

```text
Exactly Once
```

而是配合：

```text
At-Least-Once + Idempotency
```

实现业务上的可靠性。

---

# 85. Redis Failure

需要明确设计：

```text
Redis 不可用时系统怎么处理？
```

不同功能可以选择不同策略。

例如 Event Cache：

```text
Redis Down
   │
   ▼
Fallback PostgreSQL
```

而高并发库存：

```text
Redis Down
```

可能选择：

```text
Fail Fast
```

避免使用不安全 fallback 导致超卖。

---

# 86. Timeout

所有外部依赖调用都需要 Timeout。

例如：

```text
PostgreSQL

Redis

Kafka

gRPC
```

不能允许请求无限等待。

---

# 87. Retry

Retry 只用于：

```text
Transient Failure
```

例如：

```text
Temporary Network Failure
```

不应该对：

```text
Validation Error

Permission Denied

Inventory Sold Out
```

进行重试。

---

# 88. Circuit Breaker

如果后期增加 Circuit Breaker，也必须有明确理由。

例如：

```text
Notification Service
```

长时间不可用。

此时不断同步调用只会：

```text
增加延迟

消耗连接

加重故障
```

可以进行：

```text
Circuit Breaking
```

但它属于可选高级内容，而不是项目第一阶段必做项。

---

# 89. 前端范围

前端控制在少量页面：

```text
/login

/events

/events/:id

/orders

/admin
```

主要负责展示：

```text
登录

活动浏览

库存

预约

订单状态

系统监控
```

---

# 90. Admin Dashboard

后台 Dashboard 可以展示：

```text
Requests / Second

P95 Latency

Active Reservations

Orders / Minute

Available Inventory

Kafka Consumer Lag

Cache Hit Ratio

Error Rate
```

例如：

```text
┌──────────────────────────────────────┐
│ FlashFlow Admin                      │
├─────────────────┬────────────────────┤
│ RPS             │ TBD                │
│ P95             │ TBD                │
│ Orders/min      │ TBD                │
│ Cache Hit Rate  │ TBD                │
│ Kafka Lag       │ TBD                │
│ Inventory       │ TBD                │
└─────────────────┴────────────────────┘
```

---

# 91. 前端开发原则

不花大量时间做：

```text
Landing Page

Animation

Design System

几十个 CRUD 页面

复杂 CMS
```

重点仍然是：

```text
Backend Engineering
```

---

# 92. Repository Structure

推荐目录：

```text
flashflow/
│
├── cmd/
│   ├── api/
│   │   └── main.go
│   │
│   ├── order-worker/
│   │   └── main.go
│   │
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
│   ├── cache.md
│   ├── messaging.md
│   ├── reliability.md
│   ├── observability.md
│   ├── benchmarks.md
│   └── failure-testing.md
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

# 93. 开发路线

项目不要第一天直接上：

```text
Kafka

Kubernetes

gRPC

微服务

OpenTelemetry
```

应该逐阶段演进。

---

# Phase 1 — 后端基础

目标：

- [ ] 初始化 Go Project
- [ ] 设计项目目录
- [ ] PostgreSQL
- [ ] Migration
- [ ] Authentication
- [ ] Event API
- [ ] Inventory
- [ ] Reservation
- [ ] Order
- [ ] Validation
- [ ] Structured Logging
- [ ] Unit Test
- [ ] Integration Test
- [ ] Docker

初始架构：

```text
Go API
   │
   ▼
PostgreSQL
```

这一阶段重点证明：

```text
我会正确开发一个普通后端服务。
```

---

# Phase 2 — 性能优化

增加：

- [ ] Redis
- [ ] Cache Aside
- [ ] Cache TTL
- [ ] Cache Stampede Protection
- [ ] Singleflight
- [ ] Rate Limiting
- [ ] PostgreSQL Index
- [ ] Query Optimization
- [ ] Connection Pool Tuning
- [ ] k6 Load Test
- [ ] P50 / P95 / P99

架构：

```text
        Go API
        /    \
       ▼      ▼
    Redis   PostgreSQL
```

重点证明：

```text
我会测量系统性能，并根据瓶颈进行优化。
```

---

# Phase 3 — 高并发库存

增加：

- [ ] PostgreSQL Row Lock
- [ ] Redis Atomic Reservation
- [ ] Redis Lua Script
- [ ] Overselling Test
- [ ] Idempotency Key
- [ ] Reservation Expiration
- [ ] PostgreSQL vs Redis Benchmark

重点证明：

```text
我理解 concurrency、atomicity、locking 和 consistency。
```

---

# Phase 4 — 分布式系统

增加：

- [ ] Kafka
- [ ] reservation.created
- [ ] Order Worker
- [ ] Consumer Group
- [ ] Idempotent Consumer
- [ ] Retry
- [ ] DLQ
- [ ] Transactional Outbox
- [ ] gRPC
- [ ] Inventory Service
- [ ] Order Service

架构：

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

重点证明：

```text
我理解 distributed system 中的失败和一致性问题。
```

---

# Phase 5 — 可观测性

增加：

- [ ] Prometheus
- [ ] HTTP Metrics
- [ ] Business Metrics
- [ ] Redis Metrics
- [ ] PostgreSQL Metrics
- [ ] Kafka Metrics
- [ ] Grafana
- [ ] OpenTelemetry
- [ ] Distributed Tracing
- [ ] Structured Logs
- [ ] Trace ID Propagation

重点证明：

```text
我不仅会写服务，也知道上线以后怎么观察服务。
```

---

# Phase 6 — Cloud Native

增加：

- [ ] Kubernetes Deployment
- [ ] Service
- [ ] Ingress
- [ ] ConfigMap
- [ ] Secret
- [ ] Resource Request / Limit
- [ ] Liveness Probe
- [ ] Readiness Probe
- [ ] HPA

重点证明：

```text
服务可以被容器化、部署和水平扩展。
```

---

# Phase 7 — CI/CD

增加：

- [ ] GitHub Actions
- [ ] gofmt
- [ ] golangci-lint
- [ ] go vet
- [ ] Unit Test
- [ ] Race Detector
- [ ] Integration Test
- [ ] Docker Build
- [ ] Push GHCR
- [ ] Kubernetes Deploy
- [ ] Deployment Health Check

重点证明：

```text
代码从提交到部署可以自动化完成。
```

---

# Phase 8 — Reliability Testing

增加：

- [ ] Kill Consumer
- [ ] Duplicate Kafka Event
- [ ] Restart Outbox Worker
- [ ] Redis Failure
- [ ] Kafka Failure
- [ ] PostgreSQL Failure
- [ ] PostgreSQL Latency
- [ ] API Pod Kill
- [ ] Graceful Shutdown Test
- [ ] Kubernetes Recovery Test

重点证明：

```text
系统不是只在 happy path 下运行。
```

---

# Phase 9 — Frontend & Demo

增加：

- [ ] Login
- [ ] Event List
- [ ] Event Detail
- [ ] Reservation
- [ ] Order Status
- [ ] Admin Dashboard
- [ ] Real-Time Status
- [ ] Demo Workflow

---

# 94. Architecture Decision Records

项目中重要技术决定最好单独记录。

例如：

```text
为什么选择 Go？

为什么 PostgreSQL？

为什么 Redis？

为什么 Kafka？

为什么外部 REST？

为什么内部 gRPC？

为什么先 Modular Monolith？

为什么拆 Inventory Service？

为什么使用 sqlc？

为什么采用 Transactional Outbox？

为什么接受 At-Least-Once？

为什么使用 Kubernetes？
```

---

# 95. 每个架构决策的结构

推荐使用：

```text
Problem
   │
   ▼
Options
   │
   ▼
Trade-offs
   │
   ▼
Decision
   │
   ▼
Consequences
```

而不是：

```text
因为 Kafka 性能很好，所以用了 Kafka。
```

---

# 96. 核心工程挑战

项目最终重点展示以下问题。

## 防止超卖

在大量并发请求下：

```text
Reserved Inventory <= Available Inventory
```

---

## 幂等

无论：

```text
HTTP Retry

Kafka Redelivery
```

都不会产生重复业务数据。

---

## 可靠事件投递

数据库写入和 Kafka Event 不会因为单次故障永久不一致。

---

## Traffic Spike

突然的高流量可以通过：

```text
Redis

Kafka

Horizontal Scaling
```

进行缓冲和扩展。

---

## 数据库性能

通过：

```text
EXPLAIN ANALYZE

Indexes

Connection Pool

Load Test
```

实际优化。

---

## Cache Consistency

Redis 提高性能的同时，需要明确：

```text
什么时候失效

什么时候更新

允许多长时间 stale data
```

---

## Failure Recovery

服务 Crash 后：

```text
数据不会永久损坏

消息可以重新处理

系统能够恢复
```

---

## Observability

系统出现问题时，可以回答：

```text
哪里慢？

哪个服务错了？

Kafka 是否积压？

数据库是否是瓶颈？

Redis 是否命中？

哪一个请求失败？
```

---

# 97. 项目不应该变成什么

FlashFlow 不应该变成：

```text
巨大 CRUD 系统

30 个前端页面

15 个没有必要的微服务

技术名词堆砌项目

没有数据支撑的高并发项目

复制教程完成的 Demo
```

---

# 98. 核心工程原则

## Start Simple

先构建正确的简单系统。

---

## Measure Before Optimize

先测：

```text
RPS

P95

P99

CPU

DB Query
```

再优化。

---

## Design for Failure

假设：

```text
Network 会失败

Redis 会失败

Kafka 会失败

Consumer 会 Crash

请求会 Retry
```

然后设计恢复方案。

---

## Explicit over Magic

关键数据库逻辑尽量显式：

```text
SQL

Transaction

Lock

Constraint

Index
```

避免重要行为完全隐藏在 ORM 后面。

---

## Avoid Resume-Driven Architecture

不要因为简历需要：

```text
Kafka

Kubernetes

gRPC
```

就强行加入。

每一个技术必须能够回答：

```text
它解决了什么问题？
```

---

# 99. 项目完成后应该掌握的 Go 能力

```text
goroutine

channel

context.Context

sync.WaitGroup

errgroup

worker pool

interfaces

error handling

graceful shutdown

race detector

HTTP middleware
```

---

# 100. HTTP Backend 能力

```text
REST API

Middleware

Authentication

Authorization

Validation

Rate Limiting

API Versioning

Idempotency

Error Handling
```

---

# 101. PostgreSQL 能力

```text
Schema Design

Transaction

Isolation Level

Row Lock

Index

Constraint

Query Plan

Connection Pool

Cursor Pagination
```

---

# 102. Redis 能力

```text
Cache Aside

TTL

Atomic Operation

Lua Script

Rate Limiting

Cache Stampede

Hot Key
```

---

# 103. Kafka 能力

```text
Producer

Consumer

Consumer Group

Partition

Offset

Retry

DLQ

Consumer Lag

At-Least-Once Delivery
```

---

# 104. Distributed Systems 能力

```text
Idempotency

Eventual Consistency

Partial Failure

Retry

Timeout

Transactional Outbox

Async Messaging

Failure Recovery
```

---

# 105. Cloud Native 能力

```text
Docker

Kubernetes

Deployment

Service

Ingress

Health Check

HPA

Horizontal Scaling

Config

Secrets
```

---

# 106. DevOps 能力

```text
CI/CD

GitHub Actions

Automated Tests

Container Registry

Deployment Automation
```

---

# 107. Observability 能力

```text
Metrics

Logs

Traces

Prometheus

Grafana

OpenTelemetry

Jaeger / Tempo
```

---

# 108. 面试问题

完成项目后，应该可以围绕项目回答很多典型后端问题。

---

## 为什么使用 Redis？

因为系统存在：

```text
Read Heavy Workload

Hot Data

High-Concurrency Inventory
```

Redis 可以用于：

```text
Cache

Atomic Inventory Reservation

Rate Limiting
```

但 Redis 并不是所有数据的最终事实来源。

---

## 如何防止库存超卖？

项目实现并比较：

```text
PostgreSQL Transaction + Row Lock
```

和：

```text
Redis Atomic Reservation + Lua Script
```

并通过并发测试验证：

```text
Successful Reservation <= Inventory
```

---

## 如果 Kafka 重复发送消息怎么办？

Consumer 使用：

```text
Idempotency
```

数据库使用：

```text
Unique Constraint
```

保证重复事件不会创建重复订单。

---

## 如果数据库 Commit 成功但 Kafka Publish 失败怎么办？

使用：

```text
Transactional Outbox
```

将：

```text
Business Data

Outbox Event
```

写入同一个数据库事务。

---

## Consumer Crash 怎么办？

如果：

```text
DB Commit

Consumer Crash

Offset Not Commit
```

Kafka 会重新投递。

Consumer 的幂等能力防止重复执行。

---

## 为什么不用 Exactly Once？

因为分布式系统里通常更实际的是：

```text
At-Least-Once Delivery

+

Idempotent Consumer
```

实现业务意义上的 Exactly Once Effect。

---

## 为什么不是一开始就微服务？

因为：

```text
Microservices 是成本，不是目标。
```

系统先使用 Modular Monolith。

只有 Inventory、Order 等模块存在：

```text
独立扩容

故障隔离

Workload 差异
```

时才拆分。

---

## 为什么使用 Kafka？

因为订单创建不需要阻塞用户预约请求。

Kafka 可以：

```text
异步处理

流量削峰

服务解耦

独立扩容
```

---

## 为什么内部使用 gRPC？

内部服务需要：

```text
Strong Contract

Protobuf

Efficient Serialization

Low-Latency RPC
```

外部浏览器仍然使用 REST。

---

## 如何知道 Redis 确实提高了性能？

通过：

```text
k6
```

对比：

```text
Cache Disabled

vs

Cache Enabled
```

记录：

```text
RPS

P95

P99

DB Query Count

Cache Hit Ratio
```

---

## 如何排查一个慢请求？

首先查看：

```text
Grafana Metrics
```

确定：

```text
P95 / P99
```

异常。

然后通过：

```text
OpenTelemetry Trace
```

确定慢在哪一个 Span。

例如：

```text
HTTP

Redis

gRPC

Kafka

PostgreSQL
```

然后结合：

```text
Structured Logs

EXPLAIN ANALYZE
```

进一步定位。

---

# 109. 简历描述

项目完成后，可以写成：

## FlashFlow — 高并发分布式预约与订单平台

```text
Go · PostgreSQL · Redis · Kafka · gRPC · Docker · Kubernetes · OpenTelemetry · Prometheus · Grafana
```

- 基于 Go 设计并实现高并发预约系统，通过 PostgreSQL Transaction / Row Lock 和 Redis Lua Script 实现原子库存扣减，并通过并发测试防止库存超卖。
- 基于 Kafka 构建异步订单处理流程，实现 Consumer 幂等、Retry、Dead Letter Queue 和 Transactional Outbox，解决消息重复和数据库/Kafka 双写一致性问题。
- 使用 PostgreSQL Index、EXPLAIN ANALYZE、Redis Cache-Aside 和 Singleflight 优化系统性能，并使用 k6 对吞吐量、P95/P99 延迟、数据库负载和缓存命中率进行测试。
- 通过 OpenTelemetry、Prometheus、Grafana 和结构化日志建立 Metrics、Logs、Distributed Tracing 可观测体系。
- 使用 Docker 容器化服务，通过 Kubernetes Deployment、Health Check 和 HPA 实现服务部署与水平扩缩容，并通过 GitHub Actions 建立 CI/CD Pipeline。

---

# 110. 完成 Benchmark 后的简历增强

只有在真实测试完成之后，才加入：

```text
在 X 并发请求下保持零超卖

P95 从 X ms 降低到 Y ms

吞吐量从 X req/s 提升到 Y req/s

Redis Cache 将数据库查询量降低 X%

成功处理 X 次 Kafka 重复投递且未产生重复订单
```

禁止写没有测过的数据。

---

# 111. 运行项目

Clone：

```bash
git clone https://github.com/<your-username>/flashflow.git

cd flashflow
```

启动基础设施：

```bash
docker compose up -d
```

执行 Migration：

```bash
make migrate-up
```

启动 Backend：

```bash
make run
```

测试：

```bash
make test
```

Race Detector：

```bash
make test-race
```

Integration Test：

```bash
make test-integration
```

Load Test：

```bash
make load-test
```

关闭：

```bash
docker compose down
```

---

# 112. Makefile

规划命令：

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

# 113. Environment Variables

开发环境示例：

```bash
APP_ENV=development

HTTP_PORT=8080

DATABASE_URL=postgres://flashflow:flashflow@localhost:5432/flashflow

REDIS_ADDR=localhost:6379

KAFKA_BROKERS=localhost:9092

JWT_SECRET=development-only-secret

OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

生产环境 Secret 不应该提交到 Git。

---

# 114. Documentation

详细工程文档放在：

```text
docs/
```

例如：

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

README 负责：

```text
项目总览
```

docs 负责：

```text
深入解释工程设计和 Trade-Off。
```

---

# 115. Future Improvements

项目完成核心目标之后，可以进一步尝试：

```text
Kafka Schema Registry

Kafka Partition Strategy Benchmark

Redis Cluster

PostgreSQL Read Replica

CDC

Debezium

Helm

Argo CD

Chaos Engineering

Service Mesh

Multi-Region Deployment
```

但是这些都属于：

```text
Future Work
```

不是第一版项目的必须内容。

---

# 116. 项目最终目标

FlashFlow 最终希望证明的不只是：

```text
我会 Go。
```

而是：

```text
我能够完整开发一个后端系统。
```

包括：

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
Backend Implementation
     │
     ▼
Testing
     │
     ▼
Performance Optimization
     │
     ▼
Caching
     │
     ▼
Concurrency
     │
     ▼
Distributed Systems
     │
     ▼
Reliability
     │
     ▼
Observability
     │
     ▼
Containerization
     │
     ▼
CI/CD
     │
     ▼
Kubernetes
     │
     ▼
Operations
```

项目真正希望向面试官传递的是：

> 我不仅能够完成 API 和数据库 CRUD，还理解一个后端服务在真实生产环境中会面对的并发、缓存、一致性、故障、性能、部署和运维问题。

同时：

> 我能够从一个简单可工作的系统开始，根据实际问题逐步引入 Redis、Kafka、gRPC、微服务、Kubernetes 和可观测体系，而不是为了堆砌技术而增加系统复杂度。

最终衡量这个项目质量的问题不是：

> 使用了多少框架？

而是：

> 每一个架构决定能否解释？

> 每一个性能优化能否测量？

> 每一个故障场景能否测试？

> 每一个分布式问题是否真正理解？

如果这些问题都能回答，那么 FlashFlow 就不只是一个 GitHub Demo，而是一套可以直接用于后端实习面试讨论的完整工程项目。

---

# License

本项目主要用于：

```text
后端工程学习

个人作品集

技术实践

面试准备
```
