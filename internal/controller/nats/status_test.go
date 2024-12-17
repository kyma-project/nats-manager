package nats

import (
	"errors"
	"testing"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ErrTestErrorMsg = errors.New("test error")

func Test_syncNATSStatus(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name           string
		givenNATS      *nmapiv1alpha1.NATS
		wantNATSStatus nmapiv1alpha1.NATSStatus
		wantResult     bool
	}{
		{
			name: "should update the status",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRStatusInitialized(),
				testutils.WithNATSStateProcessing(),
			),
			wantNATSStatus: nmapiv1alpha1.NATSStatus{
				State: nmapiv1alpha1.StateReady,
				Conditions: []kmetav1.Condition{
					{
						Type:               string(nmapiv1alpha1.ConditionStatefulSet),
						Status:             kmetav1.ConditionTrue,
						LastTransitionTime: kmetav1.Now(),
						Reason:             string(nmapiv1alpha1.ConditionReasonProcessing),
						Message:            "",
					},
					{
						Type:               string(nmapiv1alpha1.ConditionAvailable),
						Status:             kmetav1.ConditionTrue,
						LastTransitionTime: kmetav1.Now(),
						Reason:             string(nmapiv1alpha1.ConditionReasonProcessing),
						Message:            "",
					},
				},
			},
			wantResult: false,
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

			newNATS := tc.givenNATS.DeepCopy()
			newNATS.Status = tc.wantNATSStatus

			// when
			err := reconciler.syncNATSStatus(testEnv.Context, newNATS, testEnv.Logger)

			// then
			require.NoError(t, err)
			gotNats, err := testEnv.GetNATS(tc.givenNATS.GetName(), tc.givenNATS.GetNamespace())
			require.NoError(t, err)
			require.True(t, gotNats.Status.IsEqual(tc.wantNATSStatus))
		})
	}
}

func Test_syncNATSStatusWithErr(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name           string
		givenNATS      *nmapiv1alpha1.NATS
		givenError     error
		wantNATSStatus nmapiv1alpha1.NATSStatus
		wantResult     bool
	}{
		{
			name: "should update the status with error message",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRStatusInitialized(),
				testutils.WithNATSStateProcessing(),
			),
			givenError: ErrTestErrorMsg,
			wantNATSStatus: nmapiv1alpha1.NATSStatus{
				State: nmapiv1alpha1.StateError,
				Conditions: []kmetav1.Condition{
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
						Message:            "test error",
					},
				},
			},
			wantResult: false,
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

			newNATS := tc.givenNATS.DeepCopy()
			newNATS.Status = tc.wantNATSStatus

			// when
			err := reconciler.syncNATSStatusWithErr(testEnv.Context, newNATS, tc.givenError, testEnv.Logger)

			// then
			// the original error should have being returned.
			require.Error(t, err)
			require.Equal(t, tc.givenError.Error(), err.Error())
			// now check if the error is reflected in the CR status.
			gotNats, err := testEnv.GetNATS(tc.givenNATS.GetName(), tc.givenNATS.GetNamespace())
			require.NoError(t, err)
			require.True(t, gotNats.Status.IsEqual(tc.wantNATSStatus))
		})
	}
}

func Test_updateStatus(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name           string
		givenNATS      *nmapiv1alpha1.NATS
		wantNATSStatus nmapiv1alpha1.NATSStatus
		wantResult     bool
	}{
		{
			name: "should update the status",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRStatusInitialized(),
				testutils.WithNATSStateProcessing(),
			),
			wantNATSStatus: nmapiv1alpha1.NATSStatus{
				State: nmapiv1alpha1.StateReady,
				Conditions: []kmetav1.Condition{
					{
						Type:               string(nmapiv1alpha1.ConditionStatefulSet),
						Status:             kmetav1.ConditionTrue,
						LastTransitionTime: kmetav1.Now(),
						Reason:             string(nmapiv1alpha1.ConditionReasonProcessing),
						Message:            "",
					},
					{
						Type:               string(nmapiv1alpha1.ConditionAvailable),
						Status:             kmetav1.ConditionTrue,
						LastTransitionTime: kmetav1.Now(),
						Reason:             string(nmapiv1alpha1.ConditionReasonProcessing),
						Message:            "",
					},
				},
			},
			wantResult: false,
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

			oldNATS, err := testEnv.GetNATS(tc.givenNATS.GetName(), tc.givenNATS.GetNamespace())
			require.NoError(t, err)
			newNATS := oldNATS.DeepCopy()
			newNATS.Status = tc.wantNATSStatus

			// when
			err = reconciler.updateStatus(testEnv.Context, &oldNATS, newNATS, testEnv.Logger)

			// then
			require.NoError(t, err)
			gotNats, err := testEnv.GetNATS(tc.givenNATS.GetName(), tc.givenNATS.GetNamespace())
			require.NoError(t, err)
			require.True(t, gotNats.Status.IsEqual(tc.wantNATSStatus))
		})
	}
}

func Test_watchDestinationRule(t *testing.T) {
	t.Parallel()

	// given
	testEnv := NewMockedUnitTestEnvironment(t)
	reconciler := testEnv.Reconciler

	// define mock behaviour
	testEnv.ctrlManager.On("GetCache").Return(nil)
	testEnv.ctrlManager.On("GetRESTMapper").Return(testEnv.Client.RESTMapper())
	testEnv.controller.On("Watch",
		mock.Anything,
	).Return(nil).Once()

	// when
	err := reconciler.watchDestinationRule(testEnv.Logger)

	// then
	require.NoError(t, err)
	testEnv.controller.AssertExpectations(t)
}
