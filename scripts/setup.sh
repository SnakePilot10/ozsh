#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "[setup] checking Go"
if ! command -v go >/dev/null 2>&1; then
  echo "[setup] Go 1.24+ is required" >&2
  exit 1
fi

echo "[setup] downloading Go modules"
go mod download

echo "[setup] installing Go quality tools"
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
go install golang.org/x/vuln/cmd/govulncheck@latest

if command -v pre-commit >/dev/null 2>&1; then
  echo "[setup] installing pre-commit hooks"
  pre-commit install
else
  echo "[setup] pre-commit not found; install it and run: pre-commit install"
fi

echo "[setup] ok"
