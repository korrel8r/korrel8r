#!/bin/bash
# Check for broken internal links in a generated static site directory.
# Usage: check-links.sh <site-dir> [exclude-pattern...]
# Exclude patterns are grep -E patterns for link paths to skip (e.g. "^/client/").
#
# Automatically strips the Hugo baseURL path prefix from links before checking.

set -euo pipefail

site="${1:?Usage: check-links.sh <site-dir> [exclude-pattern...]}"
shift
excludes=("$@")

# Detect Hugo baseURL path prefix from config
hugo_config="$(dirname "$site")/hugo.yaml"
base_path=""
if [[ -f "$hugo_config" ]]; then
	base_path=$(grep -oP 'baseURL:.*github\.io\K/[^/]+' "$hugo_config" 2>/dev/null || true)
fi

broken=0

while IFS= read -r file; do
	while IFS= read -r link; do
		# Check exclude patterns
		skip=false
		for pat in "${excludes[@]+"${excludes[@]}"}"; do
			if echo "$link" | grep -qE "$pat"; then
				skip=true
				break
			fi
		done
		$skip && continue

		path="${link%%#*}"

		# Strip baseURL prefix for local file lookup
		local_path="$path"
		if [[ -n "$base_path" && "$path" == "$base_path"* ]]; then
			local_path="${path#"$base_path"}"
		fi

		target="${site}${local_path}"
		if [[ ! -f "$target" && ! -f "${target%/}/index.html" && ! -f "${target}index.html" ]]; then
			echo "BROKEN: $file -> $link"
			broken=$((broken + 1))
		fi
	done < <(grep -oP 'href="\K/[^"]*' "$file" 2>/dev/null || true)
done < <(find "$site" -name '*.html')

if ((broken > 0)); then
	echo "$broken broken internal link(s) found."
	exit 1
fi
echo "All internal links OK."
