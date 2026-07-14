#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "[deploy:production] starting"

if [[ "${OZSH_ASSUME_YES:-0}" != "1" ]]; then
  read -r -p "Deploy to production? Type 'deploy production' to continue: " answer
  if [[ "$answer" != "deploy production" ]]; then
    echo "[deploy:production] cancelled"
    exit 1
  fi
fi

if [[ -n "${DEPLOY_PROD_COMMAND:-}" ]]; then
  echo "[deploy:production] running DEPLOY_PROD_COMMAND"
  bash -lc "$DEPLOY_PROD_COMMAND"
elif [[ -n "${DEPLOY_TARGET_PRODUCTION:-}" ]]; then
  echo "[deploy:production] target configured: $DEPLOY_TARGET_PRODUCTION"
  echo "[deploy:production] no deploy command configured; skipping safely"
else
  echo "[deploy:production] no production target configured; skipping safely"
fi

echo "[deploy:production] ok"
