package nats

import (
	"context"
	"fmt"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	natspkg "github.com/kyma-project/nats-manager/pkg/nats"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	StreamExistsErrorMsg = "Cannot delete NATS cluster as stream exists"
	natsClientPort       = 4222
	InstanceLabelKey     = "app.kubernetes.io/instance"
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
	if err := r.createAndConnectNatsClient(nats); err != nil {
		// delete a PVC if NATS client cannot be created
		return r.deletePVCsAndRemoveFinalizer(ctx, nats, r.logger)
	}
	// check if NATS JetStream stream exists
	streamExists, err := r.getNatsClient(nats).StreamExists()
	if err != nil {
		// delete a PVC if NATS client cannot be created
		return r.deletePVCsAndRemoveFinalizer(ctx, nats, r.logger)
	}
	if streamExists {
		// if a stream exists, do not delete the NATS cluster
		nats.Status.UpdateConditionDeletion(metav1.ConditionFalse,
			natsv1alpha1.ConditionReasonDeletionError, StreamExistsErrorMsg)
		return ctrl.Result{Requeue: true}, r.syncNATSStatus(ctx, nats, log)
	}

	return r.deletePVCsAndRemoveFinalizer(ctx, nats, r.logger)
}

// create a new NATS client instance and connect to the NATS server.
func (r *Reconciler) createAndConnectNatsClient(nats *natsv1alpha1.NATS) error {
	// create a new instance if it does not exist
	if r.getNatsClient(nats) == nil {
		r.setNatsClient(nats, natspkg.NewNatsClient(&natspkg.Config{
			URL: fmt.Sprintf("nats://%s.%s.svc.cluster.local:%d", nats.Name, nats.Namespace, natsClientPort),
		}))
	}
	return r.getNatsClient(nats).Init()
}

func (r *Reconciler) deletePVCsAndRemoveFinalizer(ctx context.Context,
	nats *natsv1alpha1.NATS, log *zap.SugaredLogger) (ctrl.Result, error) {
	// retrieve the labelSelector from the StatefulSet with the name: nats.Name
	sts, err := r.kubeClient.GetStatefulSet(ctx, nats.Name, nats.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}
	labelValue := sts.Labels[InstanceLabelKey]

	// delete PVCs with the label selector
	labelSelector := fmt.Sprintf("%s=%s", InstanceLabelKey, labelValue)
	if err := r.kubeClient.DeletePVCsWithLabel(ctx, labelSelector, nats.Namespace); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// close the nats connection and remove the client instance
	r.getNatsClient(nats).Close()
	r.setNatsClient(nats, nil)

	log.Debugf("deleted PVCs with a namespace: %s and label selector: %s", nats.Namespace, labelSelector)
	return r.removeFinalizer(ctx, nats)
}

func (r *Reconciler) getNatsClient(nats *natsv1alpha1.NATS) natspkg.Client {
	crKey := nats.Namespace + "/" + nats.Name
	return r.natsClients[crKey]
}

func (r *Reconciler) setNatsClient(nats *natsv1alpha1.NATS, newNatsClient natspkg.Client) {
	crKey := nats.Namespace + "/" + nats.Name
	r.natsClients[crKey] = newNatsClient
}
