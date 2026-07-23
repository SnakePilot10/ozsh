# Git workflow

## Branches

- `main`: stable integration and release branch. Changes arrive through pull requests.
- `feature/*`: user-facing features and larger improvements.
- `fix/*`: bug fixes and small corrective changes.
- `chore/*`: maintenance, documentation, tooling, and repository cleanup.
- `release/*`: optional release preparation when a version needs stabilization.

Keep branches short-lived and focused on one reviewable concern.

## Conventional Commits

Use this format:

```text
type(optional-scope): short imperative description
```

Common types:

- `feat`: new functionality.
- `fix`: bug correction.
- `docs`: documentation only.
- `refactor`: internal change without intended behavior changes.
- `test`: tests or test infrastructure.
- `chore`: maintenance.
- `ci`: GitHub Actions and automation.
- `build`: build, packaging, or dependencies.
- `perf`: performance improvement.

Examples:

```text
feat: add prompt theme preset
fix(shell): preserve zshrc newline during reset
ci: consolidate repository checks
```

## Change workflow

1. Update local `main`.
2. Create a focused branch.
3. Make small, reviewable changes.
4. Run the relevant checks:

   ```bash
   scripts/lint.sh --check
   scripts/test.sh
   scripts/build.sh
   scripts/healthcheck.sh
   ```

5. Update the Graphify index when it exists and the tool is available.
6. Commit with a Conventional Commit message.
7. Push the branch and open a pull request against `main`.
8. Merge only after the required CI checks pass.

Tests that touch `.zshrc`, generated shell files, plugins, or `HOME` must use an
isolated temporary home directory.

## Releases

1. Start from a green `main` branch.
2. Review `docs/release-checklist.md`.
3. Run the full local validation suite.
4. Update release notes and packaging metadata when required.
5. Create and push a signed or annotated `vX.Y.Z` tag.
6. Let the `release` workflow publish static GoReleaser artifacts, checksums,
   and keyless cosign verification material.
7. Verify the published checksums, signature bundle, certificate, and binaries
   before announcing the release.

## Recommended branch protection

Configure `main` to:

- require a pull request before merging;
- require the CI jobs to pass;
- require the branch to be current before merging;
- block force pushes and deletion;
- prevent direct pushes except for deliberate emergency recovery.

The expected CI jobs are `quality`, `security-scan`, `go-test`, and
`android-termux-cross-build`.

## Operational rules

- Never commit tokens, credentials, certificates, private keys, or real `.env` files.
- Never force-push a shared branch without explicit agreement.
- Do not bypass CI for code, packaging, installer, or shell-management changes.
- Keep generated build output, coverage files, and local analysis artifacts out of Git.
