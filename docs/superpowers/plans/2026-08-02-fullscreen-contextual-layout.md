# Fullscreen Contextual TUI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make ozsh consume the available terminal height and use the lower workspace for an expanded live preview plus contextual details on every screen.

**Architecture:** Add pure layout-measurement and composition helpers in a focused file, then route the existing screen state through new responsive renderers. Keep `Update` and all side effects unchanged; only `View` composition and screen presentation change. Wide terminals use two-column workspaces, compact Termux widths stack the same semantic regions, and short terminals collapse contextual details before hiding the preview or footer.

**Tech Stack:** Go 1.25, Bubble Tea, Lip Gloss, Bubbles textinput, VHS, GitHub Actions.

## Global Constraints

- Work on branch `fix/theme-gallery-v2` and PR #21; do not merge before real Termux visual review.
- Keep `View()` pure and preserve asynchronous work behind `tea.Cmd`.
- Use dimensions stored from `tea.WindowSizeMsg`; never hardcode a single terminal size.
- Use `lipgloss.AdaptiveColor`; do not emit raw ANSI escape sequences.
- Wide breakpoint is `width >= 72`; compact mode is `width < 72`.
- Preserve five screens in this order: Home, Prompt, Themes, Plugins, Preview.
- Preserve Review & Apply semantics and do not write `.zshrc` before final confirmation.
- Preserve Android/Termux ARM64 cross-build support.

---

### Task 1: Full-screen measurement and composition

**Files:**
- Create: `internal/tui/fullscreen_layout.go`
- Modify: `internal/tui/tui_v2_part2.go`
- Test: `internal/tui/fullscreen_layout_test.go`

**Interfaces:**
- Produces: `type layoutSpec struct`, `func (m Model) layout() layoutSpec`, `func composeFullscreen(header, body, status, footer string, spec layoutSpec) string`, `func fitHeight(value string, width, height int) string`, `func screenFooter(tab int) string`.
- Consumes: `Model.width`, `Model.height`, `panelStyle`, `fitBlock`, `renderHeader`, and `renderHint`.

- [ ] **Step 1: Write failing tests**

Add tests that set a model to `72x32` and assert the rendered panel height is at most 32 rows, the footer appears in the final four rows, and no rendered line exceeds 72 cells. Add a `58x28` compact case and a `72x18` short case.

- [ ] **Step 2: Run tests and verify failure**

Run: `go test ./internal/tui -run 'TestFullscreen|TestFooter|TestShortLayout' -count=1`

Expected: FAIL because the current panel ends at content height and has no `layoutSpec`.

- [ ] **Step 3: Implement pure layout helpers**

Create `layoutSpec` with terminal, panel, content, workspace, footer, wide, and short measurements. Compute panel height from `Model.height - panelStyle.GetVerticalFrameSize()`, clamp every dimension to at least one cell, and reserve footer/status rows before assigning workspace height.

- [ ] **Step 4: Route `View()` through full-screen composition**

Refactor `View()` so screen renderers return only workspace content. Compose header, body, transient status, and a screen-specific footer with vertical padding inserted between body and footer. Set both panel width and height through Lip Gloss.

- [ ] **Step 5: Run focused tests**

Run: `go test ./internal/tui -run 'TestFullscreen|TestFooter|TestShortLayout' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

Commit: `feat: add fullscreen TUI composition`

### Task 2: Prompt workspace with live preview and selected-segment details

**Files:**
- Create: `internal/tui/prompt_workspace.go`
- Modify: `internal/tui/tui_v2_part2.go`
- Test: `internal/tui/prompt_workspace_test.go`

**Interfaces:**
- Produces: `func (m Model) promptWorkspace(spec layoutSpec) string`, `func (m Model) promptConfigurationPanel(width int) string`, `func (m Model) promptSegmentsPanel(width, maxRows int) string`, `func (m Model) selectedSegmentPanel(width int, compact bool) string`, `func segmentDescription(name string) string`, `func segmentCondition(name string, segment config.SegmentConfig) string`.
- Consumes: `m.previewConfig()`, `prompt.Simulated`, `promptSegmentIcon`, `segmentLabel`, `m.cursor`, and `m.cfg.Prompt`.

- [ ] **Step 1: Write failing Prompt tests**

Add a wide test asserting the plain-text view contains `Configuration`, `Segments`, `Live preview`, and `Selected segment`. Add a compact test at width 58 asserting those labels occur in stacked order. Move the cursor and assert selected details change from User to Directory.

- [ ] **Step 2: Run tests and verify failure**

Run: `go test ./internal/tui -run 'TestPromptWorkspace|TestSelectedSegment' -count=1`

Expected: FAIL because the existing Prompt screen has no contextual lower panel.

- [ ] **Step 3: Implement configuration and segments panels**

Render the configuration summary with display name, layout, symbol, icons, right prompt, separator, and heavy-segment state. Render segment rows with a non-color selection marker, enabled state, icon, label, and enough contrast to distinguish focus.

- [ ] **Step 4: Implement expanded live preview**

Render the simulated prompt in a bordered panel spanning the workspace width. Give it a minimum useful height and vertically pad it when the terminal has spare rows.

- [ ] **Step 5: Implement selected-segment details**

Show state, description, icon mode/value, foreground color, bold state, and runtime condition. In short mode, collapse this to a one-line summary while retaining the selected label and state.

- [ ] **Step 6: Assemble responsive Prompt workspace**

At widths of at least 72 cells, join Configuration and Segments horizontally with a two-cell gutter, then place Live preview and Selected segment below. At compact widths, stack all four groups. Preserve name editing as a focused modal-like workspace.

- [ ] **Step 7: Run focused tests**

Run: `go test ./internal/tui -run 'TestPromptWorkspace|TestSelectedSegment' -count=1`

Expected: PASS.

- [ ] **Step 8: Commit**

Commit: `feat: expand prompt workspace`

### Task 3: Contextual layouts for Home, Themes, Plugins, and Preview

**Files:**
- Create: `internal/tui/contextual_screens.go`
- Modify: `internal/tui/tui_v2_part2.go`
- Test: `internal/tui/contextual_screens_test.go`

**Interfaces:**
- Produces: `func (m Model) homeWorkspace(spec layoutSpec) string`, `func (m Model) themesWorkspace(spec layoutSpec) string`, `func (m Model) pluginsWorkspace(spec layoutSpec) string`, `func (m Model) previewWorkspace(spec layoutSpec) string`.
- Consumes: existing theme, plugin, preview, status, and input helpers without changing their state semantics.

- [ ] **Step 1: Write failing semantic-region tests**

For every screen, assert required labels are present at wide and compact dimensions. Themes must expose list, description, palette, and preview. Plugins must expose list and selected-plugin details. Preview must expose scenarios, context, and an expanded live preview. Home must expose system summary and quick actions.

- [ ] **Step 2: Run tests and verify failure**

Run: `go test ./internal/tui -run 'TestHomeWorkspace|TestThemesWorkspace|TestPluginsWorkspace|TestPreviewWorkspace' -count=1`

Expected: FAIL because the current screens are content-height renderers.

- [ ] **Step 3: Implement Home and Themes**

Use two columns on wide terminals and stacked groups on compact terminals. Keep theme variants directly selectable. Put theme description, palette, and preview in the contextual side panel.

- [ ] **Step 4: Implement Plugins and Preview**

Show selected plugin health, trust, installation, and load information in the contextual panel. Expand the Preview screen’s live output and retain editable fields and scenario switching.

- [ ] **Step 5: Run focused tests**

Run: `go test ./internal/tui -run 'TestHomeWorkspace|TestThemesWorkspace|TestPluginsWorkspace|TestPreviewWorkspace' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

Commit: `feat: add contextual screen layouts`

### Task 4: Visual regression coverage and VHS walkthrough

**Files:**
- Modify: `internal/tui/visual_hierarchy_test.go`
- Modify: `docs/screencasts/tui-visual-hierarchy.tape`
- Test: `internal/tui/fullscreen_layout_test.go`, `internal/tui/prompt_workspace_test.go`, `internal/tui/contextual_screens_test.go`

**Interfaces:**
- Consumes all renderers from Tasks 1–3.
- Produces stable terminal-size scenarios for CI and documentation.

- [ ] **Step 1: Extend width and height invariants**

Test all five tabs at `100x34`, `72x30`, `58x28`, and `48x18`. Assert no line exceeds width and no view exceeds height.

- [ ] **Step 2: Update VHS tape**

Record navigation through Prompt, Themes, Plugins, and Preview at a wide size, then resize below 72 columns and demonstrate the stacked layout.

- [ ] **Step 3: Run the full repository test suite**

Run: `go test ./... -count=1`

Expected: PASS.

- [ ] **Step 4: Run project validation scripts**

Run:

```bash
scripts/lint.sh --check
scripts/test.sh
scripts/build.sh
scripts/healthcheck.sh
scripts/release-smoke.sh
scripts/install-smoke.sh
```

Expected: all commands exit 0.

- [ ] **Step 5: Confirm Android/Termux build**

Run: `CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -buildvcs=false ./cmd/ozsh`

Expected: exit 0.

- [ ] **Step 6: Commit**

Commit: `test: cover fullscreen responsive TUI`

### Task 5: PR verification and visual handoff

**Files:**
- Modify: PR #21 description only.

- [ ] **Step 1: Verify GitHub Actions**

Confirm quality, tests/coverage/race/build/smoke, Android/Termux, and security jobs all conclude successfully on the latest head SHA.

- [ ] **Step 2: Update PR description**

Document the full-screen layout, responsive breakpoint, contextual panels, selected-segment details, VHS tape, and exact CI evidence.

- [ ] **Step 3: Keep PR in draft**

Do not merge. Provide Termux checkout and build commands for final device review.
