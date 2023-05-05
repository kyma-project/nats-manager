package nats

import (
	"github.com/kyma-project/nats-manager/api/v1alpha1"
)

func HaveStatusReady(nats v1alpha1.NATS) bool {
	return nats.Status.State == v1alpha1.StateReady
}
