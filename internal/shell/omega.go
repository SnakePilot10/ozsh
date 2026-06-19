package shell

import "fmt"

// WriteOmega atomically replaces the generated prompt file with private
// permissions. It leaves the previous complete file intact if generation or
// writing fails before the final rename.
func WriteOmega(data []byte) error {
	if err := EnsureOzshDir(); err != nil {
		return fmt.Errorf("setup omega directory: %w", err)
	}
	if err := atomicWrite(OmegaZshPath(), data, 0o600); err != nil {
		return fmt.Errorf("write omega.zsh: %w", err)
	}
	return nil
}
