package provisioner

import (
	"fmt"
)

type NATSConfig struct {
	ClusterSize int
}

type Provisioner interface {
	Deploy(config NATSConfig) error
	Delete() error
}

type NATSProvisioner struct {
}

func (r NATSProvisioner) Deploy(_ NATSConfig) error {
	fmt.Println("NATS cluster is deployed") //nolint:forbidigo //keep temporarily
	return nil
}

func (r NATSProvisioner) Delete() error {
	fmt.Println("NATS cluster is deleted") //nolint:forbidigo //keep temprarily
	return nil
}
