#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "[deploy:staging] starting"

if [[ -n "${DEPLOY_STAGING_COMMAND:-}" ]]; then
  echo "[deploy:staging] running DEPLOY_STAGING_COMMAND"
  bash -lc "$DEPLOY_STAGING_COMMAND"
elif [[ -n "${DEPLOY_TARGET_STAGING:-}" ]]; then
  echo "[deploy:staging] target configured: $DEPLOY_TARGET_STAGING"
  echo "[deploy:staging] no deploy command configured; skipping safely"
else
  echo "[deploy:staging] no staging target configured; skipping safely"
fi

echo "[deploy:staging] ok"
