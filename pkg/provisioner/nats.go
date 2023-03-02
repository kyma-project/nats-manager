package provisioner

import (
	"fmt"
)

type NatsConfig struct {
	ClusterSize int
}

type Provisioner interface {
	Deploy(config NatsConfig) error
	Delete() error
}

type NatsProvisioner struct {
}

func (r NatsProvisioner) Deploy(config NatsConfig) error {
	fmt.Println("NATS cluster is deployed") //nolint:forbidigo //keep temporarily
	return nil
}

func (r NatsProvisioner) Delete() error {
	fmt.Println("NATS cluster is deleted") //nolint:forbidigo //keep temprarily
	return nil
}
