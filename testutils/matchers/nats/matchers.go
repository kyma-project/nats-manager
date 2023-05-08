package nats

import (
	"github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func HaveStatusReady() gomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *v1alpha1.NATS) string {
			return n.Status.State
		}, gomega.Equal(v1alpha1.StateReady))
}

func HaveCondition(condition metav1.Condition) gomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *v1alpha1.NATS) []metav1.Condition {
			return n.Status.Conditions
		},
		gomega.ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras|gstruct.IgnoreMissing, gstruct.Fields{
			"Type":    gomega.Equal(condition.Type),
			"Reason":  gomega.Equal(condition.Reason),
			"Message": gomega.Equal(condition.Message),
			"Status":  gomega.Equal(condition.Status),
		})))
}

func HaveReadyConditionStatefulSet() gomegatypes.GomegaMatcher {
	return HaveCondition(metav1.Condition{
		Type:    string(v1alpha1.ConditionStatefulSet),
		Status:  metav1.ConditionTrue,
		Reason:  string(v1alpha1.ConditionReasonStatefulSetAvailable),
		Message: "StatefulSet is ready",
	})
}

func HaveReadyConditionAvailable() gomegatypes.GomegaMatcher {
	return HaveCondition(metav1.Condition{
		Type:    string(v1alpha1.ConditionAvailable),
		Status:  metav1.ConditionTrue,
		Reason:  string(v1alpha1.ConditionReasonDeployed),
		Message: "NATS is deployed",
	})
}
