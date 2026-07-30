# Charm TUI Redesign

**Status:** Approved design
**Date:** 2026-07-30
**Branch:** `feat/charm-tui-redesign`

## Context

`ozsh` already depends on Bubble Tea v1, Bubbles v1, and Lip Gloss v1, but the current 919-line TUI model uses only the Bubbles `textinput` component. Navigation, lists, contextual help, scrolling, and operation status are implemented manually. This makes the interface harder to use in a small Termux terminal and makes the Builder difficult to extend safely.

## Goals

- Make the TUI task-oriented, keyboard-first, and fully usable at 80x24.
- Expose every user-configurable prompt, segment, theme, and plugin-item setting.
- Keep edits in an in-memory draft until the user explicitly saves or applies them.
- Replace manual UI primitives with the Bubbles components already in the dependency graph.
- Split the monolithic TUI into focused, independently tested screen models.
- Preserve the existing CLI behavior and safe shell-writing rules.

## Non-goals

- Migrating Bubble Tea, Bubbles, or Lip Gloss to v2.
- Adding Huh, Glamour, Fang, or other Charm dependencies.
- Changing prompt-generation semantics or the config schema.
- Changing the supported plugin engine from `manual`.
- Making the config schema `version` editable.
- Requiring mouse input.

## Experience and responsive layout

The root screen is a compact task menu implemented with `bubbles/list`:

1. Home
2. Build prompt
3. Preview
4. Apply
5. Doctor
6. Themes
7. Plugins

Each task opens as a full-screen view. `Esc` returns to the previous screen without discarding the current draft. `q` from Home requests application exit. If the draft is dirty, exit offers Save, Discard, and Cancel choices.

The baseline layout is 80x24. Every screen contains a short breadcrumb/header, one main content area, and contextual help generated from its actual key bindings. At 110 columns or wider, Builder, Themes, and Plugins may show a live preview beside the primary content. Below 110 columns the preview moves below the content or to a dedicated action. Below 60x18, the TUI shows a resize notice rather than a clipped, unusable form.

Mouse support may be added where Bubbles provides it naturally, but every operation must remain keyboard-accessible.

## Architecture

### Root application model

The root `AppModel` owns:

- terminal dimensions;
- navigation stack and active screen;
- `savedConfig`, the last successfully loaded or saved configuration;
- `draftConfig`, the editable deep copy shared by Builder, Preview, and Themes;
- dirty state derived from the saved and draft values;
- global status and error messages;
- save, apply, and exit-confirmation state.

Screen models own only screen-specific UI state such as cursor position, filter text, text inputs, viewport position, and spinner state. They communicate domain intentions to the root through typed Bubble Tea messages. The root remains the single owner of configuration mutation, validation, persistence, and navigation.

Representative messages include navigation, draft replacement, save request/result, apply request/result, discard confirmation, and status notification. Long-running filesystem operations return through `tea.Cmd`; screen rendering never performs writes.

### Screen decomposition

The existing `tui_v2.go` is replaced incrementally by focused files under `internal/tui/`:

- `app.go`: root model, routing, shared session state, and global messages.
- `keys.go`: canonical key bindings used by both Update logic and help rendering.
- `layout.go`: responsive dimensions and shared shell chrome.
- `screen_home.go`: task menu.
- `screen_builder.go`: segment list, prompt settings, and editor routing.
- `screen_preview.go`: simulated and real-context preview fields.
- `screen_apply.go`: change review, confirmation, spinner, and result state.
- `screen_doctor.go`: checks and safe fixes.
- `screen_themes.go`: preset/custom theme selection and editing.
- `screen_plugins.go`: plugin list, add form, enablement, and trust controls.

Shared visual styles remain centralized. Screen files expose constructors and narrow methods; they do not load configuration or touch shell files directly.

## Charm components

The redesign stays on the currently installed v1 libraries and uses:

- `bubbles/list` for Home, segments, themes, and plugins, including filtering and pagination;
- `bubbles/key` plus `bubbles/help` as the single source of truth for shortcuts and contextual help;
- `bubbles/viewport` for previews, diffs, diagnostic details, and long error output;
- `bubbles/spinner` while save, apply, plugin, or diagnostic operations run;
- `bubbles/textinput` for colors, icons, separators, preview context, and plugin fields.

A table component is intentionally omitted from the first iteration because descriptive lists adapt better to 80-column terminals.

## Complete Builder

The Builder lists every segment from the supported segment registry/default configuration, including disabled segments absent from `Prompt.Order` and `Prompt.RightOrder`.

For each segment, the user can:

- enable or disable it;
- place it exclusively in the left prompt or right prompt;
- reorder it within that side;
- edit icon, foreground, background, and bold state;
- edit `show_success` when the segment supports that setting;
- restore the segment defaults;
- see a live simulated preview.

The Builder also exposes the user-configurable prompt settings: style, newline, right-prompt enablement, heavy-segment suppression, and separator.

The left and right order slices contain no duplicates. Moving a segment to one side removes it from the other. Disabling a segment preserves its last placement and position so re-enabling it is predictable. Validation runs after each committed field edit and before save or apply.

## Themes and plugins

Themes use a filterable preset list with live preview. The user can apply a preset, edit the theme name and all theme colors, restore the saved theme, or save the current values as the draft theme.

Plugins use a filterable descriptive list. The existing operations remain available: add, enable, disable, trust, untrust, and remove. Trust actions retain explicit confirmation and explain that third-party shell code will execute. The plugin engine remains fixed to `manual`.

## Draft, save, and apply flow

On startup, configuration is loaded once and deep-cloned into `savedConfig` and `draftConfig`.

- Navigation preserves the draft and performs no writes.
- Save validates and atomically persists `config.toml`, then refreshes `savedConfig`.
- Apply validates the draft and presents a scrollable review of the planned `config.toml`, generated `omega.zsh`, and `.zshrc` changes.
- No filesystem write occurs before explicit Apply confirmation.
- During Apply, input that could trigger another operation is disabled and a spinner is shown.
- Successful Apply refreshes both saved and draft state from the applied configuration.

The underlying operations retain their existing per-file atomic writes and backup behavior. The design does not claim a cross-file transaction. If an operation fails after modifying one file, the result identifies completed and failed steps, leaves the TUI open, and provides a retry or recovery path using the existing backups.

## Errors and accessibility

Field validation errors appear beside the relevant field and remain until corrected. Operational errors appear in a scrollable detail view and never terminate the TUI. Global messages use semantic text in addition to color, so success and failure are not distinguished by color alone.

The key map drives both event matching and displayed help, preventing stale shortcut text. Focus is always visible. Lists retain cursor and filter state when navigating back. The interface must remain usable without Nerd Fonts or true-color support.

## Testing strategy

### Unit tests

- Root routing, navigation stack, dirty-state transitions, and exit confirmation.
- Each screen's key handling and emitted domain messages.
- Builder invariants for enabling, placement, ordering, reset, and validation.
- Contextual help generated from the same bindings used by Update logic.
- Layout behavior at 80x24, at 110 columns or wider, and below 60x18.

### Integration tests

All filesystem tests use an isolated temporary `HOME`.

- Edit, navigate away, return, and retain the draft without writes.
- Save, discard, cancel exit, and reload persisted configuration.
- Review and cancel Apply with no changed files.
- Confirm Apply and verify `config.toml`, `omega.zsh`, `.zshrc`, and backups.
- Inject failures and verify accurate partial-result reporting and recovery guidance.
- Exercise plugin trust confirmations without sourcing third-party code in tests.

### Repository validation

Run:

```bash
scripts/lint.sh --check
scripts/test.sh
scripts/build.sh
scripts/healthcheck.sh
```

The existing race and coverage checks must continue to pass, aggregate coverage must remain above the repository's 70% gate, and Android/Termux ARM64 cross-build must remain green. Existing non-TUI CLI tests must remain unchanged and passing.

## Acceptance criteria

- The complete workflow is usable with a keyboard at 80x24 without horizontal clipping.
- All user-editable prompt, segment, theme, and plugin-item settings are reachable.
- Navigation does not write files or lose the in-memory draft.
- No Apply-related file changes occur before explicit confirmation.
- Help text always reflects the active bindings.
- Long previews, diffs, and errors are scrollable.
- Existing CLI commands, config schema, prompt generation, and shell safety behavior remain compatible.
- The required repository checks and Termux ARM64 build pass.
