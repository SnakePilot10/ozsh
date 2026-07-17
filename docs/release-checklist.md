# Release Checklist

Use this checklist before tagging a production release.

## Local Gates

Run from the repository root:

```bash
GOCACHE=/tmp/ozsh-go-build GOMODCACHE=/tmp/ozsh-gomod go test ./...
GOCACHE=/tmp/ozsh-go-build GOMODCACHE=/tmp/ozsh-gomod go test -coverprofile=/tmp/ozsh_coverage.out ./...
GOCACHE=/tmp/ozsh-go-build GOMODCACHE=/tmp/ozsh-gomod go tool cover -func=/tmp/ozsh_coverage.out | awk '/^total:/ { if ($3 + 0 < 70) exit 1 }'
GOCACHE=/tmp/ozsh-go-build GOMODCACHE=/tmp/ozsh-gomod go build -buildvcs=false ./...
GOCACHE=/tmp/ozsh-go-build GOMODCACHE=/tmp/ozsh-gomod go test -run '^$' -bench BenchmarkRunApply -benchmem ./cmd/ozsh
GOCACHE=/tmp/ozsh-go-build GOMODCACHE=/tmp/ozsh-gomod go run golang.org/x/vuln/cmd/govulncheck@latest ./...
scripts/release-smoke.sh
scripts/install-smoke.sh
```

Expected gates:

- Total Go coverage is at least 70%.
- `BenchmarkRunApply` stays below 100ms/op.
- `govulncheck` reports 0 called vulnerabilities.
- The smoke script builds, previews, applies, validates generated Zsh syntax, and resets with a temp HOME.
- The installer smoke validates unattended install, PATH update, optional apply, and dry-run without network side effects.

## Snapshot Release

If GoReleaser is installed:

```bash
goreleaser release --snapshot --clean
```

Verify archives exist for:

- Linux amd64
- Linux arm64
- macOS amd64
- macOS arm64
- Android/Termux arm64

## Tag Release

1. Choose the release version, for example `v1.0.0`.
2. Update `CHANGELOG.md`.
3. Create and push the tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

4. Confirm the GitHub Actions `release` workflow completes. The GoReleaser job
   must wait for validation, security scan, and Android/Termux cross-build gates.
5. Download `checksums.txt` and `checksums.txt.sig` from the release artifacts.
6. Verify the checksum signature with cosign:

```bash
cosign verify-blob \
  --certificate-identity-regexp 'https://github.com/SnakePilot10/ozsh/.github/workflows/release.yml@refs/tags/v.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --signature checksums.txt.sig \
  checksums.txt
```

7. Verify the downloaded archive checksum:

```bash
sha256sum --check checksums.txt --ignore-missing
```

## v1.0 Exit Criteria

- `config.toml` includes `version = 1` and legacy configs migrate with backup.
- `ozsh apply` and `ozsh reset` are idempotent against common `.zshrc` profiles.
- `ozsh doctor --report` produces a sanitized support bundle without copying `.zshrc` content.
- GitHub Releases publish archives, `checksums.txt`, and a cosign signature.
- Clean install smoke passes on Ubuntu, macOS, and Termux.
- The public README describes the supported CLI/config stability contract.

## Homebrew Formula

Before publishing the Homebrew formula, replace:

- `v0.0.0` with the release version without the leading `v`.
- the all-zero `sha256` with the SHA256 of the release tarball.

Then run:

```bash
brew audit --strict packaging/homebrew/ozsh.rb
brew install --build-from-source packaging/homebrew/ozsh.rb
brew test ozsh
```

## AUR PKGBUILD

Before publishing the PKGBUILD, replace:

- `pkgver=0.0.0` with the release version without the leading `v`.
- `sha256sums=('SKIP')` with the SHA256 of the release tarball.

Then run in a clean Arch environment:

```bash
makepkg --clean --syncdeps --check
namcap PKGBUILD
```

## Clean Install Smoke

Run a fresh install on:

- Ubuntu latest
- macOS latest
- Termux

For each environment:

```bash
ozsh doctor
ozsh preview
ozsh apply
zsh -n ~/.config/ozsh/omega.zsh
ozsh reset
```

The new-user path is release-ready when install, preview, apply, and reset complete in under 5 minutes without manual config edits.
