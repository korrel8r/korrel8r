// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package domains

import (
	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/incident"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
 	"github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"github.com/korrel8r/korrel8r/pkg/domains/netflow"
	"github.com/korrel8r/korrel8r/pkg/domains/trace"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

var All = []korrel8r.Domain{
	k8s.Domain,
	log.Domain,
	netflow.Domain,
	trace.Domain,
	alert.Domain,
	metric.Domain,
	incident.Domain,
}
