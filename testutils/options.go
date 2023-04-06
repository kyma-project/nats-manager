package testutils

import (
	"errors"

	"github.com/kyma-project/nats-manager/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Option func(*unstructured.Unstructured) error
type NATSOption func(*v1alpha1.NATS) error

func WithName(name string) Option {
	return func(o *unstructured.Unstructured) error {
		if _, exists := o.Object["metadata"]; !exists {
			o.Object["metadata"] = make(map[string]interface{})
		}

		metadata, ok := o.Object["metadata"].(map[string]interface{})
		if !ok {
			return errors.New("failed to convert metadata to map[string]interface")
		}
		metadata["name"] = name
		return nil
	}
}

func WithNamespace(namespace string) Option {
	return func(o *unstructured.Unstructured) error {
		if _, exists := o.Object["metadata"]; !exists {
			o.Object["metadata"] = make(map[string]interface{})
		}

		metadata, ok := o.Object["metadata"].(map[string]interface{})
		if !ok {
			return errors.New("failed to convert metadata to map[string]interface")
		}
		metadata["namespace"] = namespace
		return nil
	}
}

func WithSpecReplicas(replicas int) Option {
	return func(o *unstructured.Unstructured) error {
		if _, exists := o.Object["spec"]; !exists {
			o.Object["spec"] = make(map[string]interface{})
		}

		spec, ok := o.Object["spec"].(map[string]interface{})
		if !ok {
			return errors.New("failed to convert spec to map[string]interface")
		}
		spec["replicas"] = replicas
		return nil
	}
}

func WithStatefulSetStatusAvailableReplicas(replicas int) Option {
	return func(o *unstructured.Unstructured) error {
		if _, exists := o.Object["status"]; !exists {
			o.Object["status"] = make(map[string]interface{})
		}

		status, ok := o.Object["status"].(map[string]interface{})
		if !ok {
			return errors.New("failed to convert status to map[string]interface")
		}
		status["availableReplicas"] = replicas
		return nil
	}
}

func WithStatefulSetStatusReadyReplicas(replicas int) Option {
	return func(o *unstructured.Unstructured) error {
		if _, exists := o.Object["status"]; !exists {
			o.Object["status"] = make(map[string]interface{})
		}

		status, ok := o.Object["status"].(map[string]interface{})
		if !ok {
			return errors.New("failed to convert status to map[string]interface")
		}
		status["readyReplicas"] = replicas
		return nil
	}
}

func WithNATSCRFinalizer(finalizer string) NATSOption {
	return func(nats *v1alpha1.NATS) error {
		controllerutil.AddFinalizer(nats, finalizer)
		return nil
	}
}
