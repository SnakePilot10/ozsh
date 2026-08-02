# ozsh TUI Redesign

**Date:** 2026-08-02  
**Status:** Approved  
**Target branch:** `design/ozsh-tui-redesign`

## 1. Objective

Replace the current debug-oriented Bubble Tea interface with a polished, mobile-first guided workspace for Termux and Linux. Preserve the existing safe prompt generation, backup, plugin trust, and apply behavior while making prompt customization understandable without exposing internal configuration fields.

## 2. Product principles

- Termux is a first-class platform.
- The interface must remain usable in a narrow mobile terminal without horizontal overflow.
- Preview precedes mutation. No `.zshrc` change occurs without explicit review and confirmation.
- Normal screens use plain user-facing language; paths, hashes, and raw errors live under technical details.
- Compatible Unicode rendering is the default. Nerd Font glyphs are opt-in.
- Existing installations must not receive new aliases, plugins, or visual changes silently.
- The TUI remains English-only for this release, with UI strings centralized for future localization.
- Long-running downloads and filesystem operations must not block Bubble Tea's update loop.

## 3. Navigation and responsive layout

The primary order is:

1. `Home`
2. `Prompt`
3. `Themes`
4. `Plugins`
5. `Preview`

`Apply` is a global review modal, not a tab. `Doctor` and backup recovery are actions within `Home`.

Keyboard behavior:

- `Tab` / `Shift+Tab`: next or previous primary screen when no editor owns the key.
- `1`–`5`: direct navigation when no text field owns the key.
- Arrow keys or `j` / `k`: move the local selection.
- `Enter`: open, edit, or activate the selected control.
- `Esc`: close the current editor or modal and return one level.
- `a`: open Review & Apply when no text field owns the key.
- `Ctrl+C`: quit.
- The footer shows only valid actions for the current context.

The layout has wide and compact modes derived from terminal width. Compact mode uses one column, abbreviated navigation labels, wrapped descriptions, and no side-by-side panels. It must not emit horizontally overflowing help lines. Resize events preserve the active screen, selection, and focused field.

## 4. Home

### First run

For a fresh configuration, Home presents a setup checklist:

1. Choose display name.
2. Choose prompt and theme.
3. Configure plugins.
4. Nerd Font (optional).
5. Review and apply.

The checklist links to the corresponding screen or modal and displays `Ready`, `Pending`, or `Optional`.

### Normal state

After setup, Home displays:

- Overall status: `Ready`, `Changes pending`, or `Needs attention`.
- Active theme and Circuit variant when applicable.
- Compatible or Nerd Font icon mode.
- Active plugin count and pending plugin operations.
- Last backup metadata.
- Actions: `Run Doctor`, `Manage Nerd Font`, `Restore Backup`, and `Review & Apply`.

Paths and low-level state are hidden under `Technical details`.

### Doctor

Doctor results are grouped into shell/platform, configuration, managed `.zshrc`, plugins, and font/icons. Each failure includes a user-facing explanation. A safe automatic repair is offered only when a bounded fix exists, and every mutating repair requires confirmation.

### Backup recovery

The recovery dialog lists dated backups, previews the selected target, and restores a specific backup. It creates a backup of the current state before replacement. A failed restoration reports exactly which files changed and which did not.

## 5. Prompt builder

The default visual alias is `user` on fresh Termux and Linux installations. The field is editable. An empty value means the real system username. This setting changes the generated prompt, not only its simulated preview, and never changes `$USER`, accounts, ownership, or permissions.

### Simple view

The simple view provides:

- Display name.
- One-line or two-line layout.
- Prompt symbol.
- Icon mode: `Compatible` or `Nerd Font`.
- An ordered segment list.

Supported segments remain:

- User
- Directory
- Git branch
- Command status
- Time
- Host
- Python environment
- Node.js
- Go
- Battery
- Background jobs

The list shows enabled state, a readable label, and a short description. Runtime-heavy segments show `May affect startup time`. Users can toggle and reorder entries. The prompt preview updates immediately.

### Advanced segment editor

Opening a segment exposes:

- Enabled state.
- Foreground and background color.
- Bold state.
- Compatible icon.
- Nerd Font icon.
- Supported visibility condition.
- Restore theme defaults.

Raw values such as `fg=#ff003c bold=true icon=""` are not shown in the main list. RPROMPT is disabled initially in compact terminals but remains available under advanced options with a clipping warning.

TUI changes remain an in-memory pending configuration until Review & Apply. Selecting or installing a plugin and installing a font are separate explicit side effects.

## 6. Theme catalog

A built-in theme is a complete prompt preset defining palette, layout, initial segment order and enabled state, separators, prompt symbol, Compatible icons, and Nerd Font icons.

Initial catalog:

1. Minimal
2. Pure
3. Powerline
4. Cyberpunk
5. Matrix
6. Dracula
7. Nord
8. Gruvbox
9. Catppuccin
10. Termux
11. Circuit
12. Retro

Circuit is the only theme family with variants:

- Blue
- Green
- Amber
- Red
- Mono
- Neon

The gallery shows theme name, concise description, color swatches, and a rendered preview. It does not expose raw hexadecimal rows as the primary presentation.

Selecting a theme previews it first. `Use Theme` loads its complete prompt preset while preserving display name, plugin configuration, icon mode, and unrelated preferences. User edits after selection mark it as `<Theme> · Modified`. Reloading the preset requires confirmation and resets only theme-owned prompt properties.

Preset definitions have one declarative source of truth embedded into the binary. Code and external preset files must not contain divergent copies of the same catalog.

## 7. Plugins

A fresh configuration presents this recommended base set as selected:

- `zsh-autosuggestions`
- `fzf-tab`
- `zsh-syntax-highlighting`

The first-run flow asks for one confirmation before installing dependencies, downloading the known repositories, trusting, and enabling those selections. Existing configurations do not gain recommended plugins automatically.

Each plugin displays one of:

- `Active`
- `Disabled`
- `Not installed`
- `Needs attention`

Primary actions:

- `Space`: enable or disable.
- `Enter`: details.
- `i`: install selected missing plugins.
- `u`: update.
- `x`: remove with confirmation.

Known plugins use curated repository and load-path metadata. Their source order is controlled so completion initialization precedes `fzf-tab` and `zsh-syntax-highlighting` loads last. Custom plugins retain explicit HTTPS URL, relative load file, and trust controls under an Advanced action.

When offline, prompt configuration continues and missing plugins remain pending. Applying a prompt must clearly warn about enabled plugins that cannot load.

## 8. Nerd Fonts and icon modes

`Compatible` is the default icon mode.

`Home → Manage Nerd Font` offers:

- JetBrainsMono Nerd Font (recommended).
- MesloLGS Nerd Font.
- FiraCode Nerd Font.

Each ozsh release uses a curated manifest of official assets and expected checksums. Downloads are verified before installation.

### Termux

1. Select a font.
2. Download and verify it.
3. Back up `~/.termux/font.ttf` when present.
4. Install the chosen monospaced font as `~/.termux/font.ttf`.
5. Run `termux-reload-settings`.
6. Enable Nerd Font icons.
7. Allow restoration of the previous Termux font.

### Linux

1. Download and verify it.
2. Install it in the user font directory.
3. Refresh the font cache when available.
4. Explain that the user must select it in their terminal emulator.
5. Enable Nerd Font icons only after user confirmation.

Failure at any point retains Compatible icon mode. Verified downloads may be reused from a cache.

## 9. Preview and apply

Preview is the final primary screen. It renders the pending prompt in a terminal-like block, shows active theme/variant and icon mode, and summarizes unapplied changes.

Built-in scenarios:

- Clean
- Git dirty
- Command failed
- Dev project
- Low battery

Advanced preview context editing remains available but is not the primary experience.

Wide layout shows Current and Pending side by side. Compact layout toggles between them.

`Review & Apply` shows:

- A readable change summary.
- Pending or unhealthy plugins.
- Files that will be updated.
- The `.zshrc` diff under Technical details.
- Final confirmation.

The existing safe apply ordering remains authoritative:

1. Clone the pending configuration.
2. Generate shell content from the clone.
3. Save the configuration.
4. Write `omega.zsh`.
5. Back up and inject the managed `.zshrc` block.

If block injection fails after configuration and `omega.zsh` are updated, the result explicitly says that activation in `.zshrc` remains pending. Success clears the in-session pending state and returns Home to `Ready`.

## 10. Configuration schema and migration

The configuration schema advances from v1 to v2. V2 adds at least:

- Display name.
- Icon mode.
- Prompt layout.
- Prompt symbol.
- Stable theme identifier.
- Optional Circuit variant.
- Separate Compatible and Nerd Font segment icons.
- Metadata needed to determine whether the active prompt differs from a built-in preset.

Migration rules:

- Back up the v1 file before writing v2.
- Preserve existing prompt segment values and order.
- Preserve existing plugins and trust state.
- Set migrated display name to empty so existing prompts continue showing the real username.
- Do not add recommended plugins to migrated configurations.
- Map known existing theme names without changing their rendered prompt unexpectedly.
- Reject unsupported future schema versions.
- On migration failure, leave the original file intact.

Fresh v2 configurations use `display_name = "user"`, Compatible icons, and the recommended plugin selections pending first-run confirmation.

## 11. Code organization

The current monolithic `internal/tui/tui_v2.go` is split into focused units:

```text
internal/tui/
├── app.go
├── navigation.go
├── styles.go
├── layout.go
├── components/
├── screens/
│   ├── home.go
│   ├── prompt.go
│   ├── themes.go
│   ├── plugins.go
│   └── preview.go
└── operations/
```

The root model owns screen routing, pending configuration, modal state, dimensions, and asynchronous operation messages. Screens own their local selection and editor state through explicit interfaces. Components contain reusable tab bar, status, list, modal, and form rendering.

Additional domain packages:

- `internal/themes`: embedded catalog, validation, preset application, modification detection.
- `internal/fonts`: manifest, download verification, Termux/Linux installation, cache, and restore.
- `internal/plugins`: curated catalog layered onto the existing manual plugin manager.

No network or filesystem mutation occurs directly from a View method. Slow or mutating operations run through Bubble Tea commands and return typed messages.

## 12. Error handling

- Expected user problems use short actionable messages.
- Raw paths, checksums, command output, and wrapped errors appear under Technical details and in the existing rotating log.
- Forms keep their values after validation errors.
- Busy operations disable only conflicting actions and expose progress.
- Cancellation never leaves a partially written font file as the active Termux font.
- Plugin and font download failures are recoverable without discarding prompt edits.
- Apply and restore results distinguish complete success, partial completion, and no mutation.

## 13. Verification

Required automated checks:

- V1-to-v2 migration, backup, preservation, and failure behavior.
- Fresh defaults versus migrated defaults.
- Navigation and key ownership in narrow and wide sizes.
- Resize behavior without lost selection or input focus.
- Golden prompt snapshots for all 12 themes in Compatible and Nerd Font modes, including all Circuit variants.
- Theme apply/preserve/modified behavior.
- Plugin catalog status and deterministic load phases.
- Font checksum, cache, Termux backup/install/restore, and Linux user install behavior through isolated fake services.
- Review modal and apply outcomes, including partial activation failure.
- Integration tests with an isolated `HOME`.
- Proof that preview, navigation, and Doctor read-only paths do not mutate `.zshrc`.

Release verification:

- Formatting.
- `go vet`.
- Race tests.
- ShellCheck.
- Existing test and healthcheck scripts.
- Real Linux smoke.
- Real Termux smoke at compact width.
- Manual confirmation that Compatible mode renders without Nerd Font glyphs.
