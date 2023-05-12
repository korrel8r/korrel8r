#!/bin/bash
#
# Prepend a copyright line to every .go file that does not have one.

LINE="// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE"

sed -i '1 i \'"$LINE"'\n' $(find -name *.go | xargs grep -L "$LINE")
