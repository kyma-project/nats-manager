package nats

import (
	"fmt"
	"testing"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	natsmanager "github.com/kyma-project/nats-manager/pkg/manager"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func Test_generateNatsResources(t *testing.T) {
	t.Parallel()

	givenNATS := testutils.NewNATSCR()

	testEnv := NewMockedUnitTestEnvironment(t, givenNATS)
	reconciler := testEnv.Reconciler

	instance := &chart.ReleaseInstance{
		Name:      "test1",
		Namespace: "test1",
	}
	require.Len(t, instance.RenderedManifests.Items, 0)

	// define mock behaviour
	natsResources := &chart.ManifestResources{
		Items: []*unstructured.Unstructured{
			testutils.NewNATSStatefulSetUnStruct(),
		},
	}
	testEnv.natsManager.On("GenerateNATSResources",
		instance, mock.AnythingOfType("manager.Option"), mock.AnythingOfType("manager.Option"),
	).Return(natsResources, nil).Once()

	// when
	err := reconciler.generateNatsResources(givenNATS, instance)

	// then
	require.NoError(t, err)
	require.Len(t, instance.RenderedManifests.Items, len(natsResources.Items))
	// the method should have being called
	testEnv.natsManager.AssertExpectations(t)
}

func Test_initNATSInstance(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name               string
		givenNATS          *natsv1alpha1.NATS
		wantIstioEnabled   bool
		wantRotatePassword bool
	}{
		{
			name:               "should return instance with right configurations and manifests (istio: disabled)",
			givenNATS:          testutils.NewNATSCR(),
			wantIstioEnabled:   false,
			wantRotatePassword: false,
		},
		{
			name:               "should return instance with right configurations and manifests (istio: enabled)",
			givenNATS:          testutils.NewNATSCR(),
			wantIstioEnabled:   true,
			wantRotatePassword: true,
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

			// define mocks behaviour
			testEnv.kubeClient.On("DestinationRuleCRDExists",
				mock.Anything).Return(tc.wantIstioEnabled, nil)
			if tc.wantRotatePassword {
				// if secret do not exist, then password will be rotated.
				testEnv.kubeClient.On("GetSecret",
					mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
			} else {
				// if secret exist, then password will not be rotated.
				sampleSecret := testutils.NewSecret()
				testEnv.kubeClient.On("GetSecret",
					mock.Anything, mock.Anything, mock.Anything).Return(sampleSecret, nil)
			}

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
					natsmanager.IstioEnabledKey:   tc.wantIstioEnabled,
					natsmanager.RotatePasswordKey: tc.wantRotatePassword, // do not recreate secret if it exists
				},
			)

			// when
			releaseInstance, err := reconciler.initNATSInstance(testEnv.Context, tc.givenNATS, testEnv.Logger)

			// then
			require.NoError(t, err)
			require.Len(t, releaseInstance.RenderedManifests.Items, len(natsResources.Items))
			require.Equal(t, tc.wantIstioEnabled, releaseInstance.Configuration["istio.enabled"])
			// if secret does not exist, then it should rotate password to create new secret
			require.Equal(t, tc.wantRotatePassword, releaseInstance.Configuration["auth.rotatePassword"])
		})
	}
}

func Test_handleNATSCRAllowedCheck(t *testing.T) {
	t.Parallel()

	givenAllowedNATS := testutils.NewNATSCR(
		testutils.WithNATSCRName("eventing-nats"),
		testutils.WithNATSCRNamespace("kyma-system"),
	)

	// define test cases
	testCases := []struct {
		name            string
		givenNATS       *natsv1alpha1.NATS
		wantCheckResult bool
	}{
		{
			name: "should allow NATS CR if name and namespace is correct",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRName("eventing-nats"),
				testutils.WithNATSCRNamespace("kyma-system"),
			),
			wantCheckResult: true,
		},
		{
			name: "should not allow NATS CR if name is incorrect",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRName("not-allowed-name"),
				testutils.WithNATSCRNamespace("kyma-system"),
			),
			wantCheckResult: false,
		},
		{
			name: "should not allow NATS CR if namespace is incorrect",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRName("eventing-nats"),
				testutils.WithNATSCRNamespace("not-allowed-namespace"),
			),
			wantCheckResult: false,
		},
		{
			name: "should not allow NATS CR if name and namespace, both are incorrect",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRName("not-allowed-name"),
				testutils.WithNATSCRNamespace("not-allowed-namespace"),
			),
			wantCheckResult: false,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			testEnv := NewMockedUnitTestEnvironment(t, tc.givenNATS)
			testEnv.Reconciler.allowedNATSCR = givenAllowedNATS

			// when
			result, err := testEnv.Reconciler.handleNATSCRAllowedCheck(testEnv.Context, tc.givenNATS, testEnv.Logger)

			// then
			require.NoError(t, err)
			require.Equal(t, tc.wantCheckResult, result)

			// if the NATS CR is not allowed then check if the CR status is correctly updated or not.
			gotNATS, err := testEnv.GetNATS(tc.givenNATS.Name, tc.givenNATS.Namespace)
			require.NoError(t, err)
			if !tc.wantCheckResult {
				// check nats.status.state
				require.Equal(t, natsv1alpha1.StateError, gotNATS.Status.State)

				// check nats.status.conditions
				wantConditions := []metav1.Condition{
					{
						Type:               string(natsv1alpha1.ConditionStatefulSet),
						Status:             metav1.ConditionFalse,
						LastTransitionTime: metav1.Now(),
						Reason:             string(natsv1alpha1.ConditionReasonForbidden),
						Message:            "",
					},
					{
						Type:               string(natsv1alpha1.ConditionAvailable),
						Status:             metav1.ConditionFalse,
						LastTransitionTime: metav1.Now(),
						Reason:             string(natsv1alpha1.ConditionReasonForbidden),
						Message: fmt.Sprintf(ErrSingleCRAllowedFormat, givenAllowedNATS.Name,
							givenAllowedNATS.Namespace),
					},
				}
				require.True(t, natsv1alpha1.ConditionsEquals(wantConditions, gotNATS.Status.Conditions))

				wantK8sEvent := []string{
					fmt.Sprintf("Warning Forbidden Only a single NATS CR with name: %s and namespace: %s is"+
						"allowed to be created in a Kyma cluster.", givenAllowedNATS.Name,
						givenAllowedNATS.Namespace),
				}

				// check k8s events
				gotEvents := testEnv.GetK8sEvents()
				require.Equal(t, wantK8sEvent, gotEvents)
			}
		})
	}
}
