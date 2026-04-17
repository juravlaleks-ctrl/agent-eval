#!/bin/sh
set -eu

branch="$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)"

repo_root="$(git rev-parse --show-toplevel)"

die() {
  echo "hook blocked: $1" >&2
  exit 1
}

file_has_cyrillic() {
  path="$1"
  [ -f "$path" ] || return 1
  grep -q '[А-Яа-яЁё]' "$path"
}

check_public_file() {
  rel="$1"
  path="$repo_root/$rel"
  [ -f "$path" ] || return 0

  if file_has_cyrillic "$path"; then
    die "'$rel' must remain English-only."
  fi
}

for rel in README.md CONTRIBUTING.md NOTICE LICENSE; do
  check_public_file "$rel"
done

[ -e "$repo_root/SPEC.md" ] && die "root 'SPEC.md' is obsolete; use 'docs/SPEC.md'."

if [ "$branch" = "main" ]; then
  if [ -d "$repo_root/docs" ]; then
    die "'docs/' must not be present in the worktree on branch 'main'."
  fi

  exit 0
fi

[ -d "$repo_root/docs" ] || exit 0

[ -f "$repo_root/docs/SPEC.md" ] || die "'docs/SPEC.md' is required for non-main engineering documentation."
[ -f "$repo_root/docs/.code-review-graph.json" ] || die "'docs/.code-review-graph.json' is required for doc-truth metadata."

for path in $(find "$repo_root/docs" -type f -name '*.md' -print); do
  if ! file_has_cyrillic "$path"; then
    rel="${path#"$repo_root"/}"
    die "'$rel' must remain Russian-language documentation."
  fi
done

exit 0
