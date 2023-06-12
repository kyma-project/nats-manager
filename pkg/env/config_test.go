package env

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_GetConfig(t *testing.T) {
	// given
	givenEnvs := map[string]string{}

	// when
	_, err := GetConfig()

	// then, should fail when required envs are not defined.
	require.Error(t, err)

	// given, define required envs
	givenEnvs["NATS_CHART_DIR"] = "/test1/test2"
	givenEnvs["NATS_CR_NAME"] = "name1"
	givenEnvs["NATS_CR_NAMESPACE"] = "namespace1"

	for k, v := range givenEnvs {
		t.Setenv(k, v)
	}

	// when
	config, err := GetConfig()

	// then, should pass when required envs are defined.
	require.NoError(t, err)
	require.Equal(t, givenEnvs["NATS_CHART_DIR"], config.NATSChartDir)
}
