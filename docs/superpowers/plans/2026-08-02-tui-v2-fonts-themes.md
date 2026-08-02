# ozsh TUI v2, Fonts, Themes, and Recommended Plugins Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate the supplied TUI v2, font manifest, theme catalog, and README as a tested Linux/Termux feature set.

**Architecture:** Extend the existing configuration and plugin layers rather than bypassing them. Embed declarative theme data, isolate font download/install logic behind a manager, and keep the TUI as an orchestration layer over tested package APIs.

**Tech Stack:** Go 1.25, Bubble Tea, Bubbles, Lip Gloss, BurntSushi TOML, `go:embed`, standard-library HTTP/ZIP/crypto packages, GitHub Actions.

## Global Constraints

- Linux and Android/Termux are the supported release targets.
- Persisted config changes remain atomic, private, backed up, and schema-versioned.
- Font archives are pinned and SHA-256 verified.
- Custom plugins remain explicit, HTTPS-only, reviewed, and trusted before loading.
- No direct push to `main`; publish through a draft pull request.

---

### Task 1: Configuration schema v2

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/defaults.go`
- Modify: `internal/config/io.go`
- Modify: `internal/config/validate.go`
- Test: `internal/config/io_test.go`
- Test: `internal/config/validate_test.go`

**Interfaces:**
- Produces constants `IconModeCompatible`, `IconModeNerd`, `PromptLayoutOneLine`, and `PromptLayoutTwoLine`.
- Produces v2 fields used by themes, plugins, prompt generation, and TUI.

- [ ] Write tests that load a version-1 TOML file and assert migration to version 2 with compatible icons, inferred layout, default symbol, theme ID, and selected curated plugins.
- [ ] Run `go test ./internal/config` and verify the new assertions fail because v2 fields and migration are absent.
- [ ] Add v2 types, defaults, migration normalization, and strict validation.
- [ ] Run `go test ./internal/config` and verify all config tests pass.
- [ ] Commit with `feat(config): add v2 prompt and catalog settings`.

### Task 2: Embedded theme catalog

**Files:**
- Create: `internal/themes/catalog.toml`
- Create: `internal/themes/catalog.go`
- Create: `internal/themes/catalog_test.go`

**Interfaces:**
- Produces `Preset`, `List`, `Get`, `Variants`, and `Apply` consumed by the TUI and CLI.

- [ ] Write tests for the 12 gallery IDs, six Circuit variants, duplicate rejection, immutable list results, and cloned config application.
- [ ] Run `go test ./internal/themes` and verify it fails because the package does not exist.
- [ ] Embed and parse the supplied catalog, validate it once, and implement the public API.
- [ ] Run `go test ./internal/themes` and verify all tests pass.
- [ ] Commit with `feat(themes): add declarative embedded catalog`.

### Task 3: Verified Nerd Font manager

**Files:**
- Create: `internal/fonts/manifest.go`
- Create: `internal/fonts/manager.go`
- Create: `internal/fonts/manager_test.go`

**Interfaces:**
- Produces `NewManager(home string, termux bool) *Manager`.
- Produces `(*Manager).Install(context.Context, Font, func(downloaded, total int64)) error` and `(*Manager).RestoreTermux(context.Context) error`.

- [ ] Write local-server tests for checksum mismatch, safe extraction, atomic Termux install/backup/restore, and Linux installation.
- [ ] Run `go test ./internal/fonts` and verify it fails because the manager is absent.
- [ ] Implement bounded download, SHA-256 verification, ZIP extraction, atomic writes, optional cache reload, and restoration.
- [ ] Run `go test ./internal/fonts` and verify all tests pass.
- [ ] Commit with `feat(fonts): add verified Nerd Font installer`.

### Task 4: Recommended plugin catalog

**Files:**
- Create: `internal/plugins/catalog.go`
- Modify: existing plugin manager files only where needed to expose the safe clone operation.
- Create: `internal/plugins/catalog_test.go`

**Interfaces:**
- Produces `Definition`, `Status`, `Catalog`, `FindDefinition`, `StatusFor`, and `InstallRecommended`.

- [ ] Write tests for default definitions, selected/installed/active state, item reconciliation, and syntax-highlighting-last installation order.
- [ ] Run `go test ./internal/plugins` and verify failures identify the missing catalog API.
- [ ] Implement catalog mapping by delegating downloads and persistence to the existing plugin lifecycle.
- [ ] Run `go test ./internal/plugins` and verify all tests pass.
- [ ] Commit with `feat(plugins): add recommended plugin workflow`.

### Task 5: Prompt and shell integration

**Files:**
- Modify: `internal/prompt/*` files that render user/icon/layout fields.
- Modify: `internal/shell/*` only if the TUI requires an existing read-only helper.
- Test: corresponding package tests.

**Interfaces:**
- Prompt rendering consumes v2 display name, icon mode, layout, symbol, and dual icons while preserving escaping.

- [ ] Write failing tests for display-name fallback, compatible/Nerd icon selection, one/two-line layout, symbol rendering, and plugin load order.
- [ ] Run focused prompt and shell tests and verify expected failures.
- [ ] Implement minimal rendering and helper changes.
- [ ] Run focused tests and verify they pass.
- [ ] Commit with `feat(prompt): render v2 identity layout and icons`.

### Task 6: TUI v2 integration

**Files:**
- Replace: `internal/tui/tui_v2.go`
- Modify: `internal/tui/tui_test.go`

**Interfaces:**
- Consumes configuration v2, theme catalog, font manager, plugin catalog, prompt preview, safe apply, and shell backup helpers.

- [ ] Update tests first for five tabs, review modal, font/backup dialogs, narrow rendering, theme variants, plugin selection, and immutable config cloning.
- [ ] Run `go test ./internal/tui` and verify failures reflect the old seven-tab UI.
- [ ] Replace the TUI with the supplied implementation and make only compatibility corrections required by tested package APIs.
- [ ] Run `go test ./internal/tui` and verify all tests pass.
- [ ] Commit with `feat(tui): introduce focused five-screen workflow`.

### Task 7: Documentation and full verification

**Files:**
- Replace: `README.md`
- Create: `docs/superpowers/specs/2026-08-02-tui-v2-fonts-themes-design.md`
- Create: `docs/superpowers/plans/2026-08-02-tui-v2-fonts-themes.md`

- [ ] Replace the README with the supplied version and verify every documented command or feature exists.
- [ ] Run `gofmt -w` on changed Go files.
- [ ] Run `scripts/lint.sh --check`.
- [ ] Run `scripts/test.sh` and `go test -race ./...`.
- [ ] Run `scripts/build.sh`, `scripts/healthcheck.sh`, and `CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -buildvcs=false ./cmd/ozsh`.
- [ ] Commit with `docs: document TUI v2 fonts themes and plugins`.
- [ ] Push `feat/tui-v2-fonts-themes` and open one draft PR against `main` with validation evidence and known limitations.
