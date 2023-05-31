package validation_test

import (
	"os"
	"testing"

	"github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/nats-manager/testutils/integration"
)

const projectRootDir = "../../../../../"

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
			name: `a NATS CR with a spec.cluster.size that is even should return an error`,
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSCRDefaults(),
				testutils.WithNATSClusterSize(4)),
			wantErrMsg: "size only accepts odd numbers",
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
