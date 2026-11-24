// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"net/url"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseQuery(t *testing.T) {
	d := mock.NewDomain("test", "foo", "bar")

	c, selector, err := ParseQuery(d, "test:foo:mydata")
	require.NoError(t, err)
	assert.Equal(t, "foo", c.Name())
	assert.Equal(t, "mydata", selector)

	for _, x := range []struct {
		name, query, wantErr string
	}{
		{"wrong domain", "other:foo:data", "wrong domain"},
		{"bad class", "test:nosuch:data", "class not found"},
		{"invalid format", "invalid", ""},
	} {
		t.Run(x.name, func(t *testing.T) {
			_, _, err := ParseQuery(d, x.query)
			require.Error(t, err)
			if x.wantErr != "" {
				assert.ErrorContains(t, err, x.wantErr)
			}
		})
	}
}

func TestSplitURLQueryData(t *testing.T) {
	for _, tc := range []struct {
		name, data, wantData string
		want                 url.Values
		wantErr              string
	}{
		{name: "absent", data: `selector{label="a"}`, wantData: `selector{label="a"}`},
		{name: "arbitrary key", data: `selector?tenant=x`, wantData: `selector`, want: url.Values{"tenant": {"x"}}},
		{name: "repeated key", data: `selector?k=a&k=b`, wantData: `selector`, want: url.Values{"k": {"a", "b"}}},
		{name: "encoded", data: `selector?na%6D%65=a%2Db`, wantData: `selector`, want: url.Values{"name": {"a-b"}}},
		{name: "empty section", data: `selector?`, wantData: `selector?`},
		{name: "bare key accepted by URL parser", data: `selector?flag`, wantData: `selector`, want: url.Values{"flag": {""}}},
		{name: "empty key accepted by URL parser", data: `selector?=value`, wantData: `selector`, want: url.Values{"": {"value"}}},
		{name: "semicolon rejected", data: `selector?a=x;b=y`, wantData: `selector?a=x;b=y`, wantErr: `invalid semicolon`},
		{name: "malformed escape", data: `selector?x=%zz`, wantData: `selector?x=%zz`, wantErr: `invalid URL escape`},
		{name: "rightmost candidate", data: `selector?first=x?last=y`, wantData: `selector?first=x`, want: url.Values{"last": {"y"}}},
		{name: "skip malformed candidate for valid rightmost", data: `selector?bad=%zz?last=y`, wantData: `selector?bad=%zz`, want: url.Values{"last": {"y"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotData, got, err := SplitURLQueryData(tc.data)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				assert.Equal(t, tc.wantData, gotData)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantData, gotData)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestUnmarshalQueryString(t *testing.T) {
	d := mock.NewDomain("test", "foo")
	type Data struct{ Name string }

	c, data, err := UnmarshalQueryString[Data](d, `test:foo:{"name":"hello"}`)
	require.NoError(t, err)
	assert.Equal(t, "foo", c.Name())
	assert.Equal(t, "hello", data.Name)

	_, _, err = UnmarshalQueryString[Data](d, `test:foo:not valid json or yaml`)
	assert.ErrorContains(t, err, "invalid query")
}
