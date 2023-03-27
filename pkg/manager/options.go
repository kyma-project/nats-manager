package manager

import (
	"github.com/kyma-project/nats-manager/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Option func(*unstructured.Unstructured) error

// WithOwnerReference sets the OwnerReferences of a k8s Object.
func WithOwnerReference(nats v1alpha1.Nats) Option {
	return func(o *unstructured.Unstructured) error {
		if _, exists := o.Object["metadata"]; !exists {
			o.Object["metadata"] = make(map[string]interface{}, 32)
		}

		metadata := o.Object["metadata"].(map[string]interface{})
		metadata["ownerReferences"] = []map[string]interface{}{
			{
				"apiVersion": nats.APIVersion,
				"kind":       nats.Kind,
				//"controller": true,
				"name":               nats.Name,
				"uid":                nats.UID,
				"blockOwnerDeletion": true,
			},
		}
		return nil
	}
}
