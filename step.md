# FlashFlow 实现进度

## 当前状态

| 字段 | 当前值 |
| --- | --- |
| 最后更新时间 | 2026-08-21 |
| 已完成阶段 | Phase 0 — 仓库与开发基线 |
| 已完成任务 | 7 / 97 |
| 下一任务 | 1.1 领域词汇与状态机 |
| 全局环境变更 | 无 |

## Phase 0 实施记录

### 0.1 工具链与决策基线

- 根据已配置的 Git 远端确定 Go module path 为 `github.com/renyufly/FlashFlowGo`。
- 在 `.tool-versions` 中固定 Go、Node、pnpm、PostgreSQL、Redis、Kafka、sqlc、golang-migrate、buf、protoc、golangci-lint 和 k6 的版本。
- 新增 `docs/toolchain.md` 和 ADR 0001，记录模块化单体基线、暂缓引入的服务、版本管理策略及项目内工具链方案。
- Go 1.26.6 和 golangci-lint 2.12.2 仅下载到被 Git 忽略的 `.tools/`；`scripts/bootstrap-tools.ps1` 会对两个压缩包执行 checksum 校验。未修改全局 PATH、运行时或包管理器状态。
- 验证期间将 Go module、构建及 lint 缓存保存在被 Git 忽略的 `.cache/` 中。

### 0.2 仓库骨架

- 仅创建 Phase 0 中可实际运行的边界：`cmd/api`、`internal/platform/config` 和 `internal/platform/httpserver`。
- 添加仓库级忽略规则、换行规则、编辑器规则、`CONTRIBUTING.md` 以及 MIT License/版权头约定。
- 领域模块、workers、gRPC 服务、前端、migrations 和生成代码目录延后到各自计划阶段创建，避免占位 import 和过早拆分微服务。

### 0.3 配置契约

- 实现强类型的运行环境、HTTP timeout、PostgreSQL URL 和 Secret 配置，并在启动时校验。
- `DATABASE_URL` 与 `JWT_SECRET` 为必填项；生产环境要求更高强度的 JWT Secret。
- 实现可安全写入启动日志的配置摘要，对数据库凭证和 JWT Secret 进行脱敏。
- 单元测试覆盖默认值、必填项缺失、非法 duration、非法 URL scheme、生产环境 Secret 强度以及日志脱敏。

### 0.4 进程与 HTTP 骨架

- 添加 Chi router、`/healthz`、构建版本/commit/时间字段、稳定的 JSON 错误结构、request ID 接收与生成、panic recovery 和 handler timeout。
- 设置有界的 HTTP read/write/idle/shutdown timeout，并接入 SIGINT/SIGTERM 取消信号。
- 测试健康检查与错误响应契约、在途请求的优雅排空，并验证 server 运行 goroutine 可以正常退出。
- 进程 smoke test 返回 HTTP 200，响应包含 `status`、`version`、`commit`、`build_time` 和指定的 request ID；启动日志中仅出现脱敏后的 Secret。

### 0.5 开发命令

- 添加 `dev`、`run`、`build`、`test`、`test-race`、`test-integration`、`lint`、`generate`、`migrate-*`、`docker-*`、`load-test`、`proto`、`tools` 和 `tools-check` targets。
- 所有命令保留原始输出并传播非零退出码。若后续阶段所需工具或输入尚未安装，对应命令会明确失败，不会静默跳过。
- Windows 环境缺少兼容的 64 位 C 编译器时，race tests 会在固定版本的 `golang:1.26.6-bookworm` 容器中执行，从而避免安装全局编译器。

### 0.6 格式化与静态检查

- 添加不会修改源码的 gofmt 检查、go vet、包含 goimports 的 golangci-lint v2、仓库级 Prettier 默认规则，以及为后续前端预留的 ESLint flat config 基线。
- 将本地工具、缓存、构建产物、前端依赖和生成代码排除在版本控制及格式扫描之外。

### 0.7 本地依赖基线

- 添加 `postgres` Compose profile，使用 `postgres:17.6-alpine`，配置固定开发凭证、UTC 时区、项目命名 volume、仅 loopback 可访问的端口绑定及 `pg_isready` healthcheck。
- 通过 `POSTGRES_PORT` 支持配置宿主机端口。本机 5432 已被其他服务占用，因此验收使用 55432，未停止或修改原有服务。
- 已验证容器状态为 `healthy`，且 `pg_isready` 报告 PostgreSQL 正在接受连接。
- `docker-down` 不使用 `--volumes`；文档明确要求任何手动清理操作都必须先核对项目 volume 名称。

## 验收证据

验收环境：Windows、Go 1.26.6、Docker Engine 29.2.1。以下检查均已通过：

```text
make tools                    通过（项目内 Go 与 golangci-lint）
make build                    通过
make test                     通过
make test-integration         通过
make lint                     通过（gofmt、go vet、golangci-lint：0 issues）
make test-race                通过（固定版本 Linux Go 容器）
GET /healthz                  通过（HTTP 200，包含版本字段）
Compose PostgreSQL profile    通过（状态 healthy；pg_isready 接受连接）
```

当前没有发布任何 benchmark 或性能数据声明。Phase 1 尚未开始。
