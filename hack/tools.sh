#!/usr/bin/env bash
#
# This file is part of the korrel8r project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2024 The korrel8r Contributors
#

set -eu -o pipefail

# constants
PROJECT_ROOT="$(git rev-parse --show-toplevel)"
GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"

declare -r PROJECT_ROOT GOOS GOARCH
declare -r LOCAL_BIN="$PROJECT_ROOT/tmp/bin"

# versions
declare -r KUSTOMIZE_VERSION=${KUSTOMIZE_VERSION:-v3.8.7}
declare -r OC_VERSION=${OC_VERSION:-4.13.0}
declare -r KUBECTL_VERSION=${KUBECTL_VERSION:-v1.28.4}
declare -r SWAGGER_VERSION=${SWAGGER_VERSION:-v0.30.5}
declare -r SWAG_VERSION=${SWAG_VERSION:-v1.16.3}
declare -r GOLANGCI_LINT_VERSION=${GOLANGCI_LINT_VERSION:-v1.55.2}
declare -r KIND_VERSION=${KIND_VERSION:-v0.21.0}
declare -r CONTROLLER_TOOLS_VERSION=${CONTROLLER_TOOLS_VERSION:-v0.12.1}

# install
declare -r KUSTOMIZE_INSTALL_SCRIPT="https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
declare -r OC_URL="https://mirror.openshift.com/pub/openshift-v4/clients/ocp/$OC_VERSION"

go_install() {
	local pkg="$1"
	local version="$2"
	shift 2

	echo "installing $pkg version: $version"

	GOBIN=$LOCAL_BIN go install "$pkg@$version" || {
		echo "failed to install $pkg - $version"
		return 1
	}
	echo "$pkg - $version was installed successfully"
	return 0
}

validate_version() {
	local cmd="$1"
	local version_arg="$2"
	local version_regex="$3"
	shift 3

	command -v "$cmd" >/dev/null 2>&1 || return 1

	[[ "$(eval "$cmd $version_arg" | grep -o "$version_regex")" =~ $version_regex ]] || {
		return 1
	}

	echo "$cmd matching $version_regex already installed"
}

version_kubectl() {
	kubectl version --client
}

install_kubectl() {
	local version_regex="Client Version: $KUBECTL_VERSION"

	validate_version kubectl "version --client" "$version_regex" && return 0

	echo "installing kubectl version: $KUBECTL_VERSION"
	local install_url="https://dl.k8s.io/release/$KUBECTL_VERSION/bin/$GOOS/$GOARCH/kubectl"

	curl -Lo "$LOCAL_BIN/kubectl" "$install_url" || {
		echo "failed to install kubectl"
		return 1
	}
	chmod +x "$LOCAL_BIN/kubectl"
	echo "kubectl - $KUBECTL_VERSION was installed successfully"
}

version_kustomize() {
	kustomize version
}

install_kustomize() {
	validate_version kustomize version "$KUSTOMIZE_VERSION" && return 0

	echo "installing kustomize version: $KUSTOMIZE_VERSION"
	(
		# NOTE: this handles softlinks properly
		cd "$LOCAL_BIN"
		curl -Ss $KUSTOMIZE_INSTALL_SCRIPT | bash -s -- "${KUSTOMIZE_VERSION:1}" .
	) || {
		echo "failed to install kustomize"
		return 1
	}
	echo "kustomize was installed successfully"
}

version_controller-gen() {
	controller-gen --version
}

install_controller-gen() {
	local version_regex="Version: $CONTROLLER_TOOLS_VERSION"
	validate_version controller-gen --version "$version_regex" && return 0
	go_install sigs.k8s.io/controller-tools/cmd/controller-gen "$CONTROLLER_TOOLS_VERSION"
}

version_swagger() {
	swagger version
}

install_swagger() {
	local version_regex="Version: $SWAGGER_VERSION"
	validate_version swagger version "$version_regex" && return 0
	go_install github.com/go-swagger/go-swagger/cmd/swagger "$SWAGGER_VERSION"
}

version_swag() {
	swag --version
}

install_swag() {
	local version_regex="Version: $SWAG_VERSION"
	validate_version swag --version "$version_regex" && return 0
	go_install github.com/swaggo/swag/cmd/swag "$SWAG_VERSION"
}

version_golangci-lint() {
	golangci-lint --version
}

install_golangci-lint() {
	local version_regex="Version: $GOLANGCI_LINT_VERSION"
	validate_version golangci-lint --version "$version_regex" && return 0
	go_install github.com/golangci/golangci-lint/cmd/golangci-lint "$GOLANGCI_LINT_VERSION"
}

version_kind() {
	kind --version
}

install_kind() {
	local version_regex="Version: $KIND_VERSION"
	validate_version kind --version "$version_regex" && return 0
	go_install sigs.k8s.io/kind "$KIND_VERSION"
}

version_oc() {
	oc version --client
}

install_oc() {
	local version_regex="Client Version: $OC_VERSION"

	validate_version oc "version --client" "$version_regex" && return 0

	echo "installing oc version: $OC_VERSION"
	local os="$GOOS"
	[[ $os == "darwin" ]] && os="mac"

	local install="$OC_URL/openshift-client-$os.tar.gz"
	# NOTE: tar should be extracted to a tmp dir since it also contains kubectl
	# which overwrites kubectl installed by install_kubectl above
	local oc_tmp="$LOCAL_BIN/tmp-oc"
	mkdir -p "$oc_tmp"
	curl -sNL "$install" | tar -xzf - -C "$oc_tmp" || {
		echo "failed to install oc"
		return 1
	}
	mv "$oc_tmp/oc" "$LOCAL_BIN/"
	chmod +x "$LOCAL_BIN/oc"
	rm -rf "$LOCAL_BIN/tmp-oc/"
	echo "oc was installed successfully"

}

install_all() {
	echo "installing all tools ..."
	local ret=0
	for tool in $(declare -F | cut -f3 -d ' ' | grep install_ | grep -v 'install_all'); do
		"$tool" || ret=1
	done
	return $ret
}

version_all() {

	echo "Versions"
	for version_tool in $(declare -F | cut -f3 -d ' ' | grep version_ | grep -v 'version_all'); do
		local tool="${version_tool#version_}"
		local location=""
		location="$(command -v "$tool")"
		echo "$tool -> $location"
		"$version_tool"
		echo
	done
}

main() {
	local op="${1:-all}"
	shift || true

	mkdir -p "$LOCAL_BIN"
	export PATH="$LOCAL_BIN:$PATH"

	# NOTE: skip installation if invocation is tools.sh version
	if [[ "$op" == "version" ]]; then
		version_all
		return $?
	fi

	install_"$op"
	version_"$op"
}

main "$@"
