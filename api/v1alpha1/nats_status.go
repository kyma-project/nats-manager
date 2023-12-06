package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (ns *NATSStatus) IsEqual(status NATSStatus) bool {
	thisWithoutCond := ns.DeepCopy()
	statusWithoutCond := status.DeepCopy()

	// remove conditions, so that we don't compare them
	thisWithoutCond.Conditions = []metav1.Condition{}
	statusWithoutCond.Conditions = []metav1.Condition{}

	return reflect.DeepEqual(thisWithoutCond, statusWithoutCond) &&
		ConditionsEquals(ns.Conditions, status.Conditions)
}

func (ns *NATSStatus) FindCondition(conditionType ConditionType) *metav1.Condition {
	for _, condition := range ns.Conditions {
		if string(conditionType) == condition.Type {
			return &condition
		}
	}
	return nil
}

func (ns *NATSStatus) UpdateConditionStatefulSet(status metav1.ConditionStatus, reason ConditionReason,
	message string) {
	condition := metav1.Condition{
		Type:               string(ConditionStatefulSet),
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
	meta.SetStatusCondition(&ns.Conditions, condition)
}

func (ns *NATSStatus) UpdateConditionAvailable(status metav1.ConditionStatus, reason ConditionReason,
	message string) {
	condition := metav1.Condition{
		Type:               string(ConditionAvailable),
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
	meta.SetStatusCondition(&ns.Conditions, condition)
}

func (ns *NATSStatus) SetStateReady() {
	ns.State = StateReady
	ns.UpdateConditionStatefulSet(metav1.ConditionTrue,
		ConditionReasonStatefulSetAvailable, "StatefulSet is ready")
	ns.UpdateConditionAvailable(metav1.ConditionTrue, ConditionReasonDeployed, "NATS is deployed")
}

func (ns *NATSStatus) SetStateProcessing() {
	ns.State = StateProcessing
}

func (ns *NATSStatus) SetStateWarning() {
	ns.State = StateWarning
}

func (ns *NATSStatus) SetWaitingStateForStatefulSet() {
	ns.SetStateProcessing()
	ns.UpdateConditionStatefulSet(metav1.ConditionFalse,
		ConditionReasonStatefulSetPending, "")
	ns.UpdateConditionAvailable(metav1.ConditionFalse, ConditionReasonDeploying, "")
}

func (ns *NATSStatus) SetStateError() {
	ns.State = StateError
	ns.UpdateConditionStatefulSet(metav1.ConditionFalse, ConditionReasonSyncFailError, "")
	ns.UpdateConditionAvailable(metav1.ConditionFalse, ConditionReasonProcessingError, "")
}

func (ns *NATSStatus) SetStateDeleting() {
	ns.State = StateDeleting
}

func (ns *NATSStatus) UpdateConditionDeletion(status metav1.ConditionStatus, reason ConditionReason,
	message string) {
	condition := metav1.Condition{
		Type:               string(ConditionDeleted),
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
	meta.SetStatusCondition(&ns.Conditions, condition)
}

func (ns *NATSStatus) Initialize() {
	ns.SetStateProcessing()
	ns.UpdateConditionStatefulSet(metav1.ConditionFalse, ConditionReasonProcessing, "")
	ns.UpdateConditionAvailable(metav1.ConditionFalse, ConditionReasonProcessing, "")
}

// ClearURL clears the url.
func (ns *NATSStatus) ClearURL() {
	ns.URL = ""
}

// SetURL sets the url.
func (ns *NATSStatus) SetURL(url string) {
	ns.URL = url
}
