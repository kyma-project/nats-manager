package validation_test

import (
	"os"
	"testing"

	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
	natsmatchers "github.com/kyma-project/nats-manager/testutils/matchers/nats"

	"github.com/kyma-project/nats-manager/testutils/integration"
)

const projectRootDir = "../../../../../"

const noError = ""

var testEnvironment *integration.TestEnvironment //nolint:gochecknoglobals // used in tests

// TestMain pre-hook and post-hook to run before and after all tests.
func TestMain(m *testing.M) {
	// Note: The setup will provision a single K8s env and
	// all the tests need to create and use a separate namespace

	// setup env test
	var err error
	testEnvironment, err = integration.NewTestEnvironment(projectRootDir, true)
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

func Test_Validate_CreateNatsCR(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		givenNATS  *v1alpha1.NATS
		wantErrMsg string
	}{
		{
			name: `validation of spec.cluster.size passes for odd numbers`,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSClusterSize(3)),
			wantErrMsg: noError,
		},
		{
			name: `validation of spec.cluster.size fails for even numbers`,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSClusterSize(4)),
			wantErrMsg: "size only accepts odd numbers",
		},
		{
			name: `validation of spec.cluster.size fails for numbers < 1`,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSClusterSize(-1)),
			wantErrMsg: "should be greater than or equal to 1",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			testEnvironment.EnsureNamespaceCreation(t, tc.givenNATS.GetNamespace())

			// when
			err := testEnvironment.CreateK8sResource(tc.givenNATS)

			// then
			if tc.wantErrMsg == "" {
				require.NoError(t, err)
			} else {
				require.Contains(t, err.Error(), tc.wantErrMsg)
			}
		})
	}
}

func Test_NATSCR_Defaulting(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		// We use Unstructured instead NATS, to ensure that all undefined properties are nil and not default.
		givenUnstructuredNATS unstructured.Unstructured
		wantMatches           gomegatypes.GomegaMatcher
	}{
		{
			name: "defaulting",
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "NATS",
					"apiVersion": "operator.kyma-project.io/v1alpha1",
					"metadata": map[string]interface{}{
						"name":      "name-defaulting01",
						"namespace": "namespace-defaulting01",
					},
				},
			},
			wantMatches: gomega.And(
				natsmatchers.HaveSpecClusterSize(3),
				// natsmatchers.HaveSpecResources(corev1.ResourceRequirements{
				// 	Limits: corev1.ResourceList{
				// 		"cpu":    resource.MustParse("20m"),
				// 		"memory": resource.MustParse("64Mi"),
				// 	},
				// 	Requests: corev1.ResourceList{
				// 		"cpu":    resource.MustParse("5m"),
				// 		"memory": resource.MustParse("16Mi"),
				// 	},
				// })
				natsmatchers.HaveSpecLoggingTrace(false),
				natsmatchers.HaveSpecLoggingDebug(false),
				natsmatchers.HaveSpecJetsStreamMemStorage(v1alpha1.MemStorage{
					Enabled: false,
					Size:    resource.MustParse("20Mi"),
				}),
				natsmatchers.HaveSpecJetStramFileStorage(v1alpha1.FileStorage{
					StorageClassName: "default",
					Size:             resource.MustParse("1Gi"),
				}),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewGomegaWithT(t)

			// given
			testEnvironment.EnsureNamespaceCreation(t, tc.givenUnstructuredNATS.GetNamespace())

			// when
			testEnvironment.EnsureK8sUnStructResourceCreated(t, &tc.givenUnstructuredNATS)

			// then
			testEnvironment.GetNATSAssert(g, &v1alpha1.NATS{
				ObjectMeta: metav1.ObjectMeta{Name: "name-defaulting01", Namespace: "namespace-defaulting01"},
			}).Should(tc.wantMatches)
		})
	}
}
