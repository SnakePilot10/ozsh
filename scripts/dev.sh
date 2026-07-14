#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "[dev] building ozsh"
go build -buildvcs=false -o ./bin/ozsh ./cmd/ozsh

echo "[dev] running doctor with isolated HOME"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
mkdir -p "$TMP/home"
printf 'export EDITOR=vim\n' > "$TMP/home/.zshrc"
HOME="$TMP/home" ./bin/ozsh doctor || true

echo "[dev] binary ready: ./bin/ozsh"
