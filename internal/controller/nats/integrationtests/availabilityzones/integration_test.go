package controller_test

import (
	"fmt"
	"log"
	"os"
	"testing"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/kyma-project/nats-manager/testutils/integration"
	nmtsmatchers "github.com/kyma-project/nats-manager/testutils/matchers/nats"
	"github.com/onsi/gomega"
	onsigomegatypes "github.com/onsi/gomega/types"
	"github.com/stretchr/testify/require"
	keventsv1 "k8s.io/api/events/v1"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	projectRootDir   = "../../../../../"
	nodeZoneLabelKey = "topology.kubernetes.io/zone"
)

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

	// envtest do not have any nodes created by default.
	// create k8s Nodes.
	givenNodes := []client.Object{
		testutils.NewNodeUnStruct(
			testutils.WithName("node1"),
			testutils.WithLabels(map[string]string{nodeZoneLabelKey: "east-us-1"}),
		),
		testutils.NewNodeUnStruct(
			testutils.WithName("node2"),
			testutils.WithLabels(map[string]string{nodeZoneLabelKey: "east-us-2"}),
		),
		testutils.NewNodeUnStruct(
			testutils.WithName("node3"),
			testutils.WithLabels(map[string]string{nodeZoneLabelKey: "east-us-3"}),
		),
		testutils.NewNodeUnStruct(
			testutils.WithName("node4"),
			testutils.WithLabels(map[string]string{nodeZoneLabelKey: "east-us-4"}),
		),
		testutils.NewNodeUnStruct(
			testutils.WithName("node5"),
			testutils.WithLabels(map[string]string{nodeZoneLabelKey: "east-us-5"}),
		),
	}

	nodesList, err := testEnvironment.GetNodesFromK8s()
	if err != nil {
		log.Fatal(err)
	}
	if len(nodesList.Items) == 0 {
		for _, node := range givenNodes {
			err := testEnvironment.CreateK8sResource(node)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	// run tests
	code := m.Run()

	// tear down test env
	if err = testEnvironment.TearDown(); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

func Test_DifferentAvailabilityZones(t *testing.T) {
	givenNATSPodLabels := map[string]string{
		"app.kubernetes.io/instance": "eventing",
		"app.kubernetes.io/name":     "nats",
		"kyma-project.io/dashboard":  "eventing",
	}

	// these tests should not run in parallel as the created NATS Pods are shared across tests.
	testCases := []struct {
		name                  string
		givenNATS             *nmapiv1alpha1.NATS
		givenNATSPodsNodes    []string
		givenK8sEvents        keventsv1.EventList
		givenStatefulSetReady bool
		wantMatches           onsigomegatypes.GomegaMatcher
		wantEventMatches      onsigomegatypes.GomegaMatcher
		wantEnsureK8sObjects  bool
	}{
		{
			name: "ConditionAvailabilityZones should have false status when spec.cluster.size < 3",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSClusterSize(1),
			),
			givenStatefulSetReady: true,
			wantMatches: gomega.And(
				nmtsmatchers.HaveStatusReady(),
				nmtsmatchers.HaveCondition(kmetav1.Condition{
					Type:               string(nmapiv1alpha1.ConditionAvailabilityZones),
					Status:             kmetav1.ConditionFalse,
					LastTransitionTime: kmetav1.Now(),
					Reason:             string(nmapiv1alpha1.ConditionReasonNotConfigured),
					Message:            "NATS is not configured to run in cluster mode (i.e. spec.cluster.size < 3).",
				}),
			),
		},
		{
			name: "ConditionAvailabilityZones should have pending status when statefulset is not ready",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSClusterSize(3),
			),
			givenStatefulSetReady: false,
			wantMatches: gomega.And(
				nmtsmatchers.HaveStatusProcessing(),
				nmtsmatchers.HaveCondition(kmetav1.Condition{
					Type:               string(nmapiv1alpha1.ConditionAvailabilityZones),
					Status:             kmetav1.ConditionFalse,
					LastTransitionTime: kmetav1.Now(),
					Reason:             string(nmapiv1alpha1.ConditionReasonStatefulSetPending),
					Message:            "",
				}),
			),
		},
		{
			name: "ConditionAvailabilityZones should have true status when availability zones == 3",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSClusterSize(3),
			),
			givenStatefulSetReady: true,
			givenNATSPodsNodes:    []string{"node1", "node2", "node3"},
			wantMatches: gomega.And(
				nmtsmatchers.HaveStatusReady(),
				nmtsmatchers.HaveCondition(kmetav1.Condition{
					Type:               string(nmapiv1alpha1.ConditionAvailabilityZones),
					Status:             kmetav1.ConditionTrue,
					LastTransitionTime: kmetav1.Now(),
					Reason:             string(nmapiv1alpha1.ConditionReasonDeployed),
					Message:            "NATS is deployed in different availability zones.",
				}),
			),
		},
		{
			name: "ConditionAvailabilityZones should have false status when availability zones < 3",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSClusterSize(3),
			),
			givenStatefulSetReady: true,
			givenNATSPodsNodes:    []string{"node1", "node2", "node2"},
			wantMatches: gomega.And(
				nmtsmatchers.HaveStatusWarning(),
				nmtsmatchers.HaveCondition(kmetav1.Condition{
					Type:               string(nmapiv1alpha1.ConditionAvailabilityZones),
					Status:             kmetav1.ConditionFalse,
					LastTransitionTime: kmetav1.Now(),
					Reason:             string(nmapiv1alpha1.ConditionReasonUnknown),
					Message: "NATS is not currently using enough availability " +
						"zones (Recommended: 3, current: 2).",
				}),
			),
		},
		{
			name: "ConditionAvailabilityZones should have true status when availability zones > 3",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSClusterSize(5),
			),
			givenStatefulSetReady: true,
			givenNATSPodsNodes:    []string{"node1", "node2", "node3", "node4", "node5"},
			wantMatches: gomega.And(
				nmtsmatchers.HaveStatusReady(),
				nmtsmatchers.HaveCondition(kmetav1.Condition{
					Type:               string(nmapiv1alpha1.ConditionAvailabilityZones),
					Status:             kmetav1.ConditionTrue,
					LastTransitionTime: kmetav1.Now(),
					Reason:             string(nmapiv1alpha1.ConditionReasonDeployed),
					Message:            "NATS is deployed in different availability zones.",
				}),
			),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// these tests should not run in parallel.

			// given
			g := gomega.NewGomegaWithT(t)

			// create unique namespace for this test run.
			givenNamespace := tc.givenNATS.GetNamespace()
			testEnvironment.EnsureNamespaceCreation(t, givenNamespace)

			// when
			testEnvironment.EnsureK8sResourceCreated(t, tc.givenNATS)

			// then
			testEnvironment.EnsureK8sStatefulSetExists(t, testutils.GetStatefulSetName(*tc.givenNATS), givenNamespace)

			// envtest do not create Pods for a Statefulset, so we need to create them manually.
			for i, nodeName := range tc.givenNATSPodsNodes {
				newPod := testutils.NewPodUnStruct(
					testutils.WithName(fmt.Sprintf("pod-%d", i)),
					testutils.WithNamespace(givenNamespace),
					testutils.WithLabels(givenNATSPodLabels),
					testutils.WithSpecNodeName(nodeName),
				)
				require.NoError(t, testEnvironment.CreateK8sResource(newPod))
			}
			// clean up pods, so they do not conflict with other test cases.
			defer func() {
				for i := range len(tc.givenNATSPodsNodes) {
					newPod := testutils.NewPodUnStruct(
						testutils.WithName(fmt.Sprintf("pod-%d", i)),
						testutils.WithNamespace(givenNamespace),
					)
					testEnvironment.EnsureK8sResourceDeleted(t, newPod)
				}
			}()

			if tc.givenStatefulSetReady {
				// make mock updates to deployed resources.
				makeStatefulSetReady(t, testutils.GetStatefulSetName(*tc.givenNATS), givenNamespace)
			}

			// check NATS CR status.
			testEnvironment.GetNATSAssert(g, tc.givenNATS).Should(tc.wantMatches)
		})
	}
}

func makeStatefulSetReady(t *testing.T, name, namespace string) {
	t.Helper()
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
