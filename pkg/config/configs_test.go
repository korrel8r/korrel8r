// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	c, err := Load("testdata/config.json")
	require.NoError(t, err)
	want := Configs{
		{
			Source:  "testdata/config.json",
			Include: []string{"config1.yaml", "config.json", "config2.yaml"},
			Tuning:  &Tuning{RequestTimeout: Duration{Duration: time.Second}},
		},
		{
			Source: "testdata/config1.yaml",
			Rules: []Rule{
				{Name: "rule1",
					Start:  ClassSpec{Domain: "foo", Classes: []string{"a", "b", "d", "e"}},
					Goal:   ClassSpec{Domain: "bar", Classes: []string{"z"}},
					Result: ResultSpec{Query: "what?"},
				},
			},
		},
		{
			Source: "testdata/config2.yaml",
			Rules: []Rule{
				{Name: "rule2",
					Start:  ClassSpec{Domain: "foo", Classes: []string{"d", "e"}},
					Goal:   ClassSpec{Domain: "bar", Classes: []string{"q"}},
					Result: ResultSpec{Query: "blah"}}},
		},
	}
	assert.Equal(t, want, c)
}

func TestLoad_bad_tuning(t *testing.T) {
	_, err := Load("testdata/bad-tuning.json")
	require.EqualError(t, err, "Unexpected tuning section in included configuration: testdata/config.json")
}

func TestConfigs_Expand(t *testing.T) {
	c := Configs{
		{
			Aliases: []Class{
				{Name: "x", Domain: "foo", Classes: []string{"p", "q"}},
				{Name: "y", Domain: "foo", Classes: []string{"x", "a"}},
				{Name: "z", Domain: "foo", Classes: []string{"u", "v"}},
			},
			Rules: []Rule{
				{
					Name:   "r",
					Start:  ClassSpec{Domain: "foo", Classes: []string{"y"}},
					Goal:   ClassSpec{Domain: "foo", Classes: []string{"z"}},
					Result: ResultSpec{Query: "dummy"},
				},
			},
		},
	}
	require.NoError(t, expand(c))
	want := Configs{
		{
			Rules: []Rule{
				{
					Name:   "r",
					Start:  ClassSpec{Domain: "foo", Classes: []string{"p", "q", "a"}},
					Goal:   ClassSpec{Domain: "foo", Classes: []string{"u", "v"}},
					Result: ResultSpec{Query: "dummy"},
				},
			},
		},
	}
	assert.Equal(t, want, c)
}

func TestConfigs_Expand_sameGroupDifferentDomain(t *testing.T) {
	c := Configs{
		{
			Aliases: []Class{
				{Name: "x", Domain: "foo", Classes: []string{"p", "q"}},
				{Name: "x", Domain: "bar", Classes: []string{"bbq"}},
			},
			Rules: []Rule{
				{
					Name:   "r1",
					Start:  ClassSpec{Domain: "foo", Classes: []string{"a", "x"}},
					Goal:   ClassSpec{Domain: "bar", Classes: []string{"x"}},
					Result: ResultSpec{Query: "help"},
				},
			},
		},
	}
	require.NoError(t, expand(c))
	want := Configs{
		{
			Rules: []Rule{{
				Name:   "r1",
				Start:  ClassSpec{Domain: "foo", Classes: []string{"a", "p", "q"}},
				Goal:   ClassSpec{Domain: "bar", Classes: []string{"bbq"}},
				Result: ResultSpec{Query: "help"}}},
		}}
	assert.Equal(t, want, c)
}
