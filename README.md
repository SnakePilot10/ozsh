# ozsh

> A declarative prompt builder for Zsh: configure, preview, generate, and apply.

## Status

`ozsh` is functional beta software. The CLI, Bubble Tea TUI, prompt generator,
themes, manual plugin support, backups, installer, and release packaging are in
place. The project remains pre-v1.0 while it receives broader testing on Linux
and Termux.

## Principles

- **Safe by default.** Managed `.zshrc` changes are atomic and backed up.
- **Preview before apply.** Configuration can be inspected without touching the shell.
- **Manual plugins.** Third-party shell code is never trusted implicitly.
- **Termux matters.** Android is treated as a supported environment, not an afterthought.

## Installation

Go 1.25 or newer is required for source installations.

```bash
git clone https://github.com/SnakePilot10/ozsh.git
cd ozsh
./install.sh
```

Unattended installation from a reviewed checkout:

```bash
OZSH_YES=1 OZSH_UPDATE_PATH=1 OZSH_APPLY=1 ./install.sh
```

Installer variables:

- `OZSH_REPO`: Git URL or local source path.
- `OZSH_INSTALL_DIR`: source checkout directory.
- `OZSH_BIN_DIR`: binary destination.
- `OZSH_YES=1`: allow supported dependency installation.
- `OZSH_APPLY=1`: apply the generated prompt after installation.
- `OZSH_UPDATE_PATH=1`: add the binary directory to `.zshrc` when needed.

Versioned binaries, checksums, signatures, certificates, and Sigstore bundles are
published through GitHub Releases when a `v*` tag is cut. Avoid piping the
mutable `main` branch into a shell.

## Commands

```bash
ozsh doctor
ozsh doctor --report
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
ozsh plugin untrust plugin
ozsh plugin remove plugin

ozsh tui
ozsh version
ozsh update --check
ozsh update
```

Use `--verbose` or `-v` for debug output. Logs are written to
`~/.config/ozsh/ozsh.log` and rotate at 5MB with three retained backups.
`ozsh update --check` fetches the source checkout and reports when a new version
is available. `ozsh update` fast-forwards the source checkout, rebuilds ozsh,
and replaces the currently running binary.
`ozsh doctor --report` writes a sanitized local diagnostic report to
`~/.config/ozsh/doctor-report.txt` for support requests.

## Configuration

Configuration lives at `~/.config/ozsh/config.toml`.

```toml
version = 1

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

- `user`: current username.
- `host`: hostname.
- `cwd`: current directory.
- `git`: branch inside a Git repository.
- `status`: previous command status.
- `time`: current time.
- `venv`: active Python virtual environment.
- `node`: Node.js version when `package.json` exists.
- `go`: Go version when `go.mod` exists.
- `battery`: Linux or Termux battery level.
- `jobs`: background job count.

Colors accept Zsh color names or six-digit hex values such as `#00f5ff`.
Each segment can also define a literal `icon` and foreground/background colors;
dynamic prompt text and icons are escaped before Zsh renders them.
`right_order` controls `RPROMPT`. Set `disable_heavy_segments = true` to skip
runtime-heavy segments such as Git, Node.js, Go, and battery detection.

The `version` field is the stable config schema version. Pre-v1 configs without
that field are migrated automatically after creating a timestamped backup next to
`config.toml`. Future schema versions are rejected instead of being rewritten.

## Themes

Built-in presets:

- `cyber-cyan`
- `neon-red`
- `matrix-green`

Preset files are stored under `presets/` for users and packagers.

## Manual plugins

Plugins are cloned into `~/.config/ozsh/plugins/`. Only HTTPS repository URLs
are accepted. A plugin load path must be a relative `.zsh` or `.sh` file.
Generated shell code sources a plugin only when it is enabled, explicitly
trusted, readable, a regular file under `$HOME`, and not a symlink.

Trusting a plugin means allowing third-party code to execute in Zsh. Review it
before running `ozsh plugin trust <name>`.

## TUI

`ozsh tui` opens a Bubble Tea interface with dashboard, prompt builder, editable
preview, apply, doctor, themes, and plugins views. The apply flow shows the
planned `.zshrc` diff and requires confirmation before writing.

Key controls:

| Keys | Action |
| --- | --- |
| `Tab` / `Shift+Tab`, `1`-`7` | Change view |
| `Up` / `Down`, `j` / `k` | Move through lists and fields |
| `/`, `PgUp` / `PgDn` | Filter and page lists |
| `F1` / `?` | Toggle contextual help |
| `s`, `d` | Save or discard Builder changes |
| `e` | Edit the selected prompt segment |
| `p`, `t`, `u` | Add, trust, or untrust a plugin |
| `a`, then `y` | Review and confirm Apply |
| `q` / `Ctrl+C` | Quit / force quit |

Printable keys belong to the focused input while editing. `Esc` cancels a form
or confirmation. Trusting a plugin always requires a second explicit
confirmation that shows its source and load path.

## Termux

Termux is detected through `TERMUX_VERSION` or `PREFIX`. The installer can use
`pkg` for missing Go, Zsh, and Git dependencies and never attempts `chsh`.
After `ozsh apply`, start Zsh so the managed block can source
`~/.config/ozsh/omega.zsh`.

## Development

```bash
git clone https://github.com/SnakePilot10/ozsh.git
cd ozsh
scripts/setup.sh
```

Useful checks:

```bash
scripts/lint.sh --check
scripts/test.sh
scripts/build.sh
scripts/healthcheck.sh
scripts/release-smoke.sh
scripts/install-smoke.sh
```

The CI workflow verifies module integrity and tidy state, runs formatting,
`go vet`, golangci-lint, ShellCheck, race and coverage tests on Linux, scans Go
vulnerabilities and secrets, and cross-builds for Android/Termux ARM64.

## Project structure

```text
cmd/ozsh/          CLI entrypoint and command tests
internal/config/   TOML loading, persistence, defaults, and validation
internal/logging/  local logging and rotation
internal/plugins/  manual plugin management and trust rules
internal/prompt/   prompt generation and preview
internal/shell/    environment detection, .zshrc management, and backups
internal/tui/      Bubble Tea interface
packaging/         AUR packaging
presets/           built-in themes
scripts/           development, validation, installation, and release smoke tests
```

## CI and releases

Pull requests target `main`. GitHub Actions validates every push and pull
request. Tags matching `v*` invoke GoReleaser to publish versioned artifacts and
checksums. Release checksums are signed with keyless cosign. Releases publish
`checksums.txt`, its detached signature, signing certificate, and verification
bundle; verification commands are in `docs/release-checklist.md`.

See `GIT_WORKFLOW.md` for branch and release conventions and
`docs/release-checklist.md` before publishing a version. The first stable
release should be preceded by a `v1.0.0-rc.1` tag and a short external smoke
window on Linux and Termux.

## Contributing

Use focused branches and Conventional Commits. Avoid direct pushes to `main`,
include relevant validation results in the pull request, and use an isolated
`HOME` in any test that touches `.zshrc`, generated shell files, or plugins.

## License

`ozsh` is released under the MIT License. See `LICENSE`.
