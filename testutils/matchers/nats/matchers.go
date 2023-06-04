package nats

import (
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/nats-manager/api/v1alpha1"
)

func HaveSpecClusterSize(size int) gomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *v1alpha1.NATS) int {
			return n.Spec.Cluster.Size
		}, gomega.Equal(size))
}

func HaveSpecLoggingDebug(enabled bool) gomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *v1alpha1.NATS) bool {
			return n.Spec.Logging.Debug
		}, gomega.Equal(enabled))
}

func HaveSpecLoggingTrace(enabled bool) gomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *v1alpha1.NATS) bool {
			return n.Spec.Logging.Trace
		}, gomega.Equal(enabled))
}

func HaveSpecResources(res corev1.ResourceRequirements) gomegatypes.GomegaMatcher {
	return gomega.And(
		gomega.WithTransform(
			func(n *v1alpha1.NATS) bool {
				return n.Spec.Resources.Requests.Storage().Equal(*res.Requests.Storage())
			}, gomega.BeTrue()),
		gomega.WithTransform(
			func(n *v1alpha1.NATS) bool {
				return n.Spec.Resources.Requests.Cpu().Equal(*res.Requests.Cpu())
			}, gomega.BeTrue()),
		gomega.WithTransform(
			func(n *v1alpha1.NATS) bool {
				return n.Spec.Resources.Limits.Storage().Equal(*res.Requests.Storage())
			}, gomega.BeTrue()),
		gomega.WithTransform(
			func(n *v1alpha1.NATS) bool {
				return n.Spec.Resources.Requests.Cpu().Equal(*res.Requests.Cpu())
			}, gomega.BeTrue()),
	)
}

func HaveStatusReady() gomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *v1alpha1.NATS) string {
			return n.Status.State
		}, gomega.Equal(v1alpha1.StateReady))
}

func HaveStatusProcessing() gomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *v1alpha1.NATS) string {
			return n.Status.State
		}, gomega.Equal(v1alpha1.StateProcessing))
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

func HavePendingConditionStatefulSet() gomegatypes.GomegaMatcher {
	return HaveCondition(metav1.Condition{
		Type:    string(v1alpha1.ConditionStatefulSet),
		Status:  metav1.ConditionFalse,
		Reason:  string(v1alpha1.ConditionReasonStatefulSetPending),
		Message: "",
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

func HaveDeployingConditionAvailable() gomegatypes.GomegaMatcher {
	return HaveCondition(metav1.Condition{
		Type:    string(v1alpha1.ConditionAvailable),
		Status:  metav1.ConditionFalse,
		Reason:  string(v1alpha1.ConditionReasonDeploying),
		Message: "",
	})
}
