# Transparent TUI and Extensible Plugins Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove background fills from ozsh TUI chrome and replace the hidden custom-plugin form with a safe, discoverable, keyboard-first workflow whose filesystem changes are committed only through Review & Apply.

**Architecture:** Preserve the current Bubble Tea model and fullscreen layout, but separate plugin work into repository parsing, bounded candidate discovery, staged cloning, pending filesystem changes, and transactional application. The TUI owns only wizard state and pending configuration. `internal/apply` finalizes additions and removals together with config, `omega.zsh`, and `.zshrc` writes.

**Tech Stack:** Go 1.25.12, Bubble Tea, Bubbles `textinput`, Lip Gloss, `context`, `os/exec`, TOML-backed ozsh configuration, Go tests, GitHub Actions, Android/Termux ARM64 cross-build.

## Global Constraints

- Implement PR A only. Do not add Starship or change `CurrentConfigVersion`.
- Native prompt previews may contain background ANSI codes; ozsh TUI chrome must not emit background SGR codes.
- Preserve the responsive breakpoint at `72` columns and all fullscreen width/height guarantees.
- Keep the workflow keyboard-first and usable below `72` columns.
- Accept only HTTPS repository URLs without credentials, query strings, or fragments.
- Clone with `git clone --depth 1` and a `45s` timeout.
- Store temporary checkouts under `~/.config/ozsh/plugins/.staging-*`; use mode `0700` where supported.
- Bound candidate scanning to depth `4` and `128` accepted candidates.
- Accept only regular `.plugin.zsh`, `.zsh`, and `.sh` files. Reject symlinks and symlink path components.
- Downloading never implies trust. Explicit confirmation is required.
- Add, remove, trust, load-file, and enabled-state changes stay pending until Review & Apply.
- Cloning runs through `tea.Cmd`; `Update()` and `View()` remain non-blocking.
- Use TDD and commit after each independently testable task.

## File Map

### Create

- `internal/plugins/repository.go`: repository parsing, duplicate checks, load-path validation.
- `internal/plugins/repository_test.go`: repository validation tests.
- `internal/plugins/candidates.go`: bounded candidate discovery and ranking.
- `internal/plugins/candidates_test.go`: ranking, bounds, and symlink tests.
- `internal/plugins/staging.go`: injectable shallow clone and staging cleanup.
- `internal/plugins/staging_test.go`: timeout, cancellation, permissions, cleanup.
- `internal/plugins/changes.go`: pending add/remove set and rename transaction.
- `internal/plugins/changes_test.go`: queue, commit, rollback, conflict tests.
- `internal/apply/snapshot.go`: file snapshots for rollback.
- `internal/apply/snapshot_test.go`: existing/missing-file restoration.
- `internal/tui/plugin_items.go`: flattened navigation over recommended and custom entries.
- `internal/tui/plugin_wizard.go`: wizard state and update logic.
- `internal/tui/plugin_wizard_view.go`: responsive wizard rendering.
- `internal/tui/plugin_wizard_test.go`: wizard state-machine tests.
- `internal/tui/plugin_lifecycle_test.go`: pending lifecycle tests.
- `internal/tui/apply_plugin_changes_test.go`: review snapshot and retry tests.
- `internal/tui/transparent_chrome_test.go`: background ANSI regressions.

### Modify

- `internal/tui/visual_styles.go`
- `internal/tui/fullscreen_layout.go`
- `internal/tui/contextual_screens.go`
- `internal/tui/tui_v2_part1.go`
- `internal/tui/tui_v2_part2.go`
- `internal/tui/tui_v2_part5.go`
- `internal/tui/tui_v2_part6.go`
- `internal/tui/apply_workspace.go`
- `internal/tui/contextual_screens_test.go`
- `internal/tui/visual_hierarchy_test.go`
- `internal/plugins/plugins.go`
- `internal/plugins/plugins_test.go`
- `internal/apply/apply.go`
- `internal/apply/apply_test.go`
- `docs/screencasts/tui-visual-hierarchy.tape`
- `README.md`

---

### Task 1: Remove backgrounds from TUI chrome

**Files:**
- Modify: `internal/tui/visual_styles.go`
- Modify: `internal/tui/fullscreen_layout.go`
- Create: `internal/tui/transparent_chrome_test.go`
- Modify: `internal/tui/visual_hierarchy_test.go`

**Interfaces:**
- Consumes: `renderTab`, `renderPreviewBox`, `panelStyle`, `workspaceBoxStyle`, `selectedRowStyle`.
- Produces: background-free semantic styles and `backgroundSGR` test matcher.

- [ ] **Step 1: Write the failing background regression test**

```go
package tui

import (
    "regexp"
    "strings"
    "testing"

    "github.com/snakepilot10/ozsh/internal/config"
)

var backgroundSGR = regexp.MustCompile(`\x1b\[(?:4[0-9]|10[0-7]|48;(?:2|5);[0-9;]+)m`)

func configWithoutPromptBackgrounds() *config.Config {
    cfg := config.Default()
    for name, segment := range cfg.Prompt.Segments {
        segment.BG = ""
        cfg.Prompt.Segments[name] = segment
    }
    return cfg
}

func TestTUIChromeDoesNotEmitBackgroundColors(t *testing.T) {
    for tab := range tabs {
        model := NewModel(configWithoutPromptBackgrounds())
        model.width, model.height = 100, 34
        model.setTab(tab)
        if match := backgroundSGR.FindString(model.View()); match != "" {
            t.Fatalf("tab %d emitted background SGR %q", tab, match)
        }
    }
}

func TestSelectionRemainsIdentifiableWithoutColor(t *testing.T) {
    model := NewModel(config.Default())
    model.width, model.height = 58, 28
    model.setTab(tabPlugins)
    plain := plainText(model.View())
    if !strings.Contains(plain, "› [") {
        t.Fatalf("selected row lost marker:\n%s", plain)
    }
    if !strings.Contains(renderTab("Plugins", true, false), "\x1b[4m") {
        t.Fatal("active tab lost underline styling")
    }
}
```

- [ ] **Step 2: Run the test and verify failure**

```bash
go test ./internal/tui -run 'TestTUIChromeDoesNotEmitBackgroundColors|TestSelectionRemainsIdentifiableWithoutColor' -count=1
```

Expected: the background test fails because current panel, tab, preview, or badge styles call `.Background(...)`.

- [ ] **Step 3: Remove every chrome background declaration**

Use these style definitions:

```go
tabActiveStyle = lipgloss.NewStyle().
    Foreground(visualPalette.Accent).
    Bold(true).
    Underline(true)

previewBoxStyle = lipgloss.NewStyle().
    Foreground(visualPalette.Text).
    Border(lipgloss.RoundedBorder()).
    BorderForeground(visualPalette.Border).
    Padding(0, 1)

variantBadgeStyle = lipgloss.NewStyle().
    Foreground(visualPalette.Accent)

selectedRowStyle = lipgloss.NewStyle().
    Foreground(visualPalette.Accent).
    Bold(true)

panelStyle = lipgloss.NewStyle().
    Foreground(visualPalette.Text).
    Border(lipgloss.RoundedBorder()).
    BorderForeground(visualPalette.FocusBorder).
    Padding(1, 2)
```

Run `rg '\.Background\(' internal/tui`. Remove remaining TUI-chrome uses. Prompt preview rendering in `internal/prompt` is excluded.

- [ ] **Step 4: Remove unused palette fields**

Run:

```bash
rg 'visualPalette\.(Surface|Panel)' internal/tui
```

When it returns no matches, remove `Surface` and `Panel` from the palette struct and initializer.

- [ ] **Step 5: Run TUI tests**

```bash
go test ./internal/tui -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/visual_styles.go internal/tui/fullscreen_layout.go internal/tui/transparent_chrome_test.go internal/tui/visual_hierarchy_test.go
git commit -m "fix: remove background fills from TUI chrome"
```

---

### Task 2: Extract repository validation and candidate discovery

**Files:**
- Create: `internal/plugins/repository.go`
- Create: `internal/plugins/repository_test.go`
- Create: `internal/plugins/candidates.go`
- Create: `internal/plugins/candidates_test.go`
- Modify: `internal/plugins/plugins.go`
- Modify: `internal/plugins/plugins_test.go`

**Interfaces:**

```go
type Repository struct {
    URL  string
    Name string
}

func ParseRepository(raw string) (Repository, error)
func ValidateNewRepository(cfg *config.Config, repository Repository) error
func ValidateLoadPath(load string) error

type ScanLimits struct {
    MaxDepth      int
    MaxCandidates int
}

var DefaultScanLimits = ScanLimits{MaxDepth: 4, MaxCandidates: 128}

type Candidate struct {
    RelativePath string
    Kind         string
    Depth        int
}

func DiscoverCandidates(root, repositoryName string, limits ScanLimits) ([]Candidate, error)
func ValidateCandidate(root, relative string) error
```

- [ ] **Step 1: Write failing repository tests**

```go
func TestParseRepositoryAcceptsHTTPSGitURL(t *testing.T) {
    got, err := ParseRepository("https://github.com/example/demo.git")
    if err != nil {
        t.Fatalf("ParseRepository() error = %v", err)
    }
    if got != (Repository{URL: "https://github.com/example/demo.git", Name: "demo"}) {
        t.Fatalf("ParseRepository() = %#v", got)
    }
}

func TestParseRepositoryRejectsUnsafeForms(t *testing.T) {
    cases := []string{
        "http://example.com/demo.git",
        "https://user:secret@example.com/demo.git",
        "https://example.com/demo.git?token=secret",
        "https://example.com/demo.git#readme",
    }
    for _, raw := range cases {
        if _, err := ParseRepository(raw); err == nil {
            t.Fatalf("ParseRepository(%q) error = nil", raw)
        }
    }
}
```

- [ ] **Step 2: Run repository tests and verify failure**

```bash
go test ./internal/plugins -run TestParseRepository -count=1
```

Expected: compile failure because `ParseRepository` is undefined.

- [ ] **Step 3: Implement reusable validators**

```go
func ParseRepository(raw string) (Repository, error) {
    raw = strings.TrimSpace(raw)
    parsed, err := url.Parse(raw)
    if err != nil {
        return Repository{}, fmt.Errorf("invalid plugin URL: %w", err)
    }
    if parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
        return Repository{}, fmt.Errorf("plugin URL must be an https repository URL without credentials")
    }
    if parsed.RawQuery != "" || parsed.ForceQuery {
        return Repository{}, fmt.Errorf("plugin URL must not include a query string")
    }
    if parsed.Fragment != "" {
        return Repository{}, fmt.Errorf("plugin URL must not include a fragment")
    }
    name := strings.TrimSuffix(filepath.Base(strings.TrimSuffix(parsed.Path, "/")), ".git")
    if err := validateName(name); err != nil {
        return Repository{}, err
    }
    return Repository{URL: raw, Name: name}, nil
}
```

Move current load validation behind exported `ValidateLoadPath`. Make legacy `Add` call these helpers so CLI behavior remains unchanged.

- [ ] **Step 4: Write failing candidate tests**

```go
func TestDiscoverCandidatesRanksExpectedFiles(t *testing.T) {
    root := t.TempDir()
    for _, path := range []string{
        "demo.plugin.zsh",
        "other.plugin.zsh",
        "plugin.zsh",
        "plugin.sh",
        "lib/nested.plugin.zsh",
    } {
        writeCandidate(t, root, path)
    }
    got, err := DiscoverCandidates(root, "demo", DefaultScanLimits)
    if err != nil {
        t.Fatalf("DiscoverCandidates() error = %v", err)
    }
    want := []string{"demo.plugin.zsh", "other.plugin.zsh", "plugin.zsh", "plugin.sh", "lib/nested.plugin.zsh"}
    if !reflect.DeepEqual(candidatePaths(got), want) {
        t.Fatalf("candidate paths = %#v, want %#v", candidatePaths(got), want)
    }
}

func TestDiscoverCandidatesRejectsMatchingSymlink(t *testing.T) {
    root := t.TempDir()
    target := filepath.Join(root, "real.zsh")
    if err := os.WriteFile(target, []byte("# plugin\n"), 0o600); err != nil {
        t.Fatal(err)
    }
    if err := os.Symlink(target, filepath.Join(root, "demo.plugin.zsh")); err != nil {
        t.Fatal(err)
    }
    if _, err := DiscoverCandidates(root, "demo", DefaultScanLimits); err == nil {
        t.Fatal("DiscoverCandidates() error = nil, want symlink rejection")
    }
}
```

- [ ] **Step 5: Implement bounded scanning and sorting**

Use `filepath.WalkDir`. Skip `.git`, `vendor`, `node_modules`, `docs`, `examples`, `test`, `tests`, `dist`, and `build`. Stop descending beyond depth `4`. Return an error when accepted candidates exceed `128`.

Sort by:

1. root `<repositoryName>.plugin.zsh`;
2. other root `.plugin.zsh`;
3. root `.zsh`;
4. root `.sh`;
5. nested candidates by suffix priority, then depth, then lexical path.

`ValidateCandidate` must clean the relative path, prove it stays under `root`, `Lstat` every path component, reject symlinks, and require a readable regular file.

- [ ] **Step 6: Run plugin tests**

```bash
go test ./internal/plugins -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/plugins/repository.go internal/plugins/repository_test.go internal/plugins/candidates.go internal/plugins/candidates_test.go internal/plugins/plugins.go internal/plugins/plugins_test.go
git commit -m "feat: discover safe plugin load candidates"
```

---

### Task 3: Add cancellable staged cloning

**Files:**
- Create: `internal/plugins/staging.go`
- Create: `internal/plugins/staging_test.go`

**Interfaces:**

```go
type CloneRunner interface {
    Clone(ctx context.Context, repositoryURL, destination string) error
}

type ExecCloneRunner struct{}

func (ExecCloneRunner) Clone(ctx context.Context, repositoryURL, destination string) error

type StagedRepository struct {
    Repository Repository
    StagingDir string
    FinalDir   string
    Candidates []Candidate
}

func StageRepository(ctx context.Context, cfg *config.Config, rawURL string, runner CloneRunner) (StagedRepository, error)
func (stage StagedRepository) Cleanup() error
```

- [ ] **Step 1: Write failing success and cleanup tests**

```go
type cloneFunc func(context.Context, string, string) error

func (fn cloneFunc) Clone(ctx context.Context, repositoryURL, destination string) error {
    return fn(ctx, repositoryURL, destination)
}

func TestStageRepositoryClonesAndFindsCandidates(t *testing.T) {
    t.Setenv("HOME", t.TempDir())
    runner := cloneFunc(func(_ context.Context, _ string, destination string) error {
        return os.WriteFile(filepath.Join(destination, "demo.plugin.zsh"), []byte("# plugin\n"), 0o600)
    })
    stage, err := StageRepository(context.Background(), config.Default(), "https://example.com/demo.git", runner)
    if err != nil {
        t.Fatalf("StageRepository() error = %v", err)
    }
    if !strings.HasPrefix(filepath.Base(stage.StagingDir), ".staging-") {
        t.Fatalf("staging directory = %q", stage.StagingDir)
    }
    if len(stage.Candidates) != 1 || stage.Candidates[0].RelativePath != "demo.plugin.zsh" {
        t.Fatalf("candidates = %#v", stage.Candidates)
    }
    if err := stage.Cleanup(); err != nil {
        t.Fatalf("Cleanup() error = %v", err)
    }
    if _, err := os.Stat(stage.StagingDir); !os.IsNotExist(err) {
        t.Fatalf("staging directory still exists: %v", err)
    }
}
```

- [ ] **Step 2: Write failing cancellation cleanup test**

```go
func TestStageRepositoryCancellationRemovesStagingDirectory(t *testing.T) {
    t.Setenv("HOME", t.TempDir())
    ctx, cancel := context.WithCancel(context.Background())
    cancel()
    runner := cloneFunc(func(ctx context.Context, _, _ string) error { return ctx.Err() })
    _, err := StageRepository(ctx, config.Default(), "https://example.com/demo.git", runner)
    if !errors.Is(err, context.Canceled) {
        t.Fatalf("StageRepository() error = %v", err)
    }
    matches, _ := filepath.Glob(filepath.Join(Dir(), ".staging-*"))
    if len(matches) != 0 {
        t.Fatalf("cancel left staging directories: %v", matches)
    }
}
```

- [ ] **Step 3: Run tests and verify failure**

```bash
go test ./internal/plugins -run StageRepository -count=1
```

Expected: compile failure because staging types are undefined.

- [ ] **Step 4: Implement staging**

`ExecCloneRunner.Clone` runs:

```go
cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", repositoryURL, destination)
output, err := cmd.CombinedOutput()
```

`StageRepository` parses and duplicate-checks the repository, creates `Dir()` and `.staging-*` with `0700`, invokes the runner, discovers candidates, and returns `FinalDir: filepath.Join(Dir(), repository.Name)`. Every error path removes the staging directory.

- [ ] **Step 5: Add a timeout test for `ExecCloneRunner`**

Place a fake `git` script first on `PATH` that sleeps for one second. Run with a `20ms` context timeout. Assert `ctx.Err() == context.DeadlineExceeded` and that no staging path remains.

- [ ] **Step 6: Run tests**

```bash
go test ./internal/plugins -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/plugins/staging.go internal/plugins/staging_test.go
git commit -m "feat: stage custom plugin repositories safely"
```

---

### Task 4: Model pending filesystem changes

**Files:**
- Create: `internal/plugins/changes.go`
- Create: `internal/plugins/changes_test.go`

**Interfaces:**

```go
type PendingAdd struct {
    Name          string
    RepositoryURL string
    StagingDir    string
    FinalDir      string
    Load          string
}

type PendingRemove struct {
    Name   string
    Source string
}

type ChangeSet struct {
    Adds    []PendingAdd
    Removes []PendingRemove
}

func (changes ChangeSet) Clone() ChangeSet
func (changes ChangeSet) Empty() bool
func (changes ChangeSet) Counts() (adds, removes int)
func (changes ChangeSet) RepositoryURLFor(name string) (string, bool)
func (changes ChangeSet) RootFor(name, finalSource string) string
func (changes *ChangeSet) QueueAdd(cfg *config.Config, stage StagedRepository, load string) error
func (changes *ChangeSet) QueueRemove(cfg *config.Config, name string) error
func (changes ChangeSet) Cleanup() error
func (changes ChangeSet) Begin(cfg *config.Config) (*Transaction, error)

func (transaction *Transaction) Commit() error
func (transaction *Transaction) Rollback() error
```

- [ ] **Step 1: Write queue tests**

```go
func TestChangeSetQueueAddCreatesTrustedPendingItem(t *testing.T) {
    t.Setenv("HOME", t.TempDir())
    stage := stagedFixture(t, "demo", "demo.plugin.zsh")
    cfg := config.Default()
    var changes ChangeSet
    if err := changes.QueueAdd(cfg, stage, "demo.plugin.zsh"); err != nil {
        t.Fatalf("QueueAdd() error = %v", err)
    }
    item := cfg.Plugins.Items[len(cfg.Plugins.Items)-1]
    if item.Name != "demo" || item.Source != stage.FinalDir || !item.Enabled || !item.Trusted {
        t.Fatalf("pending item = %#v", item)
    }
}

func TestQueueRemoveCancelsUnappliedAdd(t *testing.T) {
    t.Setenv("HOME", t.TempDir())
    stage := stagedFixture(t, "demo", "demo.plugin.zsh")
    cfg := config.Default()
    var changes ChangeSet
    if err := changes.QueueAdd(cfg, stage, "demo.plugin.zsh"); err != nil {
        t.Fatal(err)
    }
    if err := changes.QueueRemove(cfg, "demo"); err != nil {
        t.Fatal(err)
    }
    if !changes.Empty() || len(cfg.Plugins.Items) != 0 {
        t.Fatalf("changes=%#v items=%#v", changes, cfg.Plugins.Items)
    }
    if _, err := os.Stat(stage.StagingDir); !os.IsNotExist(err) {
        t.Fatalf("staging directory remains: %v", err)
    }
}
```

- [ ] **Step 2: Write concrete rollback test**

```go
func TestTransactionRollbackRestoresAddAndRemoval(t *testing.T) {
    t.Setenv("HOME", t.TempDir())
    cfg := config.Default()

    installed := filepath.Join(Dir(), "old")
    if err := os.MkdirAll(installed, 0o700); err != nil {
        t.Fatal(err)
    }
    if err := os.WriteFile(filepath.Join(installed, "old.zsh"), []byte("# old\n"), 0o600); err != nil {
        t.Fatal(err)
    }
    cfg.Plugins.Items = append(cfg.Plugins.Items, config.PluginItem{
        Name: "old", Enabled: true, Trusted: true, Source: installed, Load: "old.zsh",
    })

    stage := stagedFixture(t, "demo", "demo.plugin.zsh")
    var changes ChangeSet
    if err := changes.QueueAdd(cfg, stage, "demo.plugin.zsh"); err != nil {
        t.Fatal(err)
    }
    if err := changes.QueueRemove(cfg, "old"); err != nil {
        t.Fatal(err)
    }

    transaction, err := changes.Begin(cfg)
    if err != nil {
        t.Fatalf("Begin() error = %v", err)
    }
    if _, err := os.Stat(stage.FinalDir); err != nil {
        t.Fatalf("final add missing after Begin: %v", err)
    }
    if _, err := os.Stat(installed); !os.IsNotExist(err) {
        t.Fatalf("removal source still exists after Begin: %v", err)
    }

    if err := transaction.Rollback(); err != nil {
        t.Fatalf("Rollback() error = %v", err)
    }
    if _, err := os.Stat(stage.StagingDir); err != nil {
        t.Fatalf("staging add not restored: %v", err)
    }
    if _, err := os.Stat(stage.FinalDir); !os.IsNotExist(err) {
        t.Fatalf("final add remains after rollback: %v", err)
    }
    if _, err := os.Stat(installed); err != nil {
        t.Fatalf("removed plugin not restored: %v", err)
    }
}
```

- [ ] **Step 3: Run tests and verify failure**

```bash
go test ./internal/plugins -run 'ChangeSet|Transaction' -count=1
```

Expected: compile failure because change-set types are undefined.

- [ ] **Step 4: Implement queue semantics**

`QueueAdd` validates the chosen file against `stage.StagingDir`, rejects duplicate names/final paths, records `PendingAdd`, and appends:

```go
config.PluginItem{
    Name: stage.Repository.Name,
    Enabled: true,
    Trusted: true,
    Source: stage.FinalDir,
    Load: filepath.ToSlash(filepath.Clean(load)),
}
```

`QueueRemove` cancels and cleans an unapplied add, or records an installed removal and removes the item from pending config without touching disk.

- [ ] **Step 5: Implement begin, commit, and rollback**

`Begin` validates every path before mutation. Rename additions `StagingDir -> FinalDir`. Rename removals `Source -> Source + ".ozsh-remove-<nanoseconds>"`. Journal every rename. On failure, reverse completed entries.

`Rollback` reverses entries in reverse order. `Commit` deletes removal quarantine directories and leaves finalized additions intact. Combine primary and rollback errors with `errors.Join`.

- [ ] **Step 6: Add path and conflict tests**

Test duplicate adds, add/remove name conflict, staging outside `Dir()`, incorrect final path, symlink load target, and removal source outside the managed root.

- [ ] **Step 7: Run tests**

```bash
go test ./internal/plugins -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/plugins/changes.go internal/plugins/changes_test.go
git commit -m "feat: queue transactional plugin filesystem changes"
```

---

### Task 5: Apply plugin changes transactionally

**Files:**
- Create: `internal/apply/snapshot.go`
- Create: `internal/apply/snapshot_test.go`
- Modify: `internal/apply/apply.go`
- Modify: `internal/apply/apply_test.go`

**Interfaces:**

```go
type Request struct {
    Config        *config.Config
    PluginChanges plugins.ChangeSet
}

func Apply(request Request) error
func ApplyConfig(cfg *config.Config) error
```

Compatibility wrapper:

```go
func ApplyConfig(cfg *config.Config) error {
    return Apply(Request{Config: cfg})
}
```

- [ ] **Step 1: Write snapshot tests**

```go
func TestFileSnapshotRestoresExistingFile(t *testing.T) {
    path := filepath.Join(t.TempDir(), "file")
    if err := os.WriteFile(path, []byte("before"), 0o640); err != nil {
        t.Fatal(err)
    }
    snapshot, err := captureFile(path)
    if err != nil {
        t.Fatal(err)
    }
    if err := os.WriteFile(path, []byte("after"), 0o600); err != nil {
        t.Fatal(err)
    }
    if err := snapshot.Restore(); err != nil {
        t.Fatal(err)
    }
    data, _ := os.ReadFile(path)
    if string(data) != "before" {
        t.Fatalf("restored data = %q", data)
    }
}

func TestFileSnapshotRestoresMissingFileByRemoval(t *testing.T) {
    path := filepath.Join(t.TempDir(), "missing")
    snapshot, err := captureFile(path)
    if err != nil {
        t.Fatal(err)
    }
    if err := os.WriteFile(path, []byte("created"), 0o600); err != nil {
        t.Fatal(err)
    }
    if err := snapshot.Restore(); err != nil {
        t.Fatal(err)
    }
    if _, err := os.Stat(path); !os.IsNotExist(err) {
        t.Fatalf("path still exists: %v", err)
    }
}
```

- [ ] **Step 2: Implement atomic restore**

Capture existence, bytes, and permission bits. Restore through a same-directory temporary file and `os.Rename`; remove the current path when the original did not exist.

- [ ] **Step 3: Write apply finalization and rollback tests**

Add `TestApplyFinalizesPendingPluginAdd` using a staged fixture and `Request`. Add `TestApplyOmegaFailureRestoresFilesAndPluginPaths` by creating `shell.OmegaZshPath()` as a directory, then assert config bytes, `.zshrc`, staging path, final path, and removed-plugin path exactly match pre-apply state.

- [ ] **Step 4: Implement request apply order**

Use this order:

1. clone config;
2. generate prompt;
3. preflight `.zshrc`;
4. snapshot config, `omega.zsh`, and `.zshrc`;
5. begin plugin transaction;
6. save config;
7. write `omega.zsh`;
8. inject `.zshrc`;
9. commit plugin transaction.

On failure after step 5, rollback plugins and restore all file snapshots in reverse write order. Combine errors with `errors.Join`.

- [ ] **Step 5: Preserve stable error prefixes**

Use `generate prompt`, `preflight .zshrc`, `save config`, `write omega.zsh`, and `inject .zshrc`. Update old tests to assert these prefixes.

- [ ] **Step 6: Run tests**

```bash
go test ./internal/apply ./internal/plugins -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/apply/apply.go internal/apply/apply_test.go internal/apply/snapshot.go internal/apply/snapshot_test.go
git commit -m "feat: apply pending plugin changes transactionally"
```

---

### Task 6: Render recommended and custom plugin sections

**Files:**
- Create: `internal/tui/plugin_items.go`
- Modify: `internal/tui/contextual_screens.go`
- Modify: `internal/tui/tui_v2_part1.go`
- Modify: `internal/tui/fullscreen_layout.go`
- Modify: `internal/tui/contextual_screens_test.go`

**Interfaces:**

```go
type pluginItemKind uint8

const (
    pluginItemRecommended pluginItemKind = iota
    pluginItemCustom
)

type pluginListItem struct {
    Kind        pluginItemKind
    Definition  plugins.Definition
    ConfigIndex int
}

func (m Model) pluginListItems() []pluginListItem
func (m Model) selectedPluginListItem() (pluginListItem, bool)
```

- [ ] **Step 1: Write failing UI tests**

```go
func TestPluginsWorkspaceShowsCustomSectionAndAddAction(t *testing.T) {
    model := NewModel(config.Default())
    model.width, model.height = 58, 28
    model.setTab(tabPlugins)
    plain := plainText(model.View())
    for _, expected := range []string{
        "Recommended plugins",
        "Custom plugins",
        "No custom plugins yet",
        "[a] Add custom plugin",
    } {
        if !strings.Contains(plain, expected) {
            t.Fatalf("plugins screen lost %q:\n%s", expected, plain)
        }
    }
}
```

Add a custom config item, set cursor to `len(plugins.Catalog())`, and assert `selectedPluginListItem()` returns `pluginItemCustom`.

- [ ] **Step 2: Run tests and verify failure**

```bash
go test ./internal/tui -run 'TestPluginsWorkspace|TestPluginsCursor' -count=1
```

Expected: failure because the custom section and flattened model do not exist.

- [ ] **Step 3: Implement flattened navigation**

Append curated definitions first, then config items not found by `plugins.FindDefinition`. Change Plugins `selectionCount()` to `len(m.pluginListItems())`. Remove direct cursor arithmetic as callers migrate.

- [ ] **Step 4: Render sections and details**

Plain output must include:

```text
Recommended plugins
› [x] Autosuggestions [active]
  [ ] fzf-tab [missing]

Custom plugins
  No custom plugins yet

[a] Add custom plugin
```

Use foreground-only badges: `[active]`, `[disabled]`, `[untrusted]`, `[attention]`, `[pending]`. Custom details show managed path, load file, trust, enabled, active state, and repository URL from `pluginChanges.RepositoryURLFor` when available.

- [ ] **Step 5: Update Plugins key ownership**

Use this footer:

```go
return renderHint("a add custom  ·  space enable  ·  t/u trust  ·  l load file  ·  d remove  ·  Ctrl+A apply  ·  Ctrl+C quit")
```

On Plugins, plain `a` opens the wizard. `Ctrl+A` opens Review & Apply. Preserve plain `a` behavior on other screens.

- [ ] **Step 6: Run responsive tests**

```bash
go test ./internal/tui -run 'Plugins|Responsive|Bounds' -count=1
```

Expected: PASS at `100x34`, `72x30`, `58x28`, and `48x18`.

- [ ] **Step 7: Commit**

```bash
git add internal/tui/plugin_items.go internal/tui/contextual_screens.go internal/tui/tui_v2_part1.go internal/tui/fullscreen_layout.go internal/tui/contextual_screens_test.go
git commit -m "feat: expose custom plugins in the TUI"
```

---

### Task 7: Implement the add-plugin wizard

**Files:**
- Create: `internal/tui/plugin_wizard.go`
- Create: `internal/tui/plugin_wizard_view.go`
- Create: `internal/tui/plugin_wizard_test.go`
- Modify: `internal/tui/tui_v2_part1.go`
- Modify: `internal/tui/fullscreen_layout.go`
- Modify: `internal/tui/tui_v2_part6.go`

**Interfaces:**

```go
type pluginWizardStep uint8

const (
    pluginWizardClosed pluginWizardStep = iota
    pluginWizardURL
    pluginWizardCloning
    pluginWizardCandidates
    pluginWizardTrust
    pluginWizardSummary
)

type pluginWizardMode uint8

const (
    pluginWizardAdd pluginWizardMode = iota
    pluginWizardChangeLoad
)

type pluginWizardModel struct {
    Step      pluginWizardStep
    Mode      pluginWizardMode
    URL       textinput.Model
    Stage     *plugins.StagedRepository
    Candidate int
    Error     string
    RequestID uint64
    Cancel    context.CancelFunc
}

type pluginStageResult struct {
    RequestID uint64
    Stage     plugins.StagedRepository
    Err       error
}
```

Add to `Model`:

```go
pluginWizard      pluginWizardModel
pluginChanges     plugins.ChangeSet
pluginCloneRunner plugins.CloneRunner
```

Remove `pluginURL`, `pluginLoad`, `pluginFocus`, and `pluginAdvanced` after migration.

- [ ] **Step 1: Write wizard opening test**

```go
func TestPluginWizardOpensFromPluginsScreen(t *testing.T) {
    model := NewModel(config.Default())
    model.setTab(tabPlugins)
    updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
    got := updated.(Model)
    if got.pluginWizard.Step != pluginWizardURL || !got.pluginWizard.URL.Focused() {
        t.Fatalf("wizard = %#v", got.pluginWizard)
    }
}
```

- [ ] **Step 2: Write URL, result, and cancellation tests**

Test invalid HTTP URL stays on URL step with an error. Inject a blocking `CloneRunner`; Enter must move to Cloning and return a command. Esc must call cancel. A stale `RequestID` result must be ignored and its stage cleaned. A current successful result must move to Candidates.

- [ ] **Step 3: Run tests and verify failure**

```bash
go test ./internal/tui -run PluginWizard -count=1
```

Expected: compile failure because wizard types are undefined.

- [ ] **Step 4: Implement clone command**

```go
ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
m.pluginWizard.RequestID++
requestID := m.pluginWizard.RequestID
m.pluginWizard.Cancel = cancel
m.pluginWizard.Step = pluginWizardCloning
snapshot := cloneConfig(m.cfg)
runner := m.pluginCloneRunner
cmd := func() tea.Msg {
    defer cancel()
    stage, err := plugins.StageRepository(ctx, snapshot, rawURL, runner)
    return pluginStageResult{RequestID: requestID, Stage: stage, Err: err}
}
```

- [ ] **Step 5: Implement transitions**

- URL: Enter validates and starts clone; Esc closes.
- Cloning: Esc cancels; current result opens Candidates.
- Candidates: `up/k`, `down/j`, Enter opens Trust, Esc cleans stage and returns URL.
- Trust: `y/enter` opens Summary; `n/esc` returns Candidates.
- Summary: Enter calls `pluginChanges.QueueAdd`, closes, selects the new item, and reports `plugin queued; Review & Apply to activate`.

- [ ] **Step 6: Render each state**

Show URL input, clone progress, candidate paths, trust warning, final managed path, selected load file, and pending summary. Use borders and foreground styles only.

- [ ] **Step 7: Route wizard before global navigation**

```go
if m.pluginWizard.Step != pluginWizardClosed {
    return m.updatePluginWizard(msg)
}
```

Render wizard in `workspaceContent` before normal tab content.

- [ ] **Step 8: Run tests**

```bash
go test ./internal/tui -run 'PluginWizard|Plugins|Bounds|Chrome' -count=1
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/tui/plugin_wizard.go internal/tui/plugin_wizard_view.go internal/tui/plugin_wizard_test.go internal/tui/tui_v2_part1.go internal/tui/fullscreen_layout.go internal/tui/tui_v2_part6.go
git commit -m "feat: add guided custom plugin wizard"
```

---

### Task 8: Convert lifecycle actions to pending changes

**Files:**
- Modify: `internal/tui/tui_v2_part5.go`
- Modify: `internal/tui/plugin_wizard.go`
- Modify: `internal/tui/plugin_items.go`
- Modify: `internal/tui/contextual_screens.go`
- Create: `internal/tui/plugin_lifecycle_test.go`

**Interfaces:**

```go
func (m *Model) toggleCustomPlugin(item pluginListItem)
func (m *Model) trustCustomPlugin(item pluginListItem, trusted bool)
func (m *Model) openLoadFilePicker(item pluginListItem) error
func (m *Model) queueCustomPluginRemoval(item pluginListItem) error
```

- [ ] **Step 1: Write pending-state test**

```go
func TestCustomPluginToggleRemainsPending(t *testing.T) {
    t.Setenv("HOME", t.TempDir())
    cfg := customPluginConfig(t, "demo", true, true)
    model := NewModel(cfg)
    model.setTab(tabPlugins)
    model.cursor = len(plugins.Catalog())
    model.togglePluginAtCursor()
    if model.cfg.Plugins.Items[0].Enabled {
        t.Fatal("plugin remained enabled")
    }
    if _, err := os.Stat(config.Path()); !os.IsNotExist(err) {
        t.Fatalf("toggle saved before Apply: %v", err)
    }
}
```

- [ ] **Step 2: Write removal, trust, and load-file tests**

Test these exact outcomes:

- installed removal is queued while its directory remains;
- unapplied-add removal cleans staging and removes the pending add;
- load-file change updates pending config only;
- staged trust validates against staging root;
- installed trust validates against final root;
- untrusting makes pending activation false.

- [ ] **Step 3: Remove immediate saves**

Delete `config.Save` and `plugins.AddAndSave` calls from TUI lifecycle helpers. Recommended selection remains in pending `m.cfg`.

- [ ] **Step 4: Implement change-load mode**

For `pluginWizardChangeLoad`, skip URL and clone. Run `DiscoverCandidates` on `pluginChanges.RootFor(item.Name, item.Source)`, show the candidate selector, validate the chosen file, and update `cfg.Plugins.Items[item.ConfigIndex].Load` after confirmation.

- [ ] **Step 5: Implement removal confirmation**

Add:

```go
pluginRemoveConfirm bool
pluginRemoveName    string
```

On a custom row, `d` opens confirmation. `y/enter` calls `QueueRemove`; `n/esc` cancels. Recommended rows report that curated entries can be deselected but not removed.

- [ ] **Step 6: Update details actions**

Custom details show:

```text
[space] Enable/disable
[t/u]   Trust/remove trust
[l]     Change load file
[d]     Remove plugin
```

- [ ] **Step 7: Run tests**

```bash
go test ./internal/tui ./internal/plugins -run 'Plugin|ChangeSet' -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/tui/tui_v2_part5.go internal/tui/plugin_wizard.go internal/tui/plugin_items.go internal/tui/contextual_screens.go internal/tui/plugin_lifecycle_test.go
git commit -m "refactor: defer custom plugin changes until apply"
```

---

### Task 9: Integrate pending changes into Review & Apply

**Files:**
- Modify: `internal/tui/tui_v2_part1.go`
- Modify: `internal/tui/tui_v2_part2.go`
- Modify: `internal/tui/tui_v2_part6.go`
- Modify: `internal/tui/apply_workspace.go`
- Modify: `internal/tui/fullscreen_layout.go`
- Create: `internal/tui/apply_plugin_changes_test.go`

**Interfaces:**

```go
type applyResult struct {
    Config *config.Config
    Err    error
}
```

Add to `Model`:

```go
reviewedPluginChanges plugins.ChangeSet
```

- [ ] **Step 1: Write review snapshot test**

```go
func TestOpenApplyReviewSnapshotsPluginChanges(t *testing.T) {
    model := modelWithPendingPluginAdd(t)
    model.openApplyReview()
    model.pluginChanges.Adds[0].Load = "mutated.zsh"
    if model.reviewedPluginChanges.Adds[0].Load == "mutated.zsh" {
        t.Fatal("reviewed change set was not cloned")
    }
}
```

Add view assertions for `Plugin changes`, `1 add`, `0 remove`, plugin name, load file, and final path.

- [ ] **Step 2: Update apply command**

```go
func doApply(cfg *config.Config, changes plugins.ChangeSet) tea.Cmd {
    snapshot := cloneConfig(cfg)
    changeSnapshot := changes.Clone()
    return func() tea.Msg {
        err := applyop.Apply(applyop.Request{Config: snapshot, PluginChanges: changeSnapshot})
        return applyResult{Config: snapshot, Err: err}
    }
}
```

Success assigns config, clears both change sets, and reports `applied`. Failure preserves pending state for retry.

- [ ] **Step 3: Update Review & Apply content**

Show add/remove counts and each action before technical `.zshrc` details. Render from `reviewedPluginChanges`, not mutable live state.

- [ ] **Step 4: Clean staging when TUI exits**

```go
func Run() error {
    cfg, err := config.Load()
    if err != nil {
        return err
    }
    final, runErr := tea.NewProgram(NewModel(cfg), tea.WithAltScreen()).Run()
    if model, ok := final.(Model); ok {
        cleanupErr := model.pluginChanges.Cleanup()
        if runErr == nil && cleanupErr != nil {
            return cleanupErr
        }
    }
    return runErr
}
```

Clean a stale asynchronous result before ignoring it.

- [ ] **Step 5: Add retry and cleanup tests**

Verify apply failure keeps staging and pending changes; successful Apply clears them; wizard cancel deletes staging; TUI final cleanup deletes any unapplied stages.

- [ ] **Step 6: Run tests**

```bash
go test ./internal/tui ./internal/apply ./internal/plugins -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/tui/tui_v2_part1.go internal/tui/tui_v2_part2.go internal/tui/tui_v2_part6.go internal/tui/apply_workspace.go internal/tui/fullscreen_layout.go internal/tui/apply_plugin_changes_test.go
git commit -m "feat: review and apply pending plugin changes"
```

---

### Task 10: Documentation and complete verification

**Files:**
- Modify: `README.md`
- Modify: `docs/screencasts/tui-visual-hierarchy.tape`

- [ ] **Step 1: Update README**

Document recommended plugins, `[a] Add custom plugin`, URL validation, candidate selection, trust warning, pending state, Review & Apply, lifecycle actions, and `~/.config/ozsh/plugins` ownership. State that trusted shell plugins execute code in every interactive shell.

- [ ] **Step 2: Update VHS walkthrough**

Show transparent selection, open Add custom plugin, enter a local fake repository URL supported by the tape fixture, choose a candidate, reach trust review, cancel, and return to Plugins. The tape must not depend on internet access.

- [ ] **Step 3: Format and verify tidy state**

```bash
gofmt -w internal/plugins internal/apply internal/tui
go mod verify
go mod tidy -diff
```

Expected: all commands exit `0` and `go mod tidy -diff` prints no diff.

- [ ] **Step 4: Run lint exactly as CI**

```bash
scripts/lint.sh --check
```

Expected: exit `0` after gofmt, vet, shell syntax, ShellCheck, and golangci-lint checks.

- [ ] **Step 5: Run tests and coverage exactly as CI**

```bash
set -o pipefail
go test -coverprofile=coverage.out ./... 2>&1 | tee test-output.log
go tool cover -func=coverage.out | awk '/^total:/ { if ($3 + 0 < 70) { print "coverage " $3 " is below 70%"; exit 1 } }'
go test -race ./...
```

Expected: tests pass and total coverage is at least `70%`.

- [ ] **Step 6: Run builds and smoke tests exactly as CI**

```bash
go build -buildvcs=false ./...
scripts/release-smoke.sh
scripts/install-smoke.sh
CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -buildvcs=false ./cmd/ozsh
```

Expected: every command exits `0`.

- [ ] **Step 7: Run vulnerability scan**

```bash
go run "golang.org/x/vuln/cmd/govulncheck@v1.6.0" ./...
```

Expected: no reachable vulnerabilities. GitHub Actions performs the pinned Gitleaks scan.

- [ ] **Step 8: Perform the real Termux checklist**

Verify:

1. no gray filled rectangles around panel, tabs, selected rows, previews, badges, or wizard;
2. selection remains obvious with the keyboard open;
3. `Custom plugins` and `[a] Add custom plugin` remain visible at narrow width;
4. wizard cancellation removes `.staging-*`;
5. queued plugin displays `[pending]`;
6. Review & Apply lists final path and load file;
7. successful Apply moves the checkout and loads it only after trust confirmation.

- [ ] **Step 9: Commit documentation**

```bash
git add README.md docs/screencasts/tui-visual-hierarchy.tape
git commit -m "docs: explain custom plugin workflow"
```

- [ ] **Step 10: Push and open a draft PR**

```bash
git push -u origin feat/tui-plugins-polish
gh pr create --draft --base main --head feat/tui-plugins-polish --title "feat: add transparent TUI and extensible plugins" --body-file /tmp/ozsh-pr-a.md
```

The PR body must list transparent chrome, plugin wizard, transactional Apply, verification results, and the pending real-Termux visual check. Do not mark ready or merge before CI and that checklist pass.
