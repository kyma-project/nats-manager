package fixtures

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
)

const (
	NamespaceName = "kyma-system"
	CRName        = "eventing-nats"
	STSName       = CRName
	ContainerName = "nats"
	pvcLabel      = "app.kubernetes.io/name=nats"
	secLabel      = "app.kubernetes.io/name=nats"
	podLabel      = "nats_cluster=eventing-nats"
	ClusterSize   = 3
	SecretName    = "eventing-nats-secret" //nolint:gosec // This is used for test purposes only.
)

func NATSCR() *natsv1alpha1.NATS {
	return testutils.NewNATSCR(
		testutils.WithNATSCRName(CRName),
		testutils.WithNATSCRNamespace(NamespaceName),
		testutils.WithNATSClusterSize(ClusterSize),
		testutils.WithNATSFileStorage(natsv1alpha1.FileStorage{
			StorageClassName: "default",
			Size:             resource.MustParse("1Gi"),
		}),
		testutils.WithNATSMemStorage(natsv1alpha1.MemStorage{
			Enabled: false,
			Size:    resource.MustParse("20Mi"),
		}),
		testutils.WithNATSResources(corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"cpu":    resource.MustParse("20m"),
				"memory": resource.MustParse("64Mi"),
			},
			Requests: corev1.ResourceList{
				"cpu":    resource.MustParse("5m"),
				"memory": resource.MustParse("16Mi"),
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
