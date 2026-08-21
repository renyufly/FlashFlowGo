POWERSHELL ?= powershell
GO := $(shell $(POWERSHELL) -NoProfile -Command "if (Test-Path '.tools/go/bin/go.exe') { '.tools/go/bin/go.exe' } else { 'go' }")
GOLANGCI_LINT := $(shell $(POWERSHELL) -NoProfile -Command "if (Test-Path '.tools/bin/golangci-lint.exe') { '.tools/bin/golangci-lint.exe' } else { 'golangci-lint' }")
MIGRATE := $(shell $(POWERSHELL) -NoProfile -Command "if (Test-Path '.tools/bin/migrate.exe') { '.tools/bin/migrate.exe' } else { 'migrate' }")
BUF := $(shell $(POWERSHELL) -NoProfile -Command "if (Test-Path '.tools/bin/buf.exe') { '.tools/bin/buf.exe' } else { 'buf' }")
K6 := $(shell $(POWERSHELL) -NoProfile -Command "if (Test-Path '.tools/bin/k6.exe') { '.tools/bin/k6.exe' } else { 'k6' }")

.PHONY: dev run build test test-race test-integration lint generate migrate-up migrate-down migrate-status docker-build docker-up docker-down docker-logs load-test proto tools tools-check

dev run:
	$(GO) run ./cmd/api

build:
	$(POWERSHELL) -NoProfile -Command "New-Item -ItemType Directory -Force bin | Out-Null"
	$(GO) build -trimpath -o bin/flashflow-api.exe ./cmd/api

test:
	$(GO) test ./...

test-race:
	$(POWERSHELL) -NoProfile -ExecutionPolicy Bypass -File scripts/test-race.ps1

test-integration:
	$(GO) test -tags=integration ./...

lint:
	$(POWERSHELL) -NoProfile -ExecutionPolicy Bypass -File scripts/check-format.ps1
	$(GO) vet ./...
	$(POWERSHELL) -NoProfile -ExecutionPolicy Bypass -File scripts/run-lint.ps1

generate:
	$(GO) generate ./...
	$(GO) mod tidy

migrate-up:
	$(MIGRATE) -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	$(MIGRATE) -path migrations -database "$(DATABASE_URL)" down 1

migrate-status:
	$(MIGRATE) -path migrations -database "$(DATABASE_URL)" version

docker-build:
	docker compose --profile postgres build

docker-up:
	docker compose --profile postgres up -d --wait postgres

docker-down:
	docker compose --profile postgres down

docker-logs:
	docker compose --profile postgres logs postgres

load-test:
	$(K6) run loadtest/smoke.js

proto:
	$(BUF) lint
	$(BUF) generate

tools:
	$(POWERSHELL) -NoProfile -ExecutionPolicy Bypass -File scripts/bootstrap-tools.ps1

tools-check:
	$(POWERSHELL) -NoProfile -ExecutionPolicy Bypass -File scripts/check-tools.ps1
