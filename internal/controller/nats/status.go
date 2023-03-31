package nats

import (
	"context"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stype "k8s.io/apimachinery/pkg/types"
)

// syncNATSStatus syncs NATS status and updates the k8s subscription.
// Returns the relevant error.
func (r *Reconciler) syncNATSStatusWithErr(ctx context.Context,
	nats *natsv1alpha1.Nats, err error, log *zap.SugaredLogger) error {
	// set error state in status
	nats.Status.SetStateError()
	nats.Status.UpdateConditionAvailable(metav1.ConditionFalse, natsv1alpha1.ConditionReasonProcessingError, err.Error())

	return r.syncNATSStatus(ctx, nats, log)
}

// syncNATSStatus syncs NATS status and updates the k8s subscription.
func (r *Reconciler) syncNATSStatus(ctx context.Context,
	nats *natsv1alpha1.Nats, log *zap.SugaredLogger) error {
	namespacedName := &k8stype.NamespacedName{
		Name:      nats.Name,
		Namespace: nats.Namespace,
	}

	// fetch the latest NATS object, to avoid k8s conflict errors.
	actualNATS := &natsv1alpha1.Nats{}
	if err := r.Client.Get(ctx, *namespacedName, actualNATS); err != nil {
		return err
	}

	// copy new changes to the latest object
	desiredNATS := actualNATS.DeepCopy()
	desiredNATS.Status = nats.Status

	// sync subscription status with k8s
	return r.updateStatus(ctx, actualNATS, desiredNATS, log)
}

// updateStatus updates the status to k8s if modified.
func (r *Reconciler) updateStatus(ctx context.Context, oldNATS, newNATS *natsv1alpha1.Nats,
	logger *zap.SugaredLogger) error {
	// compare the status taking into consideration lastTransitionTime in conditions
	if oldNATS.Status.IsEqual(newNATS.Status) {
		return nil
	}

	// update the status for subscription in k8s
	if err := r.Status().Update(ctx, newNATS); err != nil {
		return err
	}

	logger.Debugw("Updated NATS status",
		"oldStatus", oldNATS.Status, "newStatus", newNATS.Status)

	return nil
}
