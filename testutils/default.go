package testutils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/kyma-project/nats-manager/api/v1alpha1"
)

func DefaultSpec() *v1alpha1.NATSSpec {
	quant20Mi := resource.MustParse("20Mi")
	quant1Gi := resource.MustParse("1Gi")

	return &v1alpha1.NATSSpec{
		Cluster: v1alpha1.Cluster{
			Size: 3,
		},
		JetStream: v1alpha1.JetStream{
			MemStorage: v1alpha1.MemStorage{
				Enabled: false,
				Size:    quant20Mi,
			},
			FileStorage: v1alpha1.FileStorage{
				StorageClassName: "default",
				Size:             &quant1Gi,
			},
		},
		Logging: v1alpha1.Logging{
			Debug: false,
			Trace: false,
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"cpu":    resource.MustParse("20m"),
				"memory": resource.MustParse("64Mi"),
			},
			Requests: corev1.ResourceList{
				"cpu":    resource.MustParse("5m"),
				"memory": resource.MustParse("16Mi"),
			},
		},
		Annotations: nil,
		Labels:      nil,
	}
}
