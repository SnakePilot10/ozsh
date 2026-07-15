# ozsh

> A declarative prompt builder for Zsh. Config -> Preview -> Generate -> Apply.

## Current Status

Beta funcional, no v1.0. La mayoria del CLI y la TUI estan implementados, y la
suite local pasa al 100%. Quedan validaciones externas antes de declarar v1.0.
Ver `STATUS.md` antes de probar cambios grandes.

## Philosophy

- **Motor first, TUI later.** The CLI engine works before any visual interface.
- **Never touch `.zshrc` without backup.** Every `apply` and `reset` creates a timestamped backup.
- **Manual plugins, no magic.** ozsh can source plugin files, but does not manage plugin runtime behavior.
- **Termux is a first-class citizen.** Detected automatically, no shell switching magic.

## Installation

Requires Go 1.24+.

Download a versioned binary from GitHub Releases when available, or install from source:

```bash
git clone https://github.com/SnakePilot10/ozsh.git
cd ozsh
./install.sh
```

Unattended source install:

```bash
curl -fsSL https://raw.githubusercontent.com/SnakePilot10/ozsh/main/install.sh | bash -s -- --yes --update-path --apply
```

Installer environment:

- `OZSH_REPO` - Git URL or local path to clone
- `OZSH_INSTALL_DIR` - source checkout directory
- `OZSH_BIN_DIR` - binary install directory
- `OZSH_YES=1` - install missing dependencies where supported
- `OZSH_APPLY=1` - run `ozsh apply` after installation
- `OZSH_UPDATE_PATH=1` - append the binary directory to `.zshrc` when missing

## Commands

```bash
ozsh doctor
ozsh preview
ozsh preview --real
ozsh apply
ozsh reset
ozsh theme list
ozsh theme preview cyber-cyan
ozsh theme apply cyber-cyan
ozsh plugin list
ozsh plugin add https://github.com/user/plugin.git plugin.zsh
ozsh plugin enable plugin
ozsh plugin disable plugin
ozsh plugin trust plugin
ozsh plugin remove plugin
ozsh tui
ozsh update --check
ozsh update
```

Use `--verbose` or `-v` for debug output. Logs are written to
`~/.config/ozsh/ozsh.log` and rotate at 5MB with three retained backups.
`ozsh update --check` fetches the source checkout and reports when a new version
is available.

## Configuration

Config lives at `~/.config/ozsh/config.toml`.

```toml
[prompt]
style = "simple"
newline = true
right_prompt = false
disable_heavy_segments = false
separator = "  "
order = ["user", "cwd", "git", "status"]
right_order = []
```

Available segments:

- `user` - current username (`%n`)
- `host` - hostname (`%m`)
- `cwd` - current directory (`%~`)
- `git` - branch when inside a Git repository
- `status` - last command status
- `time` - current time (`%*`)
- `venv` - active Python virtualenv
- `node` - Node.js version when `package.json` exists
- `go` - Go version when `go.mod` exists
- `battery` - Linux or Termux battery level
- `jobs` - background job count

Colors accept Zsh names (`cyan`, `red`, `default`) or hex values like
`#00f5ff`. Hex colors are emitted into generated Zsh and rendered as truecolor
ANSI in `ozsh preview`.

Right prompt segments are configured with `right_order`; enabling
`right_prompt = true` writes `RPROMPT`.

Set `disable_heavy_segments = true` to skip runtime-heavy segments (`git`,
`node`, `go`, and `battery`) in generated prompts and previews. The TUI also
uses this lighter preview mode automatically on Termux.

## Themes

Built-in presets:

- `cyber-cyan`
- `neon-red`
- `matrix-green`

Theme files also live in `presets/*.toml` for humans and packagers.

## Prompt Examples

Minimal:

```text
snake  ~/dev/ozsh
❯
```

With Git status:

```text
snake  ~/dev/ozsh  main +  ✘ 1
❯
```

With right prompt:

```text
snake  ~/dev/ozsh  main +                  14:52
❯
```

## Manual Plugins

Plugins are cloned into `~/.config/ozsh/plugins/`. Only `https` plugin URLs are
accepted. Plugin load files must be relative `.zsh` or `.sh` paths. Generated
`omega.zsh` sources plugin files only when they are enabled, readable, regular
files under `$HOME`, not symlinks, and explicitly trusted with
`ozsh plugin trust <name>`.

## TUI

`ozsh tui` opens a Bubble Tea interface with dashboard, prompt builder, editable
preview, apply, doctor, themes, and plugins tabs. The Apply tab shows
the planned `.zshrc` diff first and requires confirmation before writing.

## Screencasts

An asciinema-compatible screencast lives at `docs/screencasts/preview.cast`.
It can be converted to a GIF with `agg`.

## Templates

If `templates/<prompt.style>.zsh.tmpl` exists, ozsh renders it with Go
`text/template`. Otherwise it uses the built-in generator.

## Termux

Termux is detected automatically via `TERMUX_VERSION` or `PREFIX`. The installer
uses `pkg install` for missing `golang`, `zsh`, and `git` when available.

ozsh does not run `chsh` on Termux. Run `ozsh apply` or install with
`OZSH_APPLY=1`, start `zsh`, and let the managed block in `~/.zshrc` source
`~/.config/ozsh/omega.zsh`.

## Migration from omega-zsh-python

Start with `examples/minimal.toml`, copy over the segment order and colors you
want to preserve, then run:

```bash
ozsh preview
ozsh apply
```

## Development Stack

- Language: Go 1.24+.
- Dependency manager: Go modules.
- CLI entrypoint: `cmd/ozsh/main.go`.
- Terminal UI: Bubble Tea, Bubbles and Lip Gloss.
- Config format: TOML.
- Release tooling: GoReleaser, GitHub Releases, Homebrew formula and AUR PKGBUILD.
- CI/CD: GitHub Actions.

## Developer Setup

```bash
git clone https://github.com/SnakePilot10/ozsh.git
cd ozsh
scripts/setup.sh
```

If `pre-commit` is installed, `scripts/setup.sh` installs local hooks. The hooks
run `scripts/lint.sh` and fast tests before each commit.

## Quality Commands

```bash
scripts/lint.sh --check      # gofmt check, go vet, golangci-lint when installed
scripts/test.sh              # coverage, race tests and smoke tests
scripts/build.sh             # production binary at bin/ozsh
scripts/healthcheck.sh       # CLI healthcheck with isolated HOME
scripts/release-smoke.sh     # release smoke path
scripts/install-smoke.sh     # installer smoke path
```

Coverage currently gates at 70%. The target is to raise it toward 80% without
blocking release-candidate work.

## Project Structure

```text
cmd/ozsh/          CLI entrypoint and command tests
internal/config/   TOML config loading, saving and validation
internal/logging/  local structured logging and rotation
internal/plugins/  manual plugin management and trust rules
internal/prompt/   prompt generation, templates and preview rendering
internal/shell/    shell detection, .zshrc management and backups
internal/tui/      Bubble Tea interface
packaging/         Homebrew and AUR packaging
presets/           built-in themes
scripts/           local automation, validation, deploy and smoke tests
templates/         optional Zsh prompt templates
```

## CI/CD and Releases

Validate locally before opening a PR:

```bash
git checkout -b feature/my-change
scripts/lint.sh --check
scripts/test.sh
scripts/build.sh
scripts/healthcheck.sh
```

GitHub Actions repeats validation in a clean environment. Merges to `main`
publish the Docker image and run the production gate. Tags matching `v*` publish
GitHub Releases with GoReleaser artifacts and checksums.

Read:

- `ANALISIS_PROYECTO.md` for project analysis.
- `GIT_WORKFLOW.md` for branch and PR rules.
- `DEPLOYMENT.md` for deployment and rollback.

## Contributing

Use Conventional Commits and work from `feature/*`, `hotfix/*` or `release/*`
branches. Do not push directly to `main`. Pull requests should include local
validation output and any release or deployment notes.
