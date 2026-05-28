# inference-manager

Go HTTP service for dynamic switching of inference backends (vLLM for LLMs, diffusers for images) on DGX Spark (ARM64, Ubuntu).
Manages docker compose containers for individual backends via a REST API.

## Quickstart

```bash
# Build for linux/arm64
make build

# Run locally (uses config/models.yaml by default)
INFERENCE_CONFIG=config/models.yaml ./bin/inference-manager

# Or with go run
INFERENCE_CONFIG=config/models.yaml go run ./cmd/manager
```

## Configuration

Set `INFERENCE_CONFIG` env var to the path of your `models.yaml` (default: `config/models.yaml`).

```yaml
models:
  llm-dev:
    compose_file: /opt/inference/llm-dev/docker-compose.yml
    health_url: http://localhost:8001/v1/health/ready
    alias: llm-dev
  llm-lab:
    compose_file: /opt/inference/llm-lab/docker-compose.yml
    health_url: http://localhost:8002/v1/health/ready
    alias: llm-lab
```

## Environment Variables

| Variable              | Default                                      | Description                    |
|-----------------------|----------------------------------------------|--------------------------------|
| `INFERENCE_CONFIG`    | `config/models.yaml`                         | Path to models YAML config     |
| `INFERENCE_STATE`     | `/var/lib/inference-manager/state.json`      | Path to state persistence file |
| `INFERENCE_ADDR`      | `:8080`                                      | HTTP listen address            |

## API

### `POST /activate?model=<alias>`
Stops the current backend (if running), starts the requested one, and waits for its health check (timeout 300 s, retry every 5 s). On health timeout, attempts to restore the previous backend and returns 503.

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
Stops all backends (for maintenance). Returns 200.

## Deploy (systemd)

```bash
# Copy binary
sudo cp bin/inference-manager /usr/local/bin/inference-manager

# Copy config
sudo mkdir -p /opt/inference-manager/config
sudo cp config/models.yaml /opt/inference-manager/config/

# Install and start service
sudo cp deploy/inference-manager.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now inference-manager

# Check logs
sudo journalctl -u inference-manager -f
```

## LiteLLM Hook

Configure LiteLLM to call `deploy/litellm-hook.sh` as a `pre_call_hook` to activate the
target backend on demand before routing:

```bash
export INFERENCE_MANAGER_URL=http://localhost:8080
export MODEL_ALIAS=llm-dev
bash deploy/litellm-hook.sh
```

## Tests

```bash
make test
```
