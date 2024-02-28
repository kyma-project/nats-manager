package controller_test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	keventsv1 "k8s.io/api/events/v1"

	"github.com/onsi/gomega"

	onsigomegatypes "github.com/onsi/gomega/types"
	"github.com/stretchr/testify/require"
	kcorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	nmctrl "github.com/kyma-project/nats-manager/internal/controller/nats"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/kyma-project/nats-manager/testutils/integration"
	nmtsmatchers "github.com/kyma-project/nats-manager/testutils/matchers/nats"
)

const projectRootDir = "../../../../../"

var testEnvironment *integration.TestEnvironment //nolint:gochecknoglobals // used in tests

// TestMain pre-hook and post-hook to run before and after all tests.
func TestMain(m *testing.M) {
	// Note: The setup will provision a single K8s env and
	// all the tests need to create and use a separate namespace

	// setup env test
	var err error
	testEnvironment, err = integration.NewTestEnvironment(projectRootDir, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	// run tests
	code := m.Run()

	// tear down test env
	if err = testEnvironment.TearDown(); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

func Test_CreateNATSCR(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                  string
		givenNATS             *nmapiv1alpha1.NATS
		givenK8sEvents        keventsv1.EventList
		givenStatefulSetReady bool
		wantMatches           onsigomegatypes.GomegaMatcher
		wantEventMatches      onsigomegatypes.GomegaMatcher
		wantEnsureK8sObjects  bool
	}{
		{
			name: "NATS CR should have processing status when StatefulSet is not ready",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			givenStatefulSetReady: false,
			wantMatches: gomega.And(
				nmtsmatchers.HaveStatusProcessing(),
				nmtsmatchers.HavePendingConditionStatefulSet(),
				nmtsmatchers.HaveDeployingConditionAvailable(),
			),
			wantEventMatches: gomega.And(
				nmtsmatchers.HaveProcessingEvent(),
			),
		},
		{
			name: "NATS CR should have ready status when StatefulSet is ready",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			givenStatefulSetReady: true,
			wantMatches: gomega.And(
				nmtsmatchers.HaveStatusReady(),
				nmtsmatchers.HaveReadyConditionStatefulSet(),
				nmtsmatchers.HaveReadyConditionAvailable(),
			),
			wantEventMatches: gomega.And(
				nmtsmatchers.HaveProcessingEvent(),
				nmtsmatchers.HaveDeployingEvent(),
			),
		},
		{
			name: "should have created k8s objects as specified in NATS CR",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSLogging(true, true),
				testutils.WithNATSResources(kcorev1.ResourceRequirements{
					Limits: kcorev1.ResourceList{
						"cpu":    resource.MustParse("199m"),
						"memory": resource.MustParse("199Mi"),
					},
					Requests: kcorev1.ResourceList{
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
				testutils.WithNATSFileStorage(nmapiv1alpha1.FileStorage{
					StorageClassName: "test-sc1",
					Size:             resource.MustParse("66Gi"),
				}),
				testutils.WithNATSMemStorage(nmapiv1alpha1.MemStorage{
					Enabled: true,
					Size:    resource.MustParse("66Gi"),
				}),
			),
			givenStatefulSetReady: true,
			wantMatches: gomega.And(
				nmtsmatchers.HaveStatusReady(),
				nmtsmatchers.HaveReadyConditionStatefulSet(),
				nmtsmatchers.HaveReadyConditionAvailable(),
			),
			wantEnsureK8sObjects: true,
			wantEventMatches: gomega.And(
				nmtsmatchers.HaveProcessingEvent(),
				nmtsmatchers.HaveDeployingEvent(),
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

			// when
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenNATS)

			// then
			testEnvironment.EnsureK8sStatefulSetExists(t, testutils.GetStatefulSetName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sConfigMapExists(t, testutils.GetConfigMapName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sSecretExists(t, testutils.GetSecretName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sServiceExists(t, testutils.GetServiceName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sPodDisruptionBudgetExists(t,
				testutils.GetPodDisruptionBudgetName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sDestinationRuleExists(t,
				testutils.GetDestinationRuleName(*tc.givenNATS), givenNamespace)

			if tc.givenStatefulSetReady {
				// make mock updates to deployed resources.
				makeStatefulSetReady(t, testutils.GetStatefulSetName(*tc.givenNATS), givenNamespace)
			}

			// check NATS CR status.
			testEnvironment.GetNATSAssert(g, tc.givenNATS).Should(tc.wantMatches)

			// check kubernetes events.
			testEnvironment.GetK8sEventsAssert(g, tc.givenNATS).Should(tc.wantEventMatches)

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

			// check the url in the NATS CR status
			testEnvironment.EnsureURLInNATSStatus(t, tc.givenNATS.Name, givenNamespace)
		})
	}
}

// Test_UpdateNATSCR tests if updating the NATS CR will trigger reconciliation
// and k8s objects are updated accordingly.
func Test_UpdateNATSCR(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name            string
		givenNATS       *nmapiv1alpha1.NATS
		givenUpdateNATS *nmapiv1alpha1.NATS
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
				testutils.WithNATSResources(kcorev1.ResourceRequirements{
					Limits: kcorev1.ResourceList{
						"cpu":    resource.MustParse("199m"),
						"memory": resource.MustParse("199Mi"),
					},
					Requests: kcorev1.ResourceList{
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
				testutils.WithNATSMemStorage(nmapiv1alpha1.MemStorage{
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

			// check the url in the NATS CR status
			testEnvironment.EnsureURLInNATSStatus(t, tc.givenNATS.Name, givenNamespace)
		})
	}
}

func Test_DeleteNATSCR(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		givenNATS *nmapiv1alpha1.NATS
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
				testutils.WithNATSResources(kcorev1.ResourceRequirements{
					Limits: kcorev1.ResourceList{
						"cpu":    resource.MustParse("199m"),
						"memory": resource.MustParse("199Mi"),
					},
					Requests: kcorev1.ResourceList{
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
				testutils.WithNATSFileStorage(nmapiv1alpha1.FileStorage{
					StorageClassName: "test-sc1",
					Size:             resource.MustParse("66Gi"),
				}),
				testutils.WithNATSMemStorage(nmapiv1alpha1.MemStorage{
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

			if !*testEnvironment.EnvTestInstance.UseExistingCluster {
				// create a PVC as local envtest cluster cannot create PVCs.
				pvc := testutils.NewPVC(tc.givenNATS.Name, givenNamespace,
					map[string]string{nmctrl.InstanceLabelKey: tc.givenNATS.Name})
				testEnvironment.EnsureK8sResourceCreated(t, pvc)
			}

			testEnvironment.EnsureK8sPVCExists(t, tc.givenNATS.Name, givenNamespace)

			// when
			testEnvironment.EnsureK8sResourceDeleted(t, tc.givenNATS)

			// then
			// we expect the other resources are deleted by k8s garbage collector.
			// ensure PVC is deleted
			testEnvironment.EnsureK8sPVCNotFound(t, tc.givenNATS.Name, givenNamespace)

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
		name                 string
		givenNATS            *nmapiv1alpha1.NATS
		wantResourceDeletion []deletionFunc
	}{
		{
			name: "should recreate StatefulSet",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			wantResourceDeletion: []deletionFunc{
				deleteStatefulSetFromK8s,
			},
		},
		{
			name: "should recreate ConfigMap",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			wantResourceDeletion: []deletionFunc{
				deleteConfigMapFromK8s,
			},
		},
		{
			name: "should recreate Secret",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			wantResourceDeletion: []deletionFunc{
				deleteSecretFromK8s,
			},
		},
		{
			name: "should recreate Service",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			wantResourceDeletion: []deletionFunc{
				deleteServiceFromK8s,
			},
		},
		{
			name: "should recreate DestinationRule",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			wantResourceDeletion: []deletionFunc{
				deleteDestinationRuleFromK8s,
			},
		},
		{
			name: "should recreate PodDisruptionBudget",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			wantResourceDeletion: []deletionFunc{
				deletePodDisruptionBudgetFromK8s,
			},
		},
		{
			name: "should recreate all objects",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
			),
			wantResourceDeletion: []deletionFunc{
				deleteServiceFromK8s,
				deleteConfigMapFromK8s,
				deleteStatefulSetFromK8s,
				deleteSecretFromK8s,
				deleteDestinationRuleFromK8s,
			},
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
			testEnvironment.EnsureK8sPodDisruptionBudgetExists(t, testutils.GetPodDisruptionBudgetName(*tc.givenNATS),
				givenNamespace)
			testEnvironment.EnsureK8sDestinationRuleExists(t,
				testutils.GetDestinationRuleName(*tc.givenNATS), givenNamespace)

			// when
			ensureK8sResourceDeletion(t, *testEnvironment, tc.givenNATS.GetName(), givenNamespace)

			// then
			// ensure all k8s objects exists again
			testEnvironment.EnsureK8sStatefulSetExists(t, testutils.GetStatefulSetName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sConfigMapExists(t, testutils.GetConfigMapName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sSecretExists(t, testutils.GetSecretName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sServiceExists(t, testutils.GetServiceName(*tc.givenNATS), givenNamespace)
			testEnvironment.EnsureK8sPodDisruptionBudgetExists(t, testutils.GetPodDisruptionBudgetName(*tc.givenNATS),
				givenNamespace)
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
		givenNATS    *nmapiv1alpha1.NATS
		wantMatchers onsigomegatypes.GomegaMatcher
	}{
		{
			name: "should have reconciled again without problems",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSCRName("test1"),
				testutils.WithNATSLogging(true, true),
				testutils.WithNATSResources(kcorev1.ResourceRequirements{
					Limits: kcorev1.ResourceList{
						"cpu":    resource.MustParse("199m"),
						"memory": resource.MustParse("199Mi"),
					},
					Requests: kcorev1.ResourceList{
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
				testutils.WithNATSFileStorage(nmapiv1alpha1.FileStorage{
					StorageClassName: "test-sc1",
					Size:             resource.MustParse("66Gi"),
				}),
				testutils.WithNATSMemStorage(nmapiv1alpha1.MemStorage{
					Enabled: true,
					Size:    resource.MustParse("66Gi"),
				}),
			),
			wantMatchers: gomega.And(
				nmtsmatchers.HaveStatusReady(),
				nmtsmatchers.HaveReadyConditionStatefulSet(),
				nmtsmatchers.HaveReadyConditionAvailable(),
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

			// check the url in the NATS CR status
			testEnvironment.EnsureURLInNATSStatus(t, tc.givenNATS.Name, givenNamespace)
		})
	}
}

func makeStatefulSetReady(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		sts, err := testEnvironment.GetStatefulSetFromK8s(name, namespace)
		if err != nil {
			testEnvironment.Logger.Errorw("failed to get statefulSet", "error", err)
			return false
		}

		sts.Status.Replicas = *sts.Spec.Replicas
		sts.Status.AvailableReplicas = *sts.Spec.Replicas
		sts.Status.CurrentReplicas = *sts.Spec.Replicas
		sts.Status.ReadyReplicas = *sts.Spec.Replicas
		sts.Status.UpdatedReplicas = *sts.Spec.Replicas

		err = testEnvironment.UpdateStatefulSetStatusOnK8s(*sts)
		if err != nil {
			testEnvironment.Logger.Errorw("failed to update statefulSet status", "error", err)
			return false
		}
		return true
	}, integration.SmallTimeOut, integration.SmallPollingInterval, "failed to update status of StatefulSet")
}

type deletionFunc func(env integration.TestEnvironment, natsName, namespace string) error

func ensureK8sResourceDeletion(
	t *testing.T, env integration.TestEnvironment, natsName, namespace string, fs ...deletionFunc,
) {
	for _, f := range fs {
		require.NoError(t, f(env, natsName, namespace))
	}
}

func deleteStatefulSetFromK8s(env integration.TestEnvironment, natsName, namespace string) error {
	stsName := fmt.Sprintf(testutils.StatefulSetNameFormat, natsName)
	return env.DeleteStatefulSetFromK8s(stsName, namespace)
}

func deleteServiceFromK8s(env integration.TestEnvironment, natsName, namespace string) error {
	svcName := fmt.Sprintf(testutils.ServiceNameFormat, natsName)
	return env.DeleteServiceFromK8s(svcName, namespace)
}

func deleteConfigMapFromK8s(env integration.TestEnvironment, natsName, namespace string) error {
	cmName := fmt.Sprintf(testutils.ConfigMapNameFormat, natsName)
	return env.DeleteConfigMapFromK8s(cmName, namespace)
}

func deleteSecretFromK8s(env integration.TestEnvironment, natsName, namespace string) error {
	secName := fmt.Sprintf(testutils.SecretNameFormat, natsName)
	return env.DeleteSecretFromK8s(secName, namespace)
}

func deleteDestinationRuleFromK8s(env integration.TestEnvironment, natsName, namespace string) error {
	destName := fmt.Sprintf(testutils.DestinationRuleNameFormat, natsName)
	return env.DeleteDestinationRuleFromK8s(destName, namespace)
}

func deletePodDisruptionBudgetFromK8s(env integration.TestEnvironment, natsName, namespace string) error {
	destName := fmt.Sprintf(testutils.PodDisruptionBudgetNameFormat, natsName)
	return env.DeletePodDisruptionBudgetFromK8s(destName, namespace)
}
