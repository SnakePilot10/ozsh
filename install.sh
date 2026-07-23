#!/usr/bin/env bash
set -euo pipefail

REPO="${OZSH_REPO:-https://github.com/SnakePilot10/ozsh}"
INSTALL_DIR="${OZSH_INSTALL_DIR:-${HOME}/.local/share/ozsh}"
BIN_DIR="${OZSH_BIN_DIR:-${HOME}/.local/bin}"
YES="${OZSH_YES:-0}"
APPLY="${OZSH_APPLY:-0}"
UPDATE_PATH="${OZSH_UPDATE_PATH:-0}"
DRY_RUN=0
SKIP_BUILD=0

usage() {
    cat <<'EOF'
Usage: install.sh [--yes] [--apply] [--update-path] [--dry-run] [--skip-build]

Environment:
  OZSH_REPO         Git URL or local path to clone. Default: https://github.com/SnakePilot10/ozsh
  OZSH_INSTALL_DIR  Source checkout directory. Default: ~/.local/share/ozsh
  OZSH_BIN_DIR      Binary install directory. Default: ~/.local/bin
  OZSH_YES          Install missing dependencies when supported. Values: 1/true/yes
  OZSH_APPLY        Run ozsh apply after installing. Values: 1/true/yes
  OZSH_UPDATE_PATH  Add OZSH_BIN_DIR to ~/.zshrc when missing. Values: 1/true/yes
EOF
}

truthy() {
    case "${1:-}" in
        1|true|TRUE|yes|YES|y|Y|on|ON) return 0 ;;
        *) return 1 ;;
    esac
}

shell_single_quote() {
    local value=${1-}
    printf "'%s'" "${value//\'/\'\\\'\'}"
}

validate_bin_dir() {
    if [[ -z "$BIN_DIR" ]]; then
        echo "[x] OZSH_BIN_DIR must not be empty" >&2
        exit 1
    fi
    if [[ "$BIN_DIR" != /* ]]; then
        echo "[x] OZSH_BIN_DIR must be an absolute path: $BIN_DIR" >&2
        exit 1
    fi
    if [[ "$BIN_DIR" =~ [[:cntrl:]] ]]; then
        echo "[x] OZSH_BIN_DIR must not contain control characters" >&2
        exit 1
    fi
    if [[ -e "$BIN_DIR" && ! -d "$BIN_DIR" ]]; then
        echo "[x] OZSH_BIN_DIR exists but is not a directory: $BIN_DIR" >&2
        exit 1
    fi
}

run() {
    if [[ "$DRY_RUN" == "1" ]]; then
        printf '[dry run] %q' "$1"
        shift
        printf ' %q' "$@"
        printf '\n'
        return 0
    fi
    "$@"
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --yes|-y)
            YES=1
            ;;
        --apply)
            APPLY=1
            ;;
        --no-apply)
            APPLY=0
            ;;
        --update-path)
            UPDATE_PATH=1
            ;;
        --dry-run)
            DRY_RUN=1
            ;;
        --skip-build)
            SKIP_BUILD=1
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            echo "[x] unknown option: $1" >&2
            usage >&2
            exit 1
            ;;
    esac
    shift
done

if [[ -n "${TERMUX_VERSION:-}" ]]; then
    BIN_DIR="${OZSH_BIN_DIR:-${PREFIX}/bin}"
fi
validate_bin_dir

echo "=== ozsh installer ==="
echo
echo "repo:        $REPO"
echo "source dir:  $INSTALL_DIR"
echo "binary dir:  $BIN_DIR"
if [[ "$DRY_RUN" == "1" ]]; then
    echo "mode:        dry run"
fi
echo

install_packages() {
    if ! truthy "$YES"; then
        return 0
    fi
    if [[ $# -eq 0 ]]; then
        return 0
    fi

    local -a packages=("$@")
    if [[ -n "${TERMUX_VERSION:-}" ]] && command -v pkg >/dev/null 2>&1; then
        for i in "${!packages[@]}"; do
            [[ "${packages[$i]}" == "go" ]] && packages[i]="golang"
        done
        run pkg install -y "${packages[@]}"
    elif command -v apt-get >/dev/null 2>&1; then
        for i in "${!packages[@]}"; do
            [[ "${packages[$i]}" == "go" ]] && packages[i]="golang-go"
        done
        run sudo apt-get update
        run sudo apt-get install -y "${packages[@]}"
    elif command -v dnf >/dev/null 2>&1; then
        for i in "${!packages[@]}"; do
            [[ "${packages[$i]}" == "go" ]] && packages[i]="golang"
        done
        run sudo dnf install -y "${packages[@]}"
    elif command -v pacman >/dev/null 2>&1; then
        run sudo pacman -S --needed --noconfirm "${packages[@]}"
    else
        echo "[!] missing dependencies: $*"
        echo "    install them manually, or rerun after adding a supported package manager"
    fi
}

missing=()
command -v git >/dev/null 2>&1 || missing+=(git)
if ! command -v go >/dev/null 2>&1; then
    if [[ -n "${TERMUX_VERSION:-}" ]]; then
        missing+=(go)
    else
        echo "[x] Go 1.25+ is required. Install a supported Go toolchain before running this installer." >&2
        exit 1
    fi
fi
command -v zsh >/dev/null 2>&1 || missing+=(zsh)
if [[ ${#missing[@]} -gt 0 ]]; then
    echo "[*] Missing dependencies: ${missing[*]}"
    install_packages "${missing[@]}"
fi

for cmd in git go; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "[x] $cmd is required. Install it or rerun with OZSH_YES=1 on a supported system." >&2
        exit 1
    fi
done

if ! command -v zsh >/dev/null 2>&1; then
    echo "[!] zsh not found. Install it before running 'ozsh apply'."
fi

go_version="$(go version | awk '{print $3}' | sed 's/^go//')"
case "$go_version" in
    1.25*|1.26*|1.27*|1.28*|1.29*|1.[3-9][0-9]*|[2-9].*) ;;
    *)
        echo "[x] Go 1.25+ is required; found ${go_version:-unknown}" >&2
        exit 1
        ;;
esac

if [[ -d "$INSTALL_DIR/.git" ]]; then
    echo "[*] Updating existing installation..."
    if [[ "$DRY_RUN" == "1" ]]; then
        echo "[dry run] git -C $INSTALL_DIR pull --ff-only"
    else
        git -C "$INSTALL_DIR" pull --ff-only
    fi
elif [[ -d "$INSTALL_DIR" && -n "$(find "$INSTALL_DIR" -mindepth 1 -maxdepth 1 2>/dev/null)" ]]; then
    echo "[x] install dir exists but is not an ozsh git checkout: $INSTALL_DIR" >&2
    exit 1
else
    echo "[*] Cloning ozsh..."
    run git clone --depth 1 "$REPO" "$INSTALL_DIR"
fi

if [[ "$SKIP_BUILD" != "1" ]]; then
    echo "[*] Building ozsh..."
    if [[ "$DRY_RUN" == "1" ]]; then
        echo "[dry run] CGO_ENABLED=0 go build -buildvcs=false -o ozsh ./cmd/ozsh"
    else
        (cd "$INSTALL_DIR" && CGO_ENABLED=0 go build -buildvcs=false -o ozsh ./cmd/ozsh)
    fi
fi

run mkdir -p "$BIN_DIR"
if [[ "$SKIP_BUILD" != "1" ]]; then
    run cp "$INSTALL_DIR/ozsh" "$BIN_DIR/ozsh"
    run chmod +x "$BIN_DIR/ozsh"
fi

zshrc="${HOME}/.zshrc"
path_line="export PATH=$(shell_single_quote "$BIN_DIR"):\"\$PATH\""
default_path_line="export PATH=\"\$HOME/.local/bin:\$PATH\""
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    if truthy "$UPDATE_PATH"; then
        echo "[*] Adding $BIN_DIR to $zshrc"
        if [[ "$DRY_RUN" == "1" ]]; then
            echo "[dry run] append PATH update to $zshrc"
        else
            mkdir -p "$(dirname "$zshrc")"
            touch "$zshrc"
            if [[ "$BIN_DIR" == "${HOME}/.local/bin" ]]; then
                grep -Fqx "$default_path_line" "$zshrc" || printf '\n%s\n' "$default_path_line" >> "$zshrc"
            else
                grep -Fqx "$path_line" "$zshrc" || printf '\n%s\n' "$path_line" >> "$zshrc"
            fi
        fi
    else
        echo "[!] $BIN_DIR is not in PATH. Add this to your .zshrc:"
        if [[ "$BIN_DIR" == "${HOME}/.local/bin" ]]; then
            echo "    $default_path_line"
        else
            echo "    $path_line"
        fi
    fi
fi

echo
echo "[✓] ozsh installed to $BIN_DIR/ozsh"

if truthy "$APPLY"; then
    echo "[*] Applying ozsh prompt..."
    if [[ "$DRY_RUN" == "1" ]]; then
        echo "[dry run] HOME=$HOME $BIN_DIR/ozsh apply"
    else
        HOME="$HOME" "$BIN_DIR/ozsh" apply >/dev/null
        echo "[✓] ozsh apply complete"
    fi
fi

if [[ "$DRY_RUN" != "1" && -x "$BIN_DIR/ozsh" ]]; then
    HOME="$HOME" "$BIN_DIR/ozsh" doctor || true
fi

echo
echo "Next steps:"
echo "  ozsh doctor     # Check environment"
echo "  ozsh preview    # See prompt preview"
if ! truthy "$APPLY"; then
    echo "  ozsh apply      # Generate and apply prompt"
fi
if [[ -n "${TERMUX_VERSION:-}" ]]; then
    echo "  # Termux: ozsh does not run chsh; start zsh after applying"
fi
echo
echo "Config location: ~/.config/ozsh/config.toml"
