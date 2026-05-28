# Plan: ai-stack

## Context

New project `/home/ol1n/ai-stack` that extends the existing `llm-stack` patterns into a clean, modular AI inference platform. Three modules: LLM serving (vLLM + LiteLLM), image generation (FLUX + Qwen image editing), and OCR (Qwen2.5-VL). Docker-compose infra is strictly separated from FastAPI service code. No DB, no file storage, no ComfyUI. Pure stateless JSON APIs.

---

## Project Structure

```
/home/ol1n/ai-stack/
├── docker-compose.yml              # Root: Caddy + include: modules + shared network
├── docker-compose.llm.yaml         # vLLM lab + vLLM dev + LiteLLM
├── docker-compose.image.yaml       # image-api service
├── docker-compose.ocr.yaml         # ocr-api service
├── Caddyfile                       # Route: /v1/images/* /v1/ocr* /v1/*
├── litellm_config.yaml             # Model aliases
├── .env.example
├── .gitignore
├── services/
│   ├── image-api/
│   │   ├── Dockerfile              # FROM nvcr.io/nvidia/pytorch:25.08-py3
│   │   ├── requirements.txt
│   │   └── main.py                 # FLUX text2img + Qwen img2img endpoints
│   └── ocr-api/
│       ├── Dockerfile
│       ├── requirements.txt
│       └── main.py                 # Qwen2.5-VL OCR endpoint
├── scripts/
│   ├── download_flux.sh            # ~24 GB, gated HF model
│   └── download_qwen_vl.sh         # ~17 GB, OCR model
└── cache/                          # .gitignored
    ├── lab/     ← reuse llm-stack/cache/lab (set via .env)
    ├── dev/     ← reuse llm-stack/cache/dev (set via .env)
    ├── flux/    ← download needed
    ├── qwen-image/ ← ALREADY EXISTS at llm-stack/cache/img (QwenImageEditPlusPipeline)
    └── qwen-vl/ ← download needed
```

---

## Services & Ports

| Service     | Port | Image                                | Purpose                          |
|-------------|------|--------------------------------------|----------------------------------|
| lab         | 8000 | nvcr.io/nvidia/vllm:$VLLM_VERSION    | GPT-OSS-120B (LLM lab)           |
| dev         | 8001 | vllm/vllm-openai:gemma4-cu130        | Gemma-4-31B-IT-NVFP4 (LLM dev)  |
| litellm     | 4000 | ghcr.io/berriai/litellm:main-latest  | OpenAI-compat gateway            |
| image-api   | 8002 | built from services/image-api        | text2img + img2img               |
| ocr-api     | 8003 | built from services/ocr-api          | OCR with Qwen2.5-VL              |
| caddy       | 8080 | caddy:2-alpine                       | Public API gateway               |

---

## Docker Compose Separation

`docker-compose.yml` is the single entrypoint. Uses Docker Compose v5 `include:` to pull in module files. Owns the `internal` bridge network and Caddy.

```yaml
# docker-compose.yml (skeleton)
include:
  - docker-compose.llm.yaml
  - docker-compose.image.yaml
  - docker-compose.ocr.yaml
services:
  caddy:
    image: caddy:2-alpine
    ports: ["127.0.0.1:8080:8080"]
    volumes: [./Caddyfile:/etc/caddy/Caddyfile:ro]
    networks: [internal]
    depends_on:
      litellm: {condition: service_healthy}
      image-api: {condition: service_healthy}
      ocr-api: {condition: service_healthy}
networks:
  internal:
    driver: bridge
```

Module files each define their own services and reference `networks: [internal]` without re-declaring it.

Individual module bring-up: `docker compose -f docker-compose.ocr.yaml up -d`

---

## FastAPI Services

### image-api (`services/image-api/main.py`)

Two pipelines loaded at startup via lifespan. Controlled by `LOAD_FLUX` / `LOAD_QWEN_IMAGE` env vars (both default `1`) to allow selective loading on memory-constrained deployments.

**Endpoints:**
- `POST /v1/images/generations` — text2img via `FluxPipeline`, OpenAI-compatible shape
- `POST /v1/images/edits` — img2img via `QwenImageEditPlusPipeline`, takes `image` (base64) + `prompt`
- `GET /health` — `{status, flux_loaded, qwen_image_loaded}`

**Input/Output:** pure JSON, images as base64-encoded PNG strings. No file writes anywhere.

**Key:** `QwenImageEditPlusPipeline` requires `diffusers @ git+https://github.com/huggingface/diffusers.git`
(not on PyPI — `model_index.json` in `llm-stack/cache/img` shows `_diffusers_version: 0.36.0.dev0`).

### ocr-api (`services/ocr-api/main.py`)

Model: `Qwen2.5-VL-7B-Instruct` via `Qwen2_5_VLForConditionalGeneration` + `AutoProcessor`.

**Endpoints:**
- `POST /v1/ocr` — `{image: str (base64 or https URL), language_hint?: str, max_new_tokens?: int}` → `{text: str, model: str}`
- `GET /health`

System prompt: extract text verbatim, preserve layout, no commentary. `language_hint` appended to prompt if provided.

---

## Caddyfile Routing

```caddyfile
:8080 {
    header Access-Control-Allow-Origin "*"
    header Access-Control-Allow-Methods "GET, POST, OPTIONS"
    header Access-Control-Allow-Headers "Content-Type, Authorization"
    @options method OPTIONS
    respond @options 204

    handle /v1/images/* {
        reverse_proxy image-api:8002 {
            transport http { read_timeout 600s; write_timeout 600s }
        }
    }
    handle /v1/ocr* {
        reverse_proxy ocr-api:8003 {
            transport http { read_timeout 120s; write_timeout 120s }
        }
    }
    reverse_proxy * litellm:4000 {
        transport http { read_timeout 300s; write_timeout 300s }
        flush_interval -1
    }
}
```

---

## Model Download List

| Model                      | HF Repo                           | Size   | Status                                                |
|----------------------------|-----------------------------------|--------|-------------------------------------------------------|
| GPT-OSS-120B (lab)         | `openai/gpt-oss-120b`             | ~120 GB | **Already in** `llm-stack/cache/lab` — set via `.env` |
| Gemma-4-31B-IT-NVFP4 (dev) | `nvidia/Gemma-4-31B-IT-NVFP4`    | ~35 GB  | **Already in** `llm-stack/cache/dev` — set via `.env` |
| QwenImageEditPlusPipeline  | `Qwen/Qwen-Image-Edit-2511`       | ~54 GB  | **Already in** `llm-stack/cache/img` — set `CACHE_QWEN_IMAGE` there |
| FLUX.1-dev                 | `black-forest-labs/FLUX.1-dev`    | ~24 GB  | **Needs download** (gated) — `bash scripts/download_flux.sh` |
| Qwen2.5-VL-7B-Instruct     | `Qwen/Qwen2.5-VL-7B-Instruct`    | ~17 GB  | **Needs download** — `bash scripts/download_qwen_vl.sh` |

**GPU memory note (GB10, ~128 GB unified):** FLUX + QwenImage + QwenVL + Gemma ≈ ~130 GB total.
Use `LOAD_FLUX=0` or `LOAD_QWEN_IMAGE=0` in `.env` to drop a pipeline if needed.

---

## .env Variables

```bash
HF_TOKEN=hf_...
VLLM_VERSION=26.03.post1-py3
HF_MODEL_LAB=openai/gpt-oss-120b
HF_MODEL_DEV=nvidia/Gemma-4-31B-IT-NVFP4
CACHE_LAB=/home/ol1n/llm-stack/cache/lab
CACHE_DEV=/home/ol1n/llm-stack/cache/dev
CACHE_FLUX=/home/ol1n/ai-stack/cache/flux
CACHE_QWEN_IMAGE=/home/ol1n/llm-stack/cache/img   # already downloaded!
CACHE_QWEN_VL=/home/ol1n/ai-stack/cache/qwen-vl
FLUX_MODEL_ID=black-forest-labs/FLUX.1-dev
QWEN_IMAGE_MODEL_ID=Qwen/Qwen-Image-Edit-2511
OCR_MODEL_ID=Qwen/Qwen2.5-VL-7B-Instruct
FLUX_FP8=0
LOAD_FLUX=1
LOAD_QWEN_IMAGE=1
```

---

## Implementation Steps

1. `mkdir -p /home/ol1n/ai-stack/{services/image-api,services/ocr-api,scripts,cache/{lab,dev,flux,qwen-image,qwen-vl}}`
2. Write `docker-compose.yml`, `docker-compose.llm.yaml`, `docker-compose.image.yaml`, `docker-compose.ocr.yaml`
3. Write `Caddyfile`, `litellm_config.yaml`
4. Write `services/image-api/{Dockerfile,requirements.txt,main.py}`
5. Write `services/ocr-api/{Dockerfile,requirements.txt,main.py}`
6. Write `scripts/download_flux.sh`, `scripts/download_qwen_vl.sh`; `chmod +x scripts/*.sh`
7. Write `.env.example`, `.gitignore`
8. `cp .env.example .env` — set `CACHE_QWEN_IMAGE=/home/ol1n/llm-stack/cache/img` and fill `HF_TOKEN`
9. Download: `bash scripts/download_qwen_vl.sh` (~17 GB), then `bash scripts/download_flux.sh` (~24 GB)
10. `docker compose build`
11. Smoke-test OCR: `docker compose -f docker-compose.ocr.yaml up -d && curl http://localhost:8003/health`
12. Smoke-test image: `docker compose -f docker-compose.image.yaml up -d && curl http://localhost:8002/health`
13. Full stack: `docker compose up -d`
14. Validate: `curl http://localhost:8080/v1/models`, test each endpoint through Caddy

---

## Verification Checklist

- `GET http://localhost:8002/health` → `{flux_loaded: true, qwen_image_loaded: true}`
- `GET http://localhost:8003/health` → `{model_loaded: true}`
- `POST http://localhost:8080/v1/images/generations` `{"prompt":"a red apple"}` → base64 PNG
- `POST http://localhost:8080/v1/images/edits` `{"image":"<b64>","prompt":"make it blue"}` → edited image
- `POST http://localhost:8080/v1/ocr` `{"image":"<b64 of text scan>"}` → extracted text
- `POST http://localhost:8080/v1/chat/completions` `{"model":"llm-dev","messages":[...]}` → LLM response
