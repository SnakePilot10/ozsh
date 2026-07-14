#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

fast=0
if [[ "${1:-}" == "--fast" ]]; then
  fast=1
fi

coverage_file="${COVERAGE_FILE:-coverage.out}"
coverage_min="${COVERAGE_MIN:-70}"

echo "[test] unit tests with coverage"
go test -coverprofile="$coverage_file" ./...
go tool cover -func="$coverage_file" | awk -v min="$coverage_min" '/^total:/ { if ($3 + 0 < min) { print "coverage " $3 " is below " min "%"; exit 1 } }'

if [[ "$fast" -eq 0 ]]; then
  echo "[test] race tests"
  go test -race ./...

  echo "[test] smoke tests"
  scripts/release-smoke.sh
  scripts/install-smoke.sh
fi

echo "[test] ok"
