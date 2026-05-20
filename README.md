# nim-model-manager

Go HTTP service for dynamic switching of NVIDIA NIM models on DGX Spark (ARM64, Ubuntu).
Manages docker compose containers for individual LLM models via a REST API.

## Quickstart

```bash
# Build for linux/arm64
make build

# Run locally (uses config/models.yaml by default)
NIM_CONFIG=config/models.yaml ./bin/nim-manager

# Or with go run
NIM_CONFIG=config/models.yaml go run ./cmd/manager
```

## Configuration

Set `NIM_CONFIG` env var to the path of your `models.yaml` (default: `config/models.yaml`).

```yaml
models:
  llm-dev:
    compose_file: /opt/nim/llm-dev/docker-compose.yml
    health_url: http://localhost:8001/v1/health/ready
    alias: llm-dev
  llm-lab:
    compose_file: /opt/nim/llm-lab/docker-compose.yml
    health_url: http://localhost:8002/v1/health/ready
    alias: llm-lab
```

## Environment Variables

| Variable      | Default                             | Description                    |
|---------------|-------------------------------------|--------------------------------|
| `NIM_CONFIG`  | `config/models.yaml`                | Path to models YAML config     |
| `NIM_STATE`   | `/var/lib/nim-manager/state.json`   | Path to state persistence file |
| `NIM_ADDR`    | `:8080`                             | HTTP listen address            |

## API

### `POST /activate?model=<alias>`
Stops the current model (if running), starts the requested model, and waits for its health check (timeout 300 s, retry every 5 s). On health timeout, attempts to restore the previous model and returns 503.

**Response 200:**
```json
{"active": "llm-dev", "since": "2025-01-01T12:00:00Z"}
```

**Response 5xx:**
```json
{"error": "health check failed: ..."}
```

### `GET /status`
Returns current state.

```json
{"active": "llm-dev", "healthy": true, "since": "2025-01-01T12:00:00Z"}
```

### `GET /health`
Liveness probe. Always returns 200.

### `POST /deactivate`
Stops all models (for maintenance). Returns 200.

## Deploy (systemd)

```bash
# Copy binary
sudo cp bin/nim-manager /usr/local/bin/nim-manager

# Copy config
sudo mkdir -p /opt/nim-manager/config
sudo cp config/models.yaml /opt/nim-manager/config/

# Install and start service
sudo cp deploy/nim-manager.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now nim-manager

# Check logs
sudo journalctl -u nim-manager -f
```

## LiteLLM Hook

Configure LiteLLM to call `deploy/litellm-hook.sh` as a `pre_call_hook` to activate the
target model on demand before routing:

```bash
export NIM_MANAGER_URL=http://localhost:8080
export MODEL_ALIAS=llm-dev
bash deploy/litellm-hook.sh
```

## Tests

```bash
make test
```
