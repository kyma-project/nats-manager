package manager

import (
	"errors"

	"github.com/kyma-project/nats-manager/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Option func(*unstructured.Unstructured) error

// WithOwnerReference sets the OwnerReferences of a k8s Object.
func WithOwnerReference(nats v1alpha1.NATS) Option {
	return func(o *unstructured.Unstructured) error {
		if _, exists := o.Object["metadata"]; !exists {
			o.Object["metadata"] = make(map[string]interface{})
		}

		metadata, ok := o.Object["metadata"].(map[string]interface{})
		if !ok {
			return errors.New("failed to convert metadata to map[string]interface")
		}

		metadata["ownerReferences"] = []map[string]interface{}{
			{
				"apiVersion":         nats.APIVersion,
				"kind":               nats.Kind,
				"name":               nats.Name,
				"uid":                nats.UID,
				"blockOwnerDeletion": true,
				"controller":         true,
			},
		}
		return nil
	}
}

// WithLabel sets label on the k8s Object.
func WithLabel(key, value string) Option {
	return func(o *unstructured.Unstructured) error {
		if _, exists := o.Object["metadata"]; !exists {
			o.Object["metadata"] = make(map[string]interface{})
		}

		metadata, ok := o.Object["metadata"].(map[string]interface{})
		if !ok {
			return errors.New("failed to convert metadata to map[string]interface")
		}

		if _, exists := metadata["labels"]; !exists {
			metadata["labels"] = make(map[string]interface{})
		}

		labels, ok := metadata["labels"].(map[string]interface{})
		if !ok {
			return errors.New("failed to convert labels to map[string]interface")
		}

		labels[key] = value
		return nil
	}
}
