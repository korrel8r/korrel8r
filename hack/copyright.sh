#!/bin/bash
#
# Prepend a copyright line to every .go file that does not have one.
LINE="// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE"

# It will not include files/folders which have auto-generated go code
FILES=$(find . -type d -name 'zz_*' -prune -o -type f -name '*.go' ! -name 'zz_*' -print0 | xargs -0 grep -L "$LINE")
for file in $FILES; do sed -i '1 i '"$LINE"'\n' "$file"; done
