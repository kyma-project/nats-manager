package nats

import (
	"reflect"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	onsigomegatypes "github.com/onsi/gomega/types"
	kcorev1 "k8s.io/api/core/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func HaveSpecJetsStreamMemStorage(ms nmapiv1alpha1.MemStorage) onsigomegatypes.GomegaMatcher {
	return gomega.And(
		gomega.WithTransform(
			func(n *nmapiv1alpha1.NATS) bool {
				return n.Spec.JetStream.MemStorage.Enabled
			}, gomega.Equal(ms.Enabled)),
		gomega.WithTransform(
			func(n *nmapiv1alpha1.NATS) bool {
				return n.Spec.JetStream.MemStorage.Size.Equal(ms.Size)
			}, gomega.BeTrue()),
	)
}

func HaveSpecJetStreamFileStorage(fs nmapiv1alpha1.FileStorage) onsigomegatypes.GomegaMatcher {
	return gomega.And(
		gomega.WithTransform(
			func(n *nmapiv1alpha1.NATS) string {
				return n.Spec.JetStream.FileStorage.StorageClassName
			}, gomega.Equal(fs.StorageClassName)),
		gomega.WithTransform(
			func(n *nmapiv1alpha1.NATS) bool {
				return n.Spec.JetStream.FileStorage.Size.Equal(fs.Size)
			}, gomega.BeTrue()),
	)
}

func HaveSpecCluster(cluster nmapiv1alpha1.Cluster) onsigomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *nmapiv1alpha1.NATS) bool {
			return reflect.DeepEqual(n.Spec.Cluster, cluster)
		}, gomega.BeTrue())
}

func HaveSpecClusterSize(size int) onsigomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *nmapiv1alpha1.NATS) int {
			return n.Spec.Cluster.Size
		}, gomega.Equal(size))
}

func HaveSpecLogging(logging nmapiv1alpha1.Logging) onsigomegatypes.GomegaMatcher {
	return gomega.And(
		gomega.WithTransform(
			func(n *nmapiv1alpha1.NATS) bool {
				return n.Spec.Logging.Debug
			}, gomega.Equal(logging.Debug)),
		gomega.WithTransform(
			func(n *nmapiv1alpha1.NATS) bool {
				return n.Spec.Logging.Trace
			}, gomega.Equal(logging.Trace)),
	)
}

func HaveSpecLoggingDebug(enabled bool) onsigomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *nmapiv1alpha1.NATS) bool {
			return n.Spec.Logging.Debug
		}, gomega.Equal(enabled))
}

func HaveSpecLoggingTrace(enabled bool) onsigomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *nmapiv1alpha1.NATS) bool {
			return n.Spec.Logging.Trace
		}, gomega.Equal(enabled))
}

func HaveSpecResources(res kcorev1.ResourceRequirements) onsigomegatypes.GomegaMatcher {
	return gomega.And(
		gomega.WithTransform(
			func(n *nmapiv1alpha1.NATS) bool {
				return n.Spec.Resources.Requests.Storage().Equal(*res.Requests.Storage())
			}, gomega.BeTrue()),
		gomega.WithTransform(
			func(n *nmapiv1alpha1.NATS) bool {
				return n.Spec.Resources.Requests.Cpu().Equal(*res.Requests.Cpu())
			}, gomega.BeTrue()),
		gomega.WithTransform(
			func(n *nmapiv1alpha1.NATS) bool {
				return n.Spec.Resources.Limits.Storage().Equal(*res.Requests.Storage())
			}, gomega.BeTrue()),
		gomega.WithTransform(
			func(n *nmapiv1alpha1.NATS) bool {
				return n.Spec.Resources.Requests.Cpu().Equal(*res.Requests.Cpu())
			}, gomega.BeTrue()),
	)
}

func HaveStatusReady() onsigomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *nmapiv1alpha1.NATS) string {
			return n.Status.State
		}, gomega.Equal(nmapiv1alpha1.StateReady))
}

func HaveStatusProcessing() onsigomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *nmapiv1alpha1.NATS) string {
			return n.Status.State
		}, gomega.Equal(nmapiv1alpha1.StateProcessing))
}

func HaveStatusError() onsigomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *nmapiv1alpha1.NATS) string {
			return n.Status.State
		}, gomega.Equal(nmapiv1alpha1.StateError))
}

func HaveCondition(condition kmetav1.Condition) onsigomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(n *nmapiv1alpha1.NATS) []kmetav1.Condition {
			return n.Status.Conditions
		},
		gomega.ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras|gstruct.IgnoreMissing, gstruct.Fields{
			"Type":    gomega.Equal(condition.Type),
			"Reason":  gomega.Equal(condition.Reason),
			"Message": gomega.Equal(condition.Message),
			"Status":  gomega.Equal(condition.Status),
		})))
}

func HaveEvent(event kcorev1.Event) onsigomegatypes.GomegaMatcher {
	return gomega.WithTransform(
		func(l kcorev1.EventList) []kcorev1.Event {
			return l.Items
		}, gomega.ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras|gstruct.IgnoreMissing, gstruct.Fields{
			"Reason":  gomega.Equal(event.Reason),
			"Message": gomega.Equal(event.Message),
			"Type":    gomega.Equal(event.Type),
		})))
}

func HaveDeployedEvent() onsigomegatypes.GomegaMatcher {
	return HaveEvent(kcorev1.Event{
		Reason:  string(nmapiv1alpha1.ConditionReasonDeployed),
		Message: "StatefulSet is ready and NATS is deployed.",
		Type:    kcorev1.EventTypeNormal,
	})
}

func HaveDeployingEvent() onsigomegatypes.GomegaMatcher {
	return HaveEvent(kcorev1.Event{
		Reason:  string(nmapiv1alpha1.ConditionReasonDeploying),
		Message: "NATS is being deployed, waiting for StatefulSet to get ready.",
		Type:    kcorev1.EventTypeNormal,
	})
}

func HaveProcessingEvent() onsigomegatypes.GomegaMatcher {
	return HaveEvent(kcorev1.Event{
		Reason:  string(nmapiv1alpha1.ConditionReasonProcessing),
		Message: "Initializing NATS resource.",
		Type:    kcorev1.EventTypeNormal,
	})
}

func HaveReadyConditionStatefulSet() onsigomegatypes.GomegaMatcher {
	return HaveCondition(kmetav1.Condition{
		Type:    string(nmapiv1alpha1.ConditionStatefulSet),
		Status:  kmetav1.ConditionTrue,
		Reason:  string(nmapiv1alpha1.ConditionReasonStatefulSetAvailable),
		Message: "StatefulSet is ready",
	})
}

func HavePendingConditionStatefulSet() onsigomegatypes.GomegaMatcher {
	return HaveCondition(kmetav1.Condition{
		Type:    string(nmapiv1alpha1.ConditionStatefulSet),
		Status:  kmetav1.ConditionFalse,
		Reason:  string(nmapiv1alpha1.ConditionReasonStatefulSetPending),
		Message: "",
	})
}

func HaveForbiddenConditionStatefulSet() onsigomegatypes.GomegaMatcher {
	return HaveCondition(kmetav1.Condition{
		Type:    string(nmapiv1alpha1.ConditionStatefulSet),
		Status:  kmetav1.ConditionFalse,
		Reason:  string(nmapiv1alpha1.ConditionReasonForbidden),
		Message: "",
	})
}

func HaveReadyConditionAvailable() onsigomegatypes.GomegaMatcher {
	return HaveCondition(kmetav1.Condition{
		Type:    string(nmapiv1alpha1.ConditionAvailable),
		Status:  kmetav1.ConditionTrue,
		Reason:  string(nmapiv1alpha1.ConditionReasonDeployed),
		Message: "NATS is deployed",
	})
}

func HaveDeployingConditionAvailable() onsigomegatypes.GomegaMatcher {
	return HaveCondition(kmetav1.Condition{
		Type:    string(nmapiv1alpha1.ConditionAvailable),
		Status:  kmetav1.ConditionFalse,
		Reason:  string(nmapiv1alpha1.ConditionReasonDeploying),
		Message: "",
	})
}

func HaveForbiddenConditionAvailableWithMsg(msg string) onsigomegatypes.GomegaMatcher {
	return HaveCondition(kmetav1.Condition{
		Type:    string(nmapiv1alpha1.ConditionAvailable),
		Status:  kmetav1.ConditionFalse,
		Reason:  string(nmapiv1alpha1.ConditionReasonForbidden),
		Message: msg,
	})
}
