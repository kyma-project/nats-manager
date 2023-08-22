package events

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

// reason is the reason of an event.
type reason string

const (
	// TODO: check all the descriptions
	// ReasonDeleting is used when an object is being deleted.
	ReasonDeleting reason = "Deleting"
	// ReasonProcessing is used when an object is being processed.
	ReasonProcessing reason = "Processing"
	// ReasonDeploying is used when an object is in the process of being deployed.
	ReasonDeploying reason = "Deploying"
	// ReasonDeployed is used when an object is successfully deployed.
	ReasonDeployed reason = "Deployed"
	// ReasonFailedProcessing is used when a processing step of an object fails.
	ReasonFailedProcessing reason = "FailedProcessing"
	// ReasonForbidden is used when a forbidden action is performed on an object.
	ReasonForbidden reason = "Forbidden"
	// ReasonFailedToSyncResources is used when syncing objects resources fail.
	ReasonFailedToSyncResources reason = "FailedToSyncResources"
	// ReasonDeletionError is used when an object could not be deleted due to an error.
	ReasonDeletionError reason = "DeletionError"
)

// Normal records a normal event for an API object.
func Normal(recorder record.EventRecorder, obj runtime.Object, rn reason, msgFmt string, args ...interface{}) {
	recorder.Eventf(obj, corev1.EventTypeNormal, string(rn), msgFmt, args...)
}

// Warn records a warning event for an API object.
func Warn(recorder record.EventRecorder, obj runtime.Object, rn reason, msgFmt string, args ...interface{}) {
	recorder.Eventf(obj, corev1.EventTypeWarning, string(rn), msgFmt, args...)
}
