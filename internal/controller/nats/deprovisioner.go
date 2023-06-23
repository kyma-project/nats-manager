package nats

import (
	"context"
	"fmt"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	StreamExistsErrorMsg = "Cannot delete NATS cluster as stream exists"
	natsClientPort       = 4222
)

func (r *Reconciler) handleNATSDeletion(ctx context.Context, nats *natsv1alpha1.NATS,
	log *zap.SugaredLogger) (ctrl.Result, error) {
	// skip reconciliation for deletion if the finalizer is not set.
	if !r.containsFinalizer(nats) {
		log.Debugf("skipped reconciliation for deletion as finalizer is not set.")
		return ctrl.Result{}, nil
	}

	r.logger.Info("Deleting the NATS")
	nats.Status.SetStateDeleting()

	// create a new NATS client instance
	if err := r.createAndConnectNatsClient(ctx, nats); err != nil {
		// delete the NATS cluster in case cannot be connected
		return r.removeFinalizer(ctx, nats)
	}
	// check if NATS JetStream stream exists
	streamExists, err := r.natsClient.StreamExists()
	if err != nil {
		// delete the NATS cluster if stream cannot be checked
		return r.removeFinalizer(ctx, nats)
	}
	if streamExists {
		// if a stream exists, do not delete the NATS cluster
		nats.Status.UpdateConditionDeletion(metav1.ConditionFalse,
			natsv1alpha1.ConditionReasonDeletionError, StreamExistsErrorMsg)
		return ctrl.Result{Requeue: true}, r.syncNATSStatus(ctx, nats, log)
	}

	return r.removeFinalizer(ctx, nats)
}

// create a new NATS client instance and connect to the NATS server
func (r *Reconciler) createAndConnectNatsClient(ctx context.Context, nats *natsv1alpha1.NATS) error {
	// create a new instance if it does not exist
	if r.natsClient == nil {
		r.natsClient = NewNatsClient(&natsConfig{
			URL: fmt.Sprintf("nats://%s.%s.svc.cluster.local:%d", nats.Name, nats.Namespace, natsClientPort),
		})
	}
	if err := r.natsClient.Init(); err != nil {
		return err
	}
	return nil
}
