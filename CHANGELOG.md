# Changelog

## Unreleased

- Added config schema versioning with legacy migration backups.
- Added `ozsh doctor --report` for sanitized local diagnostics.
- Added release checksum signing with keyless cosign.
- Expanded apply/reset regression coverage for common `.zshrc` profiles.
- Documented the v1.0 release-candidate promotion path.
- Narrowed supported platforms to Linux and Termux and removed unsupported platform artifacts.
- Hardened `HOME` handling, plugin removal/trust, TUI persistence, backups, and prompt escaping.
- Added static release builds, reproducible CI tooling, AUR integrity checks, and Sigstore bundles.
- Removed the incomplete prompt-template path; legacy `omega` style configs migrate to the complete generator.

## v0.2.0

- Added prompt separators, right prompt support, optional templates, themes, manual plugin commands, logging, and expanded segment support.
- Removed the unused headers feature from config, CLI, TUI, and generated Zsh.
