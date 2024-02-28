package events

import (
	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	kcorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

// Normal records a normal event for an API object.
func Normal(recorder record.EventRecorder, obj runtime.Object, rn nmapiv1alpha1.ConditionReason, msgFmt string,
	args ...interface{},
) {
	recorder.Eventf(obj, kcorev1.EventTypeNormal, string(rn), msgFmt, args...)
}

// Warn records a warning event for an API object.
func Warn(recorder record.EventRecorder, obj runtime.Object, rn nmapiv1alpha1.ConditionReason, msgFmt string,
	args ...interface{},
) {
	recorder.Eventf(obj, kcorev1.EventTypeWarning, string(rn), msgFmt, args...)
}
