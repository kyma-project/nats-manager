package nats

import (
	"context"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *Reconciler) handleNATSDeletion(ctx context.Context, nats *natsv1alpha1.NATS,
	log *zap.SugaredLogger) (ctrl.Result, error) {
	// skip reconciliation for deletion if the finalizer is not set.
	if !r.containsFinalizer(nats) {
		log.Debugf("skipped reconciliation for deletion as finalize is not set.")
		return ctrl.Result{}, nil
	}

	r.logger.Info("Deleting the NATS")
	nats.Status.SetStateDeleting()

	// get NATS resources to de-provision
	instance, err := r.initNATSInstance(ctx, nats, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	// delete all NATS resources
	if err = r.natsManager.DeleteInstance(ctx, instance); err != nil {
		return ctrl.Result{}, r.syncNATSStatusWithErr(ctx, nats, err, log)
	}

	return r.removeFinalizer(ctx, nats)
}
