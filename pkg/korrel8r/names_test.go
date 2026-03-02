// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassSplit(t *testing.T) {
	for _, test := range []struct {
		name       string
		input      string
		wantDomain string
		wantClass  string
		wantErr    bool
	}{
		{name: "simple", input: "k8s:pod", wantDomain: "k8s", wantClass: "pod"},
		{name: "single-char", input: "a:b", wantDomain: "a", wantClass: "b"},
		{name: "with-hyphens", input: "my-domain:my-class", wantDomain: "my-domain", wantClass: "my-class"},
		{name: "with-digits", input: "log2:alert3", wantDomain: "log2", wantClass: "alert3"},
		{name: "uppercase", input: "K8s:Pod", wantDomain: "K8s", wantClass: "Pod"},
		{name: "dots", input: "k8s:Pod.v1", wantDomain: "k8s", wantClass: "Pod.v1"},
		{name: "underscore", input: "my_domain:my_class", wantDomain: "my_domain", wantClass: "my_class"},
		{name: "empty", input: "", wantErr: true},
		{name: "no-colon", input: "nocolon", wantErr: true},
		{name: "extra-colon", input: "a:b:c", wantErr: true},
		{name: "space-in-domain", input: "a b:c", wantErr: true},
		{name: "space-in-class", input: "a:b c", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			domain, class, err := ClassSplit(test.input)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.wantDomain, domain)
				assert.Equal(t, test.wantClass, class)
			}
		})
	}
}

func TestQuerySplit(t *testing.T) {
	for _, test := range []struct {
		name       string
		input      string
		wantDomain string
		wantClass  string
		wantData   string
		wantErr    bool
	}{
		{name: "simple", input: "k8s:pod:some-data", wantDomain: "k8s", wantClass: "pod", wantData: "some-data"},
		{name: "empty-data", input: "k8s:pod:", wantDomain: "k8s", wantClass: "pod", wantData: ""},
		{name: "data-with-colons", input: "k8s:pod:ns:name", wantDomain: "k8s", wantClass: "pod", wantData: "ns:name"},
		{name: "data-with-spaces", input: "log:entry:foo bar baz", wantDomain: "log", wantClass: "entry", wantData: "foo bar baz"},
		{name: "json-data", input: `k8s:Pod.v1:{namespace: "foo", name: "bar"}`, wantDomain: "k8s", wantClass: "Pod.v1", wantData: `{namespace: "foo", name: "bar"}`},
		{name: "empty", input: "", wantErr: true},
		{name: "no-colon", input: "nocolon", wantErr: true},
		{name: "one-colon", input: "a:b", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			domain, class, data, err := QuerySplit(test.input)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.wantDomain, domain)
				assert.Equal(t, test.wantClass, class)
				assert.Equal(t, test.wantData, data)
			}
		})
	}
}

func TestClassJoin(t *testing.T) {
	assert.Equal(t, "k8s:pod", ClassJoin("k8s", "pod"))
	assert.Equal(t, "a:b", ClassJoin("a", "b"))
}

func TestQueryJoin(t *testing.T) {
	assert.Equal(t, "k8s:pod:ns/name", QueryJoin("k8s", "pod", "ns/name"))
	assert.Equal(t, "a:b:", QueryJoin("a", "b", ""))
}

func TestClassSplitJoinRoundtrip(t *testing.T) {
	domain, class, err := ClassSplit("metric:prometheus")
	assert.NoError(t, err)
	assert.Equal(t, "metric:prometheus", ClassJoin(domain, class))
}

func TestQuerySplitJoinRoundtrip(t *testing.T) {
	domain, class, data, err := QuerySplit("log:entry:{severity=error}")
	assert.NoError(t, err)
	assert.Equal(t, "log:entry:{severity=error}", QueryJoin(domain, class, data))
}
