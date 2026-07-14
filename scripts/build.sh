#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

out="${1:-bin/ozsh}"
mkdir -p "$(dirname "$out")"

echo "[build] building $out"
go build -trimpath -buildvcs=false -ldflags="-s -w" -o "$out" ./cmd/ozsh

echo "[build] verifying binary"
"$out" version >/dev/null

echo "[build] ok"
