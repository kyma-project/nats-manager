package nats

import (
	"context"
	"fmt"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
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

	// get NATS resources to de-provision
	instance, err := r.initNATSInstance(ctx, nats, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	// create a new NATS client instance
	natsClientPort := 4222
	r.natsClient = NewNatsClient(&natsConfig{
		URL: fmt.Sprintf("nats://%s.%s.svc.cluster.local:%d", instance.Name, instance.Namespace, natsClientPort),
	})
	if err = r.natsClient.Init(); err != nil {
		return ctrl.Result{}, err
	}
	// check if NATS JetStream stream exists
	streamExists, err := r.natsClient.StreamExists()
	if err != nil {
		// TODO: if stream existence cannot be checked, delete anyway. Should check the error code for such errors.
		return ctrl.Result{}, err
	}
	// if streamExists, do not delete the NATS cluster
	if streamExists {
		return ctrl.Result{}, fmt.Errorf("cannot delete NATS stream exists")
	}

	// delete all NATS resources
	if err = r.natsManager.DeleteInstance(ctx, instance); err != nil {
		return ctrl.Result{}, r.syncNATSStatusWithErr(ctx, nats, err, log)
	}

	return r.removeFinalizer(ctx, nats)
}
