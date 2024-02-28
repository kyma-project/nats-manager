package manager

import (
	"fmt"
	"strings"
	"testing"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
	kcorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func Test_GenerateOverrides(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name                string
		givenNATS           *nmapiv1alpha1.NATS
		givenIstioEnabled   bool
		givenRotatePassword bool
		wantOverrides       map[string]interface{}
	}{
		{
			name: "should override when spec values are not provided in spec",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSEmptySpec(),
			),
			givenIstioEnabled:   true,
			givenRotatePassword: true,
			wantOverrides: map[string]interface{}{
				IstioEnabledKey:        true,
				RotatePasswordKey:      true,
				ClusterSizeKey:         0,
				ClusterEnabledKey:      false,
				FileStorageSizeKey:     "0",
				MemStorageEnabledKey:   false,
				DebugEnabledKey:        false,
				TraceEnabledKey:        false,
				ResourceRequestsCPUKey: "0",
				ResourceRequestsMemKey: "0",
				ResourceLimitsCPUKey:   "0",
				ResourceLimitsMemKey:   "0",
			},
		},
		{
			name: "should override when spec values are provided in spec",
			givenNATS: testutils.NewNATSCR(
				testutils.WithNATSClusterSize(5),
				testutils.WithNATSLogging(true, true),
				testutils.WithNATSFileStorage(nmapiv1alpha1.FileStorage{
					Size:             resource.MustParse("15Gi"),
					StorageClassName: "test1",
				}),
				testutils.WithNATSMemStorage(nmapiv1alpha1.MemStorage{
					Enabled: true,
					Size:    resource.MustParse("16Gi"),
				}),
				testutils.WithNATSResources(kcorev1.ResourceRequirements{
					Limits: kcorev1.ResourceList{
						"cpu":    resource.MustParse("999m"),
						"memory": resource.MustParse("999Mi"),
					},
					Requests: kcorev1.ResourceList{
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
		IstioEnabledKey:        true,
		RotatePasswordKey:      true,
		ClusterSizeKey:         float64(3),
		ClusterEnabledKey:      true,
		DebugEnabledKey:        false,
		TraceEnabledKey:        false,
		FileStorageSizeKey:     "1Gi",
		FileStorageClassKey:    "",
		MemStorageEnabledKey:   true,
		MemStorageSizeKey:      "1Gi",
		ResourceRequestsCPUKey: "40m",
		ResourceRequestsMemKey: "64Mi",
		ResourceLimitsCPUKey:   "500m",
		ResourceLimitsMemKey:   "1Gi",
		CommonLabelsKey: map[string]interface{}{
			"app.kubernetes.io/component":  "nats-manager",
			"app.kubernetes.io/created-by": "nats-manager",
			"app.kubernetes.io/managed-by": "nats-manager",
			"app.kubernetes.io/part-of":    "nats-manager",
			"control-plane":                "nats-manager",
		},
		CommonAnnotationsKey: map[string]interface{}{},
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
	t.Helper()
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
