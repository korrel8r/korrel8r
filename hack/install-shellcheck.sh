#!/usr/bin/env bash

set -eu -o pipefail

# constants
declare -r BIN="$1"
declare -r SHELLCHECK_VERSION="$2"
declare -r SHELLCHECK_URL="https://github.com/koalaman/shellcheck/releases/download/v$SHELLCHECK_VERSION/shellcheck-v$SHELLCHECK_VERSION"

GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"

install_shellcheck() {
	echo "installing shellcheck version: $SHELLCHECK_VERSION"

	local arch="$GOARCH"
	[[ $arch == "amd64" ]] && arch="x86_64"
	[[ $arch == "arm64" ]] && arch="aarch64"

	local install="$SHELLCHECK_URL.$GOOS.$arch.tar.xz"

	wget -qO- "$install" | tar -xJf - -C "$BIN"
	mv "$BIN/shellcheck-v$SHELLCHECK_VERSION/shellcheck" "$BIN"
	rm -rf "$BIN/shellcheck-v$SHELLCHECK_VERSION"

	echo "shellcheck installed to $BIN/shellcheck"
}

main() {
	mkdir -p "$BIN"
	install_shellcheck
}
main "$@"
