package nats

import (
	"context"
	"fmt"

	"github.com/kyma-project/nats-manager/pkg/events"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	natspkg "github.com/kyma-project/nats-manager/pkg/nats"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	StreamExistsErrorMsg   = "Cannot delete NATS cluster as customer stream exists"
	ConsumerExistsErrorMsg = "Cannot delete NATS cluster as stream consumer exists"
	natsClientPort         = 4222
	InstanceLabelKey       = "app.kubernetes.io/instance"
	SapStreamName          = "sap"
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
	events.Normal(r.recorder, nats, natsv1alpha1.ConditionReasonDeleting, "Deleting the NATS cluster.")

	// create a new NATS client instance.
	if err := r.createAndConnectNatsClient(nats); err != nil {
		return r.deletePVCsAndRemoveFinalizer(ctx, nats, r.logger)
	}

	customerStreamExists, err := r.customerStreamExists(nats)
	if err != nil {
		return r.deletePVCsAndRemoveFinalizer(ctx, nats, r.logger)
	}
	// if any streams exists except for 'sap' stream, block the deletion.
	if customerStreamExists {
		nats.Status.SetStateWarning()
		nats.Status.UpdateConditionDeletion(metav1.ConditionFalse,
			natsv1alpha1.ConditionReasonDeletionError, StreamExistsErrorMsg)
		events.Warn(r.recorder, nats, natsv1alpha1.ConditionReasonDeletionError, StreamExistsErrorMsg)
		return ctrl.Result{Requeue: true}, r.syncNATSStatus(ctx, nats, log)
	}

	sapStreamConsumerExists, err := r.sapStreamConsumerExists(nats)
	if err != nil {
		return r.deletePVCsAndRemoveFinalizer(ctx, nats, r.logger)
	}
	// if any 'sap' stream consumer exists, block the deletion.
	if sapStreamConsumerExists {
		nats.Status.SetStateWarning()
		nats.Status.UpdateConditionDeletion(metav1.ConditionFalse,
			natsv1alpha1.ConditionReasonDeletionError, ConsumerExistsErrorMsg)
		events.Warn(r.recorder, nats, natsv1alpha1.ConditionReasonDeletionError, ConsumerExistsErrorMsg)
		return ctrl.Result{Requeue: true}, r.syncNATSStatus(ctx, nats, log)
	}

	return r.deletePVCsAndRemoveFinalizer(ctx, nats, r.logger)
}

// check if any other stream exists except for 'sap' stream.
func (r *Reconciler) customerStreamExists(nats *natsv1alpha1.NATS) (bool, error) {
	// check if any other stream exists except for 'sap' stream.
	streams, err := r.getNatsClient(nats).GetStreams()
	if err != nil {
		return false, err
	}
	for _, stream := range streams {
		if stream.Config.Name != SapStreamName {
			return true, nil
		}
	}
	return false, nil
}

func (r *Reconciler) sapStreamConsumerExists(nats *natsv1alpha1.NATS) (bool, error) {
	// check if 'sap' stream exists.
	streams, err := r.getNatsClient(nats).GetStreams()
	if err != nil {
		return false, err
	}
	sapStreamExists := false
	for _, stream := range streams {
		if stream.Config.Name == SapStreamName {
			sapStreamExists = true
			break
		}
	}
	// if 'sap' stream does not exist, return false.
	if !sapStreamExists {
		return false, nil
	}

	return r.getNatsClient(nats).ConsumersExist(SapStreamName)
}

// create a new NATS client instance and connect to the NATS server.
func (r *Reconciler) createAndConnectNatsClient(nats *natsv1alpha1.NATS) error {
	// create a new instance if it does not exist.
	if r.getNatsClient(nats) == nil {
		r.setNatsClient(nats, natspkg.NewNatsClient(&natspkg.Config{
			URL: fmt.Sprintf("nats://%s.%s.svc.cluster.local:%d", nats.Name, nats.Namespace, natsClientPort),
		}))
	}
	return r.getNatsClient(nats).Init()
}

func (r *Reconciler) deletePVCsAndRemoveFinalizer(ctx context.Context,
	nats *natsv1alpha1.NATS, log *zap.SugaredLogger) (ctrl.Result, error) {
	labelValue := nats.Name
	// TODO: delete the following logic after migrating to modular Kyma
	if nats.Name == "eventing-nats" {
		labelValue = "eventing"
	}
	// delete PVCs with the label selector.
	labelSelector := fmt.Sprintf("%s=%s", InstanceLabelKey, labelValue)
	if err := r.kubeClient.DeletePVCsWithLabel(ctx, labelSelector, nats.Name, nats.Namespace); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// close the nats connection and remove the client instance.
	r.closeNatsClient(nats)

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

// close the nats connection and remove the client instance.
func (r *Reconciler) closeNatsClient(nats *natsv1alpha1.NATS) {
	// check if nats client exists.
	if r.getNatsClient(nats) != nil {
		r.getNatsClient(nats).Close()
		r.setNatsClient(nats, nil)
	}
}
