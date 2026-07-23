#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

BIN="$TMP/ozsh"
HOME_DIR="$TMP/home"
VERSION="${OZSH_VERSION:-dev}"
mkdir -p "$HOME_DIR"
printf 'export EDITOR=vim\n' > "$HOME_DIR/.zshrc"

echo "[smoke] build"
GOCACHE="${GOCACHE:-/tmp/ozsh-go-build}" \
GOMODCACHE="${GOMODCACHE:-/tmp/ozsh-gomod}" \
CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w -X main.version=$VERSION" -o "$BIN" "$ROOT/cmd/ozsh"

echo "[smoke] version"
actual_version="$("$BIN" version)"
if [[ "$actual_version" != "$VERSION" ]]; then
  echo "[smoke] expected version $VERSION, got $actual_version" >&2
  exit 1
fi

if [[ "$(go env GOOS)" == "linux" ]] && command -v ldd >/dev/null 2>&1; then
  echo "[smoke] static binary"
  if ldd "$BIN" >/dev/null 2>&1; then
    echo "[smoke] binary is dynamically linked" >&2
    exit 1
  fi
fi

echo "[smoke] preview"
HOME="$HOME_DIR" "$BIN" preview | grep -q '❯'

echo "[smoke] apply"
HOME="$HOME_DIR" "$BIN" apply >/dev/null
test -f "$HOME_DIR/.config/ozsh/omega.zsh"
grep -q 'ozsh_prompt()' "$HOME_DIR/.config/ozsh/omega.zsh"
grep -Fq "source \"\$HOME/.config/ozsh/omega.zsh\"" "$HOME_DIR/.zshrc"

if command -v zsh >/dev/null 2>&1; then
  echo "[smoke] generated zsh syntax"
  HOME="$HOME_DIR" zsh -n "$HOME_DIR/.config/ozsh/omega.zsh"
fi

echo "[smoke] reset"
HOME="$HOME_DIR" "$BIN" reset >/dev/null
if grep -q '# >>> ozsh >>>' "$HOME_DIR/.zshrc"; then
  echo "[smoke] reset left managed block in .zshrc" >&2
  exit 1
fi

echo "[smoke] ok"
