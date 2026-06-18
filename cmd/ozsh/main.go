package main

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "sort"
    "strings"

    "github.com/snakepilot10/ozsh/internal/config"
    "github.com/snakepilot10/ozsh/internal/logging"
    "github.com/snakepilot10/ozsh/internal/plugins"
    "github.com/snakepilot10/ozsh/internal/prompt"
    "github.com/snakepilot10/ozsh/internal/shell"
    "github.com/snakepilot10/ozsh/internal/tui"
)

var log *logging.Logger

const version = "0.2.0-dev"

func main() {
    args, verbose := parseGlobalFlags(os.Args[1:])
    log = logging.New(config.Dir(), verbose)
    log.Debug("starting ozsh with args=%v", args)

    if len(args) < 1 {
        fmt.Println("ozsh v0.1 — Prompt engine for Zsh")
        fmt.Println()
        fmt.Println("Usage: ozsh [--verbose] <command>")
        fmt.Println()
        fmt.Println("Commands:")
        fmt.Println("  preview    Show simulated prompt preview")
        fmt.Println("  apply      Generate omega.zsh and inject into .zshrc")
        fmt.Println("  doctor     Validate environment and config")
        fmt.Println("  reset      Remove ozsh block from .zshrc")
        fmt.Println("  theme      List or apply prompt themes")
        fmt.Println("  header     List or apply prompt headers")
        fmt.Println("  plugin     Manage manual plugins")
        fmt.Println("  tui        Open the terminal UI")
        fmt.Println("  version    Print version")
        fmt.Println("  update     Update source install")
        fmt.Println()
        fmt.Println("Config: ~/.config/ozsh/config.toml")
        os.Exit(0)
    }

    switch args[0] {
    case "preview":
        runPreview(args[1:])
    case "apply":
        runApply()
    case "doctor":
        runDoctor()
    case "reset":
        runReset()
    case "theme":
        runTheme(args[1:])
    case "header":
        runHeader(args[1:])
    case "plugin":
        runPlugin(args[1:])
    case "tui":
        runTUI()
    case "version":
        fmt.Println(version)
    case "update":
        runUpdate(args[1:])
    default:
        fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
        if suggestion := suggestCommand(args[0]); suggestion != "" {
            fmt.Fprintf(os.Stderr, "did you mean %q?\n", suggestion)
        }
        os.Exit(1)
    }
}

func runUpdate(args []string) {
    checkOnly := len(args) > 0 && args[0] == "--check"
    installDir := os.Getenv("OZSH_INSTALL_DIR")
    if installDir == "" {
        installDir = filepath.Join(os.Getenv("HOME"), ".local", "share", "ozsh")
    }
    if _, err := os.Stat(filepath.Join(installDir, ".git")); err != nil {
        fmt.Printf("ozsh %s\n", version)
        fmt.Printf("no source checkout found at %s\n", installDir)
        return
    }
    if checkOnly {
        cmd := exec.Command("git", "-C", installDir, "fetch", "--quiet")
        out, err := cmd.CombinedOutput()
        if err != nil {
            fmt.Fprintf(os.Stderr, "update check error: %v\n%s", err, out)
            os.Exit(1)
        }
        state, err := updateState(installDir)
        if err != nil {
            fmt.Fprintf(os.Stderr, "update check error: %v\n", err)
            os.Exit(1)
        }
        fmt.Println(state)
        return
    }
    cmd := exec.Command("git", "-C", installDir, "pull", "--ff-only")
    out, err := cmd.CombinedOutput()
    if err != nil {
        fmt.Fprintf(os.Stderr, "update error: %v\n%s", err, out)
        os.Exit(1)
    }
    fmt.Print(string(out))
}

func updateState(installDir string) (string, error) {
    local, err := gitOutput(installDir, "rev-parse", "HEAD")
    if err != nil {
        return "", fmt.Errorf("cannot read local revision: %w", err)
    }
    remote, err := gitOutput(installDir, "rev-parse", "@{u}")
    if err != nil {
        return "update check complete (no upstream configured)", nil
    }
    if local == remote {
        return "ozsh is up to date", nil
    }
    if err := exec.Command("git", "-C", installDir, "merge-base", "--is-ancestor", local, remote).Run(); err == nil {
        return "new version available (run ozsh update)", nil
    }
    return "update check complete (local checkout differs from upstream)", nil
}

func gitOutput(installDir string, args ...string) (string, error) {
    cmdArgs := append([]string{"-C", installDir}, args...)
    out, err := exec.Command("git", cmdArgs...).Output()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(out)), nil
}

func parseGlobalFlags(args []string) ([]string, bool) {
    var filtered []string
    verbose := false
    for _, arg := range args {
        switch arg {
        case "--verbose", "-v":
            verbose = true
        default:
            filtered = append(filtered, arg)
        }
    }
    return filtered, verbose
}

func runPreview(args []string) {
    cfg, err := config.Load()
    if err != nil {
        exitConfigError(err)
    }
    // Always disable heavy segments for preview to avoid slow external calls
    cfg.Prompt.DisableHeavySegments = true

    for _, arg := range args {
        if arg == "--real" {
            generated, err := prompt.Generate(cfg)
            if err != nil {
                fmt.Fprintf(os.Stderr, "generator error: %v\n", err)
                os.Exit(1)
            }
            fmt.Print(generated)
            return
        }
    }
    fmt.Println(prompt.Simulated(cfg))
}

func runApply() {
    cfg, err := config.Load()
    if err != nil {
        exitConfigError(err)
    }

    if err := shell.EnsureOzshDir(); err != nil {
        log.Error("setup failed: %v", err)
        fmt.Fprintf(os.Stderr, "setup error: %v\n", err)
        os.Exit(1)
    }

    generated, err := prompt.Generate(cfg)
    if err != nil {
        log.Error("generator failed: %v", err)
        fmt.Fprintf(os.Stderr, "generator error: %v\n", err)
        os.Exit(1)
    }

    omegaPath := shell.OmegaZshPath()
    if err := os.WriteFile(omegaPath, []byte(generated), 0644); err != nil {
        log.Error("write failed: %v", err)
        fmt.Fprintf(os.Stderr, "write error: %v\n", err)
        os.Exit(1)
    }

    if err := shell.InjectBlock(); err != nil {
        log.Error("inject failed: %v", err)
        fmt.Fprintf(os.Stderr, "inject error: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("✓ omega.zsh generated")
    fmt.Println("✓ ozsh block injected into .zshrc")
    fmt.Printf("✓ source %s to activate\n", omegaPath)
    log.Info("applied prompt to %s", omegaPath)
}

func runDoctor() {
    ok := true

    fmt.Println("ozsh doctor")
    fmt.Println()

    if shell.HasZsh() {
        fmt.Println("[✓] zsh is installed")
    } else {
        fmt.Println("[✗] zsh not found in PATH")
        ok = false
    }

    if shell.ConfigExists() {
        fmt.Println("[✓] config.toml exists")
    } else {
        fmt.Println("[✗] config.toml not found")
        ok = false
    }

    if shell.ZshrcExists() {
        fmt.Println("[✓] .zshrc exists")
    } else {
        fmt.Println("[✗] .zshrc not found")
        ok = false
    }

    if shell.HasBlock() {
        fmt.Println("[✓] ozsh block present in .zshrc")
    } else {
        fmt.Println("[ ] ozsh block not present (run 'ozsh apply')")
    }

    if shell.IsTermux() {
        fmt.Println("[✓] Termux detected")
        fmt.Printf("[✓] Termux prefix: %s\n", shell.TermuxPrefix())
        if shell.IsTermuxChroot() {
            fmt.Println("[!] termux-chroot appears active")
        }
    } else {
        fmt.Println("[✓] Linux/standard detected")
    }
    if shell.ZshIsDefaultShell() {
        fmt.Println("[✓] zsh is the default shell")
    } else if shell.IsTermux() {
        fmt.Println("[ ] zsh is not the default shell (Termux: ozsh will not run chsh)")
    } else {
        fmt.Println("[ ] zsh is not the default shell")
    }
    backups, err := shell.Backups()
    if err != nil {
        fmt.Printf("[!] backups unavailable: %v\n", err)
    } else {
        fmt.Printf("[✓] backups available: %d\n", len(backups))
    }

    fmt.Println()
    if ok {
        fmt.Println("All critical checks passed.")
    } else {
        fmt.Println("Some checks failed. Run 'ozsh apply' after fixing.")
        os.Exit(1)
    }
}

func runReset() {
    if err := shell.RemoveBlock(); err != nil {
        fmt.Fprintf(os.Stderr, "reset error: %v\n", err)
        os.Exit(1)
    }
    fmt.Println("✓ ozsh block removed from .zshrc")
    fmt.Println("  omega.zsh and config.toml preserved")
    fmt.Println("  Run 'ozsh apply' to restore")
}

func runTheme(args []string) {
    if len(args) == 0 {
        fmt.Println("Usage: ozsh theme <list|apply>")
        os.Exit(1)
    }
    switch args[0] {
    case "list":
        for name := range config.Presets {
            fmt.Println(name)
        }
    case "apply":
        if len(args) < 2 {
            fmt.Fprintln(os.Stderr, "theme apply requires a name")
            os.Exit(1)
        }
        preset, ok := config.Presets[args[1]]
        if !ok {
            fmt.Fprintf(os.Stderr, "unknown theme: %s\n", args[1])
            os.Exit(1)
        }
        cfg, err := config.Load()
        if err != nil {
            exitConfigError(err)
        }
        cfg.Theme = preset
        if user, ok := cfg.Prompt.Segments["user"]; ok {
            user.FG = preset.Accent
            cfg.Prompt.Segments["user"] = user
        }
        if status, ok := cfg.Prompt.Segments["status"]; ok {
            status.FG = preset.Error
            cfg.Prompt.Segments["status"] = status
        }
        if err := config.Save(cfg); err != nil {
            exitConfigError(err)
        }
        fmt.Printf("✓ theme applied: %s\n", preset.Name)
    case "preview":
        if len(args) < 2 {
            fmt.Fprintln(os.Stderr, "theme preview requires a name")
            os.Exit(1)
        }
        preset, ok := config.Presets[args[1]]
        if !ok {
            fmt.Fprintf(os.Stderr, "unknown theme: %s\n", args[1])
            os.Exit(1)
        }
        cfg, err := config.Load()
        if err != nil {
            exitConfigError(err)
        }
        cfg.Theme = preset
        if user, ok := cfg.Prompt.Segments["user"]; ok {
            user.FG = preset.Accent
            cfg.Prompt.Segments["user"] = user
        }
        if status, ok := cfg.Prompt.Segments["status"]; ok {
            status.FG = preset.Error
            cfg.Prompt.Segments["status"] = status
        }
        fmt.Println(prompt.Simulated(cfg))
    default:
        fmt.Fprintf(os.Stderr, "unknown theme command: %s\n", args[0])
        os.Exit(1)
    }
}

// ... (rest of file unchanged)
