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
- **Explicit plugins.** Recommended plugins are selected by default, but downloaded only after confirmation.
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
ozsh theme preview dracula
ozsh theme preview circuit --variant amber
ozsh theme apply circuit --variant neon

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
version = 2

[prompt]
style = "simple"
display_name = "user"
icon_mode = "compatible"
layout = "two-line"
symbol = "❯"
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
Each segment can define separate `compatible_icon` and `nerd_icon` values plus
foreground/background colors. Compatible icons are the default and require no
special font. Dynamic prompt text and icons are escaped before Zsh renders them.
`right_order` controls `RPROMPT`. Set `disable_heavy_segments = true` to skip
runtime-heavy segments such as Git, Node.js, Go, and battery detection.

The `version` field is the stable config schema version. Pre-v1 configs without
that field are migrated automatically after creating a timestamped backup next to
`config.toml`. Future schema versions are rejected instead of being rewritten.

## Themes

Built-in presets:

- `minimal`, `pure`, `powerline`, `cyberpunk`
- `matrix`, `dracula`, `nord`, `gruvbox`
- `catppuccin`, `termux`, `circuit`, `retro`

Circuit alone provides `blue`, `green`, `amber`, `red`, `mono`, and `neon`
variants. Use `--variant <name>` with `theme preview` or `theme apply`.

## Plugins

Fresh configurations select `zsh-autosuggestions`, `fzf-tab`, and
`zsh-syntax-highlighting`. They appear under **Recommended plugins** in the TUI.
The recommended set is curated by ozsh, remains deselectable, and is downloaded
only after explicit confirmation. Completion initializes before `fzf-tab`, and
syntax highlighting always loads last.

Custom repositories appear separately under **Custom plugins**. Press
`[a] Add custom plugin` from the Plugins screen to open the guided workflow:

1. Enter an HTTPS GitHub repository URL. Other schemes, malformed URLs,
   duplicate repositories, and destination conflicts are rejected.
2. ozsh performs a shallow clone into a private `.staging-*` directory below
   `~/.config/ozsh/plugins/`.
3. The checkout is scanned with bounded depth and file-count limits. Generated,
   dependency, VCS, symlink, and unsafe paths are excluded.
4. Choose the `.plugin.zsh`, `.zsh`, or `.sh` file that should be sourced.
5. Review the trust warning and explicitly approve the repository.
6. The plugin is shown as pending. Nothing is moved into its final directory or
   loaded by Zsh until **Review & Apply** succeeds.

The Plugins screen supports the complete pending lifecycle:

- `Space`: enable or disable the selected plugin.
- `t` / `u`: grant or remove trust from a custom plugin.
- `l`: choose a different load file from the managed checkout.
- `d`: queue removal of a custom plugin.
- `Ctrl+A`: inspect all pending configuration and filesystem changes, then
  confirm or cancel them together.

Review & Apply lists each addition or removal, including repository, load file,
and final or removed path. Apply snapshots the reviewed state and treats plugin
moves, generated shell files, configuration, and `.zshrc` injection as one
reversible transaction. A failed Apply restores the previous files and plugin
locations while preserving the pending model for retry. Cancelling leaves the
staged checkout untouched; exiting the TUI cleans any unapplied `.staging-*`
directories.

`~/.config/ozsh/plugins/` is owned and managed by ozsh. Final custom checkouts
live at `~/.config/ozsh/plugins/<plugin-name>/`; temporary clones use direct
`.staging-*` children. Do not place unrelated files in that directory or rename
managed plugin folders behind ozsh.

Generated shell code sources a plugin only when it is enabled, explicitly
trusted, readable, a regular file under `$HOME`, and not a symlink. Trust is not
a sandbox: **a trusted shell plugin executes third-party code in every
interactive Zsh session**. Review the repository and selected load file before
approving it.

The CLI plugin commands remain available for scripted administration, but the
TUI is the recommended path when adding an unfamiliar repository because it
shows candidate discovery, trust, pending state, and the final transaction
before activation.

## TUI

`ozsh tui` opens five focused screens in this order: Home, Prompt, Themes,
Plugins, and Preview. Doctor, backup recovery, and the optional Nerd Font
installer live under Home. Review & Apply is a global modal and never writes
`.zshrc` before final confirmation.

The interface uses the terminal's own background for application chrome.
Selection remains visible through accent text, borders, markers, bold text, and
underline rather than filled gray rectangles. Intentional background colors are
reserved for prompt segments that belong to the selected theme.

The layout adapts below 72 columns for Termux and other narrow terminals. The
Plugins screen keeps **Custom plugins** and `[a] Add custom plugin` discoverable
at narrow widths, while the add wizard owns keyboard focus until it is completed
or cancelled. Review & Apply displays pending plugin counts and paths before the
optional technical `.zshrc` diff.

The font dialog offers JetBrainsMono, MesloLGS, and FiraCode Nerd Font from
pinned, SHA-256-verified archives. Termux activation backs up
`~/.termux/font.ttf` and reloads settings; Linux installs the font for the
current user and asks you to select it in terminal settings.

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
internal/fonts/    verified Nerd Font download, install, and recovery
internal/logging/  local logging and rotation
internal/plugins/  curated and custom plugin management and trust rules
internal/prompt/   prompt generation and preview
internal/shell/    environment detection, .zshrc management, and backups
internal/themes/   declarative built-in theme catalog
internal/tui/      Bubble Tea interface
packaging/         AUR packaging
presets/           legacy package compatibility assets
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
