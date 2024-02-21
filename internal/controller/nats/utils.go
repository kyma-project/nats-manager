package nats

import (
	"context"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	kcontrollerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) containsFinalizer(nats *natsv1alpha1.NATS) bool {
	return controllerutil.ContainsFinalizer(nats, NATSFinalizerName)
}

func (r *Reconciler) addFinalizer(ctx context.Context, nats *natsv1alpha1.NATS) (kcontrollerruntime.Result, error) {
	controllerutil.AddFinalizer(nats, NATSFinalizerName)
	if err := r.Update(ctx, nats); err != nil {
		return kcontrollerruntime.Result{}, err
	}
	return kcontrollerruntime.Result{}, nil
}

func (r *Reconciler) removeFinalizer(ctx context.Context, nats *natsv1alpha1.NATS) (kcontrollerruntime.Result, error) {
	controllerutil.RemoveFinalizer(nats, NATSFinalizerName)
	if err := r.Update(ctx, nats); err != nil {
		return kcontrollerruntime.Result{}, err
	}

	return kcontrollerruntime.Result{}, nil
}
