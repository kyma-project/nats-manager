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
	givenEnvs["LOG_LEVEL"] = "info"
	givenEnvs["NATS_IMAGE"] = "nats-image-url"
	givenEnvs["NATS_IMAGE_FIPS"] = "nats-image-fips-url"
	givenEnvs["KYMA_FIPS_MODE_ENABLED"] = "true"

	for k, v := range givenEnvs {
		t.Setenv(k, v)
	}

	// when
	config, err := GetConfig()

	// then, should pass when required envs are defined.
	require.NoError(t, err)
	require.Equal(t, givenEnvs["NATS_CHART_DIR"], config.NATSChartDir)
	require.Equal(t, givenEnvs["NATS_IMAGE_FIPS"], config.NATSImageFIPS)
	require.Equal(t, true, config.FIPSModeEnabled)
}
