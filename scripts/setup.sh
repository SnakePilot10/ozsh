#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "[setup] checking Go"
if ! command -v go >/dev/null 2>&1; then
  echo "[setup] Go 1.25+ is required" >&2
  exit 1
fi

echo "[setup] downloading Go modules"
go mod download

echo "[setup] installing Go quality tools"
toolchain="go1.25.12"
if [[ "$(go env GOOS)" == "android" ]]; then
  # Go does not publish downloadable toolchains for Android/Termux.
  toolchain="local"
fi
GOTOOLCHAIN="$toolchain" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0
GOTOOLCHAIN="$toolchain" go install golang.org/x/vuln/cmd/govulncheck@v1.6.0

if command -v pre-commit >/dev/null 2>&1; then
  echo "[setup] installing pre-commit hooks"
  pre-commit install
else
  echo "[setup] pre-commit not found; install it and run: pre-commit install"
fi

echo "[setup] ok"
