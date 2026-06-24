package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/snakepilot10/ozsh/internal/config"
	"github.com/snakepilot10/ozsh/internal/logging"
	"github.com/snakepilot10/ozsh/internal/plugins"
	"github.com/snakepilot10/ozsh/internal/prompt"
	"github.com/snakepilot10/ozsh/internal/shell"
	"github.com/snakepilot10/ozsh/internal/tui"
)

var log *logging.Logger

const (
	version            = "0.2.0-dev"
	gitCommandTimeout  = 30 * time.Second
	gitInspectTimeout  = 10 * time.Second
)

func main() {
	args, verbose := parseGlobalFlags(os.Args[1:])
	log = logging.New(config.Dir(), verbose)
	log.Debug("starting ozsh with args=%v", args)
	if len(args) == 0 {
		printUsage()
		return
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

func printUsage() {
	fmt.Printf("ozsh %s\n\n", version)
	fmt.Println("Usage: ozsh [--verbose] <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  preview    Show simulated prompt preview")
	fmt.Println("  apply      Generate omega.zsh and inject into .zshrc")
	fmt.Println("  doctor     Validate environment and config")
	fmt.Println("  reset      Remove the ozsh block from .zshrc")
	fmt.Println("  theme      List, preview, or apply prompt themes")
	fmt.Println("  header     List, preview, or apply prompt headers")
	fmt.Println("  plugin     Manage manual plugins")
	fmt.Println("  tui        Open the terminal UI")
	fmt.Println("  version    Print version")
	fmt.Println("  update     Update a source installation")
	fmt.Println()
	fmt.Println("Config: ~/.config/ozsh/config.toml")
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

	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()
	if checkOnly {
		out, err := exec.CommandContext(ctx, "git", "-C", installDir, "fetch", "--quiet").CombinedOutput()
		if err != nil {
			updateCommandError("update check", ctx, err, out)
		}
		state, err := updateState(installDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "update check error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(state)
		return
	}

	out, err := exec.CommandContext(ctx, "git", "-C", installDir, "pull", "--ff-only").CombinedOutput()
	if err != nil {
		updateCommandError("update", ctx, err, out)
	}
	fmt.Print(string(out))
}

func updateCommandError(action string, ctx context.Context, err error, output []byte) {
	if ctx.Err() == context.DeadlineExceeded {
		fmt.Fprintf(os.Stderr, "%s error: git timed out after %s\n", action, gitCommandTimeout)
	} else {
		fmt.Fprintf(os.Stderr, "%s error: %v\n%s", action, err, output)
	}
	os.Exit(1)
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
	ctx, cancel := context.WithTimeout(context.Background(), gitInspectTimeout)
	defer cancel()
	if err := exec.CommandContext(ctx, "git", "-C", installDir, "merge-base", "--is-ancestor", local, remote).Run(); err == nil {
		return "new version available (run ozsh update)", nil
	}
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("git revision comparison timed out after %s", gitInspectTimeout)
	}
	return "update check complete (local checkout differs from upstream)", nil
}

func gitOutput(installDir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitInspectTimeout)
	defer cancel()
	cmdArgs := append([]string{"-C", installDir}, args...)
	out, err := exec.CommandContext(ctx, "git", cmdArgs...).Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("git timed out after %s", gitInspectTimeout)
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func parseGlobalFlags(args []string) ([]string, bool) {
	filtered := make([]string, 0, len(args))
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
	generated, err := prompt.Generate(cfg)
	if err != nil {
		logError("generator failed: %v", err)
		fmt.Fprintf(os.Stderr, "generator error: %v\n", err)
		os.Exit(1)
	}
	if err := shell.WriteOmega([]byte(generated)); err != nil {
		logError("write failed: %v", err)
		fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		os.Exit(1)
	}
	if err := shell.InjectBlock(); err != nil {
		logError("inject failed: %v", err)
		fmt.Fprintf(os.Stderr, "inject error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ omega.zsh generated")
	fmt.Println("✓ ozsh block injected into .zshrc")
	fmt.Printf("✓ source %s to activate\n", shell.OmegaZshPath())
	logInfo("applied prompt to %s", shell.OmegaZshPath())
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
	if cfg, err := config.Load(); err == nil && cfg != nil {
		fmt.Println("[✓] config.toml exists and is valid")
	} else {
		fmt.Printf("[✗] config.toml invalid or unavailable: %v\n", err)
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
		return
	}
	fmt.Println("Some checks failed. Run 'ozsh apply' after fixing.")
	os.Exit(1)
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
		fmt.Println("Usage: ozsh theme <list|apply|preview>")
		os.Exit(1)
	}
	switch args[0] {
	case "list":
		for _, name := range sortedThemeNames() {
			fmt.Println(name)
		}
	case "apply", "preview":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "theme %s requires a name\n", args[0])
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
		applyTheme(cfg, preset)
		if args[0] == "preview" {
			fmt.Println(prompt.Simulated(cfg))
			return
		}
		if err := config.Save(cfg); err != nil {
			exitConfigError(err)
		}
		fmt.Printf("✓ theme applied: %s\n", preset.Name)
	default:
		fmt.Fprintf(os.Stderr, "unknown theme command: %s\n", args[0])
		os.Exit(1)
	}
}

func applyTheme(cfg *config.Config, preset config.ThemeConfig) {
	cfg.Theme = preset
	if user, ok := cfg.Prompt.Segments["user"]; ok {
		user.FG = preset.Accent
		cfg.Prompt.Segments["user"] = user
	}
	if status, ok := cfg.Prompt.Segments["status"]; ok {
		status.FG = preset.Error
		cfg.Prompt.Segments["status"] = status
	}
}

func runPlugin(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: ozsh plugin <list|add|remove|enable|disable|trust|untrust>")
		fmt.Println("       ozsh plugin add <https-url> <load-file>")
		os.Exit(1)
	}
	cfg, err := config.Load()
	if err != nil {
		exitConfigError(err)
	}
	switch args[0] {
	case "list":
		if len(cfg.Plugins.Items) == 0 {
			fmt.Println("no plugins configured")
			return
		}
		for _, item := range cfg.Plugins.Items {
			state := "disabled"
			if item.Enabled {
				state = "enabled"
			}
			trust := "untrusted"
			if item.Trusted {
				trust = "trusted"
			}
			fmt.Printf("%s\t%s\t%s\t%s\n", item.Name, state, trust, pluginLoadPath(item))
		}
	case "add":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "plugin add requires an https git URL and a .zsh or .sh load file")
			os.Exit(1)
		}
		name, err := plugins.Add(cfg, args[1], args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin add error: %v\n", err)
			os.Exit(1)
		}
		if err := config.Save(cfg); err != nil {
			exitConfigError(err)
		}
		fmt.Printf("✓ plugin added: %s\n", name)
	case "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "plugin remove requires a name")
			os.Exit(1)
		}
		if err := plugins.Remove(cfg, args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "plugin remove error: %v\n", err)
			os.Exit(1)
		}
		if err := config.Save(cfg); err != nil {
			exitConfigError(err)
		}
		fmt.Printf("✓ plugin removed: %s\n", args[1])
	case "enable", "disable":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "plugin %s requires a name\n", args[0])
			os.Exit(1)
		}
		enabled := args[0] == "enable"
		if err := plugins.SetEnabled(cfg, args[1], enabled); err != nil {
			fmt.Fprintf(os.Stderr, "plugin %s error: %v\n", args[0], err)
			os.Exit(1)
		}
		if err := config.Save(cfg); err != nil {
			exitConfigError(err)
		}
		fmt.Printf("✓ plugin %s: %s\n", args[0], args[1])
	case "trust", "untrust":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "plugin %s requires a name\n", args[0])
			os.Exit(1)
		}
		trusted := args[0] == "trust"
		if err := plugins.SetTrusted(cfg, args[1], trusted); err != nil {
			fmt.Fprintf(os.Stderr, "plugin %s error: %v\n", args[0], err)
			os.Exit(1)
		}
		if err := config.Save(cfg); err != nil {
			exitConfigError(err)
		}
		fmt.Printf("✓ plugin %s: %s\n", args[0], args[1])
	default:
		fmt.Fprintf(os.Stderr, "unknown plugin command: %s\n", args[0])
		os.Exit(1)
	}
}

func runTUI() {
	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
		os.Exit(1)
	}
}

func pluginLoadPath(item config.PluginItem) string {
	if item.Load == "" {
		return "<no load file>"
	}
	return filepath.Join(item.Source, item.Load)
}

func suggestCommand(input string) string {
	commands := []string{"preview", "apply", "doctor", "reset", "theme", "header", "plugin", "tui", "version", "update"}
	sort.Slice(commands, func(i, j int) bool {
		return editDistance(input, commands[i]) < editDistance(input, commands[j])
	})
	if editDistance(input, commands[0]) <= 3 {
		return commands[0]
	}
	return ""
}

func runHeader(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: ozsh header <list|apply|preview>")
		os.Exit(1)
	}
	switch args[0] {
	case "list":
		for _, name := range sortedHeaderNames() {
			fmt.Println(name)
		}
	case "apply":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "header apply requires a name")
			os.Exit(1)
		}
		preset, ok := config.HeaderPresets[args[1]]
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown header: %s\n", args[1])
			os.Exit(1)
		}
		cfg, err := config.Load()
		if err != nil {
			exitConfigError(err)
		}
		cfg.Header = preset
		if err := config.Save(cfg); err != nil {
			exitConfigError(err)
		}
		fmt.Printf("✓ header applied: %s\n", args[1])
	case "preview":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "header preview requires a name")
			os.Exit(1)
		}
		preset, ok := config.HeaderPresets[args[1]]
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown header: %s\n", args[1])
			os.Exit(1)
		}
		fmt.Println(preset.Text)
	default:
		fmt.Fprintf(os.Stderr, "unknown header command: %s\n", args[0])
		os.Exit(1)
	}
}

func sortedThemeNames() []string {
	names := make([]string, 0, len(config.Presets))
	for name := range config.Presets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedHeaderNames() []string {
	names := make([]string, 0, len(config.HeaderPresets))
	for name := range config.HeaderPresets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func editDistance(a, b string) int {
	dp := make([][]int, len(a)+1)
	for i := range dp {
		dp[i] = make([]int, len(b)+1)
	}
	for i := 0; i <= len(a); i++ {
		dp[i][0] = i
	}
	for j := 0; j <= len(b); j++ {
		dp[0][j] = j
	}
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			dp[i][j] = min(dp[i-1][j]+1, dp[i][j-1]+1, dp[i-1][j-1]+cost)
		}
	}
	return dp[len(a)][len(b)]
}

func exitConfigError(err error) {
	logError("config error: %v", err)
	fmt.Fprintf(os.Stderr, "config error: %v\n", err)
	os.Exit(2)
}

func logError(format string, args ...any) {
	if log != nil {
		log.Error(format, args...)
	}
}

func logInfo(format string, args ...any) {
	if log != nil {
		log.Info(format, args...)
	}
}
