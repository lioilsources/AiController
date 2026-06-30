# AiController — CLAUDE.md

## Overview

Go HTTP service that manages dynamic model switching in AiStack. Activates/deactivates Docker Compose services (LLM containers) via REST API, tracks state, and health-checks containers. Runs as part of AiStack deploy as `controller` container.

Code lives in `inference-manager/` subdirectory (separate Go module).

## Build & Run

```bash
cd inference-manager

# Build
go build -o bin/manager ./cmd/manager

# Run locally
INFERENCE_CONFIG=config/models.yaml \
INFERENCE_STATE=/tmp/state.json \
./bin/manager

# Tests
go test ./...
go vet ./...
```

In production: started via AiStack `make up` (docker-compose.yml), port 8090.

## API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/activate?model=<alias>` | Start model container, wait for health |
| POST | `/deactivate?model=<alias>` | Stop model container |
| GET | `/status` | List all models and their current state |
| GET | `/health` | Liveness probe |

## Configuration

`config/models.yaml`:
```yaml
dev:
  compose_file: /home/ol1n/deploy/AiStack/deploy/docker-compose.llm.yaml
  health_url: http://dev:8000/v1/models
  alias: dev
```

- `compose_file` must be **absolute** path identical on host and inside controller container (docker.sock uses host paths)
- `health_url` is polled until container is ready
- `alias` = model name used in `?model=` query param

## Architecture

```
inference-manager/
├── cmd/manager/main.go         # HTTP server entry point
└── internal/
    ├── api/handler.go          # POST /activate, /deactivate, GET /status, /health
    ├── compose/runner.go       # docker-compose up -d / down via exec
    ├── config/config.go        # YAML config loader
    ├── health/checker.go       # HTTP health poll with timeout
    └── state/state.go          # JSON file persistence of active model
```

## Conventions

- State is persisted to JSON file (`INFERENCE_STATE` env var) — survives restarts
- Only one model can be active at a time; `activate` deactivates current first
- Logging: structured JSON via `log/slog`
- No external dependencies beyond `gopkg.in/yaml.v3`
