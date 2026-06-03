#!/bin/bash
# Prepend front matter items to a markdown file with no front matter.
# Usage: front-matter.sh FILE ITEM...
# Each ITEM is a YAML line to add (e.g. "title: My Page").

set -eu

file="$1"
shift
touch "$file"

if head -1 "$file" | grep -q '^---'; then
	echo "$file already has front matter"
	exit 1
fi

tmp=$(mktemp)
printf '%s\n' '---' "$@" '---' '<!-- Generated content, do not edit! -->' | cat - "$file" >"$tmp"
mv "$tmp" "$file"
