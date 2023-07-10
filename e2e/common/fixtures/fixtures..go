package fixtures

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
)

const (
	NamespaceName   = "kyma-system"
	CRName          = "eventing-nats"
	STSName         = CRName
	ContainerName   = "nats"
	pvcLabel        = "app.kubernetes.io/name=nats"
	secLabel        = "app.kubernetes.io/name=nats"
	podLabel        = "nats_cluster=eventing-nats"
	ClusterSize     = 3
	SecretName      = "eventing-nats-secret" //nolint:gosec // This is used for test purposes only.
	CMName          = "eventing-nats-config"
	FileStorageSize = "1Gi"
	MemStorageSize  = "500Mi"
	True            = "true"
)

func NATSCR() *natsv1alpha1.NATS {
	return testutils.NewNATSCR(
		testutils.WithNATSCRName(CRName),
		testutils.WithNATSCRNamespace(NamespaceName),
		testutils.WithNATSClusterSize(ClusterSize),
		testutils.WithNATSFileStorage(natsv1alpha1.FileStorage{
			StorageClassName: "default",
			Size:             resource.MustParse(FileStorageSize),
		}),
		testutils.WithNATSMemStorage(natsv1alpha1.MemStorage{
			Enabled: true,
			Size:    resource.MustParse(MemStorageSize),
		}),
		testutils.WithNATSResources(corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"cpu":    resource.MustParse("20m"),
				"memory": resource.MustParse("64Mi"),
			},
			Requests: corev1.ResourceList{
				"cpu":    resource.MustParse("5m"),
				"memory": resource.MustParse("64Mi"),
			},
		}),
		testutils.WithNATSLogging(
			true,
			true,
		),
	)
}

func Namespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: NamespaceName,
		},
	}
}

func PodListOpts() metav1.ListOptions {
	return metav1.ListOptions{LabelSelector: podLabel}
}

func PVCListOpts() metav1.ListOptions {
	return metav1.ListOptions{LabelSelector: pvcLabel}
}

func SecretListOpts() metav1.ListOptions {
	return metav1.ListOptions{LabelSelector: secLabel}
}
