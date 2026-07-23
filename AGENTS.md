# Repository guidance for agents

## Project

`ozsh` is a Go CLI/TUI for configuring, previewing, generating, and safely
applying declarative Zsh prompts.

Core stack:

- Go 1.25+
- Bubble Tea, Bubbles, and Lip Gloss
- TOML configuration
- GoReleaser and AUR packaging for Linux and Termux
- GitHub Actions CI

## Working rules

- Use Graphify for architecture questions when `graphify-out/graph.json` exists.
- Refresh the Graphify index after meaningful code or documentation changes when the tool is available.
- Never hard-code secrets, tokens, credentials, or private local paths.
- Do not push directly to `main`.
- Do not force-push without explicit approval.
- Keep changes small, focused, and compatible with Conventional Commits.
- Do not change `.zshrc` behavior without updating and running the relevant tests.
- Any test that touches `HOME`, `.zshrc`, generated shell files, or plugins must use an isolated temporary home directory.
- Record unresolved uncertainty in the pull request or a focused GitHub issue, not in ad hoc root-level status files.

## Validation

Run the checks relevant to the change:

```bash
scripts/lint.sh --check
scripts/test.sh
scripts/build.sh
scripts/healthcheck.sh
```

For release or installer changes, also run:

```bash
scripts/release-smoke.sh
scripts/install-smoke.sh
```

Release artifacts support Linux and Android/Termux only.
