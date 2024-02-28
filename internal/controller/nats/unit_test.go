package nats

import (
	"context"
	"testing"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	nmctrlmocks "github.com/kyma-project/nats-manager/internal/controller/nats/mocks"
	nmkchartmocks "github.com/kyma-project/nats-manager/pkg/k8s/chart/mocks"
	nmkmocks "github.com/kyma-project/nats-manager/pkg/k8s/mocks"
	nmmgrmocks "github.com/kyma-project/nats-manager/pkg/manager/mocks"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	kcorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// MockedUnitTestEnvironment provides mocked resources for unit tests.
type MockedUnitTestEnvironment struct {
	Context       context.Context
	Client        client.Client
	kubeClient    *nmkmocks.Client
	chartRenderer *nmkchartmocks.Renderer
	natsManager   *nmmgrmocks.Manager
	ctrlManager   *nmctrlmocks.Manager
	Reconciler    *Reconciler
	controller    *nmctrlmocks.Controller
	Logger        *zap.SugaredLogger
	Recorder      *record.FakeRecorder
}

func NewMockedUnitTestEnvironment(t *testing.T, objs ...client.Object) *MockedUnitTestEnvironment {
	// setup context
	ctx := context.Background()

	// setup logger
	sugaredLogger, err := testutils.NewSugaredLogger()
	require.NoError(t, err)

	// setup fake client for k8s
	newScheme := runtime.NewScheme()
	err = nmapiv1alpha1.AddToScheme(newScheme)
	require.NoError(t, err)
	err = kcorev1.AddToScheme(newScheme)
	require.NoError(t, err)
	fakeClientBuilder := fake.NewClientBuilder().WithScheme(newScheme)
	fakeClient := fakeClientBuilder.WithObjects(objs...).WithStatusSubresource(objs...).Build()
	recorder := record.NewFakeRecorder(3)

	// setup custom mocks
	chartRenderer := new(nmkchartmocks.Renderer)
	kubeClient := new(nmkmocks.Client)
	natsManager := new(nmmgrmocks.Manager)
	mockController := new(nmctrlmocks.Controller)
	mockManager := new(nmctrlmocks.Manager)

	// setup reconciler
	reconciler := NewReconciler(
		fakeClient,
		kubeClient,
		chartRenderer,
		newScheme,
		sugaredLogger,
		recorder,
		natsManager,
		nil,
	)
	reconciler.controller = mockController
	reconciler.ctrlManager = mockManager

	return &MockedUnitTestEnvironment{
		Context:       ctx,
		Client:        fakeClient,
		kubeClient:    kubeClient,
		chartRenderer: chartRenderer,
		Reconciler:    reconciler,
		controller:    mockController,
		Logger:        sugaredLogger,
		Recorder:      recorder,
		natsManager:   natsManager,
		ctrlManager:   mockManager,
	}
}

func (testEnv *MockedUnitTestEnvironment) GetNATS(name, namespace string) (nmapiv1alpha1.NATS, error) {
	var nats nmapiv1alpha1.NATS
	err := testEnv.Client.Get(testEnv.Context, ktypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &nats)
	return nats, err
}

func (testEnv *MockedUnitTestEnvironment) GetK8sEvents() []string {
	eventList := make([]string, 0, cap(testEnv.Recorder.Events))
	close(testEnv.Recorder.Events)

	for event := range testEnv.Recorder.Events {
		eventList = append(eventList, event)
	}
	return eventList
}
