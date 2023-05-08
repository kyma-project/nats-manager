package nats_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
	natsmatchers "github.com/kyma-project/nats-manager/testutils/matchers/nats"
	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	"github.com/stretchr/testify/require"
)

var testEnvironment *IntegrationTestEnvironment //nolint:gochecknoglobals // used in tests

// TestMain pre-hook and post-hook to run before and after all tests.
func TestMain(m *testing.M) {
	// Note: The setup will provision a single K8s env and
	// all the tests need to create and use a separate namespace

	// setup env test
	var err error
	testEnvironment, err = NewIntegrationTestEnvironment()
	if err != nil {
		panic(err)
	}

	// run tests
	code := m.Run()

	// tear down test env
	if err = testEnvironment.TearDown(); err != nil {
		panic(err)
	}

	os.Exit(code)
}

func Test_CreateNATSCR(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	testCases := []struct {
		name        string
		givenNATS   *v1alpha1.NATS
		wantMatches gomegatypes.GomegaMatcher
	}{
		{
			name: "should reconcile success",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			wantMatches: gomega.And(
				natsmatchers.HaveStatusReady(),
				natsmatchers.HaveReadyConditionStatefulSet(),
				natsmatchers.HaveReadyConditionAvailable(),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewGomegaWithT(t)

			// given
			// create unique namespace for this test run.
			givenNamespace := NewTestNamespace()
			require.NoError(t, testEnvironment.CreateNamespace(ctx, givenNamespace))

			// update namespace in resources.
			tc.givenNATS.Namespace = givenNamespace
			stsName := getStatefulSetName(*tc.givenNATS)

			// when
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenNATS)

			// make mock updates to deployed resources
			makeStatefulSetReady(t, stsName, givenNamespace)

			// then
			testEnvironment.GetNATSAssert(g, tc.givenNATS).Should(tc.wantMatches)
		})
	}
}

func getStatefulSetName(nats v1alpha1.NATS) string {
	return fmt.Sprintf("%s-nats", nats.Name)
}

func makeStatefulSetReady(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		sts, err := testEnvironment.GetStatefulSetFromK8s(name, namespace)
		if err != nil {
			testEnvironment.Logger.Errorw("failed to get statefulSet", err)
			return false
		}

		sts.Status.Replicas = *sts.Spec.Replicas
		sts.Status.AvailableReplicas = *sts.Spec.Replicas
		sts.Status.CurrentReplicas = *sts.Spec.Replicas
		sts.Status.ReadyReplicas = *sts.Spec.Replicas
		sts.Status.UpdatedReplicas = *sts.Spec.Replicas

		err = testEnvironment.UpdateStatefulSetStatusOnK8s(*sts)
		if err != nil {
			testEnvironment.Logger.Errorw("failed to update statefulSet status", err)
			return false
		}
		return true
	}, SmallTimeOut, SmallPollingInterval, "failed to update status of StatefulSet")
}
