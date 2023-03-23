package controller

import natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"

// isInDeletion checks if the Nats deletion timestamp is set.
func isInDeletion(nats *natsv1alpha1.Nats) bool {
	return !nats.DeletionTimestamp.IsZero()
}
