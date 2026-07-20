#!/usr/bin/env bash
set -euo pipefail

# usage: ./scripts/update-release.sh <version>
# This script updates packaging files for a release.
# It requires a git tag (e.g., v1.0.0) to exist.

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <version>" >&2
  exit 1
fi

version="$1"
tag="v${version}"

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?(\+[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]]; then
  echo "[x] Version must be SemVer without leading v: $version" >&2
  exit 1
fi

aur_pkgver="${version//-/_}"
tarball_url="https://github.com/snakepilot10/ozsh/archive/refs/tags/${tag}.tar.gz"

# Ensure tag exists
if ! git rev-parse "$tag" >/dev/null 2>&1; then
  echo "[x] Git tag $tag does not exist" >&2
  exit 1
fi

tmpdir=$(mktemp -d)
tarball="$tmpdir/ozsh-$version.tar.gz"
trap 'rm -rf "$tmpdir"' EXIT

# Download the exact tarball URL used by Homebrew and AUR.
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$tarball_url" -o "$tarball"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$tarball" "$tarball_url"
else
  echo "[x] curl or wget not found" >&2
  exit 1
fi

# Compute sha256 sum
if command -v sha256sum >/dev/null 2>&1; then
  sha=$(sha256sum "$tarball" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
  sha=$(shasum -a 256 "$tarball" | awk '{print $1}')
else
  echo "[x] sha256sum or shasum not found" >&2
  exit 1
fi

# Update AUR PKGBUILD version and sha
pkg_file="packaging/aur/PKGBUILD"
sed -i.bak \
  -e "s/^pkgver=.*/pkgver=${aur_pkgver}/" \
  -e "s/^_tagver=.*/_tagver=${version}/" \
  -e "s|^source=.*|source=(\"\$pkgname-\$pkgver.tar.gz::\$url/archive/refs/tags/v\$_tagver.tar.gz\")|" \
  -e "s/^sha256sums=.*/sha256sums=('${sha}')/" \
  "$pkg_file"

# Update Homebrew formula version and sha
formula_file="packaging/homebrew/ozsh.rb"
sed -i.bak \
  -e "s|archive/refs/tags/v.*\\.tar\\.gz|archive/refs/tags/v${version}.tar.gz|" \
  -e "s/^  version .*/  version \"${version}\"/" \
  -e "s/[0-9a-f]\{64\}/${sha}/" \
  "$formula_file"

rm -f "$pkg_file.bak" "$formula_file.bak"

echo "[\u2713] Updated PKGBUILD and Homebrew formula for version $version"
echo "SHA256: $sha"
