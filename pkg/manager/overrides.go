package manager

import natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"

const (
	MinClusterSize         = 3
	IstioEnabledKey        = "istio.enabled"
	RotatePasswordKey      = "auth.rotatePassword" //nolint:gosec // Its is not password.
	ClusterEnabledKey      = "cluster.enabled"
	ClusterSizeKey         = "cluster.replicas"
	FileStorageClassKey    = "nats.jetstream.fileStorage.storageClassName"
	FileStorageSizeKey     = "global.jetstream.fileStorage.size"
	MemStorageEnabledKey   = "nats.jetstream.memStorage.enabled"
	MemStorageSizeKey      = "nats.jetstream.memStorage.size"
	DebugEnabledKey        = "nats.logging.debug"
	TraceEnabledKey        = "nats.logging.trace"
	CommonLabelsKey        = "commonLabels"
	CommonAnnotationsKey   = "commonAnnotations"
	ResourceRequestsCPUKey = "nats.resources.requests.cpu"
	ResourceRequestsMemKey = "nats.resources.requests.memory"
	ResourceLimitsCPUKey   = "nats.resources.limits.cpu"
	ResourceLimitsMemKey   = "nats.resources.limits.memory"
)

func (m NATSManager) GenerateOverrides(spec *natsv1alpha1.NATSSpec, istioEnabled bool,
	rotatePassword bool) map[string]interface{} {
	overrides := map[string]interface{}{
		IstioEnabledKey:   istioEnabled,
		RotatePasswordKey: rotatePassword,
	}

	// clustering
	overrides[ClusterSizeKey] = spec.Cluster.Size
	overrides[ClusterEnabledKey] = true
	if spec.Cluster.Size < MinClusterSize {
		overrides[ClusterEnabledKey] = false
	}

	// file storage
	if natsv1alpha1.IsValidResourceQuantity(&spec.FileStorage.Size) {
		overrides[FileStorageSizeKey] = spec.FileStorage.Size.String()
	}
	if spec.FileStorage.StorageClassName != "" {
		overrides[FileStorageClassKey] = spec.FileStorage.StorageClassName
	}

	// memory storage
	overrides[MemStorageEnabledKey] = spec.MemStorage.Enable
	if spec.MemStorage.Enable && natsv1alpha1.IsValidResourceQuantity(&spec.MemStorage.Size) {
		overrides[MemStorageSizeKey] = spec.MemStorage.Size.String()
	}

	// logging and tracing
	overrides[DebugEnabledKey] = spec.Debug
	overrides[TraceEnabledKey] = spec.Trace

	// resources
	if natsv1alpha1.IsValidResourceQuantity(spec.Resources.Requests.Cpu()) {
		overrides[ResourceRequestsCPUKey] = spec.Resources.Requests.Cpu().String()
	}
	if natsv1alpha1.IsValidResourceQuantity(spec.Resources.Requests.Memory()) {
		overrides[ResourceRequestsMemKey] = spec.Resources.Requests.Memory().String()
	}
	if natsv1alpha1.IsValidResourceQuantity(spec.Resources.Limits.Cpu()) {
		overrides[ResourceLimitsCPUKey] = spec.Resources.Limits.Cpu().String()
	}
	if natsv1alpha1.IsValidResourceQuantity(spec.Resources.Limits.Memory()) {
		overrides[ResourceLimitsMemKey] = spec.Resources.Limits.Memory().String()
	}

	// common labels to all the deployed resources.
	if len(spec.Labels) > 0 {
		overrides[CommonLabelsKey] = spec.Labels
	}

	// common annotations to all the deployed resources.
	if len(spec.Annotations) > 0 {
		overrides[CommonAnnotationsKey] = spec.Annotations
	}

	return overrides
}
