package nats_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

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
		name                  string
		givenNATS             *v1alpha1.NATS
		givenStatefulSetReady bool
		wantMatches           gomegatypes.GomegaMatcher
	}{
		{
			name: "NATS CR should have ready status when StatefulSet is ready",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSCRName("test1"),
			),
			givenStatefulSetReady: true,
			wantMatches: gomega.And(
				natsmatchers.HaveStatusReady(),
				natsmatchers.HaveReadyConditionStatefulSet(),
				natsmatchers.HaveReadyConditionAvailable(),
			),
		},
		{
			name: "NATS CR should have processing status when StatefulSet is ready",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSCRName("test1"),
			),
			givenStatefulSetReady: false,
			wantMatches: gomega.And(
				natsmatchers.HaveStatusProcessing(),
				natsmatchers.HavePendingConditionStatefulSet(),
				natsmatchers.HaveDeployingConditionAvailable(),
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

			// when
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenNATS)

			sleepTime := 5 * time.Second
			time.Sleep(sleepTime)

			sts, err := testEnvironment.GetNATSFromK8s(tc.givenNATS.Name, givenNamespace)
			require.NoError(t, err)
			testEnvironment.Logger.Infow("sts", "sts", sts)

			// then
			testEnvironment.EnsureK8sStatefulSetExists(t, getStatefulSetName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sConfigMapExists(t, getConfigMapName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sSecretExists(t, getSecretName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sServiceExists(t, getServiceName(*tc.givenNATS), givenNamespace)

			if tc.givenStatefulSetReady {
				// make mock updates to deployed resources
				makeStatefulSetReady(t, getStatefulSetName(*tc.givenNATS), givenNamespace)
			}

			// check NATS CR status
			testEnvironment.GetNATSAssert(g, tc.givenNATS).Should(tc.wantMatches)
		})
	}
}

func getStatefulSetName(nats v1alpha1.NATS) string {
	return fmt.Sprintf("%s-nats", nats.Name)
}

func getConfigMapName(nats v1alpha1.NATS) string {
	return fmt.Sprintf("%s-nats-config", nats.Name)
}

func getSecretName(nats v1alpha1.NATS) string {
	return fmt.Sprintf("%s-nats-secret", nats.Name)
}

func getServiceName(nats v1alpha1.NATS) string {
	return fmt.Sprintf("%s-nats", nats.Name)
}

func getDestinationRuleName(nats v1alpha1.NATS) string {
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
