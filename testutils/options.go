package testutils

import (
	"errors"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	kcorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	ErrFailedToConvertStatusToMap   = errors.New("failed to convert status to map[string]interface")
	ErrFailedToConvertMetadataToMap = errors.New("failed to convert metadata to map[string]interface")
	ErrFailedToConvertSpecToMap     = errors.New("failed to convert spec to map[string]interface")
)

type (
	Option     func(*unstructured.Unstructured) error
	NATSOption func(*nmapiv1alpha1.NATS) error
)

func WithNATSCRDefaults() NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Spec = nmapiv1alpha1.NATSSpec{
			Cluster: nmapiv1alpha1.Cluster{
				Size: 1,
			},
			JetStream: nmapiv1alpha1.JetStream{
				MemStorage: nmapiv1alpha1.MemStorage{
					Enabled: false,
				},
				FileStorage: nmapiv1alpha1.FileStorage{
					StorageClassName: "default",
					Size:             resource.MustParse("1Gi"),
				},
			},
		}
		return nil
	}
}

func WithName(name string) Option {
	return func(o *unstructured.Unstructured) error {
		if _, exists := o.Object["metadata"]; !exists {
			o.Object["metadata"] = make(map[string]interface{})
		}

		metadata, ok := o.Object["metadata"].(map[string]interface{})
		if !ok {
			return ErrFailedToConvertMetadataToMap
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
			return ErrFailedToConvertMetadataToMap
		}
		metadata["namespace"] = namespace
		return nil
	}
}

func WithLabels(labels map[string]string) Option {
	return func(o *unstructured.Unstructured) error {
		if _, exists := o.Object["metadata"]; !exists {
			o.Object["metadata"] = make(map[string]interface{})
		}

		metadata, ok := o.Object["metadata"].(map[string]interface{})
		if !ok {
			return ErrFailedToConvertMetadataToMap
		}

		metadata["labels"] = labels
		return nil
	}
}

func WithSpecNodeName(nodeName string) Option {
	return func(o *unstructured.Unstructured) error {
		if _, exists := o.Object["spec"]; !exists {
			o.Object["spec"] = make(map[string]interface{})
		}

		spec, ok := o.Object["spec"].(map[string]interface{})
		if !ok {
			return ErrFailedToConvertSpecToMap
		}
		spec["nodeName"] = nodeName
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
			return ErrFailedToConvertSpecToMap
		}
		spec["replicas"] = replicas
		return nil
	}
}

func WithStatefulSetStatusCurrentReplicas(replicas int) Option {
	return func(o *unstructured.Unstructured) error {
		if _, exists := o.Object["status"]; !exists {
			o.Object["status"] = make(map[string]interface{})
		}

		status, ok := o.Object["status"].(map[string]interface{})
		if !ok {
			return ErrFailedToConvertStatusToMap
		}
		status["currentReplicas"] = replicas
		return nil
	}
}

func WithStatefulSetStatusUpdatedReplicas(replicas int) Option {
	return func(o *unstructured.Unstructured) error {
		if _, exists := o.Object["status"]; !exists {
			o.Object["status"] = make(map[string]interface{})
		}

		status, ok := o.Object["status"].(map[string]interface{})
		if !ok {
			return ErrFailedToConvertStatusToMap
		}
		status["updatedReplicas"] = replicas
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
			return ErrFailedToConvertStatusToMap
		}
		status["readyReplicas"] = replicas
		return nil
	}
}

func WithNATSCRFinalizer(finalizer string) NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		controllerutil.AddFinalizer(nats, finalizer)
		return nil
	}
}

func WithNATSCRStatusInitialized() NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Status.Initialize()
		return nil
	}
}

func WithNATSStateReady() NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Status.State = nmapiv1alpha1.StateReady
		return nil
	}
}

func WithNATSStateWarning() NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Status.State = nmapiv1alpha1.StateWarning
		return nil
	}
}

func WithNATSStateProcessing() NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Status.State = nmapiv1alpha1.StateProcessing
		return nil
	}
}

func WithNATSStateError() NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Status.State = nmapiv1alpha1.StateError
		return nil
	}
}

func WithNATSCRName(name string) NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Name = name
		return nil
	}
}

func WithNATSCRNamespace(namespace string) NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Namespace = namespace
		return nil
	}
}

func WithNATSEmptySpec() NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Spec = nmapiv1alpha1.NATSSpec{}
		return nil
	}
}

func WithNATSClusterSize(size int) NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Spec.Cluster.Size = size
		return nil
	}
}

func WithNATSLogging(debug, trace bool) NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Spec.Logging.Debug = debug
		nats.Spec.Logging.Trace = trace
		return nil
	}
}

func WithNATSMemStorage(memStorage nmapiv1alpha1.MemStorage) NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Spec.JetStream.MemStorage = memStorage
		return nil
	}
}

func WithNATSFileStorage(fileStorage nmapiv1alpha1.FileStorage) NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Spec.JetStream.FileStorage = fileStorage
		return nil
	}
}

func WithNATSCluster(cluster nmapiv1alpha1.Cluster) NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Spec.Cluster = cluster
		return nil
	}
}

func WithNATSLabels(labels map[string]string) NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Spec.Labels = labels
		return nil
	}
}

func WithNATSAnnotations(annotations map[string]string) NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Spec.Annotations = annotations
		return nil
	}
}

func WithNATSResources(resources kcorev1.ResourceRequirements) NATSOption {
	return func(nats *nmapiv1alpha1.NATS) error {
		nats.Spec.Resources = resources
		return nil
	}
}
