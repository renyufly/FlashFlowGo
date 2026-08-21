# ADR 0001: Repository, toolchain, and initial architecture

- Status: Accepted
- Date: 2026-08-21

## Context

FlashFlow needs a reproducible starting point without paying distributed-system costs before domain boundaries and correctness are proven. The Git remote is `github.com/renyufly/FlashFlowGo`, and development must not require replacing machine-wide runtimes.

## Decision

- Use module path `github.com/renyufly/FlashFlowGo`.
- Start as one Go module and one `cmd/api` process. Keep shared platform concerns in `internal/platform`; introduce domain packages only with their business phase.
- Use Chi and the Go standard library (`net/http`, `slog`).
- Pin build/runtime tools in `.tool-versions` and exact container tags in Compose.
- Support a repository-local `.tools` directory, ignored by Git. Bootstrap scripts never modify global PATH or global package-manager state.
- Keep PostgreSQL as the only Phase 0 local dependency. Redis, Kafka, observability, gRPC, workers, and frontend remain deferred.

## Consequences

The first executable is small, fast to test, and has a clear upgrade path. Developers can use globally installed matching tools or project-local binaries. A clean checkout needs an explicit checksum-verified bootstrap step. Later upgrades are intentional rather than accidental `latest` pulls.

