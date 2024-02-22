package nats

import (
	"context"
	"errors"
	"fmt"
	"testing"

	nmnats "github.com/kyma-project/nats-manager/pkg/nats"
	"go.uber.org/zap"

	nmmgr "github.com/kyma-project/nats-manager/pkg/manager"
	"github.com/kyma-project/nats-manager/pkg/nats/mocks"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	nmkmocks "github.com/kyma-project/nats-manager/pkg/k8s/mocks"
	"github.com/kyma-project/nats-manager/testutils"
	natssdk "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kcontrollerruntime "sigs.k8s.io/controller-runtime"
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
		mockNatsClientFunc     func() nmnats.Client
		wantCondition          *kmetav1.Condition
		wantNATSStatusState    string
		wantFinalizerExists    bool
		wantK8sEvents          []string
		wantResult             kcontrollerruntime.Result
	}{
		{
			name:                   "should not do anything if finalizer is not set",
			givenWithNATSCreated:   false,
			natsCrWithoutFinalizer: true,
			wantNATSStatusState:    nmapiv1alpha1.StateReady,
			wantK8sEvents:          []string{},
			wantResult:             kcontrollerruntime.Result{},
		},
		{
			name:                 "should delete resources if connection to NATS server is not established",
			givenWithNATSCreated: true,
			mockNatsClientFunc: func() nmnats.Client {
				natsClient := new(mocks.Client)
				natsClient.On("Init").Return(errors.New("connection cannot be established"))
				natsClient.On("Close").Return()
				return natsClient
			},
			wantNATSStatusState: nmapiv1alpha1.StateDeleting,
			wantK8sEvents:       []string{"Normal Deleting Deleting the NATS cluster."},
			wantResult:          kcontrollerruntime.Result{},
		},
		{
			name:                 "should delete resources if natsClients GetStreams returns unexpected error",
			givenWithNATSCreated: true,
			wantNATSStatusState:  nmapiv1alpha1.StateDeleting,
			mockNatsClientFunc: func() nmnats.Client {
				natsClient := new(mocks.Client)
				natsClient.On("Init").Return(nil)
				natsClient.On("GetStreams").Return(nil, errors.New("unexpected error"))
				natsClient.On("Close").Return()
				return natsClient
			},
			wantK8sEvents: []string{"Normal Deleting Deleting the NATS cluster."},
			wantResult:    kcontrollerruntime.Result{},
		},
		{
			name:                 "should delete resources if natsClients ConsumersExist returns unexpected error",
			givenWithNATSCreated: true,
			wantNATSStatusState:  nmapiv1alpha1.StateDeleting,
			mockNatsClientFunc: func() nmnats.Client {
				natsClient := new(mocks.Client)
				natsClient.On("Init").Return(nil)
				natsClient.On("GetStreams").Return([]*natssdk.StreamInfo{
					{
						Config: natssdk.StreamConfig{
							Name: SapStreamName,
						},
					},
				}, nil)
				natsClient.On("ConsumersExist", mock.Anything).Return(false, errors.New("unexpected error"))
				natsClient.On("Close").Return()
				return natsClient
			},
			wantK8sEvents: []string{"Normal Deleting Deleting the NATS cluster."},
			wantResult:    kcontrollerruntime.Result{},
		},
		{
			name:                 "should block deletion if non 'sap' stream exists",
			givenWithNATSCreated: true,
			wantNATSStatusState:  nmapiv1alpha1.StateWarning,
			wantCondition: &kmetav1.Condition{
				Type:               string(nmapiv1alpha1.ConditionDeleted),
				Status:             kmetav1.ConditionFalse,
				LastTransitionTime: kmetav1.Now(),
				Reason:             string(nmapiv1alpha1.ConditionReasonDeletionError),
				Message:            StreamExistsErrorMsg,
			},
			mockNatsClientFunc: func() nmnats.Client {
				natsClient := new(mocks.Client)
				natsClient.On("Init").Return(nil)
				natsClient.On("GetStreams").Return([]*natssdk.StreamInfo{
					{
						Config: natssdk.StreamConfig{
							Name: "non-sap",
						},
					},
				}, nil)
				natsClient.On("Close").Return()
				return natsClient
			},
			wantFinalizerExists: true,
			wantK8sEvents: []string{
				"Normal Deleting Deleting the NATS cluster.",
				"Warning DeletionError " + StreamExistsErrorMsg,
			},
			wantResult: kcontrollerruntime.Result{Requeue: true},
		},
		{
			name:                 "should block deletion if 'sap' stream consumer exists",
			givenWithNATSCreated: true,
			wantNATSStatusState:  nmapiv1alpha1.StateWarning,
			wantCondition: &kmetav1.Condition{
				Type:               string(nmapiv1alpha1.ConditionDeleted),
				Status:             kmetav1.ConditionFalse,
				LastTransitionTime: kmetav1.Now(),
				Reason:             string(nmapiv1alpha1.ConditionReasonDeletionError),
				Message:            ConsumerExistsErrorMsg,
			},
			mockNatsClientFunc: func() nmnats.Client {
				natsClient := new(mocks.Client)
				natsClient.On("Init").Return(nil)
				natsClient.On("GetStreams").Return([]*natssdk.StreamInfo{
					{
						Config: natssdk.StreamConfig{
							Name: SapStreamName,
						},
					},
				}, nil)
				natsClient.On("ConsumersExist", mock.Anything).Return(true, nil)
				natsClient.On("Close").Return()
				return natsClient
			},
			wantFinalizerExists: true,
			wantK8sEvents: []string{
				"Normal Deleting Deleting the NATS cluster.",
				"Warning DeletionError " + ConsumerExistsErrorMsg,
			},
			wantResult: kcontrollerruntime.Result{Requeue: true},
		},
		{
			name:                 "should delete resources if neither consumer stream nor 'sap' stream exists",
			givenWithNATSCreated: true,
			wantNATSStatusState:  nmapiv1alpha1.StateDeleting,
			mockNatsClientFunc: func() nmnats.Client {
				natsClient := new(mocks.Client)
				natsClient.On("Init").Return(nil)
				natsClient.On("GetStreams").Return([]*natssdk.StreamInfo{
					{
						Config: natssdk.StreamConfig{
							Name: SapStreamName,
						},
					},
				}, nil)
				natsClient.On("ConsumersExist", mock.Anything).Return(false, nil)
				natsClient.On("Close").Return()
				return natsClient
			},
			wantK8sEvents: []string{
				"Normal Deleting Deleting the NATS cluster.",
			},
			wantResult: kcontrollerruntime.Result{},
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			var givenNats *nmapiv1alpha1.NATS
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
				mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
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
					nmmgr.IstioEnabledKey:   false,
					nmmgr.RotatePasswordKey: true, // do not recreate secret if it exists
				},
			)

			if tc.mockNatsClientFunc != nil {
				reconciler.natsClients[nats.Namespace+"/"+nats.Name] = tc.mockNatsClientFunc()
			}

			// when
			result, err := reconciler.handleNATSDeletion(testEnv.Context, nats, testEnv.Logger)

			// then
			require.NoError(t, err)
			require.Equal(t, tc.wantResult, result)
			if tc.wantFinalizerExists {
				require.Equal(t, tc.wantNATSStatusState, nats.Status.State)
			}

			if tc.wantCondition != nil {
				gotCondition := nats.Status.FindCondition(nmapiv1alpha1.ConditionType(tc.wantCondition.Type))
				require.NotNil(t, gotCondition)
				require.True(t, nmapiv1alpha1.ConditionEquals(*gotCondition, *tc.wantCondition))
			}

			// check k8s events
			gotEvents := testEnv.GetK8sEvents()
			require.Equal(t, tc.wantK8sEvents, gotEvents)

			require.Equal(t, tc.wantFinalizerExists, controllerutil.ContainsFinalizer(nats, NATSFinalizerName))
		})
	}
}

func Test_DeletePVCsAndRemoveFinalizer(t *testing.T) {
	tests := []struct {
		name           string
		nats           *nmapiv1alpha1.NATS
		labelValue     string
		deleteErr      error
		expectedResult kcontrollerruntime.Result
		expectedErr    error
	}{
		{
			name: "delete PVCs and remove finalizer",
			nats: testutils.NewNATSCR(
				testutils.WithNATSCRName("test-nats"),
				testutils.WithNATSCRNamespace("test-namespace"),
				testutils.WithNATSCRFinalizer(NATSFinalizerName),
			),
			labelValue:     "test-nats",
			deleteErr:      nil,
			expectedResult: kcontrollerruntime.Result{},
			expectedErr:    nil,
		},
		{
			name: "labelSelector must be 'app.kubernetes.io/instance=eventing' for 'eventing-nats' nats CR name",
			nats: testutils.NewNATSCR(
				testutils.WithNATSCRName("eventing-nats"),
				testutils.WithNATSCRNamespace("kyma-system"),
				testutils.WithNATSCRFinalizer(NATSFinalizerName),
			),
			labelValue:     "eventing",
			deleteErr:      nil,
			expectedResult: kcontrollerruntime.Result{},
			expectedErr:    nil,
		},
		{
			name: "delete PVCs error",
			nats: testutils.NewNATSCR(
				testutils.WithNATSCRName("test-nats"),
				testutils.WithNATSCRNamespace("test-namespace"),
				testutils.WithNATSCRFinalizer(NATSFinalizerName),
			),
			labelValue:     "test-nats",
			deleteErr:      errors.New("delete error"),
			expectedResult: kcontrollerruntime.Result{},
			expectedErr:    errors.New("delete error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var objs []client.Object
			if tt.nats != nil {
				objs = append(objs, tt.nats)
			}

			testEnv := NewMockedUnitTestEnvironment(t, objs...)
			r := testEnv.Reconciler

			r.kubeClient.(*nmkmocks.Client).On("DeletePVCsWithLabel", mock.Anything, mock.Anything,
				tt.nats.Name, tt.nats.Namespace).Return(tt.deleteErr)
			natsClient := new(mocks.Client)
			r.setNatsClient(tt.nats, natsClient)
			r.getNatsClient(tt.nats).(*mocks.Client).On("Close").Return(nil)

			result, err := r.deletePVCsAndRemoveFinalizer(context.Background(), tt.nats, zap.NewNop().Sugar())

			require.Equal(t, tt.expectedResult, result)
			require.Equal(t, tt.expectedErr, err)
			if tt.deleteErr == nil {
				require.False(t, r.containsFinalizer(tt.nats))
			}

			labelSelector := fmt.Sprintf("%s=%s", InstanceLabelKey, tt.labelValue)
			r.kubeClient.(*nmkmocks.Client).EXPECT().DeletePVCsWithLabel(mock.Anything, labelSelector,
				tt.nats.Name, tt.nats.Namespace).Times(1)
		})
	}
}

func Test_CreateAndConnectNatsClient(t *testing.T) {
	tests := []struct {
		name        string
		nats        *nmapiv1alpha1.NATS
		initErr     error
		expectedErr error
	}{
		{
			name: "connect to existing client instance",
			nats: testutils.NewNATSCR(
				testutils.WithNATSCRName("test-nats"),
				testutils.WithNATSCRNamespace("test-namespace"),
			),
			initErr:     nil,
			expectedErr: nil,
		},
		{
			name: "init error",
			nats: testutils.NewNATSCR(
				testutils.WithNATSCRName("test-nats"),
				testutils.WithNATSCRNamespace("test-namespace"),
			),
			initErr:     errors.New("init error"),
			expectedErr: errors.New("init error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{}
			r.natsClients = make(map[string]nmnats.Client)
			r.setNatsClient(tt.nats, new(mocks.Client))
			r.getNatsClient(tt.nats).(*mocks.Client).On("Init").Return(tt.initErr)

			err := r.createAndConnectNatsClient(tt.nats)

			if err != nil {
				require.Equal(t, tt.expectedErr.Error(), err.Error())
			}
			r.getNatsClient(tt.nats).(*mocks.Client).AssertExpectations(t)
		})
	}
}

func Test_CloseNatsClient(t *testing.T) {
	tests := []struct {
		name           string
		nats           *nmapiv1alpha1.NATS
		existingClient *mocks.Client
	}{
		{
			name: "close existing client",
			nats: testutils.NewNATSCR(
				testutils.WithNATSCRName("test-nats"),
				testutils.WithNATSCRNamespace("test-namespace"),
			),
			existingClient: new(mocks.Client),
		},
		{
			name: "no existing client",
			nats: testutils.NewNATSCR(
				testutils.WithNATSCRName("test-nats"),
				testutils.WithNATSCRNamespace("test-namespace"),
			),
			existingClient: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{}
			r.natsClients = make(map[string]nmnats.Client)
			if tt.existingClient != nil {
				tt.existingClient.On("Close").Return(nil)
				r.setNatsClient(tt.nats, tt.existingClient)
			}

			r.closeNatsClient(tt.nats)
			if tt.existingClient != nil {
				tt.existingClient.AssertExpectations(t)
			}
			require.Nil(t, r.getNatsClient(tt.nats), "natsClient should be nil")
		})
	}
}
