package testutils

import (
	"github.com/kyma-project/nats-manager/api/v1alpha1"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

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

func NewNATSCR(opts ...NATSOption) *v1alpha1.NATS {
	nats := &v1alpha1.NATS{
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
