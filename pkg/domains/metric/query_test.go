// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package metric

import (
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuery_Selectors(t *testing.T) {
	for _, x := range []struct {
		query Query
		want  []string
	}{
		{`{namespace="foo"}`, []string{`{namespace="foo"}`}},
		{`blah{namespace="foo"}`, []string{`blah{namespace="foo"}`}},
		{`{__name__="fred",namespace="foo"}`, []string{`{__name__="fred",namespace="foo"}`}},
		{`count(fred{namespace="foo"}) + count(barney{name="bar"})`, []string{`fred{namespace="foo"}`, `barney{name="bar"}`}},
		{`sum by(namespace, app) (rate(received_bytes_total{component_kind="source", component_type!="internal_metrics"}[5m]))`,
			[]string{"received_bytes_total{component_kind=\"source\",component_type!=\"internal_metrics\"}"}},

		// The namespace suffix is not part of the expression.
		{`{namespace="foo"}?namespace=foo`, []string{`{namespace="foo"}`}},
		{`count(fred{namespace="foo"}) + count(barney{name="bar"})?namespace=foo&namespace=bar`,
			[]string{`fred{namespace="foo"}`, `barney{name="bar"}`}},
	} {
		t.Run(x.query.String(), func(t *testing.T) {
			got, err := x.query.Selectors()
			if assert.NoError(t, err) {
				assert.Equal(t, x.want, got)
			}
		})
	}
}

func TestQuery_Split(t *testing.T) {
	for _, x := range []struct {
		name       string
		query      string
		wantExpr   string
		wantParams url.Values // nil: no suffix.
	}{
		// No suffix: the whole query is the expression.
		{name: "plain", query: `up`, wantExpr: `up`},
		{name: "selector", query: `up{namespace="foo"}`, wantExpr: `up{namespace="foo"}`},

		// No suffix: ? and key=value inside strings and comments are part of the expression.
		{name: "double quoted", query: `up{path="/a?b=c"}`, wantExpr: `up{path="/a?b=c"}`},
		{name: "single quoted", query: `up{path='/a?b=c'}`, wantExpr: `up{path='/a?b=c'}`},
		{name: "backtick quoted", query: "up{path=`/a?b=c`}", wantExpr: "up{path=`/a?b=c`}"},
		{name: "trailing string", query: `label_replace(up, "a", "b", "c", "x?namespace=foo")`,
			wantExpr: `label_replace(up, "a", "b", "c", "x?namespace=foo")`},
		{name: "trailing comment", query: `up # ?namespace=foo`, wantExpr: `up # ?namespace=foo`},

		// Suffix: split off and parsed as parameters.
		{name: "suffix", query: `up{namespace="foo"}?namespace=foo`, wantExpr: `up{namespace="foo"}`,
			wantParams: url.Values{"namespace": []string{"foo"}}},
		{name: "suffix with dots and dashes", query: `up?namespace=my-ns.example.com`, wantExpr: `up`,
			wantParams: url.Values{"namespace": []string{"my-ns.example.com"}}},
		{name: "repeated suffix", query: `up?namespace=a&namespace=b`, wantExpr: `up`,
			wantParams: url.Values{"namespace": []string{"a", "b"}}},
		{name: "percent encoded suffix", query: `up?namespace=my%2Dns`, wantExpr: `up`,
			wantParams: url.Values{"namespace": []string{"my-ns"}}},
		{name: "suffix after commented question mark", query: "up # ?namespace=foo\n?namespace=bar", wantExpr: "up # ?namespace=foo\n",
			wantParams: url.Values{"namespace": []string{"bar"}}},
	} {
		t.Run(x.name, func(t *testing.T) {
			expr, params, err := splitQuery(x.query)
			require.NoError(t, err)
			assert.Equal(t, x.wantExpr, expr)
			assert.Equal(t, x.wantParams, params)
		})
	}
}

func TestQuery_SplitError(t *testing.T) {
	for _, x := range []struct {
		name       string
		query      string
		wantSuffix bool   // Error is a suffix error, not a PromQL error.
		wantMsg    string // Substring of the error message.
	}{
		// Malformed suffixes report suffix errors, not PromQL errors.
		{name: "unknown parameter", query: `up?tenant=x`, wantSuffix: true, wantMsg: `unknown parameter "tenant"`},
		{name: "unknown parameter with encoding", query: `up?a=%41`, wantSuffix: true, wantMsg: `unknown parameter "a"`},
		{name: "empty namespace value", query: `up?namespace=`, wantSuffix: true, wantMsg: "empty namespace value"},
		{name: "empty repeated namespace value", query: `up?namespace=a&namespace=`, wantSuffix: true, wantMsg: "empty namespace value"},
		{name: "bad encoding", query: `up?namespace=%zz`, wantSuffix: true, wantMsg: `invalid URL escape`},
		{name: "invalid namespace value", query: `up?namespace=foo!bar`, wantSuffix: true, wantMsg: `invalid namespace value "foo!bar"`},
		{name: "invalid decoded namespace value", query: `up?namespace=foo%21bar`, wantSuffix: true, wantMsg: `invalid namespace value "foo!bar"`},
		{name: "suffix without expression", query: `?namespace=`, wantSuffix: true, wantMsg: "empty namespace value"},
		{name: "suffix after broken expression with bad parameter", query: `up{} garbage?namespace=`, wantSuffix: true, wantMsg: "empty namespace value"},

		// PromQL errors are reported unchanged: validity is deferred to Selectors.
		{name: "promql error", query: `up{`, wantSuffix: false},
		{name: "bare question mark", query: `up?`, wantSuffix: false},
		{name: "valid suffix with broken expression", query: `sum(rate(foo[5m])?namespace=ns`, wantSuffix: false},
	} {
		t.Run(x.name, func(t *testing.T) {
			_, _, err := splitQuery(x.query)
			require.Error(t, err)
			assert.Equal(t, x.wantSuffix, errors.Is(err, errSuffix), "suffix error: %v", err)
			if x.wantMsg != "" {
				assert.Contains(t, err.Error(), x.wantMsg)
			}
		})
	}
}

// TestDomainQuery verifies suffix validation at parse time and round-tripping.
func TestDomainQuery(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		for _, s := range []string{
			`metric:metric:up`,
			`metric:metric:up{namespace="foo"}`,
			`metric:metric:up{namespace="foo"}?namespace=foo`,
			`metric:metric:up?namespace=a&namespace=b`,
			`metric:metric:up?namespace=my%2Dns`,
		} {
			q, err := Domain.Query(s)
			require.NoError(t, err, s)
			assert.Equal(t, s, q.String())
			q2, err := Domain.Query(q.String()) // Round trip.
			require.NoError(t, err, s)
			assert.Equal(t, q, q2)
		}
	})

	t.Run("suffix errors", func(t *testing.T) {
		for _, s := range []string{
			`metric:metric:up?tenant=x`,
			`metric:metric:up?namespace=`,
			`metric:metric:up?namespace=a&namespace=`,
			`metric:metric:up?a=%zz`,
			`metric:metric:up?namespace=foo!bar`,
		} {
			_, err := Domain.Query(s)
			require.Error(t, err, s)
			assert.True(t, errors.Is(err, errSuffix), "want suffix error: %v", err)
		}
	})

	t.Run("promql errors deferred", func(t *testing.T) {
		for _, s := range []string{
			`metric:metric:up{`,
			`metric:metric:not valid promql`,
		} {
			_, err := Domain.Query(s)
			assert.NoError(t, err, s)
		}
	})
}
