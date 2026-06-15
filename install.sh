#!/usr/bin/env bash
set -euo pipefail

REPO="https://github.com/snakepilot10/ozsh"
INSTALL_DIR="${HOME}/.local/share/ozsh"
BIN_DIR="${HOME}/.local/bin"

if [[ -n "${TERMUX_VERSION:-}" ]]; then
    BIN_DIR="${PREFIX}/bin"
    echo "[*] Termux detected"
    if command -v pkg &> /dev/null; then
        missing=()
        command -v go &> /dev/null || missing+=(golang)
        command -v zsh &> /dev/null || missing+=(zsh)
        command -v git &> /dev/null || missing+=(git)
        if [[ ${#missing[@]} -gt 0 ]]; then
            echo "[*] Installing Termux dependencies: ${missing[*]}"
            pkg install -y "${missing[@]}"
        fi
    fi
fi

echo "=== ozsh installer ==="
echo

if ! command -v go &> /dev/null; then
    echo "[✗] Go is not installed. Please install Go 1.24+ first."
    exit 1
fi

if ! command -v zsh &> /dev/null; then
    echo "[!] zsh not found. Install it before running 'ozsh apply'."
fi

if [[ -d "$INSTALL_DIR" ]]; then
    echo "[*] Updating existing installation..."
    cd "$INSTALL_DIR"
    git pull --ff-only
else
    echo "[*] Cloning ozsh..."
    git clone "$REPO" "$INSTALL_DIR"
    cd "$INSTALL_DIR"
fi

echo "[*] Building ozsh..."
go build -buildvcs=false -o ozsh ./cmd/ozsh

mkdir -p "$BIN_DIR"
cp ozsh "$BIN_DIR/ozsh"
chmod +x "$BIN_DIR/ozsh"

if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo "[!] $BIN_DIR is not in PATH. Add this to your .zshrc:"
    echo "    export PATH=\"\$PATH:$BIN_DIR\""
fi

echo
echo "[✓] ozsh installed to $BIN_DIR/ozsh"
echo
echo "Next steps:"
echo "  ozsh doctor     # Check environment"
echo "  ozsh preview    # See prompt preview"
echo "  ozsh apply      # Generate and apply prompt"
if [[ -n "${TERMUX_VERSION:-}" ]]; then
    echo "  # Termux: ozsh does not run chsh; source the generated block from zsh"
fi
echo
echo "Config location: ~/.config/ozsh/config.toml"
