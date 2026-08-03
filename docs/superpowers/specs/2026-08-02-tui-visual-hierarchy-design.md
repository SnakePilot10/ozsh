# ozsh TUI visual hierarchy refinement

## Context

The current TUI is functionally complete and works on Termux, but the interface still feels visually compressed. The logo runs into the tab strip, inactive tabs and secondary text are too dim, section boundaries are weak, and theme metadata can be mistaken for part of the prompt preview.

This refinement stays on `fix/theme-gallery-v2` and does not change the five-screen information architecture or theme behavior.

## Goals

1. Create a clear visual separation between the `ozsh` brand and navigation tabs.
2. Increase readability and scanning speed without wasting vertical space on a phone.
3. Separate theme palette metadata from the rendered prompt preview.
4. Give active, inactive, focused, informational, and disabled states distinct contrast levels.
5. Preserve responsive behavior with the Termux keyboard open.
6. Keep `View()` pure and retain all side effects inside `tea.Cmd`.

## Non-goals

- No new screens or theme families.
- No redesign of prompt generation.
- No broad refactor of unrelated application logic.
- No fixed terminal dimensions.
- No manual ANSI escape sequences.

## Visual system

### Adaptive palette

Define semantic `lipgloss.AdaptiveColor` tokens instead of isolated fixed colors:

- `accent`: active tab, focused row, important success state.
- `text`: primary labels and values.
- `subtle`: subtitles and contextual descriptions.
- `muted`: key hints and inactive metadata, still readable.
- `surface`: selected tabs and preview containers.
- `border`: default panel border.
- `focusBorder`: active modal or focused panel border.
- `danger`, `warning`, `success`: status feedback.

Dark-background values keep the existing cyan-led identity. Light-background values maintain equivalent contrast.

### Spacing scale

Use a small internal spacing scale rather than scattered literal spaces:

- `spaceXS = 1`: tightly related inline items.
- `spaceSM = 2`: logo-to-navigation and label-to-value separation.
- `spaceMD = 1 blank line`: related content groups.
- `spaceLG = 2 blank lines`: major sections only when terminal height allows.

Vertical spacing becomes responsive. Compact terminals use one blank line between major blocks; taller terminals may use two.

## Header and navigation

Build the header as two explicit Lip Gloss blocks joined horizontally:

1. `logoStyle.Render("ozsh")`
2. a tab strip rendered by `renderTabs()`

The logo receives right margin, so it is never visually fused to `Home`.

Tab states:

- Active: bold accent foreground, subtle background, horizontal padding, visible underline or lower border.
- Inactive: readable secondary foreground, no washed-out opacity effect.
- Focus/navigation context: active state remains obvious even on low-quality mobile terminal themes.

At narrow widths, tab labels remain on one line while horizontal padding is reduced. The logo-to-tabs gap never falls below two cells.

## Section hierarchy

Each screen uses shared helpers:

- `renderSectionHeader(title, subtitle)`
- `renderKeyValue(label, value)`
- `renderHint(text)`
- `renderStatus(message)`

Titles, subtitles, body content, contextual status, and global help each occupy visually separate layers. The repeated `applied` status will become a compact status line with an icon and semantic color rather than an isolated dim word.

## Theme gallery preview

The selected-theme area is split into four groups:

1. Theme name and optional variant badge.
2. Description.
3. Palette metadata labeled `Palette`.
4. Prompt output labeled `Preview` and rendered inside a subtle padded container.

Palette swatches and hex values are no longer adjacent to the prompt output. This prevents users from reading color metadata as prompt content.

The preview container has:

- one cell of horizontal padding;
- a subtle border or surface background;
- responsive width clamped to the available content width;
- no manual ANSI styling beyond what the prompt renderer already produces.

Circuit variant summary remains visible but is wrapped or truncated safely on narrow terminals.

## Prompt, Plugins, Preview, and Home screens

- Prompt: increase spacing between identity controls, segment list, prompt preview, contextual help, and status.
- Plugins: keep descriptions readable, align state labels, and strengthen active/inactive selection contrast.
- Preview: separate scenario selector, editable context, prompt output, and field help. The prompt output receives the same `Preview` container used by Themes.
- Home: align status key/value rows into stable columns and separate actions from system status.

## Responsive behavior

`tea.WindowSizeMsg` remains the source of terminal dimensions.

The layout derives an inner content width from the panel border and padding. Helpers wrap descriptions using Lip Gloss width constraints. On narrow Termux windows:

- tab padding shrinks;
- long subtitles wrap;
- theme descriptions use the available width;
- the preview box never exceeds the panel;
- nonessential vertical gaps collapse before content is hidden.

## Architecture

Introduce a focused visual layer inside `internal/tui`:

- semantic adaptive color tokens;
- reusable component styles;
- pure render helpers for header, section headers, key/value rows, status, hints, and preview boxes;
- optional layout metrics computed from `Model.width` and `Model.height`.

No render helper mutates `Model`. Existing update logic and asynchronous commands remain unchanged.

## Testing

Add regression tests for:

1. a visible gap between `ozsh` and `Home`;
2. active and inactive tabs using distinct rendered output;
3. `Palette` and `Preview` labels appearing as separate theme groups;
4. palette hex values not sharing the prompt preview line;
5. narrow-width views staying within the configured width after ANSI stripping;
6. all five screens retaining their main headings and global help;
7. light/dark adaptive color tokens being used by the visual styles;
8. existing theme gallery, race, coverage, Android/Termux build, lint, and security checks remaining green.

## Acceptance criteria

- The brand and tabs are visually separate at phone widths.
- Inactive tabs remain clearly readable.
- Screens have consistent spacing and hierarchy.
- Palette metadata cannot reasonably be confused with prompt content.
- The TUI remains usable with the Android keyboard open.
- No behavioral regression in navigation, applying, themes, plugins, preview editing, or backups.
- GitHub Actions is fully green before the PR is marked ready for review.
