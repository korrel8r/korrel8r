package main_test

import (
	"encoding/json"
	"testing"

	"os/exec"

	"github.com/alanconway/korrel8/internal/pkg/test"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/stretchr/testify/require"
)

func TestCmd_GetAlert(t *testing.T) {
	test.SkipIfNoCluster(t)
	cmd := exec.Command("go", "run", "main.go", "get", "alert", "", "-o=json")
	out, err := cmd.Output()
	require.NoError(t, test.ExitError(err))
	var result []models.GettableAlert
	require.NoError(t, json.Unmarshal(out, &result), "expect valid alerts")
	require.NotEmpty(t, result, "expect at least one alert")
}
