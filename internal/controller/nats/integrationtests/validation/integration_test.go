package validation_test

import (
	"os"
	"testing"

	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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

const (
	spec           = "spec"
	kind           = "kind"
	cluster        = "cluster"
	jetStream      = "jetStream"
	memStorage     = "memStorage"
	fileStorage    = "fileStorage"
	apiVersion     = "apiVersion"
	logging        = "logging"
	metadata       = "metadata"
	name           = "name"
	namespace      = "namespace"
	kindNATS       = "NATS"
	apiVersionNATS = "operator.kyma-project.io/v1alpha1"
)

var testEnvironment *integration.TestEnvironment //nolint:gochecknoglobals // used in tests

// TestMain pre-hook and post-hook to run before and after all tests.
func TestMain(m *testing.M) {
	// Note: The setup will provision a single K8s env and
	// all the tests need to create and use a separate namespace

	// setup env test
	var err error
	testEnvironment, err = integration.NewTestEnvironment(projectRootDir, true, nil)
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
		// We use Unstructured instead of NATS to ensure that all undefined properties are nil and not Go defaults.
		givenUnstructuredNATS unstructured.Unstructured
		wantMatches           gomegatypes.GomegaMatcher
	}{
		{
			name: "defaulting with bare minimum NATS",
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
				},
			},
			wantMatches: gomega.And(
				natsmatchers.HaveSpecClusterSize(3),
				natsmatchers.HaveSpecResources(corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						"cpu":    resource.MustParse("20m"),
						"memory": resource.MustParse("64Mi"),
					},
					Requests: corev1.ResourceList{
						"cpu":    resource.MustParse("5m"),
						"memory": resource.MustParse("16Mi"),
					},
				}),
				natsmatchers.HaveSpecLoggingTrace(false),
				natsmatchers.HaveSpecLoggingDebug(false),
				natsmatchers.HaveSpecJetsStreamMemStorage(v1alpha1.MemStorage{
					Enabled: false,
					Size:    resource.MustParse("20Mi"),
				}),
				natsmatchers.HaveSpecJetStreamFileStorage(v1alpha1.FileStorage{
					StorageClassName: "default",
					Size:             resource.MustParse("1Gi"),
				}),
			),
		},
		{
			name: "defaulting with an empty spec",
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{},
				},
			},
			wantMatches: gomega.And(
				natsmatchers.HaveSpecClusterSize(3),
				natsmatchers.HaveSpecResources(corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						"cpu":    resource.MustParse("20m"),
						"memory": resource.MustParse("64Mi"),
					},
					Requests: corev1.ResourceList{
						"cpu":    resource.MustParse("5m"),
						"memory": resource.MustParse("16Mi"),
					},
				}),
				natsmatchers.HaveSpecLoggingTrace(false),
				natsmatchers.HaveSpecLoggingDebug(false),
				natsmatchers.HaveSpecJetsStreamMemStorage(v1alpha1.MemStorage{
					Enabled: false,
					Size:    resource.MustParse("20Mi"),
				}),
				natsmatchers.HaveSpecJetStreamFileStorage(v1alpha1.FileStorage{
					StorageClassName: "default",
					Size:             resource.MustParse("1Gi"),
				}),
			),
		},
		{
			name: "defaulting with an empty spec.cluster",
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						cluster: map[string]any{},
					},
				},
			},
			wantMatches: gomega.And(
				natsmatchers.HaveSpecClusterSize(3),
			),
		},
		{
			name: "defaulting with an empty spec.jetStream",
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						jetStream: map[string]any{},
					},
				},
			},
			wantMatches: gomega.And(
				natsmatchers.HaveSpecJetsStreamMemStorage(v1alpha1.MemStorage{
					Enabled: false,
					Size:    resource.MustParse("20Mi"),
				}),
				natsmatchers.HaveSpecJetStreamFileStorage(v1alpha1.FileStorage{
					StorageClassName: "default",
					Size:             resource.MustParse("1Gi"),
				}),
			),
		},
		{
			name: "defaulting with an empty spec.jetStream.memStorage and spec.jetStream.fileStorage",
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						jetStream: map[string]any{
							memStorage:  map[string]any{},
							fileStorage: map[string]any{},
						},
					},
				},
			},
			wantMatches: gomega.And(
				natsmatchers.HaveSpecJetsStreamMemStorage(v1alpha1.MemStorage{
					Enabled: false,
					Size:    resource.MustParse("20Mi"),
				}),
				natsmatchers.HaveSpecJetStreamFileStorage(v1alpha1.FileStorage{
					StorageClassName: "default",
					Size:             resource.MustParse("1Gi"),
				}),
			),
		},
		{
			name: "defaulting with an empty spec.logging",
			givenUnstructuredNATS: unstructured.Unstructured{
				Object: map[string]any{
					kind:       kindNATS,
					apiVersion: apiVersionNATS,
					metadata: map[string]any{
						name:      testutils.GetRandK8sName(7),
						namespace: testutils.GetRandK8sName(7),
					},
					spec: map[string]any{
						logging: map[string]any{},
					},
				},
			},
			wantMatches: gomega.And(
				natsmatchers.HaveSpecLoggingTrace(false),
				natsmatchers.HaveSpecLoggingDebug(false),
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
			2
			// when
			testEnvironment.EnsureK8sUnStructResourceCreated(t, &tc.givenUnstructuredNATS)

			// then
			testEnvironment.GetNATSAssert(g, &v1alpha1.NATS{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tc.givenUnstructuredNATS.GetName(),
					Namespace: tc.givenUnstructuredNATS.GetNamespace(),
				},
			}).Should(tc.wantMatches)
		})
	}
}
