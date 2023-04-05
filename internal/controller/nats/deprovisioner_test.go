package nats

import (
	"errors"
	"testing"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func Test_handleNATSDeletion(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name                   string
		givenNATS              *natsv1alpha1.Nats
		givenWithNATSCreated   bool
		givenDeletionError     error
		wantAvailableCondition *metav1.Condition
		wantNATSStatusState    string
		wantFinalizerExists    bool
	}{
		{
			name:                 "should not do anything if finalizer is not set",
			givenWithNATSCreated: false,
			givenNATS: testutils.NewSampleNATSCR(
				testutils.WithNATSStateReady(),
			),
			wantNATSStatusState: natsv1alpha1.StateReady,
		},
		{
			name:                 "should delete nats resources",
			givenWithNATSCreated: true,
			givenNATS: testutils.NewSampleNATSCR(
				testutils.WithNATSCRStatusInitialized(),
				testutils.WithNATSStateReady(),
				testutils.WithNATSCRFinalizer(NATSFinalizerName),
			),
			wantNATSStatusState: natsv1alpha1.StateDeleting,
		},
		{
			name:                 "should update status with error when deletion fails",
			givenWithNATSCreated: true,
			givenNATS: testutils.NewSampleNATSCR(
				testutils.WithNATSCRStatusInitialized(),
				testutils.WithNATSStateReady(),
				testutils.WithNATSCRFinalizer(NATSFinalizerName),
			),
			wantNATSStatusState: natsv1alpha1.StateError,
			givenDeletionError:  errors.New("deletion failed"),
			wantAvailableCondition: &metav1.Condition{
				Type:               string(natsv1alpha1.ConditionAvailable),
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             string(natsv1alpha1.ConditionReasonProcessingError),
				Message:            "deletion failed",
			},
			wantFinalizerExists: true,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			var objs []client.Object
			if tc.givenWithNATSCreated {
				objs = append(objs, tc.givenNATS)
			}

			testEnv := NewMockedUnitTestEnvironment(t, objs...)
			reconciler := testEnv.Reconciler
			nats := tc.givenNATS.DeepCopy()

			// define mocks behaviour
			testEnv.kubeClient.On("DestinationRuleCRDExists",
				mock.Anything).Return(false, nil)
			testEnv.kubeClient.On("GetSecret",
				mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

			natsResources := &chart.ManifestResources{
				Items: []*unstructured.Unstructured{
					testutils.NewSampleNATSStatefulSetUnStruct(),
				},
			}
			testEnv.natsManager.On("GenerateNATSResources",
				mock.Anything, mock.Anything, mock.Anything).Return(natsResources, nil)

			if tc.givenDeletionError != nil {
				testEnv.natsManager.On("DeleteInstance",
					mock.Anything, mock.Anything).Return(tc.givenDeletionError)
			} else {
				testEnv.natsManager.On("DeleteInstance",
					mock.Anything, mock.Anything).Return(nil)
			}

			// when
			_, err := reconciler.handleNATSDeletion(testEnv.Context, nats, testEnv.Logger)

			// then
			require.NoError(t, err)
			require.Equal(t, tc.wantNATSStatusState, nats.Status.State)

			if tc.wantAvailableCondition != nil {
				gotCondition := nats.Status.FindCondition(natsv1alpha1.ConditionType(tc.wantAvailableCondition.Type))
				require.NotNil(t, gotCondition)
				require.True(t, natsv1alpha1.ConditionEquals(*gotCondition, *tc.wantAvailableCondition))
			}

			require.Equal(t, tc.wantFinalizerExists, controllerutil.ContainsFinalizer(nats, NATSFinalizerName))
		})
	}
}
