// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package config

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBytes(t *testing.T) {
	for _, tc := range []struct {
		json string
		want Bytes
	}{
		{`"512Mi"`, 512 * 1024 * 1024},
		{`1024`, 1024},
		{`9007199254740993`, 9007199254740993},
		{`9223372036854775807`, 9223372036854775807},
		{`"-1"`, -1},
	} {
		var got Bytes
		require.NoError(t, json.Unmarshal([]byte(tc.json), &got))
		assert.Equal(t, tc.want, got)
	}
}

func TestBytesInvalid(t *testing.T) {
	var got Bytes
	assert.Error(t, json.Unmarshal([]byte(`true`), &got))
	assert.Error(t, json.Unmarshal([]byte(`"invalid"`), &got))
	assert.Error(t, json.Unmarshal([]byte(`1.5`), &got))
	assert.Error(t, json.Unmarshal([]byte(`9223372036854775808`), &got))
}

func TestMemoryPressureDefaults(t *testing.T) {
	var tuning Tuning
	assert.Equal(t, 80, tuning.GetMemoryPressureLimit())
	assert.Equal(t, 70, tuning.GetMemoryPressureReset())

	tuning.MemoryPressureLimit = 90
	tuning.MemoryPressureReset = 75
	assert.Equal(t, 90, tuning.GetMemoryPressureLimit())
	assert.Equal(t, 75, tuning.GetMemoryPressureReset())
}
