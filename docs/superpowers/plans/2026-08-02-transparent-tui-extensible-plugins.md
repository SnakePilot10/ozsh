# Transparent TUI and Extensible Plugins Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove background fills from ozsh TUI chrome and replace the hidden custom-plugin form with a safe, discoverable, keyboard-first plugin workflow whose filesystem changes are committed only through Review & Apply.

**Architecture:** Keep the current Bubble Tea model and fullscreen layout, but split plugin work into focused domain services: repository parsing, bounded candidate discovery, staged cloning, and a pending filesystem change set. The TUI owns only wizard state and pending configuration. `internal/apply` finalizes staged additions and removals transactionally with the existing config, `omega.zsh`, and `.zshrc` writes.

**Tech Stack:** Go, Bubble Tea, Bubbles `textinput`, Lip Gloss, `context`, `os/exec`, TOML-backed ozsh configuration, Go tests, GitHub Actions, Termux ARM64 cross-build.

## Global Constraints

- This plan implements PR A only. Do not add Starship or change the configuration version/schema.
- Native prompt previews may contain background ANSI codes; ozsh TUI chrome must not emit background SGR codes.
- Preserve the responsive breakpoint at `72` columns and all fullscreen width/height guarantees.
- Keep the workflow keyboard-first and usable below `72` columns.
- Accept only HTTPS repository URLs without credentials, query strings, or fragments.
- Clone with `git clone --depth 1` and a `45s` timeout.
- Staging directories live under `~/.config/ozsh/plugins/.staging-*` and use mode `0700` where supported.
- Candidate scanning is bounded to depth `4` and at most `128` accepted candidates.
- Accept only regular `.plugin.zsh`, `.zsh`, and `.sh` files. Reject symlink candidates and symlink path components.
- Downloading a plugin never implies trust. Explicit trust confirmation is required.
- Custom-plugin additions, removals, trust changes, load-file changes, and enabled state remain pending until Review & Apply.
- Long-running cloning work runs through `tea.Cmd`; `Update()` and `View()` stay non-blocking.
- Use TDD for every task and commit after each independently testable deliverable.

---

## File Structure

### New domain files

- `internal/plugins/repository.go`: repository URL parsing, safe repository naming, duplicate checks, and reusable load-path validation.
- `internal/plugins/repository_test.go`: repository validation tests.
- `internal/plugins/candidates.go`: bounded candidate discovery, ranking, and symlink rejection.
- `internal/plugins/candidates_test.go`: candidate ordering and scan-limit tests.
- `internal/plugins/staging.go`: injectable shallow clone runner, staging directory lifecycle, and cleanup.
- `internal/plugins/staging_test.go`: clone timeout, cancellation, cleanup, and permissions tests.
- `internal/plugins/changes.go`: pending add/remove model and filesystem transaction with commit/rollback.
- `internal/plugins/changes_test.go`: queueing, finalization, quarantine, rollback, and conflict tests.
- `internal/apply/snapshot.go`: capture and restore existing config, generated prompt, and `.zshrc` files.
- `internal/apply/snapshot_test.go`: existing/missing-file restoration tests.
- `internal/tui/plugin_items.go`: one flattened selection model over recommended and custom plugin sections.
- `internal/tui/plugin_wizard.go`: wizard state, updates, commands, cancellation, and pending-change integration.
- `internal/tui/plugin_wizard_view.go`: responsive wizard rendering.
- `internal/tui/plugin_wizard_test.go`: state-machine and cancellation tests.
- `internal/tui/transparent_chrome_test.go`: ANSI background-code regression tests.

### Modified files

- `internal/tui/visual_styles.go`: remove all chrome backgrounds and add marker/border/badge styles.
- `internal/tui/fullscreen_layout.go`: route wizard/removal confirmation views and update the Plugins footer.
- `internal/tui/contextual_screens.go`: render recommended and custom sections, details, empty state, and visible add action.
- `internal/tui/tui_v2_part1.go`: replace legacy plugin form fields with wizard/change-set state and route keys/messages.
- `internal/tui/tui_v2_part5.go`: replace immediate-save custom-plugin helpers with pending lifecycle actions.
- `internal/tui/tui_v2_part6.go`: pass an apply request containing a cloned plugin change set and preserve pending state on failure.
- `internal/apply/apply.go`: add request-based transactional apply while preserving `ApplyConfig` compatibility.
- `internal/apply/apply_test.go`: plugin finalization and file rollback tests.
- `internal/plugins/plugins.go`: reuse repository/load validators and remove the direct clone path from the TUI-facing workflow.
- `internal/plugins/plugins_test.go`: retain legacy API coverage and verify refactored validators.
- `internal/tui/contextual_screens_test.go`: plugin regions, empty state, custom rows, and details.
- `internal/tui/visual_hierarchy_test.go`: transparent and monochrome-identifiable selection checks.
- `docs/screencasts/tui-visual-hierarchy.tape`: demonstrate transparent chrome and adding a custom plugin.
- `README.md`: document recommended and custom plugin workflows.

---

### Task 1: Make TUI chrome transparent

**Files:**
- Modify: `internal/tui/visual_styles.go`
- Modify: `internal/tui/fullscreen_layout.go`
- Create: `internal/tui/transparent_chrome_test.go`
- Modify: `internal/tui/visual_hierarchy_test.go`

**Interfaces:**
- Consumes: existing `renderTab`, `renderPreviewBox`, `panelStyle`, `workspaceBoxStyle`, and `selectedRowStyle` render helpers.
- Produces: `containsBackgroundSGR(value string) bool` test helper and background-free semantic styles used by all later TUI tasks.

- [ ] **Step 1: Write the failing ANSI regression test**

```go
package tui

import (
    "regexp"
    "testing"

    "github.com/snakepilot10/ozsh/internal/config"
)

var backgroundSGR = regexp.MustCompile(`\x1b\[(?:4[0-9]|10[0-7]|48;(?:2|5);[0-9;]+)m`)

func TestTUIChromeDoesNotEmitBackgroundColors(t *testing.T) {
    for tab := range tabs {
        model := NewModel(config.Default())
        model.width, model.height = 100, 34
        model.setTab(tab)
        view := model.View()
        if backgroundSGR.MatchString(view) {
            t.Fatalf("tab %d emitted a background SGR sequence: %q", tab, backgroundSGR.FindString(view))
        }
    }
}

func TestPromptPreviewBackgroundIsExcludedFromChromeRule(t *testing.T) {
    styledPrompt := "\x1b[48;2;10;20;30msegment\x1b[0m"
    chrome := renderPreviewBox("Live preview", styledPrompt, 40)
    if !backgroundSGR.MatchString(chrome) {
        t.Fatal("test fixture lost the prompt-owned background sequence")
    }
}
```

- [ ] **Step 2: Run the focused tests and confirm failure**

Run:

```bash
go test ./internal/tui -run 'TestTUIChromeDoesNotEmitBackgroundColors|TestPromptPreviewBackgroundIsExcludedFromChromeRule' -count=1
```

Expected: `TestTUIChromeDoesNotEmitBackgroundColors` fails because `panelStyle`, active tabs, preview boxes, or badges emit `Background(...)` sequences.

- [ ] **Step 3: Remove background fills from semantic styles**

Change `visual_styles.go` so the affected definitions use foreground, border, bold, underline, and padding only:

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

Remove `Surface` and `Panel` from `visualPalette` only after `rg 'visualPalette\.(Surface|Panel)' internal/tui` returns no consumers.

- [ ] **Step 4: Add a chrome-only assertion helper**

Because prompt previews may legitimately contain backgrounds, add a helper that removes the content between preview boundary markers before checking a complete screen:

```go
func stripPromptPreviewPayload(view string) string {
    lines := strings.Split(view, "\n")
    for i, line := range lines {
        if strings.Contains(plainText(line), "Live preview") || strings.Contains(plainText(line), "Preview") {
            if i+2 < len(lines) {
                lines[i+2] = plainText(lines[i+2])
            }
        }
    }
    return strings.Join(lines, "\n")
}
```

Use `backgroundSGR.MatchString(stripPromptPreviewPayload(model.View()))` in the all-tabs test so only chrome is prohibited.

- [ ] **Step 5: Verify focus remains visible without color**

Add assertions that active tabs contain underline SGR and selected rows contain `›` after ANSI stripping:

```go
func TestSelectionRemainsIdentifiableWithoutColor(t *testing.T) {
    model := NewModel(config.Default())
    model.width, model.height = 58, 28
    model.setTab(tabPlugins)
    plain := plainText(model.View())
    if !strings.Contains(plain, "› [") {
        t.Fatalf("selected plugin row lost its marker:\n%s", plain)
    }
    if !strings.Contains(renderTab("Plugins", true, false), "\x1b[4m") {
        t.Fatal("active tab lost underline styling")
    }
}
```

- [ ] **Step 6: Run TUI tests**

```bash
go test ./internal/tui -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/tui/visual_styles.go internal/tui/fullscreen_layout.go internal/tui/transparent_chrome_test.go internal/tui/visual_hierarchy_test.go
git commit -m "fix: remove background fills from TUI chrome"
```

---

### Task 2: Extract repository validation and bounded candidate discovery

**Files:**
- Create: `internal/plugins/repository.go`
- Create: `internal/plugins/repository_test.go`
- Create: `internal/plugins/candidates.go`
- Create: `internal/plugins/candidates_test.go`
- Modify: `internal/plugins/plugins.go`
- Modify: `internal/plugins/plugins_test.go`

**Interfaces:**
- Produces:

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

- [ ] **Step 1: Write failing repository parser tests**

```go
func TestParseRepositoryAcceptsHTTPSGitURL(t *testing.T) {
    got, err := ParseRepository("https://github.com/example/demo.git")
    if err != nil {
        t.Fatalf("ParseRepository() error = %v", err)
    }
    if got.Name != "demo" || got.URL != "https://github.com/example/demo.git" {
        t.Fatalf("ParseRepository() = %#v", got)
    }
}

func TestParseRepositoryRejectsCredentialsQueryAndFragment(t *testing.T) {
    cases := []string{
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

- [ ] **Step 2: Run repository tests and confirm failure**

```bash
go test ./internal/plugins -run TestParseRepository -count=1
```

Expected: compile failure because `ParseRepository` does not exist.

- [ ] **Step 3: Implement reusable repository and load validators**

Move the URL/name/load rules from `Add` into `repository.go`. Keep error messages specific and preserve `validateName` as the internal primitive:

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

Make legacy `Add` call `ParseRepository` and `ValidateLoadPath` so old CLI behavior remains covered.

- [ ] **Step 4: Write failing candidate ranking tests**

```go
func TestDiscoverCandidatesRanksExpectedLoadFiles(t *testing.T) {
    root := t.TempDir()
    writeCandidate(t, root, "demo.plugin.zsh")
    writeCandidate(t, root, "other.plugin.zsh")
    writeCandidate(t, root, "plugin.zsh")
    writeCandidate(t, root, "plugin.sh")
    writeCandidate(t, root, "lib/nested.plugin.zsh")

    got, err := DiscoverCandidates(root, "demo", DefaultScanLimits)
    if err != nil {
        t.Fatalf("DiscoverCandidates() error = %v", err)
    }
    want := []string{"demo.plugin.zsh", "other.plugin.zsh", "plugin.zsh", "plugin.sh", "lib/nested.plugin.zsh"}
    if diff := cmp.Diff(want, candidatePaths(got)); diff != "" {
        t.Fatalf("candidate order mismatch (-want +got):\n%s", diff)
    }
}

func TestDiscoverCandidatesRejectsMatchingSymlink(t *testing.T) {
    root := t.TempDir()
    target := filepath.Join(root, "real.zsh")
    os.WriteFile(target, []byte("# plugin\n"), 0o600)
    os.Symlink(target, filepath.Join(root, "demo.plugin.zsh"))
    if _, err := DiscoverCandidates(root, "demo", DefaultScanLimits); err == nil {
        t.Fatal("DiscoverCandidates() error = nil, want symlink rejection")
    }
}
```

Use a local string-slice comparison helper if the repository does not already depend on `go-cmp`; do not add a new dependency only for these tests.

- [ ] **Step 5: Implement bounded discovery and deterministic ranking**

Use `filepath.WalkDir`. Skip `.git`, `vendor`, `node_modules`, `docs`, `examples`, `test`, `tests`, `dist`, and `build` directories. Stop descending when relative depth exceeds `limits.MaxDepth`. Return an error once accepted candidates exceed `limits.MaxCandidates`.

Sort using this exact key:

```go
type candidateSortKey struct {
    nested   bool
    suffix   int
    depth    int
    path     string
}
```

Suffix priority is `.plugin.zsh = 0`, `.zsh = 1`, `.sh = 2`. Before all other keys, give root-level `<repositoryName>.plugin.zsh` a dedicated priority of `-1`.

`ValidateCandidate` must clean the relative path, keep it inside `root`, walk every path component with `os.Lstat`, reject every symlink, and require the final target to be a readable regular file.

- [ ] **Step 6: Run plugin domain tests**

```bash
go test ./internal/plugins -count=1
```

Expected: PASS, including existing `Add`, trust, and removal tests.

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
- Consumes: `Repository`, `ParseRepository`, `ValidateNewRepository`, `DiscoverCandidates`, `DefaultScanLimits`.
- Produces:

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

- [ ] **Step 1: Write failing staged-clone success and cleanup tests**

```go
type cloneFunc func(context.Context, string, string) error

func (fn cloneFunc) Clone(ctx context.Context, repositoryURL, destination string) error {
    return fn(ctx, repositoryURL, destination)
}

func TestStageRepositoryClonesAndFindsCandidates(t *testing.T) {
    t.Setenv("HOME", t.TempDir())
    cfg := config.Default()
    runner := cloneFunc(func(_ context.Context, _ string, destination string) error {
        return os.WriteFile(filepath.Join(destination, "demo.plugin.zsh"), []byte("# plugin\n"), 0o600)
    })

    stage, err := StageRepository(context.Background(), cfg, "https://example.com/demo.git", runner)
    if err != nil {
        t.Fatalf("StageRepository() error = %v", err)
    }
    if filepath.Base(stage.StagingDir)[:9] != ".staging-" {
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

- [ ] **Step 2: Write cancellation and failed-clone cleanup tests**

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

- [ ] **Step 3: Run focused tests and confirm failure**

```bash
go test ./internal/plugins -run StageRepository -count=1
```

Expected: compile failure because staging types do not exist.

- [ ] **Step 4: Implement the execution runner and staging lifecycle**

`ExecCloneRunner.Clone` runs exactly:

```go
cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", repositoryURL, destination)
output, err := cmd.CombinedOutput()
```

Wrap errors with trimmed command output. `StageRepository` must:

1. parse and duplicate-check the repository;
2. create `Dir()` and chmod it `0700`;
3. call `os.MkdirTemp(Dir(), ".staging-")` and chmod it `0700`;
4. defer cleanup until all steps succeed;
5. call the injected runner;
6. discover candidates;
7. return `StagedRepository{FinalDir: filepath.Join(Dir(), repository.Name)}`.

Do not create the final directory in this task.

- [ ] **Step 5: Add an integration-style timeout test for `ExecCloneRunner`**

Place a fake `git` script first on `PATH` that sleeps. Use `context.WithTimeout(..., 20*time.Millisecond)` and assert `errors.Is(err, context.DeadlineExceeded)` or `ctx.Err() == context.DeadlineExceeded` after the call.

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

### Task 4: Model pending plugin filesystem changes

**Files:**
- Create: `internal/plugins/changes.go`
- Create: `internal/plugins/changes_test.go`

**Interfaces:**
- Consumes: `StagedRepository`, `ValidateCandidate`, `config.PluginItem`, and `Dir()`.
- Produces:

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
func (changes *ChangeSet) QueueAdd(cfg *config.Config, stage StagedRepository, load string) error
func (changes *ChangeSet) QueueRemove(cfg *config.Config, name string) error
func (changes ChangeSet) RootFor(name, finalSource string) string
func (changes ChangeSet) Cleanup() error
func (changes ChangeSet) Begin(cfg *config.Config) (*Transaction, error)

type Transaction struct { /* private rename journal */ }
func (transaction *Transaction) Commit() error
func (transaction *Transaction) Rollback() error
```

- [ ] **Step 1: Write failing queue-add and queue-remove tests**

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
    if len(changes.Adds) != 1 || changes.Adds[0].StagingDir != stage.StagingDir {
        t.Fatalf("pending adds = %#v", changes.Adds)
    }
}

func TestQueueRemoveCancelsUnappliedAdd(t *testing.T) {
    t.Setenv("HOME", t.TempDir())
    stage := stagedFixture(t, "demo", "demo.plugin.zsh")
    cfg := config.Default()
    var changes ChangeSet
    changes.QueueAdd(cfg, stage, "demo.plugin.zsh")

    if err := changes.QueueRemove(cfg, "demo"); err != nil {
        t.Fatalf("QueueRemove() error = %v", err)
    }
    if !changes.Empty() || len(cfg.Plugins.Items) != 0 {
        t.Fatalf("unapplied add not cancelled: changes=%#v items=%#v", changes, cfg.Plugins.Items)
    }
    if _, err := os.Stat(stage.StagingDir); !os.IsNotExist(err) {
        t.Fatalf("cancelled add left staging directory: %v", err)
    }
}
```

- [ ] **Step 2: Write failing filesystem transaction tests**

```go
func TestTransactionFinalizesAddAndCommit(t *testing.T) {
    t.Setenv("HOME", t.TempDir())
    stage := stagedFixture(t, "demo", "demo.plugin.zsh")
    cfg := config.Default()
    var changes ChangeSet
    changes.QueueAdd(cfg, stage, "demo.plugin.zsh")

    transaction, err := changes.Begin(cfg)
    if err != nil {
        t.Fatalf("Begin() error = %v", err)
    }
    if _, err := os.Stat(stage.FinalDir); err != nil {
        t.Fatalf("final directory unavailable after Begin: %v", err)
    }
    if err := transaction.Commit(); err != nil {
        t.Fatalf("Commit() error = %v", err)
    }
}

func TestTransactionRollbackRestoresStagingAndRemoval(t *testing.T) {
    // Arrange one staged add and one installed plugin removal.
    // Begin must rename add staging -> final and removal source -> quarantine.
    // Rollback must restore both original paths and leave no final/quarantine path.
}
```

Replace the comment-only body above with concrete `os.MkdirAll`, `os.WriteFile`, `QueueAdd`, `QueueRemove`, `Begin`, `Rollback`, and `os.Stat` assertions before committing. The expected paths are the exact `StagingDir`, `FinalDir`, and original installed `Source` values from the fixtures.

- [ ] **Step 3: Run tests and confirm failure**

```bash
go test ./internal/plugins -run 'ChangeSet|Transaction' -count=1
```

Expected: compile failure because change-set types do not exist.

- [ ] **Step 4: Implement queue semantics**

`QueueAdd` validates the chosen candidate against `stage.StagingDir`, rejects duplicate names and final destinations, appends one pending add, and appends this exact config item:

```go
config.PluginItem{
    Name: stage.Repository.Name,
    Enabled: true,
    Trusted: true,
    Source: stage.FinalDir,
    Load: filepath.ToSlash(filepath.Clean(load)),
}
```

`QueueRemove` behaves in two branches:

- If `name` is an unapplied add, remove its config item, delete its staging directory, and remove the add from `ChangeSet`.
- Otherwise, validate the installed managed source, append one `PendingRemove`, and remove the item from the pending config without touching disk.

- [ ] **Step 5: Implement begin/commit/rollback with a rename journal**

`Begin` revalidates every add and removal before mutating anything. For each add, rename `StagingDir` to `FinalDir`. For each removal, rename `Source` to a unique sibling path ending in `.ozsh-remove-<nanoseconds>`.

Journal entries must record:

```go
type renameEntry struct {
    from string
    to   string
    kind string // "add" or "remove"
}
```

If any rename fails, reverse completed entries immediately and return an error that includes rollback failure when present.

`Rollback` reverses entries in reverse order. `Commit` removes only removal quarantine directories; it does not delete finalized additions.

- [ ] **Step 6: Add conflict and path-safety tests**

Cover duplicate add names, add/remove conflicts, a staging path outside `Dir()`, a final path not equal to `filepath.Join(Dir(), name)`, a symlink load path, and a removal source outside the managed root.

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

### Task 5: Make Review & Apply finalize plugin changes transactionally

**Files:**
- Create: `internal/apply/snapshot.go`
- Create: `internal/apply/snapshot_test.go`
- Modify: `internal/apply/apply.go`
- Modify: `internal/apply/apply_test.go`

**Interfaces:**
- Consumes: `plugins.ChangeSet`, `plugins.Transaction`, existing `config.Save`, `shell.WriteOmega`, `shell.InjectBlock`, and `prompt.Generate`.
- Produces:

```go
type Request struct {
    Config        *config.Config
    PluginChanges plugins.ChangeSet
}

func Apply(request Request) error
func ApplyConfig(cfg *config.Config) error
```

`ApplyConfig` remains as compatibility wrapper:

```go
func ApplyConfig(cfg *config.Config) error {
    return Apply(Request{Config: cfg})
}
```

- [ ] **Step 1: Write snapshot restoration tests**

```go
func TestFileSnapshotRestoresExistingFile(t *testing.T) {
    path := filepath.Join(t.TempDir(), "file")
    os.WriteFile(path, []byte("before"), 0o640)
    snapshot, err := captureFile(path)
    if err != nil { t.Fatal(err) }
    os.WriteFile(path, []byte("after"), 0o600)
    if err := snapshot.Restore(); err != nil { t.Fatal(err) }
    data, _ := os.ReadFile(path)
    if string(data) != "before" { t.Fatalf("restored data = %q", data) }
}

func TestFileSnapshotRestoresMissingFileByRemoval(t *testing.T) {
    path := filepath.Join(t.TempDir(), "missing")
    snapshot, err := captureFile(path)
    if err != nil { t.Fatal(err) }
    os.WriteFile(path, []byte("created"), 0o600)
    if err := snapshot.Restore(); err != nil { t.Fatal(err) }
    if _, err := os.Stat(path); !os.IsNotExist(err) { t.Fatalf("path still exists: %v", err) }
}
```

- [ ] **Step 2: Implement atomic snapshot restore**

`captureFile` stores existence, bytes, and permission bits. `Restore` writes through a temporary file in the same directory followed by `os.Rename`; when the original did not exist, it removes the current path.

- [ ] **Step 3: Write failing apply success and rollback tests**

Add tests that construct a real `plugins.ChangeSet` fixture:

```go
func TestApplyFinalizesPendingPluginAdd(t *testing.T) {
    home := t.TempDir()
    t.Setenv("HOME", home)
    os.WriteFile(shell.ZshrcPath(), []byte("export EDITOR=vim\n"), 0o600)
    cfg, changes, stage := pendingAddFixture(t, "demo", "demo.plugin.zsh")

    if err := Apply(Request{Config: cfg, PluginChanges: changes}); err != nil {
        t.Fatalf("Apply() error = %v", err)
    }
    if _, err := os.Stat(stage.FinalDir); err != nil {
        t.Fatalf("plugin was not finalized: %v", err)
    }
    if _, err := os.Stat(stage.StagingDir); !os.IsNotExist(err) {
        t.Fatalf("staging directory still exists: %v", err)
    }
}
```

Add a failure test by creating `shell.OmegaZshPath()` as a directory. Assert that config bytes, `.zshrc`, and plugin paths exactly match their pre-apply state after the error.

- [ ] **Step 4: Implement request-based apply order**

Use this exact order:

1. clone and validate config;
2. generate prompt text;
3. preflight `.zshrc` injection;
4. capture snapshots for `config.Path()`, `shell.OmegaZshPath()`, and `shell.ZshrcPath()`;
5. call `PluginChanges.Begin(clone)`;
6. save config;
7. write `omega.zsh`;
8. inject `.zshrc` block;
9. commit plugin transaction.

On any error after step 5, call plugin rollback and restore all three file snapshots in reverse write order. Return the original error plus every rollback error using `errors.Join`.

- [ ] **Step 5: Preserve existing error prefixes**

Keep prefixes such as `generate prompt`, `preflight .zshrc`, `save config`, `write omega.zsh`, and `inject .zshrc`. Update old tests to assert these stable prefixes rather than the previous partial-write wording.

- [ ] **Step 6: Run apply and plugin tests**

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

### Task 6: Render recommended and custom plugins as one navigable screen

**Files:**
- Create: `internal/tui/plugin_items.go`
- Modify: `internal/tui/contextual_screens.go`
- Modify: `internal/tui/tui_v2_part1.go`
- Modify: `internal/tui/fullscreen_layout.go`
- Modify: `internal/tui/contextual_screens_test.go`

**Interfaces:**
- Consumes: `plugins.Catalog`, pending `config.PluginItem` values, and `plugins.ChangeSet.RootFor`.
- Produces:

```go
type pluginItemKind uint8

const (
    pluginItemRecommended pluginItemKind = iota
    pluginItemCustom
)

type pluginListItem struct {
    Kind       pluginItemKind
    Definition plugins.Definition
    ConfigIndex int
}

func (m Model) pluginListItems() []pluginListItem
func (m Model) selectedPluginListItem() (pluginListItem, bool)
```

- [ ] **Step 1: Write failing screen-region tests**

```go
func TestPluginsWorkspaceShowsCustomSectionAndAddAction(t *testing.T) {
    model := NewModel(config.Default())
    model.width, model.height = 58, 28
    model.setTab(tabPlugins)
    plain := plainText(model.View())
    for _, expected := range []string{"Recommended plugins", "Custom plugins", "No custom plugins yet", "[a] Add custom plugin"} {
        if !strings.Contains(plain, expected) {
            t.Fatalf("plugins screen lost %q:\n%s", expected, plain)
        }
    }
}

func TestPluginsCursorCanSelectCustomPlugin(t *testing.T) {
    cfg := config.Default()
    cfg.Plugins.Items = append(cfg.Plugins.Items, config.PluginItem{Name: "demo", Source: filepath.Join(plugins.Dir(), "demo"), Load: "demo.zsh"})
    model := NewModel(cfg)
    model.setTab(tabPlugins)
    model.cursor = len(plugins.Catalog())
    selected, ok := model.selectedPluginListItem()
    if !ok || selected.Kind != pluginItemCustom {
        t.Fatalf("selected item = %#v, %v", selected, ok)
    }
}
```

- [ ] **Step 2: Run tests and confirm failure**

```bash
go test ./internal/tui -run PluginsWorkspace -count=1
```

Expected: failure because the contextual plugin screen has no custom section or add action.

- [ ] **Step 3: Implement the flattened item model**

`pluginListItems` appends every curated definition first, then every config item whose name is not curated. `selectionCount()` for `tabPlugins` becomes `len(m.pluginListItems())`.

Remove direct cursor arithmetic based on `len(plugins.Catalog()) + len(customPluginIndices())` from later code as each caller migrates.

- [ ] **Step 4: Render two sections and a details pane**

The list panel must produce this plain-text structure:

```text
Recommended plugins
› [x] Autosuggestions [active]
  [ ] fzf-tab [missing]

Custom plugins
  No custom plugins yet

[a] Add custom plugin
```

Use bracketed foreground-only badges. For custom items, show `[active]`, `[disabled]`, `[untrusted]`, `[attention]`, or `[pending]` based on config, filesystem validation, and pending changes.

The details pane must branch on item kind. Custom details include managed path, selected load file, trust, enabled, active state, and repository URL only when the pending add retains one.

- [ ] **Step 5: Update footer and key ownership**

Change the Plugins footer to:

```go
return renderHint("a add custom  ·  space enable  ·  t/u trust  ·  l load file  ·  d remove  ·  Ctrl+A apply  ·  Ctrl+C quit")
```

On the Plugins tab, plain `a` is reserved for Add custom plugin. `Ctrl+A` remains Review & Apply. On all other tabs, preserve existing plain `a` behavior.

- [ ] **Step 6: Run responsive screen tests**

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

### Task 7: Implement the custom-plugin wizard

**Files:**
- Create: `internal/tui/plugin_wizard.go`
- Create: `internal/tui/plugin_wizard_view.go`
- Create: `internal/tui/plugin_wizard_test.go`
- Modify: `internal/tui/tui_v2_part1.go`
- Modify: `internal/tui/fullscreen_layout.go`
- Modify: `internal/tui/tui_v2_part6.go`

**Interfaces:**
- Consumes: `plugins.StageRepository`, `plugins.CloneRunner`, `plugins.StagedRepository`, `plugins.ChangeSet.QueueAdd`.
- Produces:

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

type pluginWizardModel struct {
    Step       pluginWizardStep
    URL        textinput.Model
    Stage      *plugins.StagedRepository
    Candidate  int
    Error      string
    RequestID  uint64
    Cancel     context.CancelFunc
}

type pluginStageResult struct {
    RequestID uint64
    Stage     plugins.StagedRepository
    Err       error
}
```

Model additions:

```go
pluginWizard     pluginWizardModel
pluginChanges    plugins.ChangeSet
pluginCloneRunner plugins.CloneRunner
```

Remove `pluginURL`, `pluginLoad`, `pluginFocus`, and `pluginAdvanced` after all callers are migrated.

- [ ] **Step 1: Write failing wizard opening and URL tests**

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

func TestPluginWizardInvalidURLStaysOnURLStep(t *testing.T) {
    model := NewModel(config.Default())
    model.openPluginWizard()
    model.pluginWizard.URL.SetValue("http://example.com/demo.git")
    updated, _ := model.updatePluginWizard(tea.KeyMsg{Type: tea.KeyEnter})
    got := updated.(Model)
    if got.pluginWizard.Step != pluginWizardURL || got.pluginWizard.Error == "" {
        t.Fatalf("wizard = %#v", got.pluginWizard)
    }
}
```

- [ ] **Step 2: Write failing asynchronous result and cancellation tests**

Use an injected blocking clone runner and verify:

- Enter moves URL -> Cloning and returns a command.
- Esc calls the stored cancel function and returns to URL or closes the wizard.
- A result with an old `RequestID` is ignored and its returned stage is cleaned up.
- A successful current result moves to Candidates.

- [ ] **Step 3: Run tests and confirm failure**

```bash
go test ./internal/tui -run PluginWizard -count=1
```

Expected: compile failure because wizard types do not exist.

- [ ] **Step 4: Implement wizard initialization and command**

`openPluginWizard` creates a fresh `textinput.Model` with prompt `Repository URL: ` and placeholder `https://github.com/user/plugin.git`.

Starting a clone must:

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

- [ ] **Step 5: Implement candidate, trust, and summary transitions**

Keys:

- Candidates: `up/k`, `down/j`, `enter`, `esc`.
- Trust: `y/enter` confirms; `n/esc` returns to Candidates.
- Summary: `enter` calls `pluginChanges.QueueAdd`, closes wizard, selects the new custom item, and sets `m.msg = "plugin queued; Review & Apply to activate"`.
- Closing before Summary calls `Stage.Cleanup()`.

The trust screen must explicitly state that the selected shell file executes in every interactive shell.

- [ ] **Step 6: Render all wizard states responsively**

`pluginWizardView(width int)` renders:

- URL input and validation error;
- cloning progress with Cancel hint;
- candidate list with relative paths and `›` marker;
- trust review with repository URL, final path, and selected load file;
- pending summary with `[enter] Add to pending changes`.

Use borders and foreground styles only. No background fills.

- [ ] **Step 7: Route wizard before global navigation**

In `Update`, after busy dialogs but before tab navigation:

```go
if m.pluginWizard.Step != pluginWizardClosed {
    return m.updatePluginWizard(msg)
}
```

In `workspaceContent`, render the wizard before tab content whenever it is open.

- [ ] **Step 8: Run wizard and layout tests**

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

### Task 8: Convert custom-plugin lifecycle actions to pending changes

**Files:**
- Modify: `internal/tui/tui_v2_part5.go`
- Modify: `internal/tui/plugin_wizard.go`
- Modify: `internal/tui/plugin_items.go`
- Modify: `internal/tui/contextual_screens.go`
- Create or modify: `internal/tui/plugin_lifecycle_test.go`

**Interfaces:**
- Consumes: `pluginListItem`, `plugins.ChangeSet.RootFor`, `plugins.DiscoverCandidates`, `plugins.ValidateCandidate`, `plugins.ChangeSet.QueueRemove`.
- Produces:

```go
func (m *Model) toggleCustomPlugin(item pluginListItem)
func (m *Model) trustCustomPlugin(item pluginListItem, trusted bool)
func (m *Model) openLoadFilePicker(item pluginListItem) error
func (m *Model) queueCustomPluginRemoval(item pluginListItem) error
```

- [ ] **Step 1: Write failing no-immediate-save tests**

Set `HOME` to an empty temporary directory and invoke enable/trust changes. Assert `config.Path()` still does not exist while `m.cfg` changes in memory.

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
        t.Fatalf("toggle saved config before Apply: %v", err)
    }
}
```

- [ ] **Step 2: Write removal and load-file tests**

Cover:

- removing an installed plugin queues a removal and leaves its directory on disk;
- removing an unapplied add deletes staging immediately and leaves no filesystem removal;
- changing load file uses candidate discovery and updates only pending config;
- trusting a staged plugin validates against its staging root;
- trusting an installed plugin validates against its final root;
- untrusting immediately makes pending activation false.

- [ ] **Step 3: Remove direct persistence from TUI helpers**

Delete `config.Save` calls from `togglePluginAtCursor`, `trustPluginAtCursor`, `untrustPluginAtCursor`, and the old `addPluginFromInputs` path. Remove `plugins.AddAndSave` use from the TUI.

Recommended plugin selection remains pending in `m.cfg.Plugins.Selected`; installing missing curated repositories may still use the existing explicit install command, but selected/enabled state is persisted by Review & Apply.

- [ ] **Step 4: Add change-load picker mode**

Extend the wizard with a mode field:

```go
type pluginWizardMode uint8
const (
    pluginWizardAdd pluginWizardMode = iota
    pluginWizardChangeLoad
)
```

For change-load mode, skip URL and clone steps. Discover candidates from `pluginChanges.RootFor(item.Name, item.Source)`, show the same candidate selector, and update `cfg.Plugins.Items[item.ConfigIndex].Load` after confirmation.

- [ ] **Step 5: Add removal confirmation**

Add `pluginRemoveConfirm bool` and `pluginRemoveName string` to `Model`. `d` opens a foreground-only confirmation view for custom items. `y/enter` calls `QueueRemove`; `n/esc` cancels. Recommended definitions cannot be removed and should show a concise message instead.

- [ ] **Step 6: Update details actions**

For custom plugins, show:

```text
[space] Enable/disable
[t/u]   Trust/remove trust
[l]     Change load file
[d]     Remove plugin
```

For recommended plugins, show selection and installation actions only.

- [ ] **Step 7: Run lifecycle tests**

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

### Task 9: Integrate pending plugin changes into Review & Apply and exit cleanup

**Files:**
- Modify: `internal/tui/tui_v2_part1.go`
- Modify: `internal/tui/tui_v2_part2.go`
- Modify: `internal/tui/tui_v2_part6.go`
- Modify: `internal/tui/apply_workspace.go`
- Modify: `internal/tui/fullscreen_layout.go`
- Modify or create: `internal/tui/apply_plugin_changes_test.go`

**Interfaces:**
- Consumes: `apply.Request`, `plugins.ChangeSet.Clone`, `plugins.ChangeSet.Counts`, `plugins.ChangeSet.Cleanup`.
- Produces:

```go
type applyResult struct {
    Config *config.Config
    Err    error
}
```

Model additions:

```go
reviewedPluginChanges plugins.ChangeSet
```

- [ ] **Step 1: Write failing review snapshot tests**

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

Add a view assertion for `Plugin changes  1 add · 0 remove` and the staged plugin name/load file.

- [ ] **Step 2: Update apply command and result handling**

`doApply` accepts both snapshots:

```go
func doApply(cfg *config.Config, changes plugins.ChangeSet) tea.Cmd {
    request := applyop.Request{Config: cloneConfig(cfg), PluginChanges: changes.Clone()}
    return func() tea.Msg {
        err := applyop.Apply(request)
        return applyResult{Config: request.Config, Err: err}
    }
}
```

On success, assign the applied config, clear `pluginChanges` and `reviewedPluginChanges`, and report `applied`. On failure, keep pending config and changes so the user can fix or retry.

- [ ] **Step 3: Update Review & Apply content**

Show counts and per-plugin actions above technical `.zshrc` details. Include final paths for additions and names for removals. The review snapshot, not live mutable state, drives this view.

- [ ] **Step 4: Clean up staging when the TUI exits without applying**

Change `Run` to inspect the final Bubble Tea model:

```go
func Run() error {
    cfg, err := config.Load()
    if err != nil { return err }
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

Also call cleanup when a stale asynchronous stage result is ignored.

- [ ] **Step 5: Add exit-cleanup and apply-failure tests**

Test `pluginChanges.Cleanup()` through a model fixture rather than starting an interactive program. Verify apply failure preserves the stage and change set for retry; verify explicit wizard cancel removes it.

- [ ] **Step 6: Run TUI and apply tests**

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

### Task 10: Documentation, visual walkthrough, and full verification

**Files:**
- Modify: `docs/screencasts/tui-visual-hierarchy.tape`
- Modify: `README.md`
- Modify tests only when verification exposes a real regression.

**Interfaces:**
- Consumes: completed PR A behavior.
- Produces: documented keyboard workflow and reproducible visual test sequence.

- [ ] **Step 1: Update README plugin documentation**

Document:

- recommended catalog selection and installation;
- `[a] Add custom plugin`;
- URL validation;
- candidate selection;
- explicit trust warning;
- pending state and Review & Apply;
- enable, trust, change-load, and removal actions;
- managed directory `~/.config/ozsh/plugins`.

State clearly that shell plugins execute code in every interactive shell and should be reviewed before trust is granted.

- [ ] **Step 2: Update VHS walkthrough**

The tape must open Plugins, show the transparent selected row, open Add custom plugin, enter a local/fake test repository URL suitable for the recording environment, choose a candidate, stop at the trust review, cancel, and return to the plugin list. Do not require network access in CI recordings.

- [ ] **Step 3: Format and run unit tests**

```bash
gofmt -w internal/plugins internal/apply internal/tui
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 4: Run race tests**

```bash
go test -race ./... -count=1
```

Expected: PASS.

- [ ] **Step 5: Run static checks**

```bash
go mod verify
git diff --exit-code -- go.mod go.sum
go vet ./...
golangci-lint run ./...
```

Expected: all commands exit `0`.

- [ ] **Step 6: Run shell and installer checks used by CI**

Use the exact repository scripts/workflow commands already defined in `.github/workflows/ci.yml`. Confirm ShellCheck, installer smoke tests, release smoke tests, and secret/vulnerability scans pass locally where tools are available; rely on GitHub Actions for unavailable scanners.

- [ ] **Step 7: Cross-build for Termux/Android ARM64**

```bash
CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -o /tmp/ozsh-android-arm64 ./cmd/ozsh
```

Expected: build succeeds and `/tmp/ozsh-android-arm64` is non-empty.

- [ ] **Step 8: Run real Termux visual checklist**

At minimum verify:

1. no gray filled rectangles around the panel, tabs, selected rows, previews, or badges;
2. the selected row remains obvious with the keyboard open;
3. `Custom plugins` and `[a] Add custom plugin` are visible at narrow width;
4. wizard cancellation removes its staging directory;
5. a queued plugin appears as pending;
6. Review & Apply lists the pending filesystem action;
7. a successful Apply moves the checkout to its final directory and loads it only after trust confirmation.

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

The PR body must summarize transparent chrome, the custom-plugin wizard, pending transactional apply, tests, and the required real-Termux visual verification. Do not mark ready or merge until CI and the Termux checklist are complete.
