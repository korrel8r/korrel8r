// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules

import (
	"testing"

	"github.com/go-logr/logr/funcr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMissingClasses(t *testing.T) {
	var missing MissingClasses
	missing.Add("RuleA", "k8s:Missing")
	missing.Add("RuleB", "k8s:Missing", "k8s:Other")
	missing.Add("RuleC") // No classes, no-op.
	missing.DropRule("RuleB")
	missing.DropStatusRule("StatusA")
	assert.Equal(t, []string{"RuleA", "RuleB"}, missing.Classes["k8s:Missing"])

	var lines []string
	logger := funcr.New(func(_, args string) { lines = append(lines, args) }, funcr.Options{Verbosity: 2})
	missing.Log(logger)
	require.Len(t, lines, 4)
	// Sorted by class, then dropped rules.
	assert.Contains(t, lines[0], `"class"="k8s:Missing"`)
	assert.Contains(t, lines[0], `"rules"=["RuleA" "RuleB"]`)
	assert.Contains(t, lines[1], `"class"="k8s:Other"`)
	assert.Contains(t, lines[1], `"rules"=["RuleB"]`)
	assert.Contains(t, lines[2], `"msg"="dropped rules: no known start or goal classes" "rules"=["RuleB"]`)
	assert.Contains(t, lines[3], `"msg"="dropped status rules: no known start classes" "rules"=["StatusA"]`)
}

func TestMissingClassesEmpty(t *testing.T) {
	var lines []string
	logger := funcr.New(func(_, args string) { lines = append(lines, args) }, funcr.Options{Verbosity: 2})
	(&MissingClasses{}).Log(logger)
	assert.Empty(t, lines)
}
