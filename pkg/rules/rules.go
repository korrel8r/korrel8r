package rules

import (
	"github.com/alanconway/korrel8/pkg/korrel8"
)

func AddTo(r *korrel8.RuleSet) {
	r.Add(K8sToK8s()...)
	r.Add(AlertToK8s()...)
	r.Add(K8sToLoki()...)
}

func All() *korrel8.RuleSet { rs := korrel8.NewRuleSet(); AddTo(rs); return rs }
