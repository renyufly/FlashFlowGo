# Toolchain baseline

`.tool-versions` is the single version manifest. Phase 0 requires Go and golangci-lint; `scripts/bootstrap-tools.ps1` installs checksum-verified Windows binaries in ignored `.tools/`. It does not mutate global PATH.

| Tool | Pinned | First required | Isolated execution |
| --- | --- | --- | --- |
| Go | 1.26.6 | Phase 0 | `.tools/go/bin/go.exe` |
| golangci-lint | 2.12.2 | Phase 0 | `.tools/bin/golangci-lint.exe` |
| PostgreSQL | 17.6 | Phase 0 | `postgres:17.6-alpine` Compose image |
| Node / pnpm | 24.11.1 / 10.17.1 | Phase 10 | Corepack with project `packageManager` field when `web/` is created |
| Redis | 8.2.1 | Phase 4 | Fixed Compose image |
| Kafka | 4.1.0 | Phase 6 | Fixed Compose image |
| sqlc | 1.31.1 | Phase 1 | Release binary under `.tools/bin` or fixed container |
| golang-migrate | 4.19.0 | Phase 1 | Release binary under `.tools/bin` |
| buf / protoc | 1.72.0 / 32.1 | Phase 7 | Release binaries under `.tools/bin` |
| k6 | 1.3.0 | Phase 4 | Release binary under `.tools/bin` or fixed container |

Run `make tools-check` to print expected and detected versions. Missing later-phase tools produce a non-zero result by design, making incomplete setup explicit. When a phase activates a tool, extend the checksum-verified bootstrap before marking that task complete; never use floating `latest` downloads.

On Windows without a compatible 64-bit C compiler, race tests use the fixed `golang:1.26.6-bookworm` container. Tool upgrades must update `.tool-versions`, bootstrap/check scripts, Compose references, documentation, and an ADR or progress rationale together.
