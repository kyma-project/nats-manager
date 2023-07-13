package nats

import (
	"errors"
	"testing"

	natsmanager "github.com/kyma-project/nats-manager/pkg/manager"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func Test_handleNATSState(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name                  string
		givenStatefulSetReady bool
		givenNATS             *natsv1alpha1.NATS
		wantState             string
		wantConditions        []metav1.Condition
	}{
		{
			name:                  "should set correct status when StatefulSet is not ready",
			givenStatefulSetReady: false,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRName("eventing-nats"),
				testutils.WithNATSCRNamespace("kyma-system"),
			),
			wantState: natsv1alpha1.StateProcessing,
			wantConditions: []metav1.Condition{
				{
					Type:               string(natsv1alpha1.ConditionStatefulSet),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(natsv1alpha1.ConditionReasonStatefulSetPending),
					Message:            "",
				},
				{
					Type:               string(natsv1alpha1.ConditionAvailable),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(natsv1alpha1.ConditionReasonDeploying),
					Message:            "",
				},
			},
		},
		{
			name:                  "should set correct status when StatefulSet is ready",
			givenStatefulSetReady: true,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRName("eventing-nats"),
				testutils.WithNATSCRNamespace("kyma-system"),
			),
			wantState: natsv1alpha1.StateReady,
			wantConditions: []metav1.Condition{
				{
					Type:               string(natsv1alpha1.ConditionStatefulSet),
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             string(natsv1alpha1.ConditionReasonStatefulSetAvailable),
					Message:            "StatefulSet is ready",
				},
				{
					Type:               string(natsv1alpha1.ConditionAvailable),
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             string(natsv1alpha1.ConditionReasonDeployed),
					Message:            "NATS is deployed",
				},
			},
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			releaseInstance := &chart.ReleaseInstance{
				Name:      tc.givenNATS.Name,
				Namespace: tc.givenNATS.Namespace,
			}

			testEnv := NewMockedUnitTestEnvironment(t, tc.givenNATS)
			reconciler := testEnv.Reconciler

			// define mocks behaviour
			testEnv.natsManager.On("IsNATSStatefulSetReady",
				mock.Anything, mock.Anything).Return(tc.givenStatefulSetReady, nil).Once()

			// when
			_, err := reconciler.handleNATSState(testEnv.Context, tc.givenNATS, releaseInstance, testEnv.Logger)

			// then
			require.NoError(t, err)
			gotNATS, err := testEnv.GetNATS(tc.givenNATS.Name, tc.givenNATS.Namespace)
			require.NoError(t, err)
			require.Equal(t, tc.wantState, gotNATS.Status.State)
			require.True(t, natsv1alpha1.ConditionsEquals(tc.wantConditions, gotNATS.Status.Conditions))
			// mocked methods should have being called.
			testEnv.natsManager.AssertExpectations(t)
		})
	}
}

func Test_handleNATSReconcile(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name                            string
		givenStatefulSetReady           bool
		givenNATS                       *natsv1alpha1.NATS
		givenDeployError                error
		wantFinalizerCheckOnly          bool
		wantState                       string
		wantConditions                  []metav1.Condition
		wantDestinationRuleWatchStarted bool
	}{
		{
			name:                  "should set finalizer first when missing",
			givenStatefulSetReady: false,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRName("eventing-nats"),
				testutils.WithNATSCRNamespace("kyma-system"),
			),
			wantState:              natsv1alpha1.StateProcessing,
			wantFinalizerCheckOnly: true,
		},
		{
			name:                  "should set correct status when deployment fails",
			givenStatefulSetReady: false,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRName("eventing-nats"),
				testutils.WithNATSCRNamespace("kyma-system"),
				testutils.WithNATSCRFinalizer(NATSFinalizerName),
			),
			givenDeployError: errors.New("deploy error"),
			wantState:        natsv1alpha1.StateError,
			wantConditions: []metav1.Condition{
				{
					Type:               string(natsv1alpha1.ConditionStatefulSet),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(natsv1alpha1.ConditionReasonSyncFailError),
					Message:            "",
				},
				{
					Type:               string(natsv1alpha1.ConditionAvailable),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(natsv1alpha1.ConditionReasonProcessingError),
					Message:            "deploy error",
				},
			},
		},
		{
			name:                  "should set correct status when deployment is successful",
			givenStatefulSetReady: true,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRName("eventing-nats"),
				testutils.WithNATSCRNamespace("kyma-system"),
				testutils.WithNATSCRFinalizer(NATSFinalizerName),
			),
			givenDeployError: nil,
			wantState:        natsv1alpha1.StateReady,
			wantConditions: []metav1.Condition{
				{
					Type:               string(natsv1alpha1.ConditionStatefulSet),
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             string(natsv1alpha1.ConditionReasonStatefulSetAvailable),
					Message:            "StatefulSet is ready",
				},
				{
					Type:               string(natsv1alpha1.ConditionAvailable),
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             string(natsv1alpha1.ConditionReasonDeployed),
					Message:            "NATS is deployed",
				},
			},
		},
		{
			name:                  "should watch destinationRule when enabled",
			givenStatefulSetReady: true,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRName("eventing-nats"),
				testutils.WithNATSCRNamespace("kyma-system"),
				testutils.WithNATSCRFinalizer(NATSFinalizerName),
			),
			givenDeployError: nil,
			wantState:        natsv1alpha1.StateReady,
			wantConditions: []metav1.Condition{
				{
					Type:               string(natsv1alpha1.ConditionStatefulSet),
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             string(natsv1alpha1.ConditionReasonStatefulSetAvailable),
					Message:            "StatefulSet is ready",
				},
				{
					Type:               string(natsv1alpha1.ConditionAvailable),
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             string(natsv1alpha1.ConditionReasonDeployed),
					Message:            "NATS is deployed",
				},
			},
			wantDestinationRuleWatchStarted: true,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			testEnv := NewMockedUnitTestEnvironment(t, tc.givenNATS)
			reconciler := testEnv.Reconciler
			nats := tc.givenNATS.DeepCopy()

			// define mocks behaviour
			testEnv.natsManager.On("IsNATSStatefulSetReady",
				mock.Anything, mock.Anything).Return(tc.givenStatefulSetReady, nil)
			testEnv.kubeClient.On("DestinationRuleCRDExists",
				mock.Anything).Return(tc.wantDestinationRuleWatchStarted, nil)
			testEnv.controller.On("Watch",
				mock.Anything, mock.Anything,
				mock.Anything, mock.Anything,
				mock.Anything).Return(nil)
			testEnv.kubeClient.On("GetSecret",
				mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
			natsResources := &chart.ManifestResources{
				Items: []*unstructured.Unstructured{
					testutils.NewNATSStatefulSetUnStruct(
						testutils.WithName(tc.givenNATS.GetName()),
						testutils.WithNamespace(tc.givenNATS.GetNamespace()),
					),
				},
			}
			testEnv.natsManager.On("GenerateNATSResources",
				mock.Anything, mock.Anything, mock.Anything).Return(natsResources, nil)
			testEnv.natsManager.On("DeployInstance",
				mock.Anything, mock.Anything).Return(tc.givenDeployError)
			testEnv.natsManager.On("GenerateOverrides",
				mock.Anything, mock.Anything, mock.Anything).Return(
				map[string]interface{}{
					natsmanager.IstioEnabledKey:   tc.wantDestinationRuleWatchStarted,
					natsmanager.RotatePasswordKey: true, // do not recreate secret if it exists
				},
			)
			if tc.wantDestinationRuleWatchStarted {
				testEnv.ctrlManager.On("GetCache").Return(nil)
				testEnv.ctrlManager.On("GetRESTMapper").Return(testEnv.Client.RESTMapper())
			}

			// when
			_, err := reconciler.handleNATSReconcile(testEnv.Context, nats, testEnv.Logger)

			// then
			if tc.givenDeployError != nil {
				// the original error should have being returned, so another reconciliation is triggered.
				require.Error(t, err)
				require.Equal(t, tc.givenDeployError.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.wantDestinationRuleWatchStarted, reconciler.destinationRuleWatchStarted)
			gotNATS, err := testEnv.GetNATS(tc.givenNATS.Name, tc.givenNATS.Namespace)
			require.NoError(t, err)
			require.Equal(t, tc.wantState, gotNATS.Status.State)
			if tc.wantFinalizerCheckOnly {
				require.True(t, reconciler.containsFinalizer(&gotNATS))
				return
			}

			// check further
			require.True(t, natsv1alpha1.ConditionsEquals(tc.wantConditions, gotNATS.Status.Conditions))
		})
	}
}
