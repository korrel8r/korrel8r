// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package otel

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInt_Marshal(t *testing.T) {
	for _, x := range []struct {
		i Int
		s string
	}{
		{i: 10, s: `10`},
		{i: -3, s: `-3`},
		{i: 999999999999999999, s: `"999999999999999999"`},   // too big for int32
		{i: -999999999999999999, s: `"-999999999999999999"`}, // too big for int32
	} {
		t.Run(x.s, func(t *testing.T) {
			b, err := json.Marshal(x.i)
			if assert.NoError(t, err) {
				assert.Equal(t, string(b), x.s)
			}
		})
	}
}
func TestInt_Unmarshal(t *testing.T) {
	for _, x := range []struct {
		i Int
		s string
	}{
		{i: 10, s: `10`},
		{i: -3, s: `-3`},
		{i: 999999999999999999, s: `"999999999999999999"`},
		{i: -999999999999999999, s: `"-999999999999999999"`},
		{i: 999999999999999999, s: `999999999999999999`},
		{i: -999999999999999999, s: `-999999999999999999`},
	} {
		t.Run(x.s, func(t *testing.T) {
			var i Int
			if assert.NoError(t, json.Unmarshal([]byte(x.s), &i)) {
				assert.Equal(t, x.i, i)
			}
		})
	}
}

func TestUnixNanoTime(t *testing.T) {
	ns := UnixNanoTime{time.Now()}

	b, err := json.Marshal(ns)
	if assert.NoError(t, err) {
		assert.Equal(t, string(b), fmt.Sprintf(`"%v"`, ns.UnixNano()))
		assert.Equal(t, string(b), ns.String())
	}

	var v UnixNanoTime
	if assert.NoError(t, json.Unmarshal(b, &v)) {
		assert.True(t, ns.Equal(v.Time), "want: %v, got: %v", ns, v.Time)
	}
}

func TestNanoDuration(t *testing.T) {
	d := NanoDuration{time.Second}
	b, err := json.Marshal(d)
	if assert.NoError(t, err) {
		assert.Equal(t, "1000000000", string(b))
		assert.Equal(t, "1000000000", d.String())
	}
	var d2 NanoDuration
	if assert.NoError(t, json.Unmarshal(b, &d2)) {
		assert.Equal(t, d, d2)
	}
}

func TestMilliDuration(t *testing.T) {
	d := MilliDuration{time.Second}
	b, err := json.Marshal(d)
	if assert.NoError(t, err) {
		assert.Equal(t, "1000", string(b))
		assert.Equal(t, "1000", d.String())
	}
	var d2 MilliDuration
	if assert.NoError(t, json.Unmarshal(b, &d2)) {
		assert.Equal(t, d, d2)
	}
}
