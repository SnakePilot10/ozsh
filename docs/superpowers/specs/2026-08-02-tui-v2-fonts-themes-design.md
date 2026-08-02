# ozsh TUI v2, Fonts, Themes, and Recommended Plugins Design

## Goal

Integrate the supplied TUI v2 and README into `ozsh` as a coherent, buildable feature set for Linux and Termux. The feature adds a five-screen TUI, a declarative embedded theme catalog, verified Nerd Font installation, recommended plugin selection and installation, and a backward-compatible configuration migration.

## Scope

The implementation replaces `internal/tui/tui_v2.go`, updates `README.md`, and adds the supplied `internal/fonts/manifest.go` and `internal/themes/catalog.toml`. Supporting Go code, tests, and configuration changes are included so the attached files are not published as disconnected scaffolding.

## Architecture

### Configuration schema v2

`internal/config` remains the single owner of persisted TOML. Schema version 2 adds:

- Prompt display name, icon mode, layout, symbol, and separate compatible/Nerd icons.
- Theme ID and variant while retaining the existing display name and palette.
- A list of selected curated plugin IDs while retaining concrete plugin items.

Version 0 and version 1 files are migrated in memory, backed up with the existing timestamped mechanism, validated, and rewritten as version 2. Future versions remain rejected.

### Theme catalog

`internal/themes/catalog.toml` is embedded with `go:embed`. A small loader decodes it once, validates identifiers, variants, colors, layouts, segment order, and duplicate `(id, variant)` pairs, then exposes immutable copies through:

- `List() []Preset`
- `Get(id, variant string) (Preset, bool)`
- `Variants(id string) []string`
- `Apply(base *config.Config, preset Preset) *config.Config`

`List` exposes one gallery row per theme ID. Circuit appears once and its variants are selected through `Variants` and `Get`.

### Font manager

`internal/fonts` owns verified download and installation. `Manager.Install` downloads the pinned ZIP with an HTTP client bound to the caller context, enforces a size limit, verifies the archive SHA-256, extracts only the configured regular font file, and installs it atomically.

- Termux destination: `~/.termux/font.ttf`, preserving one recoverable backup and invoking `termux-reload-settings` when available.
- Linux destination: `~/.local/share/fonts/ozsh/<font-file>.ttf`, followed by `fc-cache -f` when available.

`RestoreTermux` restores the previous Termux font atomically. No arbitrary archive path or symlink is trusted.

### Recommended plugins

`internal/plugins/catalog.go` defines the curated catalog for `zsh-autosuggestions`, `fzf-tab`, and `zsh-syntax-highlighting`. Fresh configurations select all three but do not download them until confirmation. Existing plugin lifecycle and trust validation remain authoritative.

The catalog layer maps curated definitions to concrete `config.PluginItem` records, reports selected/installed/active state, and installs selected definitions through the existing bounded clone path. Syntax highlighting remains last in generated load order, and `fzf-tab` remains after completion initialization.

### TUI

The supplied TUI becomes the package implementation. Its tabs are Home, Prompt, Themes, Plugins, and Preview. Doctor, backup recovery, font management, and Review & Apply are actions or modals. Narrow layouts remain usable below 72 columns.

## Error handling

- Catalog initialization failures are deterministic and surfaced by tests; embedded production data must validate at startup.
- Downloads fail closed on HTTP errors, size overflow, checksum mismatch, malformed ZIPs, missing assets, unsafe paths, or atomic replacement failures.
- Configuration migration never overwrites the source before a successful backup.
- Plugin install failures preserve the selected configuration and identify the failed plugin.
- TUI operations return user-readable messages and never apply `.zshrc` before final confirmation.

## Testing

Add focused tests for:

1. Configuration v1 migration, v2 defaults, enum validation, icon fallback, and selected-plugin validation.
2. Theme catalog parsing, gallery de-duplication, variant lookup, immutable results, and application to a cloned config.
3. Font manifest immutability, checksum rejection, safe extraction, Termux backup/restore, and Linux destination behavior using local HTTP test servers and temporary HOME directories.
4. Curated plugin status and deterministic installation ordering without network access.
5. TUI navigation, modal state, narrow rendering, theme application, plugin selection, and config cloning.
6. Full repository formatting, vet, unit/race tests, build, healthcheck, and Android/Termux cross-build through existing scripts and CI.

## Non-goals

- Supporting macOS or Windows.
- Adding arbitrary font URLs.
- Automatically trusting third-party custom plugins.
- Replacing the existing safe apply, backup, logging, or plugin clone implementations.
