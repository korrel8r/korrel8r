#!/bin/bash
# Convert generated markdown files to asciidoc, remove the original .md file.
KRAMDOC=$1
shift

for MD in "$@"; do
	ADOC=${MD/.md/.adoc}
	sed 's/^###### Auto generated .*//' "$MD" | "$KRAMDOC" --heading-offset=-1 -o "$ADOC" -
	rm "$MD"
done
