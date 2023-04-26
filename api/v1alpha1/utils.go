package v1alpha1

import (
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConditionsEquals checks if two list of conditions are equal.
func ConditionsEquals(existing, expected []metav1.Condition) bool {
	// not equal if length is different
	if len(existing) != len(expected) {
		return false
	}

	// compile map of Conditions per ConditionType
	existingMap := make(map[ConditionType]metav1.Condition, len(existing))
	for _, value := range existing {
		existingMap[ConditionType(value.Type)] = value
	}

	for _, value := range expected {
		if !ConditionEquals(existingMap[ConditionType(value.Type)], value) {
			return false
		}
	}

	return true
}

// ConditionEquals checks if two conditions are equal.
func ConditionEquals(existing, expected metav1.Condition) bool {
	isTypeEqual := existing.Type == expected.Type
	isStatusEqual := existing.Status == expected.Status
	isReasonEqual := existing.Reason == expected.Reason
	isMessageEqual := existing.Message == expected.Message

	if !isStatusEqual || !isReasonEqual || !isMessageEqual || !isTypeEqual {
		return false
	}

	return true
}

func IsValidResourceQuantity(quantity *k8sresource.Quantity) bool {
	return quantity != nil && quantity.String() != "<nil>" && quantity.String() != ""
}
