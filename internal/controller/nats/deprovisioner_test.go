package nats

import (
	"errors"
	"testing"

	"github.com/kyma-project/nats-manager/pkg/nats"

	"github.com/kyma-project/nats-manager/internal/controller/nats/mocks"
	natsmanager "github.com/kyma-project/nats-manager/pkg/manager"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func Test_handleNATSDeletion(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name                   string
		givenWithNATSCreated   bool
		natsCrWithoutFinalizer bool
		mockNatsClientFunc     func() nats.Client
		wantCondition          *metav1.Condition
		wantNATSStatusState    string
		wantFinalizerExists    bool
		wantResult             ctrl.Result
	}{
		{
			name:                   "should not do anything if finalizer is not set",
			givenWithNATSCreated:   false,
			natsCrWithoutFinalizer: true,
			wantNATSStatusState:    natsv1alpha1.StateReady,
			wantResult:             ctrl.Result{},
		},
		{
			name:                 "should delete resources if connection to NATS server is not established",
			givenWithNATSCreated: true,
			mockNatsClientFunc: func() nats.Client {
				natsClient := new(mocks.Client)
				natsClient.On("Init").Return(errors.New("connection cannot be established"))
				natsClient.On("Close").Return()
				return natsClient
			},
			wantNATSStatusState: natsv1alpha1.StateDeleting,
			wantResult:          ctrl.Result{},
		},
		{
			name:                 "should delete resources if natsClients StreamExists returns error",
			givenWithNATSCreated: true,
			wantNATSStatusState:  natsv1alpha1.StateDeleting,
			mockNatsClientFunc: func() nats.Client {
				natsClient := new(mocks.Client)
				natsClient.On("Init").Return(nil)
				natsClient.On("StreamExists").Return(false, errors.New("unexpected error"))
				natsClient.On("Close").Return()
				return natsClient
			},
			wantResult: ctrl.Result{},
		},
		{
			name:                 "should add deleted condition with error when stream exists",
			givenWithNATSCreated: true,
			wantNATSStatusState:  natsv1alpha1.StateDeleting,
			wantCondition: &metav1.Condition{
				Type:               string(natsv1alpha1.ConditionDeleted),
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             string(natsv1alpha1.ConditionReasonDeletionError),
				Message:            StreamExistsErrorMsg,
			},
			mockNatsClientFunc: func() nats.Client {
				natsClient := new(mocks.Client)
				natsClient.On("Init").Return(nil)
				natsClient.On("StreamExists").Return(true, nil)
				return natsClient
			},
			wantFinalizerExists: true,
			wantResult:          ctrl.Result{Requeue: true},
		},
		{
			name:                 "should delete resources if stream does not exist",
			givenWithNATSCreated: true,
			mockNatsClientFunc: func() nats.Client {
				natsClient := new(mocks.Client)
				natsClient.On("Init").Return(nil)
				natsClient.On("StreamExists").Return(false, nil)
				natsClient.On("Close").Return()
				return natsClient
			},
			wantNATSStatusState: natsv1alpha1.StateDeleting,
			wantResult:          ctrl.Result{},
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			var givenNats *natsv1alpha1.NATS
			if tc.natsCrWithoutFinalizer {
				givenNats = testutils.NewNATSCR(
					testutils.WithNATSCRStatusInitialized(),
					testutils.WithNATSStateReady(),
				)
			} else {
				givenNats = testutils.NewNATSCR(
					testutils.WithNATSCRStatusInitialized(),
					testutils.WithNATSStateReady(),
					testutils.WithNATSCRFinalizer(NATSFinalizerName),
				)
			}
			var objs []client.Object
			if tc.givenWithNATSCreated {
				objs = append(objs, givenNats)
			}

			testEnv := NewMockedUnitTestEnvironment(t, objs...)
			reconciler := testEnv.Reconciler

			nats := givenNats.DeepCopy()

			// define mocks behaviour
			testEnv.kubeClient.On("DestinationRuleCRDExists",
				mock.Anything).Return(false, nil)
			testEnv.kubeClient.On("GetSecret",
				mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
			testEnv.kubeClient.On("DeletePVCsWithLabel",
				mock.Anything, mock.Anything, mock.Anything).Return(nil)
			testEnv.kubeClient.On("GetStatefulSet",
				mock.Anything, mock.Anything, mock.Anything).Return(testutils.NewStatefulSet(
				"test-nats", "test-namespace", map[string]string{"app.kubernetes.io/instance": "test-nats"}), nil)

			natsResources := &chart.ManifestResources{
				Items: []*unstructured.Unstructured{
					testutils.NewNATSStatefulSetUnStruct(),
				},
			}
			testEnv.natsManager.On("GenerateNATSResources",
				mock.Anything, mock.Anything, mock.Anything).Return(natsResources, nil)
			testEnv.natsManager.On("GenerateOverrides",
				mock.Anything, mock.Anything, mock.Anything).Return(
				map[string]interface{}{
					natsmanager.IstioEnabledKey:   false,
					natsmanager.RotatePasswordKey: true, // do not recreate secret if it exists
				},
			)

			if tc.mockNatsClientFunc != nil {
				reconciler.natsClients[nats.Namespace+"/"+nats.Name] = tc.mockNatsClientFunc()
			}

			// when
			result, err := reconciler.handleNATSDeletion(testEnv.Context, nats, testEnv.Logger)

			// then
			require.NoError(t, err)
			require.Equal(t, tc.wantNATSStatusState, nats.Status.State)
			require.Equal(t, tc.wantResult, result)

			if tc.wantCondition != nil {
				gotCondition := nats.Status.FindCondition(natsv1alpha1.ConditionType(tc.wantCondition.Type))
				require.NotNil(t, gotCondition)
				require.True(t, natsv1alpha1.ConditionEquals(*gotCondition, *tc.wantCondition))
			}

			require.Equal(t, tc.wantFinalizerExists, controllerutil.ContainsFinalizer(nats, NATSFinalizerName))
		})
	}
}
