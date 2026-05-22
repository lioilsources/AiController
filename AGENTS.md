# AGENTS.md

Guidance for AI coding agents (Claude Code et al.) working in this repository.

## What this is

`nim-model-manager` — a small Go HTTP service that dynamically switches which
NVIDIA NIM LLM model is running on a DGX Spark box (ARM64, Ubuntu). Only one
model runs at a time; the service stops the current model's docker compose
stack and brings up the requested one, waiting for its health check before
declaring the switch successful. Intended to be driven on demand (e.g. by a
LiteLLM `pre_call_hook`).

Module path: `github.com/ol1n/nim-model-manager` (Go 1.22).

## Layout

| Path                          | Responsibility                                              |
|-------------------------------|-------------------------------------------------------------|
| `cmd/manager/main.go`         | Entrypoint: load config, verify active model on boot, serve |
| `internal/api/handler.go`     | HTTP handlers; serializes activate/deactivate with a mutex  |
| `internal/compose/runner.go`  | Wraps `docker compose -f <file> up/down`                    |
| `internal/config/config.go`   | Loads `models.yaml` into `Config`                           |
| `internal/health/checker.go`  | `WaitHealthy` polls a URL until 200 or timeout              |
| `internal/state/state.go`     | Persists active model to JSON (atomic write via tmp+rename) |
| `config/models.yaml`          | Model registry (alias -> compose file + health URL)         |
| `deploy/`                     | systemd unit + LiteLLM activation hook                      |

## Architecture notes

- The `Runner` interface in `internal/compose` is the seam used for testing —
  `internal/api/handler_test.go` injects a fake runner. Keep handler logic
  testable through that interface; don't call `docker` directly from `api`.
- `activate` is the core flow: stop previous (if different) → up requested →
  `WaitHealthy` → persist state. On failure after starting, it calls
  `tryRestore` to bring the previous model back. Preserve this rollback
  behavior when editing.
- All state mutations go through `state.Store` and are guarded by
  `Handler.mu`. A single in-flight activation/deactivation at a time is an
  intentional invariant.
- Health timeout is 300s, polled every 5s (`health.Default*`). Boot-time
  verification in `main.go` uses a shorter 10s window.
- Logging is structured `slog` JSON to stdout. Match that style.

## Configuration (env vars)

| Variable     | Default                           | Purpose                  |
|--------------|-----------------------------------|--------------------------|
| `NIM_CONFIG` | `config/models.yaml`              | Path to models YAML      |
| `NIM_STATE`  | `/var/lib/nim-manager/state.json` | State persistence file   |
| `NIM_ADDR`   | `:8080`                           | HTTP listen address      |

## HTTP API

- `POST /activate?model=<alias>` — switch to model, wait for health.
- `GET  /status` — current `State` JSON.
- `GET  /health` — liveness, always 200.
- `POST /deactivate` — stop all model stacks.

## Build / test / lint

```bash
make build   # CGO_ENABLED=0 GOOS=linux GOARCH=arm64 -> bin/nim-manager
make test    # go test ./...
make lint    # golangci-lint run ./...
```

Run locally: `NIM_CONFIG=config/models.yaml go run ./cmd/manager`

The build target cross-compiles for `linux/arm64` (the DGX Spark target).
Run `make test` before pushing; add table-driven tests next to the package
they cover (see existing `*_test.go`).

## Conventions

- Standard library first; the only third-party dep is `gopkg.in/yaml.v3`.
  Avoid adding dependencies without good reason.
- Wrap errors with `fmt.Errorf("...: %w", err)` and surface them in handlers
  via `writeError` as JSON.
- Keep the deploy artifacts (`deploy/nim-manager.service`, env var defaults,
  README API docs) in sync when you change config keys or the API surface.
