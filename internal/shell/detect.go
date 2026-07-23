package shell

import (
	"os"
	"os/exec"
	"path/filepath"
)

func IsTermux() bool {
	return os.Getenv("TERMUX_VERSION") != "" ||
		os.Getenv("PREFIX") == "/data/data/com.termux/files/usr"
}

func IsTermuxChroot() bool {
	if !IsTermux() {
		return false
	}
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		if entry == "/usr/bin" || entry == "/bin" {
			return true
		}
	}
	return false
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
	return os.Getenv("HOME")
}

func ZshrcPath() string {
	if Home() == "" {
		return ""
	}
	return filepath.Join(Home(), ".zshrc")
}

func OmegaDir() string {
	if Home() == "" {
		return ""
	}
	return filepath.Join(Home(), ".config", "ozsh")
}

func OmegaZshPath() string {
	if OmegaDir() == "" {
		return ""
	}
	return filepath.Join(OmegaDir(), "omega.zsh")
}

func BackupsDir() string {
	if OmegaDir() == "" {
		return ""
	}
	return filepath.Join(OmegaDir(), "backups")
}

func HasZsh() bool {
	_, err := exec.LookPath("zsh")
	return err == nil
}

func ZshrcExists() bool {
	if ZshrcPath() == "" {
		return false
	}
	_, err := os.Stat(ZshrcPath())
	return err == nil
}

func ConfigExists() bool {
	if OmegaDir() == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(OmegaDir(), "config.toml"))
	return err == nil
}
