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
	givenEnvs["ALPINE_IMAGE"] = "alpine-image-url"
	givenEnvs["ALPINE_IMAGE_FIPS"] = "alpine-image-fips-url"
	givenEnvs["PROMETHEUS_NATS_EXPORTER_IMAGE"] = "prometheus-image-url"
	givenEnvs["PROMETHEUS_NATS_EXPORTER_IMAGE_FIPS"] = "prometheus-image-fips-url"
	givenEnvs["NATS_SERVER_CONFIG_RELOADER_IMAGE"] = "srvr-cfg-rldr-image-url"
	givenEnvs["NATS_SERVER_CONFIG_RELOADER_IMAGE_FIPS"] = "srvr-cfg-rldr-image-fips-url"
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
	require.Equal(t, givenEnvs["ALPINE_IMAGE_FIPS"], config.AlpineImageFIPS)
	require.Equal(t, givenEnvs["PROMETHEUS_NATS_EXPORTER_IMAGE_FIPS"], config.PrometheusExporterImageFIPS)
	require.Equal(t, givenEnvs["NATS_SERVER_CONFIG_RELOADER_IMAGE_FIPS"], config.NATSSrvCfgReloaderImageFIPS)
	require.Equal(t, true, config.FIPSModeEnabled)
}
