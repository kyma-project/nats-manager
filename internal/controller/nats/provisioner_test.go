package nats

import (
	"errors"
	"testing"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	nmmgr "github.com/kyma-project/nats-manager/pkg/manager"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var ErrDeployErrorMsg = errors.New("deploy error")

func Test_handleNATSState(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name                  string
		givenStatefulSetReady bool
		givenNATS             *nmapiv1alpha1.NATS
		wantState             string
		wantConditions        []kmetav1.Condition
		wantK8sEvents         []string
	}{
		{
			name:                  "should set correct status when StatefulSet is not ready",
			givenStatefulSetReady: false,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRName("eventing-nats"),
				testutils.WithNATSCRNamespace("kyma-system"),
			),
			wantState: nmapiv1alpha1.StateProcessing,
			wantConditions: []kmetav1.Condition{
				{
					Type:               string(nmapiv1alpha1.ConditionStatefulSet),
					Status:             kmetav1.ConditionFalse,
					LastTransitionTime: kmetav1.Now(),
					Reason:             string(nmapiv1alpha1.ConditionReasonStatefulSetPending),
					Message:            "",
				},
				{
					Type:               string(nmapiv1alpha1.ConditionAvailable),
					Status:             kmetav1.ConditionFalse,
					LastTransitionTime: kmetav1.Now(),
					Reason:             string(nmapiv1alpha1.ConditionReasonDeploying),
					Message:            "",
				},
			},
			wantK8sEvents: []string{
				"Normal Deploying NATS is being deployed, waiting for StatefulSet to get ready.",
			},
		},
		{
			name:                  "should set correct status when StatefulSet is ready",
			givenStatefulSetReady: true,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRName("eventing-nats"),
				testutils.WithNATSCRNamespace("kyma-system"),
			),
			wantState: nmapiv1alpha1.StateReady,
			wantConditions: []kmetav1.Condition{
				{
					Type:               string(nmapiv1alpha1.ConditionStatefulSet),
					Status:             kmetav1.ConditionTrue,
					LastTransitionTime: kmetav1.Now(),
					Reason:             string(nmapiv1alpha1.ConditionReasonStatefulSetAvailable),
					Message:            "StatefulSet is ready",
				},
				{
					Type:               string(nmapiv1alpha1.ConditionAvailable),
					Status:             kmetav1.ConditionTrue,
					LastTransitionTime: kmetav1.Now(),
					Reason:             string(nmapiv1alpha1.ConditionReasonDeployed),
					Message:            "NATS is deployed",
				},
			},
			wantK8sEvents: []string{
				"Normal Deployed StatefulSet is ready and NATS is deployed.",
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
			require.True(t, nmapiv1alpha1.ConditionsEquals(tc.wantConditions, gotNATS.Status.Conditions))

			// check k8s events
			gotEvents := testEnv.GetK8sEvents()
			require.Equal(t, tc.wantK8sEvents, gotEvents)

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
		givenNATS                       *nmapiv1alpha1.NATS
		givenDeployError                error
		wantFinalizerCheckOnly          bool
		wantState                       string
		wantConditions                  []kmetav1.Condition
		wantK8sEvents                   []string
		wantDestinationRuleWatchStarted bool
	}{
		{
			name:                  "should set finalizer first when missing",
			givenStatefulSetReady: false,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRName("eventing-nats"),
				testutils.WithNATSCRNamespace("kyma-system"),
			),
			wantState:              nmapiv1alpha1.StateProcessing,
			wantFinalizerCheckOnly: true,
			wantK8sEvents:          []string{"Normal Processing Initializing NATS resource."},
		},
		{
			name:                  "should set correct status when deployment fails",
			givenStatefulSetReady: false,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRName("eventing-nats"),
				testutils.WithNATSCRNamespace("kyma-system"),
				testutils.WithNATSCRFinalizer(NATSFinalizerName),
			),
			givenDeployError: ErrDeployErrorMsg,
			wantState:        nmapiv1alpha1.StateError,
			wantConditions: []kmetav1.Condition{
				{
					Type:               string(nmapiv1alpha1.ConditionStatefulSet),
					Status:             kmetav1.ConditionFalse,
					LastTransitionTime: kmetav1.Now(),
					Reason:             string(nmapiv1alpha1.ConditionReasonSyncFailError),
					Message:            "",
				},
				{
					Type:               string(nmapiv1alpha1.ConditionAvailable),
					Status:             kmetav1.ConditionFalse,
					LastTransitionTime: kmetav1.Now(),
					Reason:             string(nmapiv1alpha1.ConditionReasonProcessingError),
					Message:            "deploy error",
				},
			},
			wantK8sEvents: []string{
				"Normal Processing Initializing NATS resource.",
				"Warning FailedProcessing Error while NATS resources were deployed: deploy error",
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
			wantState:        nmapiv1alpha1.StateReady,
			wantConditions: []kmetav1.Condition{
				{
					Type:               string(nmapiv1alpha1.ConditionStatefulSet),
					Status:             kmetav1.ConditionTrue,
					LastTransitionTime: kmetav1.Now(),
					Reason:             string(nmapiv1alpha1.ConditionReasonStatefulSetAvailable),
					Message:            "StatefulSet is ready",
				},
				{
					Type:               string(nmapiv1alpha1.ConditionAvailable),
					Status:             kmetav1.ConditionTrue,
					LastTransitionTime: kmetav1.Now(),
					Reason:             string(nmapiv1alpha1.ConditionReasonDeployed),
					Message:            "NATS is deployed",
				},
			},
			wantK8sEvents: []string{
				"Normal Processing Initializing NATS resource.",
				"Normal Deployed StatefulSet is ready and NATS is deployed.",
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
			wantState:        nmapiv1alpha1.StateReady,
			wantConditions: []kmetav1.Condition{
				{
					Type:               string(nmapiv1alpha1.ConditionStatefulSet),
					Status:             kmetav1.ConditionTrue,
					LastTransitionTime: kmetav1.Now(),
					Reason:             string(nmapiv1alpha1.ConditionReasonStatefulSetAvailable),
					Message:            "StatefulSet is ready",
				},
				{
					Type:               string(nmapiv1alpha1.ConditionAvailable),
					Status:             kmetav1.ConditionTrue,
					LastTransitionTime: kmetav1.Now(),
					Reason:             string(nmapiv1alpha1.ConditionReasonDeployed),
					Message:            "NATS is deployed",
				},
			},
			wantDestinationRuleWatchStarted: true,
			wantK8sEvents: []string{
				"Normal Processing Initializing NATS resource.",
				"Normal Deployed StatefulSet is ready and NATS is deployed.",
			},
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
					nmmgr.IstioEnabledKey:   tc.wantDestinationRuleWatchStarted,
					nmmgr.RotatePasswordKey: true, // do not recreate secret if it exists
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

			gotNATS, err := testEnv.GetNATS(tc.givenNATS.Name, tc.givenNATS.Namespace)
			require.NoError(t, err)
			if tc.wantFinalizerCheckOnly {
				require.True(t, reconciler.containsFinalizer(&gotNATS))
				return
			}

			require.Equal(t, tc.wantDestinationRuleWatchStarted, reconciler.destinationRuleWatchStarted)
			require.Equal(t, tc.wantState, gotNATS.Status.State)

			// check k8s events
			gotEvents := testEnv.GetK8sEvents()
			require.Equal(t, tc.wantK8sEvents, gotEvents)

			// check further
			require.True(t, nmapiv1alpha1.ConditionsEquals(tc.wantConditions, gotNATS.Status.Conditions))
		})
	}
}
