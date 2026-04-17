#!/bin/sh
set -eu

branch="$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)"
[ "$branch" = "main" ] || exit 0

repo_root="$(git rev-parse --show-toplevel)"

cleanup_path() {
  path="$1"

  git rm -r -q --cached --ignore-unmatch -- "$path" >/dev/null 2>&1 || true
  rm -rf "$repo_root/$path"
}

git ls-files --cached --others --exclude-standard -- '*.md' |
while IFS= read -r path; do
  [ -n "$path" ] || continue
  [ "$path" = "README.md" ] && continue
  cleanup_path "$path"
done

cleanup_path "docs"
