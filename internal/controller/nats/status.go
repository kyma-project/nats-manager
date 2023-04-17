package nats

import (
	"context"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/pkg/k8s"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8stype "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// syncNATSStatus syncs NATS status and updates the k8s subscription.
// Returns the relevant error.
func (r *Reconciler) syncNATSStatusWithErr(ctx context.Context,
	nats *natsv1alpha1.NATS, err error, log *zap.SugaredLogger) error {
	// set error state in status
	nats.Status.SetStateError()
	nats.Status.UpdateConditionAvailable(metav1.ConditionFalse, natsv1alpha1.ConditionReasonProcessingError, err.Error())

	return r.syncNATSStatus(ctx, nats, log)
}

// syncNATSStatus syncs NATS status and updates the k8s subscription.
func (r *Reconciler) syncNATSStatus(ctx context.Context,
	nats *natsv1alpha1.NATS, log *zap.SugaredLogger) error {
	namespacedName := &k8stype.NamespacedName{
		Name:      nats.Name,
		Namespace: nats.Namespace,
	}

	// fetch the latest NATS object, to avoid k8s conflict errors.
	actualNATS := &natsv1alpha1.NATS{}
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
func (r *Reconciler) updateStatus(ctx context.Context, oldNATS, newNATS *natsv1alpha1.NATS,
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

// watchDestinationRule watches DestinationRules.
// It triggers reconciliation for NATS CR.
func (r *Reconciler) watchDestinationRule(logger *zap.SugaredLogger) error {
	// define DestinationRule type.
	destinationRuleType := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       k8s.DestinationRuleKind,
			"apiVersion": k8s.DestinationRuleAPIVersion,
		},
	}

	// define label selector for "managed-by".
	labelSelectorPredicate, err := predicate.LabelSelectorPredicate(
		metav1.LabelSelector{
			MatchLabels: map[string]string{
				ManagedByLabelKey: ManagedByLabelValue,
			},
		})
	if err != nil {
		return err
	}

	// define ignore creation predicate
	ignoreCreationPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			logger.Debugw("Ignoring create event for DestinationRule", "name", e.Object.GetName(),
				"namespace", e.Object.GetNamespace(), "kind", e.Object.GetObjectKind())
			return false
		},
	}

	// start watcher for DestinationRules.
	return r.controller.Watch(
		&source.Kind{Type: destinationRuleType},
		&handler.EnqueueRequestForOwner{OwnerType: &natsv1alpha1.NATS{}, IsController: true},
		labelSelectorPredicate,
		predicate.ResourceVersionChangedPredicate{},
		ignoreCreationPredicate,
	)
}
