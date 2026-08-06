# Transparent TUI and Extensible Plugins Plan: Mandatory Test Fixtures

This appendix is part of `2026-08-02-transparent-tui-extensible-plugins.md`. Use these exact helpers when the main plan references their names. Keeping them here prevents duplicated setup from drifting across packages.

## `internal/plugins/candidates_test.go`

```go
func writeCandidate(t *testing.T, root, relative string) {
    t.Helper()
    path := filepath.Join(root, filepath.FromSlash(relative))
    if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
        t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
    }
    if err := os.WriteFile(path, []byte("# plugin\n"), 0o600); err != nil {
        t.Fatalf("WriteFile(%s) error = %v", path, err)
    }
}

func candidatePaths(candidates []Candidate) []string {
    paths := make([]string, len(candidates))
    for i, candidate := range candidates {
        paths[i] = candidate.RelativePath
    }
    return paths
}
```

## `internal/plugins/changes.go`

The transaction type referenced by the main plan is:

```go
type renameEntry struct {
    from string
    to   string
    kind string
}

type Transaction struct {
    entries   []renameEntry
    committed bool
}
```

`Commit` and `Rollback` must reject a second terminal operation with `transaction already completed`.

## `internal/plugins/changes_test.go`

```go
func stagedFixture(t *testing.T, name, load string) StagedRepository {
    t.Helper()
    root := Dir()
    if root == "" {
        t.Fatal("plugins.Dir() returned empty path")
    }
    if err := os.MkdirAll(root, 0o700); err != nil {
        t.Fatalf("MkdirAll(plugin root) error = %v", err)
    }
    staging, err := os.MkdirTemp(root, ".staging-")
    if err != nil {
        t.Fatalf("MkdirTemp(staging) error = %v", err)
    }
    if err := os.Chmod(staging, 0o700); err != nil {
        t.Fatalf("Chmod(staging) error = %v", err)
    }
    writeCandidate(t, staging, load)
    candidates, err := DiscoverCandidates(staging, name, DefaultScanLimits)
    if err != nil {
        t.Fatalf("DiscoverCandidates() error = %v", err)
    }
    return StagedRepository{
        Repository: Repository{URL: "https://example.com/" + name + ".git", Name: name},
        StagingDir: staging,
        FinalDir:   filepath.Join(root, name),
        Candidates: candidates,
    }
}
```

## `internal/apply/apply_test.go`

```go
func pendingAddRequestFixture(t *testing.T, name, load string) (Request, plugins.StagedRepository) {
    t.Helper()
    root := plugins.Dir()
    if root == "" {
        t.Fatal("plugins.Dir() returned empty path")
    }
    if err := os.MkdirAll(root, 0o700); err != nil {
        t.Fatalf("MkdirAll(plugin root) error = %v", err)
    }
    staging, err := os.MkdirTemp(root, ".staging-")
    if err != nil {
        t.Fatalf("MkdirTemp(staging) error = %v", err)
    }
    loadPath := filepath.Join(staging, filepath.FromSlash(load))
    if err := os.MkdirAll(filepath.Dir(loadPath), 0o700); err != nil {
        t.Fatalf("MkdirAll(load parent) error = %v", err)
    }
    if err := os.WriteFile(loadPath, []byte("# plugin\n"), 0o600); err != nil {
        t.Fatalf("WriteFile(load) error = %v", err)
    }
    candidates, err := plugins.DiscoverCandidates(staging, name, plugins.DefaultScanLimits)
    if err != nil {
        t.Fatalf("DiscoverCandidates() error = %v", err)
    }
    stage := plugins.StagedRepository{
        Repository: plugins.Repository{URL: "https://example.com/" + name + ".git", Name: name},
        StagingDir: staging,
        FinalDir:   filepath.Join(root, name),
        Candidates: candidates,
    }
    cfg := config.Default()
    var changes plugins.ChangeSet
    if err := changes.QueueAdd(cfg, stage, load); err != nil {
        t.Fatalf("QueueAdd() error = %v", err)
    }
    return Request{Config: cfg, PluginChanges: changes}, stage
}
```

## `internal/tui/plugin_lifecycle_test.go`

```go
func customPluginConfig(t *testing.T, name string, enabled, trusted bool) *config.Config {
    t.Helper()
    root := plugins.Dir()
    if root == "" {
        t.Fatal("plugins.Dir() returned empty path")
    }
    source := filepath.Join(root, name)
    if err := os.MkdirAll(source, 0o700); err != nil {
        t.Fatalf("MkdirAll(source) error = %v", err)
    }
    load := name + ".zsh"
    if err := os.WriteFile(filepath.Join(source, load), []byte("# plugin\n"), 0o600); err != nil {
        t.Fatalf("WriteFile(load) error = %v", err)
    }
    cfg := config.Default()
    cfg.Plugins.Items = append(cfg.Plugins.Items, config.PluginItem{
        Name: name, Enabled: enabled, Trusted: trusted, Source: source, Load: load,
    })
    return cfg
}
```

Every test calling this helper must set `HOME` before calling it:

```go
t.Setenv("HOME", t.TempDir())
cfg := customPluginConfig(t, "demo", true, true)
```

## `internal/tui/apply_plugin_changes_test.go`

```go
func modelWithPendingPluginAdd(t *testing.T) Model {
    t.Helper()
    root := plugins.Dir()
    if root == "" {
        t.Fatal("plugins.Dir() returned empty path")
    }
    if err := os.MkdirAll(root, 0o700); err != nil {
        t.Fatalf("MkdirAll(plugin root) error = %v", err)
    }
    staging, err := os.MkdirTemp(root, ".staging-")
    if err != nil {
        t.Fatalf("MkdirTemp(staging) error = %v", err)
    }
    load := "demo.plugin.zsh"
    if err := os.WriteFile(filepath.Join(staging, load), []byte("# plugin\n"), 0o600); err != nil {
        t.Fatalf("WriteFile(load) error = %v", err)
    }
    candidates, err := plugins.DiscoverCandidates(staging, "demo", plugins.DefaultScanLimits)
    if err != nil {
        t.Fatalf("DiscoverCandidates() error = %v", err)
    }
    stage := plugins.StagedRepository{
        Repository: plugins.Repository{URL: "https://example.com/demo.git", Name: "demo"},
        StagingDir: staging,
        FinalDir:   filepath.Join(root, "demo"),
        Candidates: candidates,
    }
    model := NewModel(config.Default())
    if err := model.pluginChanges.QueueAdd(model.cfg, stage, load); err != nil {
        t.Fatalf("QueueAdd() error = %v", err)
    }
    return model
}
```

Every caller must set `HOME` first:

```go
t.Setenv("HOME", t.TempDir())
model := modelWithPendingPluginAdd(t)
```

## Fixture verification

After adding these helpers, run:

```bash
go test ./internal/plugins ./internal/apply ./internal/tui -count=1
```

The helpers must not write outside the test `HOME`, use network access, or persist configuration unless the individual test explicitly invokes Apply.
