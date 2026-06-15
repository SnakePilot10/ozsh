package shell

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func IsTermux() bool {
	return os.Getenv("TERMUX_VERSION") != "" ||
		os.Getenv("PREFIX") == "/data/data/com.termux/files/usr"
}

func IsTermuxChroot() bool {
	return IsTermux() && strings.Contains(os.Getenv("PATH"), "/usr/bin")
}

func ZshIsDefaultShell() bool {
	shell := os.Getenv("SHELL")
	return filepath.Base(shell) == "zsh"
}

func TermuxPrefix() string {
	if p := os.Getenv("PREFIX"); p != "" {
		return p
	}
	return "/data/data/com.termux/files/usr"
}

func Home() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE")
	}
	return ""
}

func ZshrcPath() string {
	return filepath.Join(Home(), ".zshrc")
}

func OmegaDir() string {
	return filepath.Join(Home(), ".config", "ozsh")
}

func OmegaZshPath() string {
	return filepath.Join(OmegaDir(), "omega.zsh")
}

func BackupsDir() string {
	return filepath.Join(OmegaDir(), "backups")
}

func HasZsh() bool {
	_, err := exec.LookPath("zsh")
	return err == nil
}

func ZshrcExists() bool {
	_, err := os.Stat(ZshrcPath())
	return err == nil
}

func ConfigExists() bool {
	_, err := os.Stat(filepath.Join(OmegaDir(), "config.toml"))
	return err == nil
}
