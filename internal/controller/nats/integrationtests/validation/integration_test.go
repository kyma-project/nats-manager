package validation_test

import (
	"os"
	"testing"

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

func Test_ValidateOddNumber(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{
			name: "Validation test 1",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Test run 123")
		})
	}
}
