package testutils

import (
	"fmt"
	"math/rand"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kyma-project/nats-manager/api/v1alpha1"
)

const (
	// for Random string generation.
	charset       = "abcdefghijklmnopqrstuvwxyz0123456789"
	randomNameLen = 5

	NameFormat                = "name-%s"
	NamespaceFormat           = "namespace-%s"
	StatefulSetNameFormat     = "%s-nats"
	ConfigMapNameFormat       = "%s-nats-config"
	SecretNameFormat          = "%s-nats-secret" //nolint:gosec // only for test purpose
	ServiceNameFormat         = "%s-nats"
	DestinationRuleNameFormat = "%s-nats"
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
	return fmt.Sprintf("name-%s", GetRandString(length))
}

func NewNamespace(name string) *apiv1.Namespace {
	namespace := apiv1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
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
			panic(err)
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
			panic(err)
		}
	}
	return obj
}

func NewSecret(opts ...Option) *apiv1.Secret {
	sampleSecret := apiv1.Secret{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(
		NewSecretUnStruct(opts...).UnstructuredContent(), &sampleSecret)
	if err != nil {
		panic(err)
	}
	return &sampleSecret
}

func NewNATSCR(opts ...NATSOption) *v1alpha1.NATS {
	name := fmt.Sprintf(NameFormat, GetRandString(randomNameLen))
	namespace := fmt.Sprintf(NamespaceFormat, GetRandString(randomNameLen))

	nats := &v1alpha1.NATS{
		// Name, UUID, Kind, APIVersion
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1alpha1",
			Kind:       "NATS",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       "1234-5678-1234-5678",
		},
	}

	for _, opt := range opts {
		if err := opt(nats); err != nil {
			panic(err)
		}
	}

	return nats
}

func NewDestinationRuleCRD() *apiextensionsv1.CustomResourceDefinition {
	result := &apiextensionsv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "destinationrules.networking.istio.io",
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Names:                 apiextensionsv1.CustomResourceDefinitionNames{},
			Scope:                 "Namespaced",
			PreserveUnknownFields: false,
		},
	}

	return result
}

func GetStatefulSetName(nats v1alpha1.NATS) string {
	return fmt.Sprintf(StatefulSetNameFormat, nats.GetName())
}

func GetConfigMapName(nats v1alpha1.NATS) string {
	return fmt.Sprintf(ConfigMapNameFormat, nats.Name)
}

func GetSecretName(nats v1alpha1.NATS) string {
	return fmt.Sprintf(SecretNameFormat, nats.Name)
}

func GetServiceName(nats v1alpha1.NATS) string {
	return fmt.Sprintf(ServiceNameFormat, nats.Name)
}

func GetDestinationRuleName(nats v1alpha1.NATS) string {
	return fmt.Sprintf(DestinationRuleNameFormat, nats.Name)
}

func FindContainer(containers []apiv1.Container, name string) *apiv1.Container {
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
func NewPVC(name, namespace string, labels map[string]string) *apiv1.PersistentVolumeClaim {
	return &apiv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: apiv1.PersistentVolumeClaimSpec{
			AccessModes: []apiv1.PersistentVolumeAccessMode{apiv1.ReadWriteOnce},
			Resources: apiv1.ResourceRequirements{
				Requests: apiv1.ResourceList{
					apiv1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
}

func NewStatefulSet(name, namespace string, labels map[string]string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}
