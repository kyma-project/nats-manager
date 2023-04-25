package manager

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func Test_GenerateOverrides(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name                string
		givenNATS           *v1alpha1.NATS
		givenIstioEnabled   bool
		givenRotatePassword bool
		wantOverrides       map[string]interface{}
	}{
		{
			name: "should not override when spec values are not provided",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSEmptySpec(),
			),
			givenIstioEnabled:   true,
			givenRotatePassword: true,
			wantOverrides: map[string]interface{}{
				IstioEnabledKey:      true,
				RotatePasswordKey:    true,
				ClusterSizeKey:       0,
				ClusterEnabledKey:    false,
				MemStorageEnabledKey: false,
				DebugEnabledKey:      false,
				TraceEnabledKey:      false,
			},
		},
		{
			name: "should override when spec values are provided",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSClusterSize(5),
				testutils.WithNATSLogging(true, true),
				testutils.WithNATSFileStorage(v1alpha1.FileStorage{
					Size:             "15Gi", //nolint:typecheck // used in tests.
					StorageClassName: "test1",
				}),
				testutils.WithNATSMemStorage(v1alpha1.MemStorage{
					Enable: true,
					Size:   "16Gi", //nolint:typecheck // used in tests.
				}),
				testutils.WithNATSResources(corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						"cpu":    resource.MustParse("999m"),
						"memory": resource.MustParse("999Mi"),
					},
					Requests: corev1.ResourceList{
						"cpu":    resource.MustParse("919m"),
						"memory": resource.MustParse("919Mi"),
					},
				}),
				testutils.WithNATSLabels(map[string]string{
					"key1": "value1",
				}),
				testutils.WithNATSAnnotations(map[string]string{
					"key2": "value2",
				}),
			),
			givenIstioEnabled:   true,
			givenRotatePassword: true,
			wantOverrides: map[string]interface{}{
				IstioEnabledKey:        true,
				RotatePasswordKey:      true,
				ClusterSizeKey:         5,
				ClusterEnabledKey:      true,
				DebugEnabledKey:        true,
				TraceEnabledKey:        true,
				FileStorageClassKey:    "test1",
				FileStorageSizeKey:     "15Gi",
				MemStorageEnabledKey:   true,
				MemStorageSizeKey:      "16Gi",
				ResourceRequestsCPUKey: "919m",
				ResourceRequestsMemKey: "919Mi",
				ResourceLimitsCPUKey:   "999m",
				ResourceLimitsMemKey:   "999Mi",
				CommonLabelsKey: map[string]string{
					"key1": "value1",
				},
				CommonAnnotationsKey: map[string]string{
					"key2": "value2",
				},
			},
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			manager := NewNATSManger(nil, nil, nil)

			// when
			overrides := manager.GenerateOverrides(&tc.givenNATS.Spec, tc.givenIstioEnabled, tc.givenRotatePassword)

			// then
			require.Equal(t, tc.wantOverrides, overrides)
		})
	}
}

// Test_Overrides_Keys checks if the keys are correct as they are defined in actual NATS helm chart.
func Test_Overrides_Keys(t *testing.T) {
	t.Parallel()

	// given
	natsChartDir := "../../resources/nats"
	helmChart, err := loader.Load(natsChartDir)
	require.NoError(t, err)

	// these are the default values as defined in NATS helm chart.
	keysToTest := map[string]interface{}{
		IstioEnabledKey:        false,
		RotatePasswordKey:      true,
		ClusterSizeKey:         float64(1),
		ClusterEnabledKey:      false,
		DebugEnabledKey:        true,
		TraceEnabledKey:        true,
		FileStorageSizeKey:     "1Gi",
		FileStorageClassKey:    "",
		MemStorageEnabledKey:   true,
		MemStorageSizeKey:      "1Gi",
		ResourceRequestsCPUKey: "5m",
		ResourceRequestsMemKey: "16Mi",
		ResourceLimitsCPUKey:   "20m",
		ResourceLimitsMemKey:   "64Mi",
		CommonLabelsKey:        map[string]interface{}{},
		CommonAnnotationsKey:   map[string]interface{}{},
	}

	// run test cases
	for key := range keysToTest {
		key := key
		t.Run(fmt.Sprintf("Testing key: %s", key), func(t *testing.T) {
			t.Parallel()

			// when
			gotValue := getValueFromNestedMap(t, key, helmChart.Values)
			require.Equal(t, keysToTest[key], gotValue)
		})
	}
}

func getValueFromNestedMap(t *testing.T, key string, data map[string]interface{}) interface{} {
	tokens := strings.Split(key, ".")
	lastNestedData := data
	for depth, token := range tokens {
		switch depth {
		case len(tokens) - 1: // last token reached, stop nesting
			return lastNestedData[token]
		default:
			var ok bool
			lastNestedData, ok = lastNestedData[token].(map[string]interface{})
			require.True(t, ok, "failed to convert to map[string]interface{}")
		}
	}
	return nil
}
