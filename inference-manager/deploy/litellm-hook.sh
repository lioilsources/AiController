#!/usr/bin/env bash
# Called by LiteLLM as a pre_call_hook to activate the target inference backend.
# Usage: INFERENCE_MANAGER_URL=http://localhost:8080 MODEL_ALIAS=llm-dev ./litellm-hook.sh
set -euo pipefail

INFERENCE_MANAGER_URL="${INFERENCE_MANAGER_URL:-http://localhost:8080}"
MODEL_ALIAS="${MODEL_ALIAS:-}"

if [[ -z "$MODEL_ALIAS" ]]; then
  echo "ERROR: MODEL_ALIAS not set" >&2
  exit 1
fi

echo "Activating model: $MODEL_ALIAS"
response=$(curl -sf -X POST "${INFERENCE_MANAGER_URL}/activate?model=${MODEL_ALIAS}" \
  -H "Content-Type: application/json" \
  --max-time 360)

echo "Manager response: $response"
