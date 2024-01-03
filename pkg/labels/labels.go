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

// SelectorInstanceNATS returns a labelselector for instance ("app.kubernetes.io/instance") as used
// by the nats-manager.
func SelectorInstanceNATS() labels.Selector {
	return labels.SelectorFromSet(map[string]string{KeyInstance: ValueNATSManager})
}

// SelectorCreatedByNATS returns a labelselector for created-by ("app.kubernetes.io/created-by") as used
// by the nats-manager.
func SelectorCreatedByNATS() labels.Selector {
	return labels.SelectorFromSet(map[string]string{KeyCreatedBy: ValueNATSManager})
}

// SelectorCreatedByNATS returns a labelselector for created-by ("app.kubernetes.io/created-by") as used
// by the nats-manager.
func SelectorManagedByNATS() labels.Selector {
	return labels.SelectorFromSet(map[string]string{KeyCreatedBy: ValueNATSManager})
}
