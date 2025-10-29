// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeLabel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Valid label", "valid_label_123", "valid_label_123"},
		{"Label with colon", "app:version", "app:version"},
		{"Label starting with number", "123invalid", "_23invalid"},
		{"Label with special chars", "app-name.service", "app_name_service"},
		{"Label with spaces", "my app", "my_app"},
		{"Empty string", "", ""},
		{"Only invalid chars", "-.@", "___"},
		{"Mixed valid/invalid", "app-1.2.3", "app_1_2_3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeLabel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeLabels(t *testing.T) {
	t.Run("valid string map", func(t *testing.T) {
		input := map[string]string{
			"valid_key":         "value1",
			"app/location":      "value2",
			"app.version":       "value3",
			"app-name":          "value4",
			"many./-separators": "value5",
		}

		result, err := SafeLabels(input)
		assert.NoError(t, err)

		resultMap := result.(map[string]string)
		expected := map[string]string{
			"valid_key":         "value1",
			"app_location":      "value2",
			"app_version":       "value3",
			"app_name":          "value4",
			"many___separators": "value5",
		}
		assert.Equal(t, expected, resultMap)
	})

	t.Run("valid string interface map", func(t *testing.T) {
		input := map[string]any{
			"123test":   "value1",
			"app.name":  42,
			"valid_key": true,
		}

		result, err := SafeLabels(input)
		assert.NoError(t, err)

		resultMap := result.(map[string]any)
		expected := map[string]any{
			"_23test":   "value1",
			"app_name":  42,
			"valid_key": true,
		}
		assert.Equal(t, expected, resultMap)
	})

	t.Run("invalid non-string key map", func(t *testing.T) {
		input := map[int]string{
			1: "value1",
			2: "value2",
		}

		result, err := SafeLabels(input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expecting map[string]T")
		assert.Nil(t, result)
	})

	t.Run("non-map input", func(t *testing.T) {
		input := "not a map"

		result, err := SafeLabels(input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expecting map[string]T")
		assert.Nil(t, result)
	})
}

func TestLogType(t *testing.T) {
	for _, x := range [][]string{
		{"default", "infrastructure"},
		{"openshift", "infrastructure"},
		{"openshift-", "infrastructure"},
		{"openshift-foo", "infrastructure"},
		{"kube", "infrastructure"}, {"kube", "infrastructure"},
		{"kube-", "infrastructure"},
		{"kube-foo", "infrastructure"},
		{"foo", "application"},
		{"foo-kube", "application"},
		{"foo-openshift", "application"},
	} {
		t.Run(x[0], func(t *testing.T) {
			assert.Equal(t, x[1], logTypeForNamespace(x[0]))
		})
	}
}
