// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Code generated from quicktemplate rules by hack/gen-applyfuncs.sh. DO NOT EDIT.

package quickrules

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/valyala/quicktemplate"
)

// applyFuncs maps rule names to the generated quicktemplate stream functions that render them.
var applyFuncs = map[string]func(qw *quicktemplate.Writer, start korrel8r.Object){
	"AlertToDeployment":  StreamAlertToDeployment,
	"LogToPod":           StreamLogToPod,
	"MetricToDeployment": StreamMetricToDeployment,
	"MetricToPod":        StreamMetricToPod,
}
