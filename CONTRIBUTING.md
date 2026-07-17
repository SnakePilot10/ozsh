# Contributing

Thanks for helping improve `ozsh`. Keep changes focused, test shell-facing
behavior with an isolated `HOME`, and use Conventional Commits.

## Local Setup

```bash
scripts/setup.sh
```

## Validation

Run the project checks before sending changes:

```bash
scripts/lint.sh --check
scripts/test.sh
scripts/build.sh
scripts/healthcheck.sh
```

For release, installer, or packaging changes, also run:

```bash
scripts/release-smoke.sh
scripts/install-smoke.sh
```

## Shell Safety Rules

- Keep prompt generation deterministic.
- Any command that writes `.zshrc` must create a backup first.
- Tests that touch `.zshrc`, generated shell files, plugins, or `HOME` must use a temporary home directory.
- Do not source third-party plugin code unless it is explicitly trusted and passes the existing path checks.
- Preserve Termux behavior; do not add `chsh` flows for Termux.

## Pull Requests

- Target `main`.
- Use focused branches and Conventional Commits, for example `fix: preserve zshrc newline`.
- Include the validation commands you ran.
- Document unresolved risk in the PR or a focused issue.
