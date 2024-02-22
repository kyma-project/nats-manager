package fixtures

import (
	kcorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
)

const (
	NamespaceName         = "kyma-system"
	ManagerDeploymentName = "nats-manager"
	CRName                = "eventing-nats"
	STSName               = CRName
	ContainerName         = "nats"
	pvcLabel              = "app.kubernetes.io/name=nats"
	podLabel              = "nats_cluster=eventing-nats"
	ClusterSize           = 3
	SecretName            = "eventing-nats-secret" //nolint:gosec // This is used for test purposes only.
	CMName                = "eventing-nats-config"
	FileStorageSize       = "1Gi"
	MemStorageSize        = "1Gi"
	True                  = "true"
	PriorityClassName     = "nats-manager-priority-class"
)

func NATSCR() *nmapiv1alpha1.NATS {
	return testutils.NewNATSCR(
		testutils.WithNATSCRName(CRName),
		testutils.WithNATSCRNamespace(NamespaceName),
		testutils.WithNATSClusterSize(ClusterSize),
		testutils.WithNATSFileStorage(nmapiv1alpha1.FileStorage{
			StorageClassName: "default",
			Size:             resource.MustParse(FileStorageSize),
		}),
		testutils.WithNATSMemStorage(nmapiv1alpha1.MemStorage{
			Enabled: true,
			Size:    resource.MustParse(MemStorageSize),
		}),
		testutils.WithNATSResources(kcorev1.ResourceRequirements{
			Limits: kcorev1.ResourceList{
				"cpu":    resource.MustParse("20m"),
				"memory": resource.MustParse("2Gi"),
			},
			Requests: kcorev1.ResourceList{
				"cpu":    resource.MustParse("5m"),
				"memory": resource.MustParse("2Gi"),
			},
		}),
		testutils.WithNATSLogging(
			true,
			true,
		),
	)
}

func Namespace() *kcorev1.Namespace {
	return &kcorev1.Namespace{
		ObjectMeta: kmetav1.ObjectMeta{
			Name: NamespaceName,
		},
	}
}

func PodListOpts() kmetav1.ListOptions {
	return kmetav1.ListOptions{LabelSelector: podLabel}
}

func PVCListOpts() kmetav1.ListOptions {
	return kmetav1.ListOptions{LabelSelector: pvcLabel}
}
