#!/bin/bash
# Generate a lazy change log.

TAG=$1 # The release tag that is about to be applied.

cat <<EOF
# Change log for project Korrel8r

This is the project's commit log. It is placeholder until a more user-readable change log is available.

This project uses semantic versioning (https://semver.org)

## Version $TAG

EOF

git log --format="%d- %s" --decorate-refs=refs/tags/* | sed 's/^ *(tag: \(.*\))\(.*\)$/\n## Version \1\n\n\2/'
