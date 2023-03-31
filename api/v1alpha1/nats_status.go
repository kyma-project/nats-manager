package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (ns *NatsStatus) IsEqual(status NatsStatus) bool {
	thisWithoutCond := ns.DeepCopy()
	statusWithoutCond := status.DeepCopy()

	// remove conditions, so that we don't compare them
	thisWithoutCond.Conditions = []metav1.Condition{}
	statusWithoutCond.Conditions = []metav1.Condition{}

	return reflect.DeepEqual(thisWithoutCond, statusWithoutCond) &&
		ConditionsEquals(ns.Conditions, status.Conditions)
}

func (ns *NatsStatus) FindCondition(conditionType ConditionType) *metav1.Condition {
	for _, condition := range ns.Conditions {
		if string(conditionType) == condition.Type {
			return &condition
		}
	}
	return nil
}

func (ns *NatsStatus) UpdateConditionStatefulSet(status metav1.ConditionStatus, reason ConditionReason,
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

func (ns *NatsStatus) UpdateConditionAvailable(status metav1.ConditionStatus, reason ConditionReason,
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

func (ns *NatsStatus) SetStateReady() {
	ns.State = StateReady
	ns.UpdateConditionStatefulSet(metav1.ConditionTrue,
		ConditionReasonStatefulSetAvailable, "StatefulSet is ready!")
	ns.UpdateConditionAvailable(metav1.ConditionTrue, ConditionReasonDeployed, "NATS is deployed!")
}

func (ns *NatsStatus) SetStateProcessing() {
	ns.State = StateProcessing
}

func (ns *NatsStatus) SetStateStatefulSetWaiting() {
	ns.SetStateProcessing()
	ns.UpdateConditionStatefulSet(metav1.ConditionFalse,
		ConditionReasonStatefulSetPending, "Waiting")
	ns.UpdateConditionAvailable(metav1.ConditionFalse, ConditionReasonDeploying, "")
}

func (ns *NatsStatus) SetStateError() {
	ns.State = StateError
	ns.UpdateConditionStatefulSet(metav1.ConditionFalse, ConditionReasonSyncFailError, "")
	ns.UpdateConditionAvailable(metav1.ConditionFalse, ConditionReasonProcessingError, "")
}

func (ns *NatsStatus) SetStateDeleting() {
	ns.State = StateDeleting
}

func (ns *NatsStatus) Initialize() {
	ns.SetStateProcessing()
	ns.UpdateConditionStatefulSet(metav1.ConditionFalse, ConditionReasonProcessing, "")
	ns.UpdateConditionAvailable(metav1.ConditionFalse, ConditionReasonProcessing, "")
}
