package manager

import (
	"fmt"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	MinClusterSize                   = 3
	IstioEnabledKey                  = "istio.enabled"
	RotatePasswordKey                = "auth.rotatePassword" //nolint:gosec // Its is not password.
	ClusterEnabledKey                = "cluster.enabled"
	ClusterSizeKey                   = "cluster.replicas"
	FileStorageClassKey              = "nats.jetstream.fileStorage.storageClassName"
	FileStorageSizeKey               = "global.jetstream.fileStorage.size"
	MemStorageEnabledKey             = "nats.jetstream.memStorage.enabled"
	MemStorageSizeKey                = "nats.jetstream.memStorage.size"
	DebugEnabledKey                  = "nats.logging.debug"
	TraceEnabledKey                  = "nats.logging.trace"
	CommonLabelsKey                  = "commonLabels"
	CommonAnnotationsKey             = "commonAnnotations"
	ResourceRequestsCPUKey           = "nats.resources.requests.cpu"
	ResourceRequestsMemKey           = "nats.resources.requests.memory"
	ResourceLimitsCPUKey             = "nats.resources.limits.cpu"
	ResourceLimitsMemKey             = "nats.resources.limits.memory"
	NatsImageUrl                     = "global.natsImageUrl"
	AlpineImageUrl                   = "global.alpineImageUrl"
	PrometheusNATSExporterImageUrl   = "global.prometheusNatsExporterImageUrl"
	NATSServerConfigReloaderImageUrl = "global.natsServerConfigReloaderImageUrl"

	CloudProviderAlicloud = "alicloud"

	defaultFileSizeAlicloud = "20Gi"
	defaultFileSizeDefault  = "1Gi"

	minFileSizeAlicloud = "20Gi"
)

// resolveFileStorageSize returns the effective file storage size given the spec value and cloud provider.
// If the spec value is empty it is defaulted; if it is set it is validated against the minimum.
// An error is returned when the spec value is below the minimum.
func resolveFileStorageSize(spec *nmapiv1alpha1.NATSSpec, cloudProvider string) (resource.Quantity, error) {
	isAlicloud := cloudProvider == CloudProviderAlicloud

	if spec.FileStorage.Size.IsZero() {
		// No value supplied – apply provider-specific default.
		if isAlicloud {
			return resource.MustParse(defaultFileSizeAlicloud), nil
		}
		return resource.MustParse(defaultFileSizeDefault), nil
	}

	// Value supplied – validate minimum only for alicloud.
	if isAlicloud {
		minimum := resource.MustParse(minFileSizeAlicloud)
		if spec.FileStorage.Size.Cmp(minimum) < 0 {
			return resource.Quantity{}, fmt.Errorf(
				"spec.jetStream.fileStorage.size %s is below the minimum required size of %s for alicloud",
				spec.FileStorage.Size.String(), minFileSizeAlicloud,
			)
		}
	}
	return spec.FileStorage.Size, nil
}

func (m NATSManager) GenerateOverrides(spec *nmapiv1alpha1.NATSSpec, istioEnabled bool,
	rotatePassword bool, cloudProvider string,
) (map[string]any, error) {
	overrides := map[string]any{
		IstioEnabledKey:   istioEnabled,
		RotatePasswordKey: rotatePassword,
	}

	// clustering
	overrides[ClusterSizeKey] = spec.Cluster.Size
	overrides[ClusterEnabledKey] = true
	if spec.Cluster.Size < MinClusterSize {
		overrides[ClusterEnabledKey] = false
	}

	// file storage – resolve size with defaulting and validation
	fileStorageSize, err := resolveFileStorageSize(spec, cloudProvider)
	if err != nil {
		return nil, err
	}
	overrides[FileStorageSizeKey] = fileStorageSize.String()
	if spec.FileStorage.StorageClassName != "" {
		overrides[FileStorageClassKey] = spec.FileStorage.StorageClassName
	}

	// memory storage
	overrides[MemStorageEnabledKey] = spec.MemStorage.Enabled
	if spec.MemStorage.Enabled {
		overrides[MemStorageSizeKey] = spec.MemStorage.Size.String()
	}

	// logging and tracing
	overrides[DebugEnabledKey] = spec.Debug
	overrides[TraceEnabledKey] = spec.Trace

	// resources
	if spec.Resources.Requests.Cpu() != nil {
		overrides[ResourceRequestsCPUKey] = spec.Resources.Requests.Cpu().String()
	}
	if spec.Resources.Requests.Memory() != nil {
		overrides[ResourceRequestsMemKey] = spec.Resources.Requests.Memory().String()
	}
	if spec.Resources.Limits.Cpu() != nil {
		overrides[ResourceLimitsCPUKey] = spec.Resources.Limits.Cpu().String()
	}
	if spec.Resources.Limits.Memory() != nil {
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

	if m.images.NATS != "" {
		overrides[NatsImageUrl] = m.images.NATS
	}
	if m.images.Alpine != "" {
		overrides[AlpineImageUrl] = m.images.Alpine
	}
	if m.images.PrometheusExporter != "" {
		overrides[PrometheusNATSExporterImageUrl] = m.images.PrometheusExporter
	}
	if m.images.NATSConfigReloader != "" {
		overrides[NATSServerConfigReloaderImageUrl] = m.images.NATSConfigReloader
	}

	return overrides, nil
}
