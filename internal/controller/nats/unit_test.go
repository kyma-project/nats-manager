package nats

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/pkg/k8s"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	chartmocks "github.com/kyma-project/nats-manager/pkg/k8s/chart/mocks"
	k8smocks "github.com/kyma-project/nats-manager/pkg/k8s/mocks"
	"github.com/kyma-project/nats-manager/pkg/manager"
	managermocks "github.com/kyma-project/nats-manager/pkg/manager/mocks"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// MockedUnitTestEnvironment provides mocked resources for unit tests.
type MockedUnitTestEnvironment struct {
	Context       context.Context
	Client        client.Client
	kubeClient    k8s.Client
	chartRenderer chart.Renderer
	natsManager   manager.Manager
	Reconciler    *Reconciler
	Logger        *zap.SugaredLogger
	Recorder      *record.FakeRecorder
}

func NewMockedUnitTestEnvironment(t *testing.T, objs ...client.Object) *MockedUnitTestEnvironment {
	// setup context
	ctx := context.Background()

	// setup logger
	sugaredLogger, err := testutils.NewTestSugaredLogger()
	require.NoError(t, err)

	// setup fake client for k8s
	newScheme := runtime.NewScheme()
	err = natsv1alpha1.AddToScheme(newScheme)
	require.NoError(t, err)
	fakeClientBuilder := fake.NewClientBuilder().WithScheme(newScheme)
	fakeClient := fakeClientBuilder.WithObjects(objs...).Build()
	recorder := &record.FakeRecorder{}

	// setup custom mocks
	chartRenderer := chartmocks.NewRenderer(t)
	kubeClient := k8smocks.NewClient(t)
	natsManager := managermocks.NewManager(t)

	// setup reconciler
	reconciler := NewReconciler(
		fakeClient,
		kubeClient,
		chartRenderer,
		newScheme,
		sugaredLogger,
		recorder,
		natsManager,
	)

	return &MockedUnitTestEnvironment{
		Context:       ctx,
		Client:        fakeClient,
		kubeClient:    kubeClient,
		chartRenderer: chartRenderer,
		Reconciler:    reconciler,
		Logger:        sugaredLogger,
		Recorder:      recorder,
		natsManager:   natsManager,
	}
}

func (testEnv *MockedUnitTestEnvironment) GetNATS(name, namespace string) (natsv1alpha1.Nats, error) {
	var nats natsv1alpha1.Nats
	err := testEnv.Client.Get(testEnv.Context, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &nats)
	return nats, err
}
