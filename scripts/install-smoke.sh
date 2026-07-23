#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

HOME_DIR="$TMP/home"
INSTALL_DIR="$TMP/install"
BIN_DIR="$TMP/bin"
FAKE_BIN="$TMP/fake-bin"
LOG="$TMP/install.log"
mkdir -p "$HOME_DIR" "$BIN_DIR" "$FAKE_BIN"
printf 'export EDITOR=vim\n' > "$HOME_DIR/.zshrc"

cat > "$FAKE_BIN/git" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" == "clone" ]]; then
  shift
  if [[ "${1:-}" == "--depth" ]]; then
    shift 2
  fi
  src="$1"
  dst="$2"
  if [[ "$src" == https://github.com/* ]]; then
    echo "unexpected network clone: $src" >&2
    exit 66
  fi
  mkdir -p "$dst"
  cp -a "$src"/. "$dst"/
  exit 0
fi
if [[ "$1" == "pull" ]]; then
  exit 0
fi
echo "unexpected git command: $*" >&2
exit 67
SH
chmod +x "$FAKE_BIN/git"

cat > "$FAKE_BIN/go" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" == "version" ]]; then
  echo "go version go1.25.12 linux/amd64"
  exit 0
fi
if [[ "$1" == "build" ]]; then
  out=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -o)
        out="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  if [[ -z "$out" ]]; then
    echo "missing go build -o" >&2
    exit 68
  fi
  cat > "$out" <<'BIN'
#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
  version)
    echo "0.2.0-dev"
    ;;
  doctor)
    echo "All critical checks passed."
    ;;
  apply)
    mkdir -p "$HOME/.config/ozsh"
    printf 'ozsh_prompt() {\n}\n' > "$HOME/.config/ozsh/omega.zsh"
    {
      printf '# BEGIN ozsh\n'
      printf 'source "$HOME/.config/ozsh/omega.zsh"\n'
      printf '# END ozsh\n'
    } >> "$HOME/.zshrc"
    echo "applied"
    ;;
  *)
    echo "fake ozsh"
    ;;
esac
BIN
  chmod +x "$out"
  exit 0
fi
echo "unexpected go command: $*" >&2
exit 69
SH
chmod +x "$FAKE_BIN/go"

cat > "$FAKE_BIN/zsh" <<'SH'
#!/usr/bin/env bash
exit 0
SH
chmod +x "$FAKE_BIN/zsh"

echo "[install-smoke] unattended install + apply"
HOME="$HOME_DIR" \
PATH="$FAKE_BIN:$PATH" \
OZSH_REPO="$ROOT" \
OZSH_INSTALL_DIR="$INSTALL_DIR" \
OZSH_BIN_DIR="$BIN_DIR" \
OZSH_YES=1 \
OZSH_APPLY=1 \
OZSH_UPDATE_PATH=1 \
"$ROOT/install.sh" > "$LOG"

test -x "$BIN_DIR/ozsh"
test -f "$INSTALL_DIR/install.sh"
grep -q 'All critical checks passed.' "$LOG"
grep -q 'ozsh apply complete' "$LOG"
grep -Fq "export PATH='$BIN_DIR':\"\$PATH\"" "$HOME_DIR/.zshrc"
grep -Fq "source \"\$HOME/.config/ozsh/omega.zsh\"" "$HOME_DIR/.zshrc"

echo "[install-smoke] dry run"
HOME="$HOME_DIR" \
PATH="$FAKE_BIN:$PATH" \
OZSH_REPO="$ROOT" \
OZSH_INSTALL_DIR="$TMP/dry-install" \
OZSH_BIN_DIR="$TMP/dry-bin" \
"$ROOT/install.sh" --dry-run > "$TMP/dry.log"

grep -q 'dry run' "$TMP/dry.log"
test ! -e "$TMP/dry-install"
test ! -e "$TMP/dry-bin/ozsh"

echo "[install-smoke] safe PATH quoting"
HOSTILE_BIN="$TMP/bin with ' quotes \" and \\$HOME \$(cmd) \`tick\` \\ slash"
EXPECTED_PATH_LINE="export PATH=$(printf "'%s'" "${HOSTILE_BIN//\'/\'\\\'\'}"):\"\$PATH\""
HOME="$HOME_DIR" \
PATH="$FAKE_BIN:$PATH" \
OZSH_REPO="$ROOT" \
OZSH_INSTALL_DIR="$TMP/quote-install" \
OZSH_BIN_DIR="$HOSTILE_BIN" \
"$ROOT/install.sh" --dry-run > "$TMP/quote.log"
grep -Fq "$EXPECTED_PATH_LINE" "$TMP/quote.log"

echo "[install-smoke] reject unsafe BIN_DIR"
if HOME="$HOME_DIR" PATH="$FAKE_BIN:$PATH" OZSH_BIN_DIR="relative/bin" "$ROOT/install.sh" --dry-run > "$TMP/bad-relative.log" 2>&1; then
  echo "relative OZSH_BIN_DIR was accepted" >&2
  exit 1
fi
if HOME="$HOME_DIR" PATH="$FAKE_BIN:$PATH" OZSH_BIN_DIR="$TMP/bad
bin" "$ROOT/install.sh" --dry-run > "$TMP/bad-control.log" 2>&1; then
  echo "control character OZSH_BIN_DIR was accepted" >&2
  exit 1
fi

echo "[install-smoke] ok"
