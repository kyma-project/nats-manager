package nats_test

import (
	"context"
	"os"
	"testing"

	"github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/kyma-project/nats-manager/testutils/integration"
	natsmatchers "github.com/kyma-project/nats-manager/testutils/matchers/nats"
	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var testEnvironment *integration.TestEnvironment //nolint:gochecknoglobals // used in tests

// TestMain pre-hook and post-hook to run before and after all tests.
func TestMain(m *testing.M) {
	// Note: The setup will provision a single K8s env and
	// all the tests need to create and use a separate namespace

	// setup env test
	var err error
	testEnvironment, err = integration.NewTestEnvironment()
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
			name: "NATS CR should have processing status when StatefulSet is not ready",
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
			givenNamespace := integration.NewTestNamespace()
			require.NoError(t, testEnvironment.CreateNamespace(ctx, givenNamespace))

			// update namespace in resources.
			tc.givenNATS.Namespace = givenNamespace

			// when
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenNATS)

			// then
			testEnvironment.EnsureK8sStatefulSetExists(t, integration.GetStatefulSetName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sConfigMapExists(t, integration.GetConfigMapName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sSecretExists(t, integration.GetSecretName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sServiceExists(t, integration.GetServiceName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sDestinationRuleExists(t,
				integration.GetDestinationRuleName(*tc.givenNATS), givenNamespace)

			if tc.givenStatefulSetReady {
				// make mock updates to deployed resources.
				makeStatefulSetReady(t, integration.GetStatefulSetName(*tc.givenNATS), givenNamespace)
			}

			// check NATS CR status.
			testEnvironment.GetNATSAssert(g, tc.givenNATS).Should(tc.wantMatches)
		})
	}
}

// Test_UpdateNATSCR tests if updating the NATS CR will trigger reconciliation
// and k8s objects are updated accordingly.
func Test_UpdateNATSCR(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	testCases := []struct {
		name            string
		givenNATS       *v1alpha1.NATS
		givenUpdateNATS *v1alpha1.NATS
	}{
		{
			name: "NATS CR should have ready status when StatefulSet is ready",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSCRName("test1"),
			),
			givenUpdateNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSCRName("test1"),
				testutils.WithNATSResources(corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						"cpu":    resource.MustParse("199m"),
						"memory": resource.MustParse("199Mi"),
					},
					Requests: corev1.ResourceList{
						"cpu":    resource.MustParse("99m"),
						"memory": resource.MustParse("99Mi"),
					},
				}),
				testutils.WithNATSLabels(map[string]string{
					"test-key1": "value1",
				}),
				testutils.WithNATSAnnotations(map[string]string{
					"test-key2": "value2",
				}),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// g := gomega.NewGomegaWithT(t)

			// given
			// create unique namespace for this test run.
			givenNamespace := integration.NewTestNamespace()
			require.NoError(t, testEnvironment.CreateNamespace(ctx, givenNamespace))

			// update namespace in resources.
			tc.givenNATS.Namespace = givenNamespace
			tc.givenUpdateNATS.Namespace = givenNamespace

			// create NATS CR.
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenNATS)
			testEnvironment.EnsureK8sStatefulSetExists(t, integration.GetStatefulSetName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sConfigMapExists(t, integration.GetConfigMapName(*tc.givenNATS), givenNamespace)

			// get NATS CR.
			nats, err := testEnvironment.GetNATSFromK8s(tc.givenNATS.Name, givenNamespace)
			require.NoError(t, err)

			// when
			// update NATS CR.
			newNATS := nats.DeepCopy()
			newNATS.Spec = tc.givenUpdateNATS.Spec
			testEnvironment.EnsureK8sResourceUpdated(t, newNATS)

			// then
			testEnvironment.EnsureNATSSpecClusterSizeReflected(t, *tc.givenUpdateNATS)
			testEnvironment.EnsureNATSSpecResourcesReflected(t, *tc.givenUpdateNATS)
			testEnvironment.EnsureK8sStatefulSetHasLabels(t, integration.GetStatefulSetName(*tc.givenNATS),
				givenNamespace, tc.givenUpdateNATS.Spec.Labels)
			testEnvironment.EnsureK8sStatefulSetHasAnnotations(t, integration.GetStatefulSetName(*tc.givenNATS),
				givenNamespace, tc.givenUpdateNATS.Spec.Annotations)
		})
	}
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
	}, integration.SmallTimeOut, integration.SmallPollingInterval, "failed to update status of StatefulSet")
}
