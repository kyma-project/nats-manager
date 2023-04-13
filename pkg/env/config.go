package env

import (
	"github.com/kelseyhightower/envconfig"
)

// Config represents the environment config for the NATS Manager.
type Config struct {
	NATSChartDir string `envconfig:"NATS_CHART_DIR" required:"true"`
}

func GetConfig() (Config, error) {
	cfg := Config{}
	if err := envconfig.Process("", &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
