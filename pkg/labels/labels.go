package labels

import (
	"k8s.io/apimachinery/pkg/labels"
)

const (
	// Kubernetes label keys used by nats-manager.
	KeyComponent = "app.kubernetes.io/component"
	KeyCreatedBy = "app.kubernetes.io/created-by"
	KeyInstance  = "app.kubernetes.io/instance"
	KeyManagedBy = "app.kubernetes.io/managed-by"
	KeyName      = "app.kubernetes.io/name"
	KeyPartOf    = "app.kubernetes.io/part-of"
	KeyDashboard = "kyma-project.io/dashboard"

	// Kubernetes label values used by nats-manager.
	ValueNATS        = "nats"
	ValueNATSManager = "nats-manager"
)

// SelectorManagedByNATS returns a labelselector for managed-by ("app.kubernetes.io/managed-by") as used
// by the nats-manager.
func SelectorManagedByNATS() labels.Selector {
	return labels.SelectorFromSet(map[string]string{KeyManagedBy: ValueNATSManager})
}
