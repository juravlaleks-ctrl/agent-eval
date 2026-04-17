#!/usr/bin/env bash
set -euo pipefail

if [[ $# -gt 0 ]]; then
  input="$*"
else
  input="$(cat)"
fi

normalized="$(
  printf '%s' "$input" |
    perl -0pe 's/\A\s+//; s/\s+\z//; $_ = lc $_'
)"

printf '%s' "$normalized" | shasum -a 256 | awk '{print $1}'
