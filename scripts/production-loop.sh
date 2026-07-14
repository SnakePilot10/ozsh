#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/production-loop.sh "type(scope): message"

Examples:
  scripts/production-loop.sh "feat: add production loop automation"
  scripts/production-loop.sh "fix(shell): preserve zshrc backups"
USAGE
}

commit_msg="${1:-}"
if [[ -z "$commit_msg" ]]; then
  usage
  exit 2
fi

if ! [[ "$commit_msg" =~ ^(feat|fix|docs|refactor|test|chore|ci|build|perf)(\([a-z0-9._-]+\))?:\ .+ ]]; then
  echo "[loop] commit message must follow Conventional Commits" >&2
  usage
  exit 2
fi

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "[loop] not inside a Git repository" >&2
  exit 1
fi

branch="$(git branch --show-current)"
if [[ ! "$branch" == feature/* ]]; then
  echo "[loop] current branch must be feature/*, got: $branch" >&2
  exit 1
fi

if [[ -n "$(git status --porcelain=v1 --untracked-files=no)" ]]; then
  echo "[loop] tracked files already modified; continuing after validation"
fi

echo "[loop] branch: $branch"
echo "[loop] lint"
scripts/lint.sh --check

echo "[loop] tests"
scripts/test.sh

echo "[loop] build"
scripts/build.sh

echo "[loop] healthcheck"
scripts/healthcheck.sh

if command -v graphify >/dev/null 2>&1; then
  if [[ -f graphify-out/graph.json ]]; then
    echo "[loop] graphify update"
    graphify update .
  else
    echo "[loop] graphify installed, but graphify-out/graph.json is absent; skipping graph update"
  fi
else
  echo "[loop] graphify not installed; skipping graph update"
fi

if [[ -z "$(git status --porcelain)" ]]; then
  echo "[loop] no changes to commit"
  exit 0
fi

echo "[loop] staging changes"
git add -A

echo "[loop] commit"
git_name="$(git config user.name || true)"
git_email="$(git config user.email || true)"

if [[ -z "$git_name" ]]; then
  git_name="${GIT_AUTHOR_NAME:-}"
fi
if [[ -z "$git_email" ]]; then
  git_email="${GIT_AUTHOR_EMAIL:-}"
fi
if [[ -z "$git_name" ]] && command -v gh >/dev/null 2>&1; then
  git_name="$(gh api user --jq '.login' 2>/dev/null || true)"
fi
if [[ -z "$git_email" ]] && [[ -n "$git_name" ]]; then
  git_email="${git_name}@users.noreply.github.com"
fi
if [[ -z "$git_name" ]]; then
  git_name="ozsh automation"
fi
if [[ -z "$git_email" ]]; then
  git_email="ozsh-automation@users.noreply.github.com"
fi

git -c user.name="$git_name" -c user.email="$git_email" commit -m "$commit_msg"

echo "[loop] push"
git push -u origin "$branch"

base="develop"
if ! git ls-remote --exit-code --heads origin develop >/dev/null 2>&1; then
  base="main"
fi

echo "[loop] done"
echo "[loop] open PR: gh pr create --base $base --head $branch --fill"
