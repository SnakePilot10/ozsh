# Fullscreen Contextual TUI Layout Design

## Goal

Turn the current content-height ozsh panel into a responsive full-screen application that uses the available terminal height for a live preview and contextual details, while remaining comfortable in Termux with the Android keyboard open.

## Approved direction

The approved reference is the Circuit Neon mockup discussed on 2026-08-02. It is a visual target, not a pixel-perfect browser reproduction. The implementation must preserve terminal-cell constraints and existing keyboard behavior.

## Global layout

The application is divided into four vertical regions:

1. Header with the `ozsh` brand, five tabs, and a compact help affordance.
2. Main workspace that expands to consume all remaining height.
3. Optional transient status line.
4. Footer anchored to the bottom of the panel.

The panel uses the dimensions stored from `tea.WindowSizeMsg`. `View()` remains pure and derives all layout measurements from `Model.width` and `Model.height`.

## Responsive breakpoints

- **Wide:** `width >= 72`. The workspace may use two columns.
- **Compact:** `width < 72`. Sections stack vertically and avoid horizontal tables.
- **Short:** usable content height below 22 rows. Contextual details collapse before the live preview and global footer.

No line may exceed the terminal width. The panel must never render a negative width or height.

## Prompt screen

### Wide layout

- Top row: Configuration on the left and Segments on the right.
- Middle: a live prompt preview spanning the full workspace width.
- Bottom: details for the currently selected segment.

The selected segment is visually prominent and displays:

- enabled state;
- display label;
- description;
- compatible or Nerd Font icon;
- foreground color;
- bold state;
- runtime condition.

### Compact layout

The same information is rendered as stacked groups in this order:

1. Configuration summary.
2. Segments list.
3. Live preview.
4. Selected segment details.

When height is constrained, segment details collapse to a one-line summary.

## Other screens

- **Home:** system summary and quick actions use two columns when wide, then stack when compact.
- **Themes:** theme list uses the left column; description, palette, and preview use the right column. Compact mode stacks them.
- **Plugins:** selected plugin details occupy the contextual lower panel and explain installation, health, trust, and load state.
- **Preview:** scenarios and editable context use the upper workspace; the live prompt preview expands below them.

## Visual language

Use semantic `lipgloss.AdaptiveColor` tokens. The active element must remain identifiable without color. Borders, padding, labels, badges, selection markers, and whitespace establish hierarchy. Do not add raw ANSI escape codes.

## Footer

The footer is pinned to the bottom of the panel. It presents screen-specific actions plus global help and quit keys. It must remain visible whenever terminal height permits.

## Architecture

- `Model.Update` continues to own state transitions and side effects through `tea.Cmd`.
- `Model.View` stays pure.
- Layout calculations live in focused helpers returning immutable measurement values.
- Screen renderers consume a layout specification instead of mutating model state.
- Contextual-detail renderers are independent from list renderers so they can collapse responsively.

## Testing

Add regression tests for:

- panel height matching the available terminal height;
- footer placement near the bottom;
- wide Prompt screen containing Configuration, Segments, Preview, and Details;
- compact Prompt screen preserving those four semantic regions;
- selected segment details changing with cursor movement;
- all five screens staying inside width and height bounds;
- short-terminal collapse behavior;
- Android/Termux ARM64 cross-build.

## Non-goals

- Mouse interaction.
- Pixel-identical reproduction of the browser mockup.
- New configuration fields unrelated to layout.
- Changing `.zshrc` application semantics.
- Merging PR #21 before visual testing on the real Termux device.
