# Contributing

FlashFlow evolves in the order defined by `plan.md`. Read `Idea.md` and `plan.md`, keep changes within the current phase, and record verified progress in `step.md`.

Run `make build`, `make test`, `make test-race`, and `make lint` before submitting. Integration, migration, protocol, and load commands must also pass when their phase makes them applicable.

Go uses `gofmt`/`goimports`; other text uses the root EditorConfig and Prettier defaults. Generated code stays separate and is never manually edited. The repository MIT `LICENSE` applies to all source, so per-file copyright boilerplate is intentionally omitted.

Never commit `.env`, credentials, tokens, private payloads, local toolchains, or benchmark claims without raw measured evidence.

