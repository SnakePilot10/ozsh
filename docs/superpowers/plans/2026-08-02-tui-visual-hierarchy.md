# ozsh TUI Visual Hierarchy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refine the existing five-screen ozsh TUI so branding, navigation, sections, metadata, previews, status, and help remain clearly separated and readable on narrow Termux terminals.

**Architecture:** Add a small pure visual layer in `internal/tui` containing semantic adaptive colors, reusable Lip Gloss styles, and width-aware render helpers. Keep all existing state transitions and commands intact, then migrate the current screen renderers to those helpers under regression tests.

**Tech Stack:** Go 1.25.12, Bubble Tea, Lip Gloss, Bubbles textinput, VHS, GitHub Actions.

## Global Constraints

- Keep the existing five screens: Home, Prompt, Themes, Plugins, Preview.
- Keep `View()` pure and keep all side effects inside `tea.Cmd`.
- Continue deriving dimensions from `tea.WindowSizeMsg`; do not hardcode a required terminal size.
- Use `lipgloss.AdaptiveColor` for semantic visual tokens.
- Do not emit raw ANSI escape codes.
- Preserve Termux usability with the Android keyboard open.
- Do not change prompt generation, theme families, plugin behavior, backup behavior, or apply semantics.
- Keep the PR as a draft until visual testing and all CI jobs pass.

---

### Task 1: Visual regression tests

**Files:**
- Create: `internal/tui/visual_hierarchy_test.go`

**Interfaces:**
- Consumes: `NewModel(*config.Config) Model`, `Model.View() string`, `Model.themes() string`, `Model.setTab(int)`.
- Produces: regression coverage for header spacing, tab contrast, preview grouping, all-screen headings, and narrow rendering.

- [ ] **Step 1: Write failing tests for header and tab hierarchy**

```go
func TestHeaderSeparatesBrandFromTabs(t *testing.T) {
    model := NewModel(config.Default())
    model.width, model.height = 72, 30
    first := strings.Split(model.View(), "\n")[0]
    if strings.Contains(first, "ozshHome") || !strings.Contains(first, "ozsh  ") {
        t.Fatalf("header lacks stable brand gap: %q", first)
    }
}

func TestActiveAndInactiveTabsRenderDifferently(t *testing.T) {
    model := NewModel(config.Default())
    active := renderTab("Home", true, false)
    inactive := renderTab("Prompt", false, false)
    if active == inactive || lipgloss.Width(active) == 0 || lipgloss.Width(inactive) == 0 {
        t.Fatalf("tab states are not visually distinct: active=%q inactive=%q", active, inactive)
    }
}
```

- [ ] **Step 2: Write failing tests for Palette and Preview separation**

```go
func TestThemeDetailsSeparatePaletteFromPreview(t *testing.T) {
    model := NewModel(config.Default())
    model.setTab(tabThemes)
    model.width, model.height = 72, 32
    view := model.themes()
    palette := strings.Index(view, "Palette")
    preview := strings.Index(view, "Preview")
    if palette < 0 || preview < 0 || palette >= preview {
        t.Fatalf("theme metadata hierarchy missing:\n%s", view)
    }
    paletteLine := strings.Split(view[palette:], "\n")[1]
    if strings.Contains(paletteLine, "~/dev/ozsh") {
        t.Fatalf("prompt output leaked into palette metadata: %q", paletteLine)
    }
}
```

- [ ] **Step 3: Write failing narrow-width and all-screen tests**

```go
func TestFiveScreensRemainReadableAtNarrowWidth(t *testing.T) {
    headings := []string{"Welcome", "Prompt", "Theme gallery", "Plugins", "Preview"}
    for tab, heading := range headings {
        model := NewModel(config.Default())
        model.width, model.height = 58, 28
        model.setTab(tab)
        view := model.View()
        if !strings.Contains(view, heading) || !strings.Contains(view, "apply") || !strings.Contains(view, "quit") {
            t.Fatalf("tab %d lost hierarchy or help:\n%s", tab, view)
        }
        for _, line := range strings.Split(view, "\n") {
            if lipgloss.Width(line) > model.width {
                t.Fatalf("tab %d line width %d exceeds terminal %d: %q", tab, lipgloss.Width(line), model.width, line)
            }
        }
    }
}
```

- [ ] **Step 4: Run tests and confirm they fail**

Run: `go test ./internal/tui -run 'TestHeaderSeparatesBrandFromTabs|TestActiveAndInactiveTabsRenderDifferently|TestThemeDetailsSeparatePaletteFromPreview|TestFiveScreensRemainReadableAtNarrowWidth'`

Expected: FAIL because semantic tab renderers and labeled preview groups do not exist yet.

- [ ] **Step 5: Commit tests**

```bash
git add internal/tui/visual_hierarchy_test.go
git commit -m "test: define TUI visual hierarchy expectations"
```

---

### Task 2: Semantic visual layer

**Files:**
- Create: `internal/tui/visual_styles.go`
- Modify: `internal/tui/tui_v2.go`
- Test: `internal/tui/visual_hierarchy_test.go`

**Interfaces:**
- Produces: `renderTab(label string, active bool, compact bool) string`, `renderHeader(active int, width int) string`, `renderSectionHeader(title, subtitle string) string`, `renderKeyValue(label, value string) string`, `renderHint(text string) string`, `renderStatus(text string, failed bool) string`, `renderPreviewBox(label, content string, width int) string`, `innerWidth(terminalWidth int) int`.

- [ ] **Step 1: Define adaptive semantic colors and component styles**

```go
var palette = struct {
    Accent, Text, Subtle, Muted, Surface, Border, FocusBorder lipgloss.AdaptiveColor
    Success, Warning, Danger                                lipgloss.AdaptiveColor
}{
    Accent:      lipgloss.AdaptiveColor{Light: "#006A73", Dark: "#27E6E6"},
    Text:        lipgloss.AdaptiveColor{Light: "#20242B", Dark: "#F2F4F8"},
    Subtle:      lipgloss.AdaptiveColor{Light: "#4D5968", Dark: "#A8B0C0"},
    Muted:       lipgloss.AdaptiveColor{Light: "#667085", Dark: "#7F899D"},
    Surface:     lipgloss.AdaptiveColor{Light: "#E9F5F5", Dark: "#121B20"},
    Border:      lipgloss.AdaptiveColor{Light: "#779092", Dark: "#31575C"},
    FocusBorder: lipgloss.AdaptiveColor{Light: "#007F89", Dark: "#27E6E6"},
    Success:     lipgloss.AdaptiveColor{Light: "#16794A", Dark: "#5FD79A"},
    Warning:     lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#E5C07B"},
    Danger:      lipgloss.AdaptiveColor{Light: "#B42318", Dark: "#FF5C75"},
}
```

- [ ] **Step 2: Implement pure header, section, status, and preview helpers**

Use `lipgloss.JoinHorizontal`, style margins rather than literal alignment padding, and clamp preview width to at least 12 cells and at most the current content width.

- [ ] **Step 3: Replace fixed global colors in `tui_v2.go`**

Set `accentStyle`, `mutedStyle`, `errorStyle`, and `panelStyle` from the semantic adaptive palette. Keep the identifiers temporarily for compatibility with existing renderers and tests.

- [ ] **Step 4: Run the focused tests**

Run: `go test ./internal/tui -run 'TestHeaderSeparatesBrandFromTabs|TestActiveAndInactiveTabsRenderDifferently'`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/visual_styles.go internal/tui/tui_v2.go internal/tui/visual_hierarchy_test.go
git commit -m "feat: add adaptive TUI visual system"
```

---

### Task 3: Header, panel, status, and help composition

**Files:**
- Modify: `internal/tui/tui_v2_part2.go`
- Test: `internal/tui/visual_hierarchy_test.go`

**Interfaces:**
- Consumes: visual helpers from Task 2.
- Produces: a width-aware `View()` with separated branding/navigation and semantic status/help layers.

- [ ] **Step 1: Replace inline logo/tab concatenation**

Use:

```go
b.WriteString(renderHeader(m.tab, m.contentWidth()))
b.WriteString("\n\n")
```

Remove `Model.renderTabs()` after all callers migrate.

- [ ] **Step 2: Render transient status semantically**

Replace the detached dim message with `renderStatus(m.msg, failed)` and keep exactly one blank line around it.

- [ ] **Step 3: Render global help as its own footer layer**

Use `renderHint("Ctrl+A apply  ·  ? help  ·  Ctrl+C quit")` so the footer remains readable but secondary.

- [ ] **Step 4: Fix width accounting**

Derive content width from panel frame size, make the panel no wider than the terminal, and retain ANSI-aware `lipgloss.Width` checks.

- [ ] **Step 5: Run focused tests**

Run: `go test ./internal/tui -run 'TestHeaderSeparatesBrandFromTabs|TestFiveScreensRemainReadableAtNarrowWidth'`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/tui_v2_part2.go internal/tui/visual_hierarchy_test.go
git commit -m "feat: clarify TUI header status and help"
```

---

### Task 4: Home, Prompt, and Preview hierarchy

**Files:**
- Modify: `internal/tui/tui_v2_part2.go`
- Test: `internal/tui/visual_hierarchy_test.go`

**Interfaces:**
- Consumes: `renderSectionHeader`, `renderKeyValue`, `renderHint`, and `renderPreviewBox`.
- Produces: clearly grouped Home, Prompt, and Preview screens.

- [ ] **Step 1: Refactor Home into Status and Actions groups**

Render stable key/value rows and leave one blank line between system status and actions.

- [ ] **Step 2: Refactor Prompt controls and segment list**

Use the shared section header, readable labels, one blank line before Segments, and a labeled `Preview` box before the screen-specific hint.

- [ ] **Step 3: Refactor Preview scenarios, context fields, and output**

Render `Scenarios`, `Context`, and the final `Preview` as separate groups. Keep editing behavior unchanged.

- [ ] **Step 4: Run screen tests**

Run: `go test ./internal/tui -run 'TestFiveScreensRemainReadableAtNarrowWidth'`

Expected: PASS for Home, Prompt, and Preview.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/tui_v2_part2.go internal/tui/visual_hierarchy_test.go
git commit -m "feat: add hierarchy to core TUI screens"
```

---

### Task 5: Theme and plugin visual grouping

**Files:**
- Modify: `internal/tui/tui_v2_part3.go`
- Test: `internal/tui/visual_hierarchy_test.go`
- Test: `internal/tui/theme_gallery_regression_test.go`

**Interfaces:**
- Consumes: visual helpers from Task 2.
- Produces: separate theme identity, description, Palette, Preview, and hint groups; readable plugin states and descriptions.

- [ ] **Step 1: Split theme details into labeled groups**

Render selected theme identity and variant badge, description, `Palette`, then `Preview`. Put prompt output only inside the preview box.

- [ ] **Step 2: Keep Circuit summary narrow-safe**

Use a width-constrained style for the Circuit variant summary rather than allowing it to bleed into the list.

- [ ] **Step 3: Improve plugin row hierarchy**

Render selected rows with a focus marker/style, make state labels readable, and keep descriptions secondary but not washed out.

- [ ] **Step 4: Run regression tests**

Run: `go test ./internal/tui -run 'TestThemeGallery|TestThemeDetailsSeparatePaletteFromPreview|TestFiveScreensRemainReadableAtNarrowWidth'`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/tui_v2_part3.go internal/tui/visual_hierarchy_test.go internal/tui/theme_gallery_regression_test.go
git commit -m "feat: separate theme metadata from prompt previews"
```

---

### Task 6: VHS visual demo and full verification

**Files:**
- Create: `docs/screencasts/tui-visual-hierarchy.tape`
- Modify: `docs/superpowers/specs/2026-08-02-tui-visual-hierarchy-design.md` only if implementation details changed.

**Interfaces:**
- Consumes: final TUI behavior.
- Produces: reproducible visual walkthrough for Termux-sized rendering.

- [ ] **Step 1: Add VHS tape**

```tape
Output docs/screencasts/tui-visual-hierarchy.gif
Set Shell "zsh"
Set FontSize 16
Set Width 720
Set Height 1280
Set TypingSpeed 35ms

Type "go run ./cmd/ozsh tui"
Enter
Sleep 2s
Type "3"
Sleep 2s
Type "]"
Sleep 1s
Type "5"
Sleep 2s
Type "2"
Sleep 2s
Type "1"
Sleep 2s
Ctrl+C
```

- [ ] **Step 2: Run format and focused package tests**

Run: `gofmt -w internal/tui/*.go && go test ./internal/tui ./internal/themes ./internal/prompt`

Expected: PASS.

- [ ] **Step 3: Run complete verification**

Run: `go test -race ./...`

Run: `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1`

Run: `go build -buildvcs=false ./...`

Run: `CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -buildvcs=false ./cmd/ozsh`

Run: `scripts/lint.sh --check`

Expected: all commands pass and total coverage remains at or above 70%.

- [ ] **Step 4: Update PR description with visual refinement and evidence**

Mention the semantic adaptive palette, spacing system, separated Palette/Preview blocks, narrow Termux checks, and final CI run.

- [ ] **Step 5: Commit**

```bash
git add docs/screencasts/tui-visual-hierarchy.tape docs/superpowers/specs/2026-08-02-tui-visual-hierarchy-design.md
git commit -m "docs: add TUI visual hierarchy demo"
```
