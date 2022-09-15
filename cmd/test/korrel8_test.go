//go:build cluster

package test

import (
	"encoding/json"
	"testing"

	"os/exec"

	"github.com/alanconway/korrel8/internal/pkg/test"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/stretchr/testify/require"
)

func TestKorrel8_Cmd_GetAlert(t *testing.T) {
	cmd := exec.Command("go", "run", "../korrel8/main.go", "get", "alert", "", "-o=json")
	out, err := cmd.Output()
	require.NoError(t, test.ExecError(err))
	var result []models.GettableAlert
	require.NoError(t, json.Unmarshal(out, &result), "expect valid alerts")
	require.NotEmpty(t, result, "expect at least one alert")
}
