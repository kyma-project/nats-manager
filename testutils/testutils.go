package testutils

import (
	"github.com/kyma-project/nats-manager/api/v1alpha1"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func NewTestLogger() (*zap.Logger, error) {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.EncoderConfig.TimeKey = "timestamp"
	loggerConfig.Encoding = "json"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("Jan 02 15:04:05.000000000")

	return loggerConfig.Build()
}

func NewTestSugaredLogger() (*zap.SugaredLogger, error) {
	logger, err := NewTestLogger()
	if err != nil {
		return nil, err
	}
	return logger.Sugar(), nil
}

func NewSampleNATSStatefulSetUnStruct(opts ...SampleOption) *unstructured.Unstructured {
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

func NewSampleSecretUnStruct(opts ...SampleOption) *unstructured.Unstructured {
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

func NewSampleSecret(opts ...SampleOption) *apiv1.Secret {
	sampleSecret := apiv1.Secret{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(
		NewSampleSecretUnStruct(opts...).UnstructuredContent(), &sampleSecret)
	if err != nil {
		panic(err)
	}
	return &sampleSecret
}

func NewSampleNATSCR(opts ...SampleNATSOption) *v1alpha1.Nats {
	nats := &v1alpha1.Nats{
		// Name, UUID, Kind, APIVersion
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1alpha1",
			Kind:       "Nats",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-object1",
			Namespace: "test-ns1",
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
