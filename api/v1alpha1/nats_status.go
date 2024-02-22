package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/api/meta"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (ns *NATSStatus) IsEqual(status NATSStatus) bool {
	thisWithoutCond := ns.DeepCopy()
	statusWithoutCond := status.DeepCopy()

	// remove conditions, so that we don't compare them
	thisWithoutCond.Conditions = []kmetav1.Condition{}
	statusWithoutCond.Conditions = []kmetav1.Condition{}

	return reflect.DeepEqual(thisWithoutCond, statusWithoutCond) &&
		ConditionsEquals(ns.Conditions, status.Conditions)
}

func (ns *NATSStatus) FindCondition(conditionType ConditionType) *kmetav1.Condition {
	for _, condition := range ns.Conditions {
		if string(conditionType) == condition.Type {
			return &condition
		}
	}
	return nil
}

func (ns *NATSStatus) UpdateConditionStatefulSet(status kmetav1.ConditionStatus, reason ConditionReason,
	message string,
) {
	condition := kmetav1.Condition{
		Type:               string(ConditionStatefulSet),
		Status:             status,
		LastTransitionTime: kmetav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
	meta.SetStatusCondition(&ns.Conditions, condition)
}

func (ns *NATSStatus) UpdateConditionAvailable(status kmetav1.ConditionStatus, reason ConditionReason,
	message string,
) {
	condition := kmetav1.Condition{
		Type:               string(ConditionAvailable),
		Status:             status,
		LastTransitionTime: kmetav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
	meta.SetStatusCondition(&ns.Conditions, condition)
}

func (ns *NATSStatus) SetStateReady() {
	ns.State = StateReady
	ns.UpdateConditionStatefulSet(kmetav1.ConditionTrue,
		ConditionReasonStatefulSetAvailable, "StatefulSet is ready")
	ns.UpdateConditionAvailable(kmetav1.ConditionTrue, ConditionReasonDeployed, "NATS is deployed")
}

func (ns *NATSStatus) SetStateProcessing() {
	ns.State = StateProcessing
}

func (ns *NATSStatus) SetStateWarning() {
	ns.State = StateWarning
}

func (ns *NATSStatus) SetWaitingStateForStatefulSet() {
	ns.SetStateProcessing()
	ns.UpdateConditionStatefulSet(kmetav1.ConditionFalse,
		ConditionReasonStatefulSetPending, "")
	ns.UpdateConditionAvailable(kmetav1.ConditionFalse, ConditionReasonDeploying, "")
}

func (ns *NATSStatus) SetStateError() {
	ns.State = StateError
	ns.UpdateConditionStatefulSet(kmetav1.ConditionFalse, ConditionReasonSyncFailError, "")
	ns.UpdateConditionAvailable(kmetav1.ConditionFalse, ConditionReasonProcessingError, "")
}

func (ns *NATSStatus) SetStateDeleting() {
	ns.State = StateDeleting
}

func (ns *NATSStatus) UpdateConditionDeletion(status kmetav1.ConditionStatus, reason ConditionReason,
	message string,
) {
	condition := kmetav1.Condition{
		Type:               string(ConditionDeleted),
		Status:             status,
		LastTransitionTime: kmetav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
	meta.SetStatusCondition(&ns.Conditions, condition)
}

func (ns *NATSStatus) Initialize() {
	ns.SetStateProcessing()
	ns.UpdateConditionStatefulSet(kmetav1.ConditionFalse, ConditionReasonProcessing, "")
	ns.UpdateConditionAvailable(kmetav1.ConditionFalse, ConditionReasonProcessing, "")
}

// ClearURL clears the url.
func (ns *NATSStatus) ClearURL() {
	ns.URL = ""
}

// SetURL sets the url.
func (ns *NATSStatus) SetURL(url string) {
	ns.URL = url
}
