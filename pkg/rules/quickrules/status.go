// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package quickrules

import (
	"fmt"
	"strings"

	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/status"
)

type compiledStatus struct {
	name  string
	start []korrel8r.Class
	apply func(korrel8r.Object) []string
}

func (s *compiledStatus) Name() string            { return s.name }
func (s *compiledStatus) Start() []korrel8r.Class { return s.start }
func (s *compiledStatus) Apply(o korrel8r.Object) (result []string, err error) {
	defer func() {
		if p := recover(); p != nil {
			result, err = nil, fmt.Errorf("%v", p)
		}
	}()
	return s.apply(o), nil
}

// StatusRules returns compiled status rules, using d to resolve classes.
func StatusRules(d *korrel8r.Domains) []status.Rule {
	type spec struct {
		name    string
		domain  string
		classes []string
		apply   func(korrel8r.Object) []string
	}
	specs := []spec{
		{name: "HealthStatus", domain: "k8s", apply: healthStatus},
		{name: "HasFinalizer", domain: "k8s", apply: hasFinalizer},
		{name: "EventType", domain: "k8s", classes: []string{"Event.v1", "Event.v1.events.k8s.io"}, apply: eventType},
		{name: "AlertSeverity", domain: "alert", apply: alertSeverity},
		{name: "LogSeverity", domain: "log", apply: logSeverity},
	}
	var result []status.Rule
	for _, s := range specs {
		start, missing := classes(d, s.name, &classSpec{Domain: s.domain, Classes: s.classes})
		if len(start) == 0 {
			logger.V(1).Info("skipped compiled status rule: no known start classes", "rule", s.name, "missingClasses", missing)
			continue
		}
		result = append(result, &compiledStatus{name: s.name, start: start, apply: s.apply})
	}
	return result
}

func healthStatus(o korrel8r.Object) []string {
	if s := k8s.HealthStatus(o.(k8s.Object)); s != "" {
		return []string{s}
	}
	return nil
}

func hasFinalizer(o korrel8r.Object) []string {
	obj := o.(map[string]any)
	metadata, _ := obj["metadata"].(map[string]any)
	if fins, _ := metadata["finalizers"].([]any); len(fins) > 0 {
		return []string{"Finalizer"}
	}
	return nil
}

func eventType(o korrel8r.Object) []string {
	obj := o.(map[string]any)
	if t, _ := obj["type"].(string); t != "" && t != "Normal" {
		return []string{t}
	}
	return nil
}

func alertSeverity(o korrel8r.Object) []string {
	a := o.(*alert.Object)
	if s := a.Labels["severity"]; s != "" && s != "none" {
		return []string{s}
	}
	return nil
}

var errorLevels = map[string]string{
	"error": "Error", "err": "Error", "ERROR": "Error", "ERR": "Error", "Error": "Error",
	"critical": "Error", "fatal": "Error", "CRITICAL": "Error", "FATAL": "Error",
}
var warningLevels = map[string]string{
	"warning": "Warning", "warn": "Warning", "WARNING": "Warning", "WARN": "Warning", "Warning": "Warning",
}

func logSeverity(o korrel8r.Object) []string {
	l := o.(log.Object)
	s := l["level"]
	if s == "" {
		s = l["severity_text"]
	}
	s = strings.TrimSpace(s)
	if v, ok := errorLevels[s]; ok {
		return []string{v}
	}
	if v, ok := warningLevels[s]; ok {
		return []string{v}
	}
	return nil
}
