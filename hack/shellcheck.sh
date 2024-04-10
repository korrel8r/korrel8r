#!/usr/bin/env bash

set -eu -o pipefail

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"

# constants
declare -r PROJECT_ROOT
declare -r LOCAL_BIN="$PROJECT_ROOT/tmp/bin"

# versions
declare -r SHELLCHECK_VERSION=${SHELLCHECK_VERSION:-0.10.0}

# install
declare -r SHELLCHECK_URL="https://github.com/koalaman/shellcheck/releases/download/v$SHELLCHECK_VERSION/shellcheck-v$SHELLCHECK_VERSION"

install_shellcheck() {
	echo "installing shellcheck version: $SHELLCHECK_VERSION"

	local arch="$GOARCH"
	[[ $arch == "amd64" ]] && arch="x86_64"
	[[ $arch == "arm64" ]] && arch="aarch64"

	local install="$SHELLCHECK_URL.$GOOS.$arch.tar.xz"

	wget -qO- "$install" | tar -xJf - -C "$LOCAL_BIN"
	mv "$LOCAL_BIN/shellcheck-v$SHELLCHECK_VERSION/shellcheck" "$LOCAL_BIN"
	rm -rf "$LOCAL_BIN/shellcheck-v$SHELLCHECK_VERSION"

	echo "shellcheck installed to $LOCAL_BIN/shellcheck"
}

main() {
	mkdir -p "$LOCAL_BIN"
	export PATH="$LOCAL_BIN:$PATH"

	install_shellcheck
}
main "$@"
