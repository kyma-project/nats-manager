package env

import (
	"github.com/kelseyhightower/envconfig"
)

// Config represents the environment config for the NATS Manager.
type Config struct {
	LogLevel        string `envconfig:"LOG_LEVEL"         required:"true"`
	NATSChartDir    string `envconfig:"NATS_CHART_DIR"    required:"true"`
	NATSCRName      string `envconfig:"NATS_CR_NAME"      required:"true"`
	NATSCRNamespace string `envconfig:"NATS_CR_NAMESPACE" required:"true"`
	FIPSModeEnabled bool   `envconfig:"KYMA_FIPS_MODE_ENABLED" default:"false"`
	NATSImage       string `envconfig:"NATS_IMAGE"        required:"true"`
	NATSImageFIPS   string `envconfig:"NATS_IMAGE_FIPS"   required:"true"`
}

func GetConfig() (Config, error) {
	cfg := Config{}
	if err := envconfig.Process("", &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
