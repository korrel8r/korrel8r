// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules

import (
	"maps"
	"slices"

	"github.com/go-logr/logr"
)

// MissingClasses collects a report of rules affected by unknown classes while building rules.
// Classes are strings of the form "domain:class", or "domain:*" for a wildcard in an unknown domain.
// The zero value is ready to use.
type MissingClasses struct {
	// Classes holds the names of rules that refer to each unknown class.
	// A rule that refers to an unknown class is still used if it has other known start and goal classes.
	Classes map[string][]string
	// Dropped rules are not used because they have no known start or goal classes.
	Dropped []string
	// DroppedStatus rules are not used because they have no known start classes.
	DroppedStatus []string
}

// Add records that rule refers to the unknown classes.
func (m *MissingClasses) Add(rule string, classes ...string) {
	if len(classes) > 0 && m.Classes == nil {
		m.Classes = map[string][]string{}
	}
	for _, class := range classes {
		m.Classes[class] = append(m.Classes[class], rule)
	}
}

// DropRule records a rule that is not used because it has no known start or goal classes.
func (m *MissingClasses) DropRule(rule string) { m.Dropped = append(m.Dropped, rule) }

// DropStatusRule records a status rule that is not used because it has no known start classes.
func (m *MissingClasses) DropStatusRule(rule string) { m.DroppedStatus = append(m.DroppedStatus, rule) }

// Log logs the report: rules for each unknown class sorted by class, then dropped rules.
func (m *MissingClasses) Log(logger logr.Logger) {
	logger = logger.V(2)
	for _, class := range slices.Sorted(maps.Keys(m.Classes)) {
		logger.Info("rules reference unknown class", "class", class, "rules", m.Classes[class])
	}
	if len(m.Dropped) > 0 {
		logger.Info("dropped rules: no known start or goal classes", "rules", m.Dropped)
	}
	if len(m.DroppedStatus) > 0 {
		logger.Info("dropped status rules: no known start classes", "rules", m.DroppedStatus)
	}
}
