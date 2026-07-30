# Charm TUI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the monolithic tab-based TUI with a modular, task-oriented, Termux-first interface that exposes the complete prompt configuration and preserves explicit save/apply safety.

**Architecture:** A root Bubble Tea model owns routing, a saved configuration, an in-memory draft, global operations, and modal state. Focused screen models own only UI state and emit typed messages to the root. Existing Bubble Tea, Bubbles, and Lip Gloss v1 dependencies provide lists, contextual help, key bindings, viewports, spinners, and text inputs.

**Tech Stack:** Go 1.25, Bubble Tea v1.3.10, Bubbles v1.0.0, Lip Gloss v1.1.0, BurntSushi/toml v1.6.0.

## Global Constraints

- Bubble Tea, Bubbles, and Lip Gloss remain on v1; do not add Huh, Glamour, Fang, or another Charm dependency.
- The baseline terminal is exactly 80x24; width >= 110 enables a side preview and sizes below 60x18 show a resize notice.
- Every operation remains keyboard-accessible and must not require Nerd Fonts, true color, or mouse input.
- `config.Config.Version` and `config.PluginConfig.Engine` remain system-managed and are not editable.
- Navigation must not write files or discard the in-memory draft.
- Apply must not write any file before explicit confirmation.
- Existing CLI commands, config schema version 1, prompt generation, and shell safety behavior remain compatible.
- Tests touching `HOME`, `.zshrc`, generated shell files, or plugins use `t.TempDir()` plus `t.Setenv("HOME", home)`.
- Implement every production behavior with a failing test first and commit each task separately using Conventional Commits.

---

## File map

- `internal/config/clone.go`: canonical deep copy used by apply and TUI state.
- `internal/config/io.go`: side-effect-free TOML encoding plus existing atomic persistence.
- `internal/apply/plan.go`: read-only Apply preview and structured step results.
- `internal/apply/apply.go`: compatible `ApplyConfig` wrapper over detailed execution.
- `internal/tui/app.go`: root Bubble Tea model and global message handling.
- `internal/tui/session.go`: saved/draft state and dirty transitions.
- `internal/tui/screen.go`: route, screen interface, screen context, and domain messages.
- `internal/tui/keys.go`: canonical global and screen key bindings.
- `internal/tui/layout.go`: 80x24-first responsive dimensions.
- `internal/tui/styles.go`: shared semantic styles.
- `internal/tui/screen_home.go`: task menu.
- `internal/tui/builder_model.go`: pure Builder configuration operations.
- `internal/tui/screen_builder.go`: segment list, settings editor, and live preview.
- `internal/tui/screen_preview.go`: editable preview context and viewport.
- `internal/tui/screen_themes.go`: filterable themes and complete color editor.
- `internal/tui/screen_plugins.go`: filterable plugin list, draft controls, and explicit add/remove operations.
- `internal/tui/screen_apply.go`: plan viewport, confirmation, spinner, and structured result.
- `internal/tui/screen_doctor.go`: checks, fix confirmation, and result details.
- `internal/tui/*_test.go`: focused unit and isolated-HOME integration tests.
- `internal/tui/tui_v2.go`: removed after all behavior has migrated.
- `README.md`: updated TUI behavior and shortcuts.

---

### Task 1: Side-effect-free Apply planning and structured execution

**Files:**
- Create: `internal/config/clone.go`
- Create: `internal/config/clone_test.go`
- Modify: `internal/config/io.go:88-146`
- Modify: `internal/config/io_test.go`
- Create: `internal/apply/plan.go`
- Create: `internal/apply/plan_test.go`
- Modify: `internal/apply/apply.go:14-48`
- Modify: `internal/apply/apply_test.go`

**Interfaces:**
- Produces: `config.Clone(*config.Config) *config.Config`.
- Produces: `config.Encode(*config.Config) ([]byte, error)` without filesystem writes or caller mutation.
- Produces: `apply.BuildPlan(*config.Config) (apply.Plan, error)`.
- Produces: `apply.ApplyConfigDetailed(*config.Config) apply.Result` while retaining `ApplyConfig(*config.Config) error`.

- [ ] **Step 1: Write failing clone and encoding tests**

```go
func TestCloneOwnsNestedValues(t *testing.T) {
	original := Default()
	clone := Clone(original)
	clone.Prompt.Order[0] = "time"
	clone.Prompt.Segments["user"] = SegmentConfig{Enabled: false}
	clone.Plugins.Items = append(clone.Plugins.Items, PluginItem{Name: "demo"})
	if original.Prompt.Order[0] == "time" || !original.Prompt.Segments["user"].Enabled || len(original.Plugins.Items) != 0 {
		t.Fatal("Clone shared nested configuration state")
	}
}

func TestEncodeDoesNotWriteOrMutate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := Default()
	before := Clone(cfg)
	data, err := Encode(cfg)
	if err != nil || !bytes.Contains(data, []byte("version = 1")) {
		t.Fatalf("Encode() data=%q err=%v", data, err)
	}
	if !reflect.DeepEqual(cfg, before) {
		t.Fatal("Encode mutated its input")
	}
	if _, err := os.Stat(Path()); !os.IsNotExist(err) {
		t.Fatalf("Encode wrote config.toml: %v", err)
	}
}
```

- [ ] **Step 2: Run the focused config tests and verify RED**

Run: `go test ./internal/config -run 'Test(CloneOwnsNestedValues|EncodeDoesNotWriteOrMutate)' -count=1`

Expected: compilation fails because `Clone` and `Encode` do not exist.

- [ ] **Step 3: Implement canonical cloning and encoding, then make Save reuse Encode**

```go
func Clone(cfg *Config) *Config {
	if cfg == nil {
		return Default()
	}
	clone := *cfg
	clone.Prompt.Order = append([]string(nil), cfg.Prompt.Order...)
	clone.Prompt.RightOrder = append([]string(nil), cfg.Prompt.RightOrder...)
	clone.Prompt.Segments = make(map[string]SegmentConfig, len(cfg.Prompt.Segments))
	for name, segment := range cfg.Prompt.Segments {
		clone.Prompt.Segments[name] = segment
	}
	clone.Plugins.Items = append([]PluginItem(nil), cfg.Plugins.Items...)
	return &clone
}

func Encode(cfg *Config) ([]byte, error) {
	clone := Clone(cfg)
	if clone.Version == 0 {
		clone.Version = CurrentConfigVersion
	}
	if err := Validate(clone); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	var out bytes.Buffer
	if err := toml.NewEncoder(&out).Encode(clone); err != nil {
		return nil, fmt.Errorf("failed to encode config: %w", err)
	}
	return out.Bytes(), nil
}
```

Change `Save` to call `Encode`, write the returned bytes to its already secured temporary file, sync, close, and atomically rename it. Preserve directory and file modes.

- [ ] **Step 4: Run config tests and verify GREEN**

Run: `go test ./internal/config -count=1`

Expected: PASS.

- [ ] **Step 5: Write failing Apply plan and partial-result tests**

```go
func TestBuildPlanHasNoSideEffects(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(config.Default())
	if err != nil {
		t.Fatal(err)
	}
	if plan.ConfigTOML == "" || !strings.Contains(plan.OmegaZsh, "ozsh_prompt()") || !strings.Contains(plan.ZshrcDiff, "+ # >>> ozsh >>>") {
		t.Fatalf("incomplete plan: %+v", plan)
	}
	if _, err := os.Stat(config.Path()); !os.IsNotExist(err) {
		t.Fatalf("BuildPlan wrote config: %v", err)
	}
}

func TestDetailedApplyReportsCompletedConfigBeforeOmegaFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".config", "ozsh", "omega.zsh"), 0o700); err != nil {
		t.Fatal(err)
	}
	result := ApplyConfigDetailed(config.Default())
	if result.Failed != StepOmega || !slices.Equal(result.Completed, []Step{StepConfig}) || result.Err == nil {
		t.Fatalf("result=%+v", result)
	}
}
```

- [ ] **Step 6: Run the Apply tests and verify RED**

Run: `go test ./internal/apply -run 'Test(BuildPlanHasNoSideEffects|DetailedApplyReportsCompletedConfigBeforeOmegaFailure)' -count=1`

Expected: compilation fails because `Plan`, `Step`, `Result`, `BuildPlan`, and `ApplyConfigDetailed` do not exist.

- [ ] **Step 7: Implement Apply planning and detailed results**

```go
type Step string

const (
	StepConfig Step = "config.toml"
	StepOmega  Step = "omega.zsh"
	StepZshrc  Step = ".zshrc"
)

type Plan struct {
	ConfigTOML string
	OmegaZsh   string
	ZshrcDiff  string
}

type Result struct {
	Completed []Step
	Failed    Step
	Err       error
}

func (r Result) OK() bool { return r.Err == nil }
```

`BuildPlan` must clone the config, call `config.Encode`, `prompt.Generate`, and `shell.PreviewInjectBlock`, and return rendered config, generated shell, and `shell.DiffLines(before, after)`. `ApplyConfigDetailed` runs the same preflight before writes, then records `StepConfig`, `StepOmega`, and `StepZshrc` immediately after each successful operation. Keep compatibility:

```go
func ApplyConfig(cfg *config.Config) error {
	return ApplyConfigDetailed(cfg).Err
}
```

- [ ] **Step 8: Run Apply and full package tests**

Run: `go test ./internal/apply ./internal/config -count=1`

Expected: PASS.

- [ ] **Step 9: Commit Task 1**

```bash
git add internal/config internal/apply
git commit -m "refactor: expose safe apply planning"
```

---

### Task 2: Saved/draft session state

**Files:**
- Create: `internal/tui/session.go`
- Create: `internal/tui/session_test.go`
- Modify: `internal/tui/remediation_test.go:84-107`

**Interfaces:**
- Consumes: `config.Clone` from Task 1.
- Produces: `NewSession(*config.Config) *Session`.
- Produces: `Draft() *config.Config`, `Saved() *config.Config`, `ReplaceDraft(*config.Config) error`, `MarkSaved(*config.Config)`, `Discard()`, and `Dirty() bool`.

- [ ] **Step 1: Write failing session transition tests**

```go
func TestSessionTracksReplaceSaveAndDiscard(t *testing.T) {
	session := NewSession(config.Default())
	draft := session.Draft()
	draft.Prompt.Separator = " | "
	if err := session.ReplaceDraft(draft); err != nil || !session.Dirty() {
		t.Fatalf("ReplaceDraft err=%v dirty=%t", err, session.Dirty())
	}
	session.Discard()
	if session.Dirty() || session.Draft().Prompt.Separator == " | " {
		t.Fatal("Discard did not restore saved config")
	}
	draft = session.Draft()
	draft.Prompt.Separator = " :: "
	if err := session.ReplaceDraft(draft); err != nil {
		t.Fatal(err)
	}
	session.MarkSaved(draft)
	if session.Dirty() || session.Saved().Prompt.Separator != " :: " {
		t.Fatal("MarkSaved did not synchronize state")
	}
}

func TestSessionRejectsInvalidDraftWithoutMutation(t *testing.T) {
	session := NewSession(config.Default())
	invalid := session.Draft()
	invalid.Prompt.Separator = ""
	if err := session.ReplaceDraft(invalid); err == nil {
		t.Fatal("ReplaceDraft accepted invalid config")
	}
	if session.Dirty() {
		t.Fatal("invalid draft changed session")
	}
}
```

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/tui -run '^TestSession' -count=1`

Expected: compilation fails because `Session` and `NewSession` do not exist.

- [ ] **Step 3: Implement Session as the sole configuration owner**

```go
type Session struct {
	saved *config.Config
	draft *config.Config
}

func NewSession(cfg *config.Config) *Session {
	base := config.Clone(cfg)
	return &Session{saved: base, draft: config.Clone(base)}
}

func (s *Session) Saved() *config.Config { return config.Clone(s.saved) }
func (s *Session) Draft() *config.Config { return config.Clone(s.draft) }
func (s *Session) Dirty() bool { return !reflect.DeepEqual(s.saved, s.draft) }
func (s *Session) Discard() { s.draft = config.Clone(s.saved) }
func (s *Session) MarkSaved(cfg *config.Config) {
	s.saved = config.Clone(cfg)
	s.draft = config.Clone(cfg)
}
```

`ReplaceDraft` clones, validates the clone, and replaces `s.draft` only after validation succeeds.

- [ ] **Step 4: Run focused and package tests**

Run: `go test ./internal/tui -run 'Test(Session|CloneConfig)' -count=1`

Expected: PASS after changing the old clone test to assert `config.Clone` or removing the duplicated `cloneConfig` assertion.

- [ ] **Step 5: Commit Task 2**

```bash
git add internal/tui/session.go internal/tui/session_test.go internal/tui/remediation_test.go
git commit -m "refactor: add TUI draft session"
```

---

### Task 3: Root router, responsive layout, Home list, and canonical help

**Files:**
- Create: `internal/tui/screen.go`
- Create: `internal/tui/app.go`
- Create: `internal/tui/keys.go`
- Create: `internal/tui/layout.go`
- Create: `internal/tui/styles.go`
- Create: `internal/tui/screen_home.go`
- Create: `internal/tui/app_test.go`
- Create: `internal/tui/layout_test.go`
- Create: `internal/tui/screen_home_test.go`
- Modify: `internal/tui/tui_v2.go:67-99` to delegate `Run` and `NewModel` to the new root while migration is in progress.

**Interfaces:**
- Consumes: `Session` from Task 2.
- Produces: `type Route`, `type Screen`, `type ScreenContext`, `type Model`, `NewModel(*config.Config) *Model`, and `Run() error`.
- Produces typed messages: `NavigateMsg`, `BackMsg`, `DraftChangedMsg`, `SaveRequestMsg`, `SaveResultMsg`, `QuitRequestMsg`, and `StatusMsg`.

- [ ] **Step 1: Write failing layout and routing tests**

```go
func TestNewLayoutBreakpoints(t *testing.T) {
	base := NewLayout(80, 24)
	if base.TooSmall || base.Wide || base.ContentWidth <= 0 || base.ContentHeight <= 0 {
		t.Fatalf("80x24 layout=%+v", base)
	}
	if !NewLayout(110, 24).Wide {
		t.Fatal("110 columns did not enable wide layout")
	}
	if !NewLayout(59, 18).TooSmall || !NewLayout(80, 17).TooSmall {
		t.Fatal("small terminal did not request resize")
	}
}

func TestRootNavigationPreservesDraft(t *testing.T) {
	model := NewModel(config.Default())
	draft := model.session.Draft()
	draft.Prompt.Separator = " | "
	updated, _ := model.Update(DraftChangedMsg{Config: draft})
	model = updated.(*Model)
	updated, _ = model.Update(NavigateMsg{To: RoutePreview})
	model = updated.(*Model)
	updated, _ = model.Update(BackMsg{})
	model = updated.(*Model)
	if model.route != RouteHome || model.session.Draft().Prompt.Separator != " | " {
		t.Fatalf("route=%v draft=%q", model.route, model.session.Draft().Prompt.Separator)
	}
}
```

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/tui -run 'Test(NewLayoutBreakpoints|RootNavigationPreservesDraft)' -count=1`

Expected: compilation fails because the new layout, root model, and messages do not exist.

- [ ] **Step 3: Define routes, screen contract, layout, and messages**

```go
type Route uint8

const (
	RouteHome Route = iota
	RouteBuilder
	RoutePreview
	RouteApply
	RouteDoctor
	RouteThemes
	RoutePlugins
)

type ScreenContext struct {
	Draft  *config.Config
	Layout Layout
	Busy   bool
}

type Screen interface {
	Init() tea.Cmd
	SetSize(Layout)
	Update(tea.Msg, ScreenContext) tea.Cmd
	View(ScreenContext) string
	ShortHelp() []key.Binding
	FullHelp() [][]key.Binding
}

type NavigateMsg struct{ To Route }
type BackMsg struct{}
type DraftChangedMsg struct{ Config *config.Config }
type SaveRequestMsg struct{}
type SaveResultMsg struct {
	Config *config.Config
	Err    error
}
type QuitRequestMsg struct{}
type StatusMsg struct {
	Text string
	Err  error
}
```

`NewLayout` reserves one header row, two footer rows, and the remaining area for content. It sets `Wide` at width >= 110 and `TooSmall` at width < 60 or height < 18.

- [ ] **Step 4: Implement the root model and save command**

```go
type Model struct {
	session *Session
	route   Route
	history []Route
	screens map[Route]Screen
	layout  Layout
	help    help.Model
	busy    bool
	status  StatusMsg
	modal   modalKind
}

func NewModel(cfg *config.Config) *Model {
	m := &Model{session: NewSession(cfg), route: RouteHome, help: help.New()}
	m.screens = newScreens(m.session.Draft())
	return m
}

func saveCmd(cfg *config.Config) tea.Cmd {
	snapshot := config.Clone(cfg)
	return func() tea.Msg {
		err := config.Save(snapshot)
		return SaveResultMsg{Config: snapshot, Err: err}
	}
}
```

The root handles window sizes, typed domain messages, busy state, save/discard/cancel and quit modals, then delegates ordinary input to the active screen. `View` returns a resize notice when `layout.TooSmall`; otherwise it renders breadcrumb, active content, semantic status, and `help.View(activeScreen)`.

- [ ] **Step 5: Write the failing Home selection and help consistency test**

```go
func TestHomeEnterNavigatesToSelectedTask(t *testing.T) {
	home := NewHomeScreen(80, 20)
	home.list.Select(1)
	cmd := home.Update(tea.KeyMsg{Type: tea.KeyEnter}, ScreenContext{})
	msg := cmd()
	if msg != (NavigateMsg{To: RouteBuilder}) {
		t.Fatalf("message=%#v", msg)
	}
}

func TestHomeHelpUsesExecutableBindings(t *testing.T) {
	home := NewHomeScreen(80, 20)
	for _, binding := range home.ShortHelp() {
		if !binding.Enabled() || len(binding.Keys()) == 0 {
			t.Fatalf("invalid binding: %#v", binding)
		}
	}
}
```

- [ ] **Step 6: Implement Home with `bubbles/list` and `bubbles/key/help`**

```go
type homeItem struct {
	route       Route
	title, desc string
}

func (i homeItem) Title() string       { return i.title }
func (i homeItem) Description() string { return i.desc }
func (i homeItem) FilterValue() string { return i.title + " " + i.desc }
```

Construct seven items in the approved order. Enter emits `NavigateMsg`; `q` emits `QuitRequestMsg`; list filtering, pagination, and movement remain delegated to `list.Model`. Define each key once with `key.NewBinding` and return those same bindings from help methods.

- [ ] **Step 7: Run focused and full TUI tests**

Run: `go test ./internal/tui -run 'Test(NewLayout|Root|Home)' -count=1`

Expected: PASS.

- [ ] **Step 8: Commit Task 3**

```bash
git add internal/tui
git commit -m "feat: add task-oriented TUI shell"
```

---

### Task 4: Pure Builder configuration operations

**Files:**
- Create: `internal/tui/builder_model.go`
- Create: `internal/tui/builder_model_test.go`

**Interfaces:**
- Produces: `type PromptSide`, `AllSegmentNames`, `SegmentSide`, `SetSegmentEnabled`, `PlaceSegment`, `MoveSegment`, `UpdateSegment`, and `ResetSegmentAppearance`.
- All functions receive a configuration and return a validated clone; no function mutates its argument.

- [ ] **Step 1: Write failing Builder invariant tests**

```go
func TestAllSegmentNamesIncludesDisabledUnorderedSegments(t *testing.T) {
	cfg := config.Default()
	names := AllSegmentNames(cfg)
	for _, want := range []string{"user", "host", "cwd", "git", "status", "time", "venv", "node", "go", "battery", "jobs"} {
		if !slices.Contains(names, want) {
			t.Fatalf("missing %q in %v", want, names)
		}
	}
}

func TestEnableUnplacedSegmentAddsItToLeftPrompt(t *testing.T) {
	cfg, err := SetSegmentEnabled(config.Default(), "time", true)
	if err != nil || !cfg.Prompt.Segments["time"].Enabled || !slices.Contains(cfg.Prompt.Order, "time") {
		t.Fatalf("cfg=%+v err=%v", cfg.Prompt, err)
	}
}

func TestPlaceSegmentIsExclusiveAndMoveKeepsOrderValid(t *testing.T) {
	cfg, err := PlaceSegment(config.Default(), "git", SideRight)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(cfg.Prompt.Order, "git") || !slices.Contains(cfg.Prompt.RightOrder, "git") {
		t.Fatalf("left=%v right=%v", cfg.Prompt.Order, cfg.Prompt.RightOrder)
	}
	moved, err := MoveSegment(cfg, "git", -1)
	if err != nil || moved.Prompt.RightOrder[0] != "git" {
		t.Fatalf("right=%v err=%v", moved.Prompt.RightOrder, err)
	}
}
```

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/tui -run 'Test(AllSegmentNames|EnableUnplaced|PlaceSegment)' -count=1`

Expected: compilation fails because Builder operations do not exist.

- [ ] **Step 3: Implement immutable Builder operations**

```go
type PromptSide uint8

const (
	SideUnassigned PromptSide = iota
	SideLeft
	SideRight
)

type SegmentUpdate struct {
	Icon        string
	FG          string
	BG          string
	Bold        bool
	ShowSuccess bool
}
```

Use `config.Clone` at the start of each operation. Enabling an unassigned segment appends it to `Prompt.Order`; disabling preserves placement. `PlaceSegment` removes every occurrence from both orders and appends it to the chosen side. `MoveSegment` swaps with the adjacent item only inside the current side. `UpdateSegment` changes only the selected segment fields. `ResetSegmentAppearance` restores icon, colors, bold, and `show_success` from `config.Default()` while preserving enabled state, side, and position. Validate before returning.

- [ ] **Step 4: Add invalid-name, boundary, duplicate, and reset tests**

```go
func TestResetSegmentAppearancePreservesPlacementAndEnabled(t *testing.T) {
	cfg := config.Default()
	cfg, _ = PlaceSegment(cfg, "git", SideRight)
	cfg, _ = SetSegmentEnabled(cfg, "git", true)
	cfg.Prompt.Segments["git"] = config.SegmentConfig{Enabled: true, FG: "#123456", Icon: "G"}
	got, err := ResetSegmentAppearance(cfg, "git")
	if err != nil || SegmentSide(got, "git") != SideRight || !got.Prompt.Segments["git"].Enabled || got.Prompt.Segments["git"].FG != config.Default().Prompt.Segments["git"].FG {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}
```

- [ ] **Step 5: Run Builder model tests**

Run: `go test ./internal/tui -run 'Test(AllSegment|Enable|Place|Move|Reset|UpdateSegment)' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit Task 4**

```bash
git add internal/tui/builder_model.go internal/tui/builder_model_test.go
git commit -m "feat: add complete prompt builder model"
```

---

### Task 5: Complete Builder screen and responsive live preview

**Files:**
- Create: `internal/tui/screen_builder.go`
- Create: `internal/tui/screen_builder_test.go`
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/screen.go`

**Interfaces:**
- Consumes: Builder operations from Task 4 and `prompt.Simulated`.
- Produces: `NewBuilderScreen(*config.Config) *BuilderScreen` implementing `Screen`.

- [ ] **Step 1: Write failing screen tests for complete segment visibility and draft messages**

```go
func TestBuilderShowsUnorderedDisabledSegments(t *testing.T) {
	screen := NewBuilderScreen(config.Default())
	screen.SetSize(NewLayout(80, 24))
	view := screen.View(ScreenContext{Draft: config.Default(), Layout: NewLayout(80, 24)})
	if !strings.Contains(view, "battery") || !strings.Contains(view, "disabled") {
		t.Fatalf("builder view missing disabled segment:\n%s", view)
	}
}

func TestBuilderToggleEmitsValidatedDraft(t *testing.T) {
	screen := NewBuilderScreen(config.Default())
	screen.selectByName("user")
	cmd := screen.Update(tea.KeyMsg{Type: tea.KeySpace}, ScreenContext{Draft: config.Default()})
	msg, ok := cmd().(DraftChangedMsg)
	if !ok || msg.Config.Prompt.Segments["user"].Enabled {
		t.Fatalf("message=%#v", cmd())
	}
}
```

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/tui -run '^TestBuilder(Shows|Toggle)' -count=1`

Expected: compilation fails because `BuilderScreen` does not exist.

- [ ] **Step 3: Implement list and screen modes**

```go
type builderMode uint8

const (
	builderSegments builderMode = iota
	builderSegmentEditor
	builderPromptSettings
)

type BuilderScreen struct {
	list     list.Model
	mode     builderMode
	inputs   []textinput.Model
	focus    int
	viewport viewport.Model
	layout   Layout
	errText  string
	keys     builderKeyMap
}
```

Populate `list.Model` from `AllSegmentNames`. The item title includes enabled state and side; the description includes foreground, background, bold, and icon without relying on color alone. Space toggles, `l`/`r` place a segment, `J`/`K` reorder, Enter opens the editor, `g` opens general prompt settings, and `Esc` closes the editor before emitting `BackMsg` from the segment list.

- [ ] **Step 4: Add failing field-validation and responsive-view tests**

```go
func TestBuilderRejectsInvalidColorBesideField(t *testing.T) {
	screen := NewBuilderScreen(config.Default())
	screen.openEditor("user")
	screen.setFieldValue("fg", "not-a-color")
	cmd := screen.commitEditor(ScreenContext{Draft: config.Default()})
	if cmd != nil || !strings.Contains(screen.View(ScreenContext{Draft: config.Default(), Layout: NewLayout(80, 24)}), "invalid color") {
		t.Fatal("invalid color was committed or not displayed")
	}
}

func TestBuilderWideLayoutAddsSidePreview(t *testing.T) {
	screen := NewBuilderScreen(config.Default())
	wide := NewLayout(110, 24)
	screen.SetSize(wide)
	view := screen.View(ScreenContext{Draft: config.Default(), Layout: wide})
	if !strings.Contains(view, "Preview") {
		t.Fatalf("wide view missing side preview:\n%s", view)
	}
}
```

- [ ] **Step 5: Implement segment and general settings editors**

Use `textinput.Model` for icon, foreground, background, and separator. Use key-controlled choices for booleans, side, and style. Keep raw invalid text inside the screen; emit `DraftChangedMsg` only after `UpdateSegment` or `config.Validate` succeeds. General settings expose style, newline, right prompt, heavy-segment suppression, and separator. Render the preview with `prompt.Simulated(ctx.Draft)` below the list at 80 columns and beside it at width >= 110.

- [ ] **Step 6: Run Builder screen and package tests**

Run: `go test ./internal/tui -run '^TestBuilder' -count=1`

Expected: PASS.

- [ ] **Step 7: Commit Task 5**

```bash
git add internal/tui/app.go internal/tui/screen.go internal/tui/screen_builder.go internal/tui/screen_builder_test.go
git commit -m "feat: build complete responsive prompt editor"
```

---

### Task 6: Preview and Themes screens

**Files:**
- Create: `internal/tui/screen_preview.go`
- Create: `internal/tui/screen_preview_test.go`
- Create: `internal/tui/screen_themes.go`
- Create: `internal/tui/screen_themes_test.go`
- Modify: `internal/tui/app.go`

**Interfaces:**
- Produces: `NewPreviewScreen() *PreviewScreen` and `NewThemesScreen(*config.Config) *ThemesScreen` implementing `Screen`.
- Themes emit only validated `DraftChangedMsg`; Preview never mutates configuration.

- [ ] **Step 1: Write failing Preview focus, validation, and viewport tests**

```go
func TestPreviewEditsOnlyFocusedField(t *testing.T) {
	screen := NewPreviewScreen()
	before := screen.inputs[1].Value()
	screen.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}, ScreenContext{Draft: config.Default()})
	if !strings.HasSuffix(screen.inputs[0].Value(), "x") || screen.inputs[1].Value() != before {
		t.Fatal("preview input routing changed an unfocused field")
	}
}

func TestPreviewReportsInvalidExitStatus(t *testing.T) {
	screen := NewPreviewScreen()
	screen.focus = previewExitStatus
	screen.inputs[previewExitStatus].SetValue("x")
	screen.syncContext()
	if !strings.Contains(screen.errText, "exit status must be an integer") {
		t.Fatalf("error=%q", screen.errText)
	}
}
```

- [ ] **Step 2: Run Preview tests and verify RED**

Run: `go test ./internal/tui -run '^TestPreview' -count=1`

Expected: compilation fails because `PreviewScreen` does not exist.

- [ ] **Step 3: Implement Preview with text inputs and viewport**

Move `sanitizeInput`, `previewInputs`, and Termux preview simplification out of `tui_v2.go`. The fields are username, current directory, Git branch, and integer exit status. Render `prompt.SimulatedWithContext` into `viewport.Model`; route typing only to the focused field and route scrolling to the viewport when it has focus.

- [ ] **Step 4: Write failing theme draft and full-color tests**

```go
func TestThemePresetEmitsDraftWithoutMutatingInput(t *testing.T) {
	base := config.Default()
	screen := NewThemesScreen(base)
	cmd := screen.applyPreset("neon-red", ScreenContext{Draft: base})
	msg := cmd().(DraftChangedMsg)
	if msg.Config.Theme.Name != "neon-red" || base.Theme.Name == "neon-red" {
		t.Fatalf("draft=%q base=%q", msg.Config.Theme.Name, base.Theme.Name)
	}
}

func TestThemeEditorExposesAllThemeFields(t *testing.T) {
	screen := NewThemesScreen(config.Default())
	for _, field := range []string{"name", "accent", "background", "muted", "success", "warning", "error"} {
		if !screen.hasField(field) {
			t.Fatalf("missing theme field %q", field)
		}
	}
}
```

- [ ] **Step 5: Implement Themes with list, editor, validation, and live preview**

Use a filterable `list.Model` for sorted presets. Enter applies the selected preset to a cloned draft; `e` opens seven `textinput.Model` fields; `r` restores the saved/draft screen snapshot; and `Esc` cancels raw field edits. Call `config.Validate` before emitting the draft. At width >= 110 render the simulated prompt beside the list; otherwise render it below or inside a viewport.

- [ ] **Step 6: Run focused and full TUI tests**

Run: `go test ./internal/tui -run 'Test(Preview|Theme)' -count=1`

Expected: PASS.

- [ ] **Step 7: Commit Task 6**

```bash
git add internal/tui/screen_preview.go internal/tui/screen_preview_test.go internal/tui/screen_themes.go internal/tui/screen_themes_test.go internal/tui/app.go
git commit -m "feat: add preview and theme screens"
```

---

### Task 7: Plugins screen with safe dirty-state coordination

**Files:**
- Create: `internal/tui/screen_plugins.go`
- Create: `internal/tui/screen_plugins_test.go`
- Create: `internal/tui/plugin_ops.go`
- Create: `internal/tui/plugin_ops_test.go`
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/screen.go`

**Interfaces:**
- Produces: `NewPluginsScreen(*config.Config) *PluginsScreen` implementing `Screen`.
- Produces: `PluginOpRequestMsg` and `PluginOpResultMsg` for explicit add/remove disk operations.
- Enable, disable, trust, and untrust produce `DraftChangedMsg` and do not persist config.

- [ ] **Step 1: Write failing draft-control and trust-confirmation tests**

```go
func TestPluginEnableChangesDraftOnly(t *testing.T) {
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{{Name: "demo", Source: filepath.Join(t.TempDir(), "demo"), Load: "plugin.zsh"}}
	screen := NewPluginsScreen(cfg)
	cmd := screen.setEnabled("demo", true, ScreenContext{Draft: cfg})
	msg := cmd().(DraftChangedMsg)
	if !msg.Config.Plugins.Items[0].Enabled || cfg.Plugins.Items[0].Enabled {
		t.Fatal("plugin enable did not remain draft-only")
	}
}

func TestPluginTrustRequiresConfirmation(t *testing.T) {
	screen := NewPluginsScreen(config.Default())
	if cmd := screen.requestTrust("demo"); cmd != nil || screen.mode != pluginConfirmTrust {
		t.Fatal("trust did not open confirmation")
	}
}
```

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/tui -run '^TestPlugin(Enable|Trust)' -count=1`

Expected: compilation fails because the new Plugins screen does not exist.

- [ ] **Step 3: Implement list, add form, draft controls, and confirmations**

Use `list.Model` for filtering and selection, two `textinput.Model` values for HTTPS URL and load path, and a spinner during add/remove. Item descriptions always include enabled/disabled and trusted/untrusted text. Trust uses `plugins.SetTrusted` on a cloned draft only after `y`; untrust, enable, and disable use cloned drafts immediately. Remove requires a second explicit confirmation because it deletes the checkout.

- [ ] **Step 4: Write failing root dirty-guard and operation-result tests**

```go
func TestDirtySessionBlocksPluginDiskOperation(t *testing.T) {
	model := NewModel(config.Default())
	draft := model.session.Draft()
	draft.Prompt.Separator = " | "
	_ = model.session.ReplaceDraft(draft)
	updated, cmd := model.Update(PluginOpRequestMsg{Kind: PluginAdd, URL: "https://example.com/demo.git", Load: "plugin.zsh"})
	model = updated.(*Model)
	if cmd != nil || model.modal != modalResolveDirtyPluginOp {
		t.Fatal("dirty plugin operation was not blocked")
	}
}

func TestPluginOperationResultRefreshesSavedAndDraft(t *testing.T) {
	model := NewModel(config.Default())
	cfg := config.Default()
	cfg.Plugins.Items = []config.PluginItem{{Name: "demo"}}
	updated, _ := model.Update(PluginOpResultMsg{Config: cfg})
	model = updated.(*Model)
	if model.session.Dirty() || len(model.session.Saved().Plugins.Items) != 1 {
		t.Fatal("plugin result did not synchronize session")
	}
}
```

- [ ] **Step 5: Implement explicit plugin operation commands**

```go
type PluginOpKind uint8

const (
	PluginAdd PluginOpKind = iota
	PluginRemove
)

type PluginOpRequestMsg struct {
	Kind PluginOpKind
	Name string
	URL  string
	Load string
}

type PluginOpResultMsg struct {
	Config *config.Config
	Err    error
}
```

If the session is dirty, the root opens Save/Discard/Cancel before running add/remove. Once clean, `pluginOpCmd` clones `session.Saved()`, calls `plugins.AddAndSave` or `plugins.RemoveAndSave`, and returns the resulting configuration. A successful result replaces both saved and draft state. A failed result preserves the session and shows the error. No test invokes a real network clone; integration tests cover invalid URLs and local filesystem safety, while existing plugin package tests retain clone behavior coverage.

- [ ] **Step 6: Run Plugins and full TUI tests**

Run: `go test ./internal/tui -run '^TestPlugin' -count=1`

Expected: PASS.

- [ ] **Step 7: Commit Task 7**

```bash
git add internal/tui/screen_plugins.go internal/tui/screen_plugins_test.go internal/tui/plugin_ops.go internal/tui/plugin_ops_test.go internal/tui/app.go internal/tui/screen.go
git commit -m "feat: add safe plugin management screen"
```

---

### Task 8: Apply and Doctor screens

**Files:**
- Create: `internal/tui/screen_apply.go`
- Create: `internal/tui/screen_apply_test.go`
- Create: `internal/tui/screen_doctor.go`
- Create: `internal/tui/screen_doctor_test.go`
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/screen.go`

**Interfaces:**
- Consumes: `apply.BuildPlan` and `apply.ApplyConfigDetailed` from Task 1.
- Produces: `ApplyRequestMsg`, `ApplyPlanMsg`, `ApplyConfirmMsg`, and `ApplyResultMsg`.
- Produces: `DoctorCheck`, `collectDoctorChecks() []DoctorCheck`, and `fixDoctorCmd(*config.Config) tea.Cmd` inside package `tui`.

- [ ] **Step 1: Write failing no-write-before-confirmation test**

```go
func TestApplyPlanAndCancelDoNotWrite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	zshrc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(zshrc, []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	model := NewModel(config.Default())
	updated, cmd := model.Update(ApplyRequestMsg{})
	model = updated.(*Model)
	planMsg := cmd().(ApplyPlanMsg)
	updated, _ = model.Update(planMsg)
	model = updated.(*Model)
	updated, _ = model.Update(BackMsg{})
	model = updated.(*Model)
	if _, err := os.Stat(config.Path()); !os.IsNotExist(err) {
		t.Fatalf("apply review wrote config: %v", err)
	}
	data, _ := os.ReadFile(zshrc)
	if strings.Contains(string(data), "ozsh") {
		t.Fatal("apply review changed .zshrc")
	}
}
```

- [ ] **Step 2: Run and verify RED**

Run: `go test ./internal/tui -run '^TestApplyPlanAndCancel' -count=1`

Expected: compilation fails because Apply messages and screen do not exist.

- [ ] **Step 3: Implement Apply states, viewport, confirmation, spinner, and result details**

```go
type applyMode uint8

const (
	applyIdle applyMode = iota
	applyReview
	applyConfirm
	applyRunning
	applyFinished
)

type ApplyPlanMsg struct {
	Plan apply.Plan
	Err  error
}

type ApplyResultMsg struct {
	Config *config.Config
	Result apply.Result
}
```

`ApplyRequestMsg` captures `session.Draft()` and calls `apply.BuildPlan` in a command. Review renders all three plan sections in `viewport.Model`. Only `y` from `applyConfirm` starts `apply.ApplyConfigDetailed` against a captured clone. While running, ignore operation-triggering keys and animate `spinner.Model`. Success calls `session.MarkSaved`; failure preserves the draft and lists completed step, failed step, error text, and whether retry is safe.

- [ ] **Step 4: Write failing Doctor check and explicit-fix tests**

```go
func TestDoctorViewUsesTextForPassAndFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	screen := NewDoctorScreen()
	view := screen.View(ScreenContext{Draft: config.Default(), Layout: NewLayout(80, 24)})
	if !strings.Contains(view, "PASS") || !strings.Contains(view, "FAIL") {
		t.Fatalf("doctor view lacks semantic status:\n%s", view)
	}
}

func TestDoctorDoesNotFixBeforeConfirmation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	screen := NewDoctorScreen()
	screen.Update(tea.KeyMsg{Type: tea.KeyEnter}, ScreenContext{Draft: config.Default()})
	if _, err := os.Stat(config.Path()); !os.IsNotExist(err) {
		t.Fatalf("doctor wrote before confirmation: %v", err)
	}
}
```

- [ ] **Step 5: Implement Doctor checks and confirmed fixes**

```go
type DoctorCheck struct {
	ID      string
	Label   string
	Detail  string
	Passed  bool
	Fixable bool
}
```

Collect Zsh availability, valid config, `.zshrc`, managed block, default shell, Termux state, and backup count without writes. Enter on a fixable item opens confirmation. Confirmed fixes use the current draft when creating a missing config, create a missing `.zshrc` with mode `0o600`, and direct managed-block fixes to Apply instead of injecting silently. Render checks as PASS/FAIL/INFO text and place long details in a viewport.

- [ ] **Step 6: Run Apply, Doctor, and isolated-HOME tests**

Run: `go test ./internal/tui -run 'Test(Apply|Doctor)' -count=1`

Expected: PASS.

- [ ] **Step 7: Commit Task 8**

```bash
git add internal/tui/screen_apply.go internal/tui/screen_apply_test.go internal/tui/screen_doctor.go internal/tui/screen_doctor_test.go internal/tui/app.go internal/tui/screen.go
git commit -m "feat: add confirmed apply and doctor screens"
```

---

### Task 9: Legacy removal, integration coverage, documentation, and repository validation

**Files:**
- Delete: `internal/tui/tui_v2.go`
- Rewrite: `internal/tui/tui_test.go`
- Rewrite: `internal/tui/remediation_test.go`
- Create: `internal/tui/integration_test.go`
- Modify: `README.md:129-135`

**Interfaces:**
- Retains public package entrypoint `Run() error`.
- Retains command behavior in `cmd/ozsh/main_v2.go` without changing CLI parsing.

- [ ] **Step 1: Write the complete isolated-HOME workflow test before deleting legacy code**

```go
func TestTUIWorkflowRetainsDraftAndAppliesOnlyAfterConfirmation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	zshrc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(zshrc, []byte("export EDITOR=vim\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	model := NewModel(config.Default())
	draft := model.session.Draft()
	draft.Prompt.Separator = " | "
	updated, _ := model.Update(DraftChangedMsg{Config: draft})
	model = updated.(*Model)
	updated, _ = model.Update(NavigateMsg{To: RoutePreview})
	model = updated.(*Model)
	if _, err := os.Stat(config.Path()); !os.IsNotExist(err) {
		t.Fatalf("navigation wrote config: %v", err)
	}
	updated, cmd := model.Update(ApplyRequestMsg{})
	model = updated.(*Model)
	updated, _ = model.Update(cmd())
	model = updated.(*Model)
	updated, cmd = model.Update(ApplyConfirmMsg{})
	model = updated.(*Model)
	updated, _ = model.Update(cmd())
	model = updated.(*Model)
	if model.session.Dirty() {
		t.Fatal("successful apply left dirty session")
	}
	saved, err := config.LoadExisting()
	if err != nil || saved.Prompt.Separator != " | " {
		t.Fatalf("saved separator=%q err=%v", saved.Prompt.Separator, err)
	}
}
```

- [ ] **Step 2: Run the integration test and verify its pre-cleanup state**

Run: `go test ./internal/tui -run '^TestTUIWorkflow' -count=1`

Expected: PASS using only new root and screen code. If it reaches a legacy helper, migrate that helper before proceeding.

- [ ] **Step 3: Remove the monolith and migrate remaining behavioral tests**

Delete `tui_v2.go`. Keep tests for all behavior that remains part of the approved design; remove assertions tied only to numeric tabs, shared cursors, stale manual help strings, and the old concrete value-model shape. Ensure `Run` lives in `app.go`:

```go
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(NewModel(cfg), tea.WithAltScreen()).Run()
	return err
}
```

- [ ] **Step 4: Add layout rendering and memory regression coverage**

```go
func TestEveryScreenRendersAt80x24WithoutWideLayout(t *testing.T) {
	model := NewModel(config.Default())
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model = updated.(*Model)
	for route := RouteHome; route <= RoutePlugins; route++ {
		model.route = route
		view := model.View()
		for _, line := range strings.Split(ansi.Strip(view), "\n") {
			if runewidth.StringWidth(line) > 80 {
				t.Fatalf("route=%v width=%d line=%q", route, runewidth.StringWidth(line), line)
			}
		}
	}
}

func TestModelViewUsesLessThan50MB(t *testing.T) {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_ = NewModel(config.Default()).View()
	runtime.ReadMemStats(&after)
	if used := after.Alloc - before.Alloc; used > 50*1024*1024 {
		t.Fatalf("TUI view allocated %d bytes", used)
	}
}
```

Use the already indirect `github.com/charmbracelet/x/ansi` and `github.com/mattn/go-runewidth`; do not add a dependency.

- [ ] **Step 5: Update the README TUI section**

Document the task menu, draft indicator, Save/Discard/Cancel exit flow, full Builder settings, 80x24 baseline, contextual help, Apply review, and the fact that plugin add/remove are explicit disk operations that resolve dirty drafts first.

- [ ] **Step 6: Run format and focused tests**

Run: `gofmt -w internal/config internal/apply internal/tui`

Run: `go test ./internal/config ./internal/apply ./internal/tui -count=1`

Expected: PASS.

- [ ] **Step 7: Run repository validation**

Run: `scripts/lint.sh --check`

Run: `scripts/test.sh`

Run: `scripts/build.sh`

Run: `scripts/healthcheck.sh`

Expected: every command exits 0, aggregate coverage remains >= 70%, race tests pass, and Android/Termux ARM64 cross-build remains green.

- [ ] **Step 8: Inspect the final diff and ensure scope discipline**

Run: `git diff --check main...HEAD`

Run: `git diff --stat main...HEAD`

Expected: no whitespace errors; changes are confined to the approved config/apply refactors, `internal/tui`, README, design, and plan documents.

- [ ] **Step 9: Commit Task 9**

```bash
git add README.md internal/tui
git commit -m "refactor: complete modular Charm TUI"
```

- [ ] **Step 10: Prepare the draft pull request**

Use the repository PR template if present. Summarize the Termux-first navigation, complete Builder, draft safety, Apply confirmation, plugin dirty-state rule, and validation results. Keep the PR in draft until GitHub Actions passes.
