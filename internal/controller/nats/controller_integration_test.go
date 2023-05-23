package nats_test

import (
	"os"
	"testing"
	"time"

	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/kyma-project/nats-manager/testutils/integration"
	natsmatchers "github.com/kyma-project/nats-manager/testutils/matchers/nats"
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

	testCases := []struct {
		name                  string
		givenNATS             *v1alpha1.NATS
		givenStatefulSetReady bool
		wantMatches           gomegatypes.GomegaMatcher
		wantEnsureK8sObjects  bool
	}{
		// TODO "NATS CR should set default values"
		// Check that a cr without spec will have spec.cluster.size=3
		{
			name: "NATS CR should have processing status when StatefulSet is not ready",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			givenStatefulSetReady: false,
			wantMatches: gomega.And(
				natsmatchers.HaveStatusProcessing(),
				natsmatchers.HavePendingConditionStatefulSet(),
				natsmatchers.HaveDeployingConditionAvailable(),
			),
		},
		{
			name: "NATS CR should have ready status when StatefulSet is ready",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			givenStatefulSetReady: true,
			wantMatches: gomega.And(
				natsmatchers.HaveStatusReady(),
				natsmatchers.HaveReadyConditionStatefulSet(),
				natsmatchers.HaveReadyConditionAvailable(),
			),
		},
		{
			name: "should have created k8s objects as specified in NATS CR",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSLogging(true, true),
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
				testutils.WithNATSFileStorage(v1alpha1.FileStorage{
					StorageClassName: "test-sc1",
					Size:             resource.MustParse("66Gi"),
				}),
				testutils.WithNATSMemStorage(v1alpha1.MemStorage{
					Enabled: true,
					Size:    resource.MustParse("66Gi"),
				}),
			),
			givenStatefulSetReady: true,
			wantMatches: gomega.And(
				natsmatchers.HaveStatusReady(),
				natsmatchers.HaveReadyConditionStatefulSet(),
				natsmatchers.HaveReadyConditionAvailable(),
			),
			wantEnsureK8sObjects: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewGomegaWithT(t)

			// given
			// create unique namespace for this test run.
			givenNamespace := tc.givenNATS.GetNamespace()
			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// when
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenNATS)

			// then
			testEnvironment.EnsureK8sStatefulSetExists(t, testutils.GetStatefulSetName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sConfigMapExists(t, testutils.GetConfigMapName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sSecretExists(t, testutils.GetSecretName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sServiceExists(t, testutils.GetServiceName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sDestinationRuleExists(t,
				testutils.GetDestinationRuleName(*tc.givenNATS), givenNamespace)

			if tc.givenStatefulSetReady {
				// make mock updates to deployed resources.
				makeStatefulSetReady(t, testutils.GetStatefulSetName(*tc.givenNATS), givenNamespace)
			}

			// check NATS CR status.
			testEnvironment.GetNATSAssert(g, tc.givenNATS).Should(tc.wantMatches)

			if tc.wantEnsureK8sObjects {
				testEnvironment.EnsureNATSSpecClusterSizeReflected(t, *tc.givenNATS)
				testEnvironment.EnsureNATSSpecResourcesReflected(t, *tc.givenNATS)
				testEnvironment.EnsureNATSSpecDebugTraceReflected(t, *tc.givenNATS)
				testEnvironment.EnsureK8sStatefulSetHasLabels(t, testutils.GetStatefulSetName(*tc.givenNATS),
					givenNamespace, tc.givenNATS.Spec.Labels)
				testEnvironment.EnsureK8sStatefulSetHasAnnotations(t, testutils.GetStatefulSetName(*tc.givenNATS),
					givenNamespace, tc.givenNATS.Spec.Annotations)
				testEnvironment.EnsureNATSSpecMemStorageReflected(t, *tc.givenNATS)
				testEnvironment.EnsureNATSSpecFileStorageReflected(t, *tc.givenNATS)
			}
		})
	}
}

// Test_ValidateNATSCR_Creation tests the validation of NATS CR creation, as it is defined in
// `api/v1alpha1/nats_type.go`.
func Test_ValidateNATSCR_Creation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	testCases := []struct {
		name        string
		givenNATS   *v1alpha1.NATS
		errMatchers gomegatypes.GomegaMatcher
	}{
		{
			name: "the validation of the default NATS CR should not cause any errors",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			errMatchers: gomega.And(
				gomega.BeNil(),
			),
		},
		// TODO: creation with spec.cluster.size = 2 causes even-number-error.
		// TODO: creation with spec.cluster.size = 5 causes no even-number-error.
		// TODO: creation with spec.memStorage.enabled is  true causes error because .size must be set too.
		// TODO: creation with spec.memStorage.size set to a value causes no error.
		// TODO: creation with spec.memStorage.enabled and .size both set to a value causes no error.
		// TODO: creation with spec.fileStorage.storageClassName and .size both set to a value causes no error.
		// TODO: creation with spec.fileStorage.storageClassName set to a value and .size not causes an error.
		// TODO: creation with spec.fileStorage.size set to a value and .storageClassName not causes an error.
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
			err := testEnvironment.K8sResourceCreatedWithErr(tc.givenNATS)

			// then
			g.Expect(err, tc.errMatchers)
		})
	}
}

//nolint:lll
// TODO
// func Test_ValidateNATSCR_Change(t *testing.T) {
// TODO: deletion of spec.cluster.size causes no error, because defaulting.
// TODO: deletion of spec.memStorage.size when .enabled=true causes error because .size must be set too.
// TODO: deletion of spec.memStorage.enabled when .size is not set causes no error.
// TODO: change of spec.memStorage.enabled=false to =true while .size is not set causes error because .size must be set.
// TODO: change of spec when spec.fileStorage.storageClassName causes error.
// TODO: change of spec when spec.fileStorage.size causes error.
// TODO: deletion of spec when spec.fileStorage.storageClassName and .size both set to a value causes error.
// TODO: deletion of spec.jetStream when spec.jetStream.fileStorage.storageClassName and .size both set to a value causes error.
// TODO: deletion of spec.jetStream.fileStorage when spec.jetStream.fileStorage.storageClassName and .size both set to a value causes error.
// }

// Test_UpdateNATSCR tests if updating the NATS CR will trigger reconciliation
// and k8s objects are updated accordingly.
func Test_UpdateNATSCR(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name            string
		givenNATS       *v1alpha1.NATS
		givenUpdateNATS *v1alpha1.NATS
	}{
		{
			name: "NATS CR should have ready status when StatefulSet is ready",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSCRName("name-stays-the-same-1"),
				testutils.WithNATSCRNamespace("namespace-stays-the-same-1"),
			),
			givenUpdateNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSCRName("name-stays-the-same-1"),
				testutils.WithNATSCRNamespace("namespace-stays-the-same-1"),
				testutils.WithNATSLogging(true, true),
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
				testutils.WithNATSMemStorage(v1alpha1.MemStorage{
					Enabled: true,
					Size:    resource.MustParse("66Gi"),
				}),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			// Create Namespace in k8s.
			givenNamespace := tc.givenNATS.GetNamespace()
			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// create NATS CR.
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenNATS)
			testEnvironment.EnsureK8sStatefulSetExists(t, testutils.GetStatefulSetName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sConfigMapExists(t, testutils.GetConfigMapName(*tc.givenNATS), givenNamespace)

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
			testEnvironment.EnsureNATSSpecDebugTraceReflected(t, *tc.givenUpdateNATS)
			testEnvironment.EnsureK8sStatefulSetHasLabels(t, testutils.GetStatefulSetName(*tc.givenNATS),
				givenNamespace, tc.givenUpdateNATS.Spec.Labels)
			testEnvironment.EnsureK8sStatefulSetHasAnnotations(t, testutils.GetStatefulSetName(*tc.givenNATS),
				givenNamespace, tc.givenUpdateNATS.Spec.Annotations)
			testEnvironment.EnsureNATSSpecMemStorageReflected(t, *tc.givenUpdateNATS)
		})
	}
}

func Test_DeleteNATSCR(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		givenNATS *v1alpha1.NATS
	}{
		{
			name: "should delete all k8s objects",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
		},
		{
			name: "should delete all k8s objects with full NATS CR",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSLogging(true, true),
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
				testutils.WithNATSFileStorage(v1alpha1.FileStorage{
					StorageClassName: "test-sc1",
					Size:             resource.MustParse("66Gi"),
				}),
				testutils.WithNATSMemStorage(v1alpha1.MemStorage{
					Enabled: true,
					Size:    resource.MustParse("66Gi"),
				}),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			// create unique namespace for this test run.
			givenNamespace := tc.givenNATS.GetNamespace()
			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// create NATS CR
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenNATS)

			// ensure all k8s objects exists
			testEnvironment.EnsureK8sStatefulSetExists(t, testutils.GetStatefulSetName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sConfigMapExists(t, testutils.GetConfigMapName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sSecretExists(t, testutils.GetSecretName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sServiceExists(t, testutils.GetServiceName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sDestinationRuleExists(t,
				testutils.GetDestinationRuleName(*tc.givenNATS), givenNamespace)

			// when
			testEnvironment.EnsureK8sResourceDeleted(t, tc.givenNATS)

			// then
			// ensure all k8s objects are deleted
			testEnvironment.EnsureK8sStatefulSetNotFound(t,
				testutils.GetStatefulSetName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sConfigMapNotFound(t, testutils.GetConfigMapName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sSecretNotFound(t, testutils.GetSecretName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sServiceNotFound(t, testutils.GetServiceName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sDestinationRuleNotFound(t,
				testutils.GetDestinationRuleName(*tc.givenNATS), givenNamespace)

			// ensure NATS CR is deleted.
			testEnvironment.EnsureK8sNATSNotFound(t, tc.givenNATS.Name, givenNamespace)
		})
	}
}

// Test_WatcherNATSCRK8sObjects tests that deleting the k8s objects deployed by NATS CR
// should trigger reconciliation.
func Test_WatcherNATSCRK8sObjects(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                        string
		givenNATS                   *v1alpha1.NATS
		wantStatefulSetDeletion     bool
		wantConfigMapDeletion       bool
		wantSecretDeletion          bool
		wantServiceDeletion         bool
		wantDestinationRuleDeletion bool
	}{
		{
			name: "should recreate StatefulSet",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			wantStatefulSetDeletion: true,
		},
		{
			name: "should recreate ConfigMap",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			wantConfigMapDeletion: true,
		},
		{
			name: "should recreate Secret",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			wantSecretDeletion: true,
		},
		{
			name: "should recreate Service",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			wantServiceDeletion: true,
		},
		{
			name: "should recreate DestinationRule",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			wantDestinationRuleDeletion: true,
		},
		{
			name: "should recreate all objects",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			wantServiceDeletion:         true,
			wantConfigMapDeletion:       true,
			wantStatefulSetDeletion:     true,
			wantSecretDeletion:          true,
			wantDestinationRuleDeletion: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			// create unique namespace for this test run.
			givenNamespace := tc.givenNATS.GetNamespace()
			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// create NATS CR
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenNATS)

			// ensure all k8s objects exists
			testEnvironment.EnsureK8sStatefulSetExists(t, testutils.GetStatefulSetName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sConfigMapExists(t, testutils.GetConfigMapName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sSecretExists(t, testutils.GetSecretName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sServiceExists(t, testutils.GetServiceName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sDestinationRuleExists(t,
				testutils.GetDestinationRuleName(*tc.givenNATS), givenNamespace)

			// when
			if tc.wantStatefulSetDeletion {
				err := testEnvironment.DeleteStatefulSetFromK8s(testutils.GetStatefulSetName(*tc.givenNATS),
					givenNamespace)
				require.NoError(t, err)
			}
			if tc.wantConfigMapDeletion {
				err := testEnvironment.DeleteConfigMapFromK8s(testutils.GetConfigMapName(*tc.givenNATS),
					givenNamespace)
				require.NoError(t, err)
			}
			if tc.wantSecretDeletion {
				err := testEnvironment.DeleteSecretFromK8s(testutils.GetSecretName(*tc.givenNATS),
					givenNamespace)
				require.NoError(t, err)
			}
			if tc.wantServiceDeletion {
				err := testEnvironment.DeleteServiceFromK8s(testutils.GetServiceName(*tc.givenNATS),
					givenNamespace)
				require.NoError(t, err)
			}
			if tc.wantDestinationRuleDeletion {
				err := testEnvironment.DeleteDestinationRuleFromK8s(testutils.GetDestinationRuleName(*tc.givenNATS),
					givenNamespace)
				require.NoError(t, err)
			}

			// then
			// ensure all k8s objects exists again
			testEnvironment.EnsureK8sStatefulSetExists(t, testutils.GetStatefulSetName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sConfigMapExists(t, testutils.GetConfigMapName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sSecretExists(t, testutils.GetSecretName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sServiceExists(t, testutils.GetServiceName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sDestinationRuleExists(t,
				testutils.GetDestinationRuleName(*tc.givenNATS), givenNamespace)
		})
	}
}

// Test_DoubleReconcileNATSCR tests that controller should be able to reconcile NATS again.
func Test_DoubleReconcileNATSCR(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		givenNATS    *v1alpha1.NATS
		wantMatchers gomegatypes.GomegaMatcher
	}{
		{
			name: "should have reconciled again without problems",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSCRName("test1"),
				testutils.WithNATSLogging(true, true),
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
				testutils.WithNATSFileStorage(v1alpha1.FileStorage{
					StorageClassName: "test-sc1",
					Size:             resource.MustParse("66Gi"),
				}),
				testutils.WithNATSMemStorage(v1alpha1.MemStorage{
					Enabled: true,
					Size:    resource.MustParse("66Gi"),
				}),
			),
			wantMatchers: gomega.And(
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
			givenNamespace := tc.givenNATS.GetNamespace()
			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// first reconcile
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenNATS)

			// check all k8s objects exists.
			testEnvironment.EnsureK8sStatefulSetExists(t, testutils.GetStatefulSetName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sConfigMapExists(t, testutils.GetConfigMapName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sSecretExists(t, testutils.GetSecretName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sServiceExists(t, testutils.GetServiceName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sDestinationRuleExists(t,
				testutils.GetDestinationRuleName(*tc.givenNATS), givenNamespace)

			// check all k8s objects are correctly specified according to NATS CR Spec.
			testEnvironment.EnsureNATSSpecClusterSizeReflected(t, *tc.givenNATS)
			testEnvironment.EnsureNATSSpecResourcesReflected(t, *tc.givenNATS)
			testEnvironment.EnsureNATSSpecDebugTraceReflected(t, *tc.givenNATS)
			testEnvironment.EnsureK8sStatefulSetHasLabels(t, testutils.GetStatefulSetName(*tc.givenNATS),
				givenNamespace, tc.givenNATS.Spec.Labels)
			testEnvironment.EnsureK8sStatefulSetHasAnnotations(t, testutils.GetStatefulSetName(*tc.givenNATS),
				givenNamespace, tc.givenNATS.Spec.Annotations)
			testEnvironment.EnsureNATSSpecMemStorageReflected(t, *tc.givenNATS)
			testEnvironment.EnsureNATSSpecFileStorageReflected(t, *tc.givenNATS)

			// make mock updates to deployed resources.
			makeStatefulSetReady(t, testutils.GetStatefulSetName(*tc.givenNATS), givenNamespace)

			// check NATS CR status.
			testEnvironment.GetNATSAssert(g, tc.givenNATS).Should(tc.wantMatchers)

			// when
			// add label to trigger second reconciliation.
			sts, err := testEnvironment.GetStatefulSetFromK8s(testutils.GetStatefulSetName(*tc.givenNATS),
				givenNamespace)
			require.NoError(t, err)
			sts.Labels = make(map[string]string)
			sts.Labels["test"] = "true"
			testEnvironment.EnsureK8sResourceUpdated(t, sts)

			// wait for any possible changes to take effect.
			time.Sleep(integration.BigPollingInterval)

			// then
			// check again all k8s objects are correctly specified according to NATS CR Spec.
			testEnvironment.EnsureNATSSpecClusterSizeReflected(t, *tc.givenNATS)
			testEnvironment.EnsureNATSSpecResourcesReflected(t, *tc.givenNATS)
			testEnvironment.EnsureNATSSpecDebugTraceReflected(t, *tc.givenNATS)
			testEnvironment.EnsureK8sStatefulSetHasLabels(t, testutils.GetStatefulSetName(*tc.givenNATS),
				givenNamespace, tc.givenNATS.Spec.Labels)
			testEnvironment.EnsureK8sStatefulSetHasAnnotations(t, testutils.GetStatefulSetName(*tc.givenNATS),
				givenNamespace, tc.givenNATS.Spec.Annotations)
			testEnvironment.EnsureNATSSpecMemStorageReflected(t, *tc.givenNATS)
			testEnvironment.EnsureNATSSpecFileStorageReflected(t, *tc.givenNATS)

			// check NATS CR status again.
			testEnvironment.GetNATSAssert(g, tc.givenNATS).Should(tc.wantMatchers)
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
