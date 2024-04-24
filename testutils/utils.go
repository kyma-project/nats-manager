package testutils

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	kappsv1 "k8s.io/api/apps/v1"
	kcorev1 "k8s.io/api/core/v1"
	kapiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// for Random string generation.
	charset       = "abcdefghijklmnopqrstuvwxyz0123456789"
	randomNameLen = 5

	NameFormat                    = "name-%s"
	NamespaceFormat               = "namespace-%s"
	StatefulSetNameFormat         = "%s-nats"
	ConfigMapNameFormat           = "%s-nats-config"
	SecretNameFormat              = "%s-nats-secret" //nolint:gosec // only for test purpose
	ServiceNameFormat             = "%s-nats"
	PodDisruptionBudgetNameFormat = "%s-nats"
	DestinationRuleNameFormat     = "%s-nats"
)

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec,gochecknoglobals // used in tests

// GetRandString returns a random string of the given length.
func GetRandString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// GetRandK8sName returns a valid name for K8s objects.
func GetRandK8sName(length int) string {
	return "name-" + GetRandString(length)
}

func NewNamespace(name string) *kcorev1.Namespace {
	namespace := kcorev1.Namespace{
		TypeMeta: kmetav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name: name,
		},
	}
	return &namespace
}

func NewLogger() (*zap.Logger, error) {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.EncoderConfig.TimeKey = "timestamp"
	loggerConfig.Encoding = "json"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("Jan 02 15:04:05.000000000")

	return loggerConfig.Build()
}

func NewSugaredLogger() (*zap.SugaredLogger, error) {
	logger, err := NewLogger()
	if err != nil {
		return nil, err
	}
	return logger.Sugar(), nil
}

func NewNATSStatefulSetUnStruct(opts ...Option) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "StatefulSet",
			"apiVersion": "apps/v1",
			"metadata": map[string]interface{}{
				"name":      "test1",
				"namespace": "test1",
			},
		},
	}

	for _, opt := range opts {
		if err := opt(obj); err != nil {
			log.Fatal(err)
		}
	}
	return obj
}

func NewSecretUnStruct(opts ...Option) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "Secret",
			"apiVersion": "v1",
			"metadata": map[string]interface{}{
				"name":      "test1",
				"namespace": "test1",
			},
		},
	}

	for _, opt := range opts {
		if err := opt(obj); err != nil {
			log.Fatal(err)
		}
	}
	return obj
}

func NewNodeUnStruct(opts ...Option) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "Node",
			"apiVersion": "v1",
			"metadata": map[string]interface{}{
				"name": "test1",
			},
		},
	}

	for _, opt := range opts {
		if err := opt(obj); err != nil {
			log.Fatal(err)
		}
	}
	return obj
}

func NewPodUnStruct(opts ...Option) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "Pod",
			"apiVersion": "v1",
			"metadata": map[string]interface{}{
				"name":      "test1",
				"namespace": "test1",
			},
		},
	}

	for _, opt := range opts {
		if err := opt(obj); err != nil {
			log.Fatal(err)
		}
	}
	return obj
}

func NewSecret(opts ...Option) *kcorev1.Secret {
	sampleSecret := kcorev1.Secret{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(
		NewSecretUnStruct(opts...).UnstructuredContent(), &sampleSecret)
	if err != nil {
		log.Fatal(err)
	}
	return &sampleSecret
}

func NewNATSCR(opts ...NATSOption) *nmapiv1alpha1.NATS {
	name := fmt.Sprintf(NameFormat, GetRandString(randomNameLen))
	namespace := fmt.Sprintf(NamespaceFormat, GetRandString(randomNameLen))

	nats := &nmapiv1alpha1.NATS{
		// Name, UUID, Kind, APIVersion
		TypeMeta: kmetav1.TypeMeta{
			APIVersion: "v1alpha1",
			Kind:       "NATS",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       "1234-5678-1234-5678",
		},
	}

	for _, opt := range opts {
		if err := opt(nats); err != nil {
			log.Fatal(err)
		}
	}

	return nats
}

func NewDestinationRuleCRD() *kapiextv1.CustomResourceDefinition {
	result := &kapiextv1.CustomResourceDefinition{
		TypeMeta: kmetav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: kmetav1.ObjectMeta{
			Name: "destinationrules.networking.istio.io",
		},
		Spec: kapiextv1.CustomResourceDefinitionSpec{
			Names:                 kapiextv1.CustomResourceDefinitionNames{},
			Scope:                 "Namespaced",
			PreserveUnknownFields: false,
		},
	}

	return result
}

func GetStatefulSetName(nats nmapiv1alpha1.NATS) string {
	return fmt.Sprintf(StatefulSetNameFormat, nats.GetName())
}

func GetConfigMapName(nats nmapiv1alpha1.NATS) string {
	return fmt.Sprintf(ConfigMapNameFormat, nats.Name)
}

func GetSecretName(nats nmapiv1alpha1.NATS) string {
	return fmt.Sprintf(SecretNameFormat, nats.Name)
}

func GetServiceName(nats nmapiv1alpha1.NATS) string {
	return fmt.Sprintf(ServiceNameFormat, nats.Name)
}

func GetPodDisruptionBudgetName(nats nmapiv1alpha1.NATS) string {
	return fmt.Sprintf(PodDisruptionBudgetNameFormat, nats.Name)
}

func GetDestinationRuleName(nats nmapiv1alpha1.NATS) string {
	return fmt.Sprintf(DestinationRuleNameFormat, nats.Name)
}

func FindContainer(containers []kcorev1.Container, name string) *kcorev1.Container {
	for _, container := range containers {
		if container.Name == name {
			return &container
		}
	}
	return nil
}

func GetDestinationRuleGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1alpha3",
		Resource: "destinationrules",
	}
}

// NewPVC creates a new PVC object with the given name, namespace, and label.
func NewPVC(name, namespace string, labels map[string]string) *kcorev1.PersistentVolumeClaim {
	return &kcorev1.PersistentVolumeClaim{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: kcorev1.PersistentVolumeClaimSpec{
			AccessModes: []kcorev1.PersistentVolumeAccessMode{kcorev1.ReadWriteOnce},
			Resources: kcorev1.VolumeResourceRequirements{
				Requests: kcorev1.ResourceList{
					kcorev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
}

func NewStatefulSet(name, namespace string, labels map[string]string) *kappsv1.StatefulSet {
	return &kappsv1.StatefulSet{
		ObjectMeta: kmetav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}
