package controller_test

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/kyma-project/nats-manager/testutils/integration"
	natsmatchers "github.com/kyma-project/nats-manager/testutils/matchers/nats"
	"github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
)

const projectRootDir = "../../../../../"

var testEnvironment *integration.TestEnvironment //nolint:gochecknoglobals // used in tests

// define allowed NATS CR.
//
//nolint:gochecknoglobals // used in tests
var givenAllowedNATS = testutils.NewNATSCR(
	testutils.WithNATSCRName("eventing-nats"),
	testutils.WithNATSCRNamespace("kyma-system"),
)

// TestMain pre-hook and post-hook to run before and after all tests.
func TestMain(m *testing.M) {
	// Note: The setup will provision a single K8s env and
	// all the tests need to create and use a separate namespace

	// setup env test
	var err error
	testEnvironment, err = integration.NewTestEnvironment(projectRootDir, false, givenAllowedNATS)
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

// Test_PreventMultipleNATSCRs tests that only single NATS CR is allowed to be reconciled in a kyma cluster.
func Test_PreventMultipleNATSCRs(t *testing.T) {
	t.Parallel()

	errMsg := fmt.Sprintf("Only a single NATS CR with name: %s and namespace: %s is allowed"+
		"to be created in a Kyma cluster.", "eventing-nats",
		"kyma-system")

	testCases := []struct {
		name        string
		givenNATS   *v1alpha1.NATS
		wantMatches gomegatypes.GomegaMatcher
	}{
		{
			name: "should allow NATS CR if name and namespace is correct",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSCRName(givenAllowedNATS.Name),
				testutils.WithNATSCRNamespace(givenAllowedNATS.Namespace),
			),
			wantMatches: gomega.And(
				natsmatchers.HaveStatusProcessing(),
				natsmatchers.HavePendingConditionStatefulSet(),
				natsmatchers.HaveDeployingConditionAvailable(),
			),
		},
		{
			name: "should not allow NATS CR if name is incorrect",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSCRName("not-allowed-name"),
				testutils.WithNATSCRNamespace("kyma-system"),
			),
			wantMatches: gomega.And(
				natsmatchers.HaveStatusError(),
				natsmatchers.HaveForbiddenConditionStatefulSet(),
				natsmatchers.HaveForbiddenConditionAvailableWithMsg(errMsg),
			),
		},
		{
			name: "should not allow NATS CR if namespace is incorrect",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSCRName("eventing-nats"),
				testutils.WithNATSCRNamespace("not-allowed-namespace"),
			),
			wantMatches: gomega.And(
				natsmatchers.HaveStatusError(),
				natsmatchers.HaveForbiddenConditionStatefulSet(),
				natsmatchers.HaveForbiddenConditionAvailableWithMsg(errMsg),
			),
		},
		{
			name: "should not allow NATS CR if name and namespace, both are incorrect",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSCRName("not-allowed-name"),
				testutils.WithNATSCRNamespace("not-allowed-namespace"),
			),
			wantMatches: gomega.And(
				natsmatchers.HaveStatusError(),
				natsmatchers.HaveForbiddenConditionStatefulSet(),
				natsmatchers.HaveForbiddenConditionAvailableWithMsg(errMsg),
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
			// check NATS CR status.
			testEnvironment.GetNATSAssert(g, tc.givenNATS).Should(tc.wantMatches)
		})
	}
}
