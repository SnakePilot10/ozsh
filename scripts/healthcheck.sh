#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

BIN="${OZSH_BIN:-$ROOT/bin/ozsh}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

mkdir -p "$TMP/home"
printf 'export EDITOR=vim\n' > "$TMP/home/.zshrc"

if [[ ! -x "$BIN" ]]; then
  echo "[healthcheck] binary not found at $BIN; building temporary binary"
  mkdir -p "$ROOT/bin"
  go build -buildvcs=false -o "$BIN" ./cmd/ozsh
fi

echo "[healthcheck] version"
"$BIN" version >/dev/null

echo "[healthcheck] preview"
HOME="$TMP/home" "$BIN" preview >/dev/null

echo "[healthcheck] apply"
HOME="$TMP/home" "$BIN" apply >/dev/null
test -f "$TMP/home/.config/ozsh/omega.zsh"
grep -Fq "source \"\$HOME/.config/ozsh/omega.zsh\"" "$TMP/home/.zshrc"

if command -v zsh >/dev/null 2>&1; then
  echo "[healthcheck] generated zsh syntax"
  HOME="$TMP/home" zsh -n "$TMP/home/.config/ozsh/omega.zsh"
fi

echo "[healthcheck] ok"
