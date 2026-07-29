# Release Checklist

Use this checklist before tagging a production release.

## Local Gates

Run from the repository root:

```bash
GOCACHE=/tmp/ozsh-go-build GOMODCACHE=/tmp/ozsh-gomod go test ./...
GOCACHE=/tmp/ozsh-go-build GOMODCACHE=/tmp/ozsh-gomod go test -race ./...
GOCACHE=/tmp/ozsh-go-build GOMODCACHE=/tmp/ozsh-gomod go test -coverprofile=/tmp/ozsh_coverage.out ./...
GOCACHE=/tmp/ozsh-go-build GOMODCACHE=/tmp/ozsh-gomod go tool cover -func=/tmp/ozsh_coverage.out | awk '/^total:/ { if ($3 + 0 < 70) exit 1 }'
GOCACHE=/tmp/ozsh-go-build GOMODCACHE=/tmp/ozsh-gomod go build -buildvcs=false ./...
GOCACHE=/tmp/ozsh-go-build GOMODCACHE=/tmp/ozsh-gomod go test -run '^$' -bench BenchmarkRunApply -benchmem ./cmd/ozsh
go mod verify
go mod tidy -diff
GOCACHE=/tmp/ozsh-go-build GOMODCACHE=/tmp/ozsh-gomod go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
scripts/release-smoke.sh
scripts/install-smoke.sh
scripts/lint.sh --check
```

Expected gates:

- Total Go coverage is at least 70%.
- `BenchmarkRunApply` stays below 100ms/op.
- `govulncheck` reports 0 called vulnerabilities.
- The smoke script builds, previews, applies, validates generated Zsh syntax, and resets with a temp HOME.
- The installer smoke validates unattended install, PATH update, optional apply, and dry-run without network side effects.

## Snapshot Release

If GoReleaser is installed, skip keyless signing for the local snapshot:

```bash
goreleaser release --snapshot --clean --skip=sign
```

Verify archives exist for:

- Linux amd64
- Linux arm64
- Android/Termux arm64

All release binaries must be built with `CGO_ENABLED=0`. On Linux, extract an
archive and confirm that `ozsh version` reports the snapshot version and that
`ldd ozsh` reports it is not a dynamic executable.

## Release Candidate

Use a release candidate before the first stable v1.0 tag.

1. Choose an RC tag, starting with `v1.0.0-rc.1`.
2. Confirm `CHANGELOG.md` lists the release candidate scope under `Unreleased`.
3. Create and push the RC tag:

```bash
git tag v1.0.0-rc.1
git push origin v1.0.0-rc.1
```

4. Confirm the gated `release` workflow completes.
5. Download the generated archives, `checksums.txt`, `checksums.txt.sig`,
   `checksums.txt.pem`, and `checksums.txt.bundle`.
6. Verify the checksum signature and keyless identity with the bundle:

```bash
cosign verify-blob \
  --certificate-identity-regexp 'https://github.com/SnakePilot10/ozsh/.github/workflows/release.yml@refs/tags/v.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --bundle checksums.txt.bundle \
  checksums.txt
```

Also verify the separately published certificate and signature:

```bash
cosign verify-blob \
  --certificate-identity-regexp 'https://github.com/SnakePilot10/ozsh/.github/workflows/release.yml@refs/tags/v.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate checksums.txt.pem \
  --signature checksums.txt.sig \
  checksums.txt
```

7. Verify at least one downloaded archive against `checksums.txt`:

```bash
sha256sum --check checksums.txt --ignore-missing
```

8. Install the RC on fresh Linux and Termux environments.
9. Run the clean install smoke steps below on each environment.
10. Leave the RC open for one to two weeks. Promote to `v1.0.0` only if no data-loss, security, install, or migration issue is reported.

## Tag Release

1. Choose the release version, for example `v1.0.0`, after a successful RC window.
2. Update `CHANGELOG.md`.
3. Create and push the tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

4. Confirm the GitHub Actions `release` workflow completes. The GoReleaser job
   must wait for validation, security scan, and Android/Termux cross-build gates.
5. Download `checksums.txt`, `checksums.txt.sig`, `checksums.txt.pem`, and
   `checksums.txt.bundle` from the release artifacts.
6. Verify the checksum signature with the cosign bundle:

```bash
cosign verify-blob \
  --certificate-identity-regexp 'https://github.com/SnakePilot10/ozsh/.github/workflows/release.yml@refs/tags/v.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --bundle checksums.txt.bundle \
  checksums.txt
```

Then repeat verification with the detached material:

```bash
cosign verify-blob \
  --certificate-identity-regexp 'https://github.com/SnakePilot10/ozsh/.github/workflows/release.yml@refs/tags/v.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate checksums.txt.pem \
  --signature checksums.txt.sig \
  checksums.txt
```

7. Verify the downloaded archive checksum:

```bash
sha256sum --check checksums.txt --ignore-missing
```

## v1.0 Exit Criteria

- `config.toml` includes `version = 2` and legacy configs migrate with backup.
- `ozsh apply` and `ozsh reset` are idempotent against common `.zshrc` profiles.
- `ozsh doctor --report` produces a sanitized support bundle without copying `.zshrc` content.
- GitHub Releases publish static archives, `checksums.txt`, and cosign signature,
  certificate, and bundle artifacts.
- Clean install smoke passes on Ubuntu and Termux.
- A `v1.0.0-rc.1` or later RC has completed the release workflow and external smoke window.
- The public README describes the supported CLI/config stability contract.

## AUR PKGBUILD

Update the package from the tagged GitHub tarball. The script computes the real
SHA256 and preserves the original tag spelling for prerelease source directories:

```bash
scripts/update-release.sh 1.0.0-rc.1
```

Then run in a clean Arch environment:

```bash
makepkg --clean --syncdeps --check
namcap PKGBUILD
```

## Clean Install Smoke

Run a fresh install on:

- Ubuntu latest
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
