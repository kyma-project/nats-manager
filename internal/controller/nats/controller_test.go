package nats

import (
	"testing"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func Test_generateNatsResources(t *testing.T) {
	t.Parallel()

	givenNATS := testutils.NewSampleNATSCR()

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
			testutils.NewSampleNATSStatefulSetUnStruct(),
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
		name             string
		givenNATS        *natsv1alpha1.Nats
		wantIstioEnabled bool
		wantSecretExists bool
	}{
		{
			name:             "should return instance with right configurations and manifests (istio: disabled)",
			givenNATS:        testutils.NewSampleNATSCR(),
			wantIstioEnabled: false,
			wantSecretExists: false,
		},
		{
			name:             "should return instance with right configurations and manifests (istio: enabled)",
			givenNATS:        testutils.NewSampleNATSCR(),
			wantIstioEnabled: true,
			wantSecretExists: true,
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
			if tc.wantSecretExists {
				sampleSecret := testutils.NewSampleSecret()
				testEnv.kubeClient.On("GetSecret",
					mock.Anything, mock.Anything, mock.Anything).Return(sampleSecret, nil)
			} else {
				testEnv.kubeClient.On("GetSecret",
					mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
			}

			natsResources := &chart.ManifestResources{
				Items: []*unstructured.Unstructured{
					testutils.NewSampleNATSStatefulSetUnStruct(),
				},
			}
			testEnv.natsManager.On("GenerateNATSResources",
				mock.Anything, mock.Anything, mock.Anything).Return(natsResources, nil)

			// when
			releaseInstance, err := reconciler.initNATSInstance(testEnv.Context, tc.givenNATS, testEnv.Logger)

			// then
			require.NoError(t, err)
			require.Len(t, releaseInstance.RenderedManifests.Items, len(natsResources.Items))
			require.Equal(t, tc.wantIstioEnabled, releaseInstance.Configuration["istio.enabled"])
			// if secret does not exist, then it should rotate password to create new secret
			require.Equal(t, !tc.wantSecretExists, releaseInstance.Configuration["auth.rotatePassword"])
		})
	}
}
