package nats

import (
	"context"
	"testing"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/require"
)

func Test_containsFinalizer(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name       string
		givenNats  *nmapiv1alpha1.NATS
		wantResult bool
	}{
		{
			name:       "should return false when finalizer is missing",
			givenNats:  testutils.NewNATSCR(),
			wantResult: false,
		},
		{
			name:       "should return true when finalizer is present",
			givenNats:  testutils.NewNATSCR(testutils.WithNATSCRFinalizer(NATSFinalizerName)),
			wantResult: true,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			testEnv := NewMockedUnitTestEnvironment(t)
			reconciler := testEnv.Reconciler

			// when, then
			require.Equal(t, tc.wantResult, reconciler.containsFinalizer(tc.givenNats))
		})
	}
}

func Test_addFinalizer(t *testing.T) {
	t.Parallel()

	t.Run("should add finalizer", func(t *testing.T) {
		t.Parallel()

		// given
		givenNats := testutils.NewNATSCR()

		testEnv := NewMockedUnitTestEnvironment(t, givenNats)
		reconciler := testEnv.Reconciler

		// when
		_, err := reconciler.addFinalizer(context.Background(), givenNats)

		// then
		require.NoError(t, err)
		gotNats, err := testEnv.GetNATS(givenNats.GetName(), givenNats.GetNamespace())
		require.NoError(t, err)
		require.True(t, reconciler.containsFinalizer(&gotNats))
	})
}

func Test_removeFinalizer(t *testing.T) {
	t.Parallel()

	t.Run("should remove finalizer", func(t *testing.T) {
		t.Parallel()

		// given
		givenNats := testutils.NewNATSCR(testutils.WithNATSCRFinalizer(NATSFinalizerName))

		testEnv := NewMockedUnitTestEnvironment(t, givenNats)
		reconciler := testEnv.Reconciler

		// when
		_, err := reconciler.removeFinalizer(context.Background(), givenNats)

		// then
		require.NoError(t, err)
		gotNats, err := testEnv.GetNATS(givenNats.GetName(), givenNats.GetNamespace())
		require.NoError(t, err)
		require.False(t, reconciler.containsFinalizer(&gotNats))
	})
}
