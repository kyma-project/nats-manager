package nats_test

import (
	"context"
	"fmt"
	"github.com/kyma-project/nats-manager/testutils"
	natsmatchers "github.com/kyma-project/nats-manager/testutils/matchers/nats"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

var testEnvironment *IntegrationTestEnvironment

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

	// create unique namespace for this test run
	testNamespace := NewTestNamespace()
	testEnvironment.CreateNamespace(ctx, testNamespace)

	natsCR := testutils.NewNATSCR(
		testutils.WithNATSCRDefaults(),
		testutils.WithNATSCRNamespace(testNamespace),
	)
	stsName := fmt.Sprintf("%s-nats", natsCR.Name) // name -> test-object1-nats, name -> test-object1-nats, namespace -> ns-icveu

	testEnvironment.EnsureK8sResourceCreated(t, natsCR)

	makeStatefulSetReady(t, stsName, natsCR.Namespace)

	require.Eventually(t, func() bool {
		nats, err := testEnvironment.GetNATSFromK8s(natsCR.Name, natsCR.Namespace)
		require.NoError(t, err)
		return natsmatchers.HaveStatusReady(nats)
	}, BigTimeOut, SmallPollingInterval)
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
	}, SmallTimeOut, SmallPollingInterval)

}
