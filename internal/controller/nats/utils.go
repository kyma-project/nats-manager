package nats

import (
	"context"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) containsFinalizer(nats *natsv1alpha1.Nats) bool {
	return controllerutil.ContainsFinalizer(nats, natsFinalizerName)
}

func (r *Reconciler) addFinalizer(ctx context.Context, nats *natsv1alpha1.Nats) (ctrl.Result, error) {
	controllerutil.AddFinalizer(nats, natsFinalizerName)
	if err := r.Update(ctx, nats); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) removeFinalizer(ctx context.Context, nats *natsv1alpha1.Nats) (ctrl.Result, error) {
	controllerutil.RemoveFinalizer(nats, natsFinalizerName)
	if err := r.Update(ctx, nats); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

