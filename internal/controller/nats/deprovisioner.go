package nats

import (
	"context"
	"fmt"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	StreamExistsErrorMsg = "Cannot delete NATS cluster as stream exists"
	natsClientPort       = 4222
	instanceLabelKey     = "app.kubernetes.io/instance"
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
		// delete a PVC if NATS client cannot be created
		return r.deletePVCsAndRemoveFinalizer(ctx, r.Client, nats, r.logger)
	}
	// check if NATS JetStream stream exists
	streamExists, err := r.natsClient.StreamExists()
	if err != nil {
		// delete a PVC if NATS client cannot be created
		return r.deletePVCsAndRemoveFinalizer(ctx, r.Client, nats, r.logger)
	}
	if streamExists {
		// if a stream exists, do not delete the NATS cluster
		nats.Status.UpdateConditionDeletion(metav1.ConditionFalse,
			natsv1alpha1.ConditionReasonDeletionError, StreamExistsErrorMsg)
		return ctrl.Result{Requeue: true}, r.syncNATSStatus(ctx, nats, log)
	}

	return r.deletePVCsAndRemoveFinalizer(ctx, r.Client, nats, r.logger)
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

func (r *Reconciler) deletePVCsAndRemoveFinalizer(ctx context.Context, client client.Client, nats *natsv1alpha1.NATS, log *zap.SugaredLogger) (ctrl.Result, error) {
	if err := deletePVCsWithLabel(ctx, client, nats, log); err != nil {
		return ctrl.Result{}, err
	}
	return r.removeFinalizer(ctx, nats)
}

func deletePVCsWithLabel(ctx context.Context, c client.Client, nats *natsv1alpha1.NATS, log *zap.SugaredLogger) error {
	// create a new labels.Selector object for the label selector
	labelSelector := fmt.Sprintf("%s=%s", instanceLabelKey, nats.Name)
	selector, err := labels.Parse(labelSelector)
	if err != nil {
		return err
	}

	// create a new list of PVC objects that match the label selector
	pvcList := &corev1.PersistentVolumeClaimList{}
	err = c.List(ctx, pvcList, &client.ListOptions{
		Namespace:     nats.Namespace,
		LabelSelector: selector,
	})
	if err != nil {
		return fmt.Errorf("failed to list PVCs: %w", err)
	}

	if len(pvcList.Items) == 0 {
		log.Debug("No PVCs found")
		return nil
	}

	// delete each PVC in the list
	for _, pvc := range pvcList.Items {
		err = c.Delete(ctx, &pvc)
		if err != nil {
			return fmt.Errorf("failed to delete PVC: %w", err)
		}
		log.Debugf("PVC deleted: %s/%s", pvc.Namespace, pvc.Name)
	}

	return nil
}
