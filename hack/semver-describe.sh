#!/bin/bash
# Return a semantic pre-release version based on the $(git describe) and the current branch.
TAG=$(git describe)
if [[ $TAG =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)-([0-9]+)?.*$ ]]; then
	echo "${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.$((BASH_REMATCH[3] + 1))-dev.$(git branch --show-current).${BASH_REMATCH[4]}"
else
	echo "Not a valid release tag: $TAG "
fi
