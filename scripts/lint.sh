#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

check_only=0
if [[ "${1:-}" == "--check" ]]; then
  check_only=1
fi

echo "[lint] gofmt"
if [[ "$check_only" -eq 1 ]]; then
  unformatted="$(gofmt -l .)"
  if [[ -n "$unformatted" ]]; then
    echo "$unformatted" >&2
    echo "[lint] files need gofmt" >&2
    exit 1
  fi
else
  gofmt -w .
fi

echo "[lint] go vet"
go vet ./...

echo "[lint] shell syntax"
bash -n install.sh scripts/*.sh

if command -v shellcheck >/dev/null 2>&1; then
  echo "[lint] shellcheck"
  shellcheck install.sh scripts/*.sh
else
  echo "[lint] shellcheck not found; install shellcheck to lint shell scripts"
fi

if command -v golangci-lint >/dev/null 2>&1; then
  echo "[lint] golangci-lint"
  golangci-lint run
else
  echo "[lint] golangci-lint not found; run scripts/setup.sh to install it"
fi

echo "[lint] ok"
