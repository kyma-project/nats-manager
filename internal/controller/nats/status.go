package nats

import (
	"context"
	"errors"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/pkg/k8s"
	"go.uber.org/zap"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	ktypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// syncNATSStatus syncs NATS status.
// Returns the relevant error.
func (r *Reconciler) syncNATSStatusWithErr(ctx context.Context,
	nats *nmapiv1alpha1.NATS, err error, log *zap.SugaredLogger,
) error {
	// set error state in status
	nats.Status.SetStateError()
	nats.Status.UpdateConditionAvailable(kmetav1.ConditionFalse, nmapiv1alpha1.ConditionReasonProcessingError, err.Error())

	// return the original error so the controller triggers another reconciliation.
	return errors.Join(err, r.syncNATSStatus(ctx, nats, log))
}

// syncNATSStatus syncs NATS status.
func (r *Reconciler) syncNATSStatus(ctx context.Context,
	nats *nmapiv1alpha1.NATS, log *zap.SugaredLogger,
) error {
	namespacedName := &ktypes.NamespacedName{
		Name:      nats.Name,
		Namespace: nats.Namespace,
	}

	// fetch the latest NATS object, to avoid k8s conflict errors.
	actualNATS := &nmapiv1alpha1.NATS{}
	if err := r.Client.Get(ctx, *namespacedName, actualNATS); err != nil {
		return err
	}

	// copy new changes to the latest object
	desiredNATS := actualNATS.DeepCopy()
	desiredNATS.Status = nats.Status

	// sync nats resource status with k8s
	return r.updateStatus(ctx, actualNATS, desiredNATS, log)
}

// updateStatus updates the status to k8s if modified.
func (r *Reconciler) updateStatus(ctx context.Context, oldNATS, newNATS *nmapiv1alpha1.NATS,
	logger *zap.SugaredLogger,
) error {
	// compare the status taking into consideration lastTransitionTime in conditions
	if oldNATS.Status.IsEqual(newNATS.Status) {
		return nil
	}

	// update the status for nats resource
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
	selector, err := kmetav1.LabelSelectorAsSelector(&kmetav1.LabelSelector{
		MatchLabels: map[string]string{
			ManagedByLabelKey: ManagedByLabelValue,
		},
	})
	if err != nil {
		return err
	}
	labelSelectorPredicate := predicate.NewTypedPredicateFuncs[*unstructured.Unstructured](
		func(o *unstructured.Unstructured) bool {
			return selector.Matches(labels.Set(o.GetLabels()))
		},
	)

	// define ignore creation predicate
	ignoreCreationPredicate := predicate.TypedFuncs[*unstructured.Unstructured]{
		CreateFunc: func(e event.TypedCreateEvent[*unstructured.Unstructured]) bool {
			logger.Debugw("Ignoring create event for DestinationRule", "name", e.Object.GetName(),
				"namespace", e.Object.GetNamespace(), "kind", e.Object.GetObjectKind())
			return false
		},
	}

	// define handler.
	objectHandler := handler.TypedEnqueueRequestForOwner[*unstructured.Unstructured](r.scheme,
		r.ctrlManager.GetRESTMapper(),
		destinationRuleType,
		handler.OnlyControllerOwner(),
	)

	return r.controller.Watch(
		source.Kind(
			r.ctrlManager.GetCache(),
			destinationRuleType,
			objectHandler,
			labelSelectorPredicate,
			ignoreCreationPredicate,
		))
}
