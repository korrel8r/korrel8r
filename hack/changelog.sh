#!/bin/bash
# Generate a lazy change log.

cat <<EOF
# Change log for project Korrel8r

This is the project's commit log. It is placeholder until a more user-readable change log is available.

This project uses semantic versioning (https://semver.org)
EOF

LATEST=$(git describe --abbrev=0) # The latest release tag.
git log --format="%d- %s" --decorate-refs=refs/tags/* $LATEST | \
    sed 's/^ *(tag: \(.*\))\(.*\)$/\n## Version \1\n\n\2/'
