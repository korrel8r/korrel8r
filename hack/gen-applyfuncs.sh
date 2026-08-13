#!/bin/bash
#
# Generate pkg/rules/quickrules/applyfuncs.go from the compiled quicktemplate rules.
#
# The applyFuncs map associates each compiled rule with the generated stream function that
# renders it. Rule names are taken from the {% func %} declarations in *.qtpl: a rule named
# X is applied by the generated StreamX function. The names must match the YAML rule metadata,
# which quickrules.parseRuleAnnotations verifies at runtime.
#
# Usage: hack/gen-applyfuncs.sh QTPL_DIR > OUTPUT.go
set -euo pipefail

DIR=${1:?usage: $0 QTPL_DIR > OUTPUT.go}

names=$(sed -nE 's/.*\{\%[[:space:]]*func[[:space:]]+([A-Za-z_][A-Za-z0-9_]*).*/\1/p' "$DIR"/*.qtpl | sort -u)
if [ -z "$names" ]; then
	echo "error: no rule functions found in $DIR" >&2
	exit 1
fi

cat <<'EOF'
// Code generated from quicktemplate rules by hack/gen-applyfuncs.sh. DO NOT EDIT.

package quickrules

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/valyala/quicktemplate"
)

// applyFuncs maps rule names to the generated quicktemplate stream functions that render them.
var applyFuncs = map[string]func(qw *quicktemplate.Writer, start korrel8r.Object){
EOF
for name in $names; do
	printf '\t"%s": Stream%s,\n' "$name" "$name"
done
echo '}'
