package env

import (
	"github.com/kelseyhightower/envconfig"
)

// Config represents the environment config for the NATS Manager.
type Config struct {
	LogLevel                    string `envconfig:"LOG_LEVEL"         required:"true"`
	NATSChartDir                string `envconfig:"NATS_CHART_DIR"    required:"true"`
	NATSCRName                  string `envconfig:"NATS_CR_NAME"      required:"true"`
	NATSCRNamespace             string `envconfig:"NATS_CR_NAMESPACE" required:"true"`
	FIPSModeEnabled             bool   `envconfig:"KYMA_FIPS_MODE_ENABLED" default:"false"`
	NATSImage                   string `envconfig:"NATS_IMAGE"        required:"true"`
	NATSImageFIPS               string `envconfig:"NATS_IMAGE_FIPS"   required:"true"`
	NATSSrvCfgReloaderImage     string `envconfig:"NATS_SERVER_CONFIG_RELOADER_IMAGE"        required:"true"`
	NATSSrvCfgReloaderImageFIPS string `envconfig:"NATS_SERVER_CONFIG_RELOADER_IMAGE_FIPS"   required:"true"`
	PrometheusExporterImage     string `envconfig:"PROMETHEUS_NATS_EXPORTER_IMAGE"        required:"true"`
	PrometheusExporterImageFIPS string `envconfig:"PROMETHEUS_NATS_EXPORTER_IMAGE_FIPS"   required:"true"`
	AlpineImage                 string `envconfig:"ALPINE_IMAGE"        required:"true"`
	AlpineImageFIPS             string `envconfig:"ALPINE_IMAGE_FIPS"   required:"true"`
}

func GetConfig() (Config, error) {
	cfg := Config{}
	if err := envconfig.Process("", &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

type ContainerImages struct {
	NATS               string
	Alpine             string
	PrometheusExporter string
	NATSConfigReloader string
}

func (cfg Config) GetImageConfig() ContainerImages {
	if cfg.FIPSModeEnabled {
		return ContainerImages{
			NATS:               cfg.NATSImageFIPS,
			Alpine:             cfg.AlpineImageFIPS,
			PrometheusExporter: cfg.PrometheusExporterImageFIPS,
			NATSConfigReloader: cfg.NATSSrvCfgReloaderImageFIPS,
		}
	}
	return ContainerImages{
		NATS:               cfg.NATSImage,
		Alpine:             cfg.AlpineImage,
		PrometheusExporter: cfg.PrometheusExporterImage,
		NATSConfigReloader: cfg.NATSSrvCfgReloaderImage,
	}
}
