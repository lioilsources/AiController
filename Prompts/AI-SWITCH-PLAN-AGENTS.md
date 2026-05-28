# AGENTS.md

## Projekt
Go HTTP service pro dynamické přepínání NVIDIA NIM modelů na DGX Spark (ARM64, Ubuntu).
Spravuje docker compose kontejnery pro jednotlivé LLM modely přes REST API.

## Jazyk a styl
- Veškerý kód v Go 1.22+
- Komentáře v angličtině, commit messages v angličtině
- Žádné externí frameworky pro HTTP (stdlib `net/http`)
- Logování přes `log/slog` (structured JSON)
- Chyby wrappovat s `fmt.Errorf("...: %w", err)`

## Architektura
- `cmd/manager/main.go` — vstupní bod, načte config, spustí HTTP server
- `internal/compose/runner.go` — volá `docker compose` jako subprocess
- `internal/health/checker.go` — HTTP GET s exponential backoff
- `internal/state/state.go` — čte/píše `/var/lib/nim-manager/state.json`
- `internal/api/handler.go` — REST handlery, volá compose + health + state

## Konfigurace
Načítá se z `config/models.yaml` (cesta přes env `NIM_CONFIG`).
Příklad struktury:
  models:
    llm-dev:
      compose_file: /opt/nim/llm-dev/docker-compose.yml
      health_url: http://localhost:8001/v1/health/ready
      alias: llm-dev
    llm-lab:
      compose_file: /opt/nim/llm-lab/docker-compose.yml
      health_url: http://localhost:8002/v1/health/ready
      alias: llm-lab

## API endpointy
POST /activate?model=<alias>
  - Zastaví aktuální model (pokud běží)
  - Spustí požadovaný model
  - Čeká na healthcheck (timeout 300s, retry každých 5s)
  - Vrátí 200 OK nebo 5xx s JSON chybou

GET /status
  - Vrátí JSON: {"active": "llm-dev", "healthy": true, "since": "<RFC3339>"}

GET /health
  - Liveness probe samotného manageru, vždy 200

POST /deactivate
  - Zastaví všechny modely (pro maintenance)

## Chování
- Souběžné requesty na /activate jsou serializovány přes mutex
- Stav se persistuje do state.json okamžitě po úspěšném swapu
- Po startu service se přečte state.json a ověří se, zda model skutečně běží
- Pokud healthcheck selže po 300s, vrátí se 503 a zkusí se restartovat předchozí model

## Testy
- Unit testy pro health/checker.go (mock HTTP server)
- Unit testy pro state/state.go (tmp soubor)
- Integration test pro api/handler.go (mock compose runner přes interface)
- Spouštět: go test ./...

## Build
- `make build` → binárka pro linux/arm64
- `make lint` → golangci-lint
- Nevyužívat CGO

## Co NEimplementovat
- Autentizaci (řeší Caddy reverse proxy před managerem)
- Scheduling / queue modelů
- Metriky (Prometheus) — nice to have, ale ne v MVP