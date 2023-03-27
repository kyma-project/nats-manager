package chart

import (
	"encoding/json"
	"github.com/kyma-project/nats-manager/testutils"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"

	"github.com/stretchr/testify/require"
)

const (
	testChartName = "component-1"
)

var chartDir = filepath.Join("test", "resources", testChartName)

func Test_getChartConfiguration(t *testing.T) {
	t.Parallel()

	logger, err := testutils.NewTestLogger()
	require.NoError(t, err)
	sugaredLogger := logger.Sugar()

	t.Run("Get chart configurations", func(t *testing.T) {
		t.Parallel()

		// given
		helmRenderer := HelmRenderer{
			chartPath: chartDir,
			logger:    sugaredLogger,
			helmChart: loadHelmChart(t),
		}

		var expected map[string]interface{}
		err = json.Unmarshal([]byte(`{
			"config": {
				"key1": "value1 from values.yaml",
				"key2": "value2 from values.yaml"
			},
			"showKey2": false
		}`), &expected)

		// when
		got := helmRenderer.getChartConfiguration()

		// then
		require.NoError(t, err)
		require.Equal(t, expected, got)
	})
}

func Test_overrideChartConfiguration(t *testing.T) {
	t.Parallel()

	logger, err := testutils.NewTestLogger()
	require.NoError(t, err)
	sugaredLogger := logger.Sugar()

	// define test cases
	testCases := []struct {
		name       string
		overrides  map[string]interface{}
		wantValues []byte
	}{
		{
			name:      "should return default values when no overrides are provided",
			overrides: map[string]interface{}{},
			wantValues: []byte(`{
				"config": {
					"key1": "value1 from values.yaml",
					"key2": "value2 from values.yaml"
				},
				"showKey2": false
			}`),
		},
		{
			name: "should override values as provided",
			overrides: map[string]interface{}{
				"config.key1": "123.4",
				"config.key2": "overridden",
			},
			wantValues: []byte(`{
				"config": {
					"key1": "123.4",
					"key2": "overridden"
				},
				"showKey2": false
			}`),
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			helmRenderer := HelmRenderer{
				chartPath: chartDir,
				logger:    sugaredLogger,
				helmChart: loadHelmChart(t),
			}

			releaseInstance := &ReleaseInstance{
				Configuration: tc.overrides,
			}

			var expected map[string]interface{}
			err = json.Unmarshal(tc.wantValues, &expected)

			// when
			gotValues, err := helmRenderer.overrideChartConfiguration(releaseInstance)

			// then
			require.NoError(t, err)
			require.Equal(t, expected, gotValues)
		})
	}
}

func Test_RenderManifest(t *testing.T) {
	logger, err := testutils.NewTestLogger()
	require.NoError(t, err)
	sugaredLogger := logger.Sugar()

	t.Run("Should render the template as correct string", func(t *testing.T) {
		// given
		helm, err := NewHelmRenderer(chartDir, sugaredLogger)
		require.NoError(t, err)

		releaseInstance := &ReleaseInstance{
			Name:      testChartName,
			Namespace: "test",
			Configuration: map[string]interface{}{
				"config.key2": "value2 from override",
				"showKey2":    true,
			},
		}

		// when
		got, err := helm.RenderManifest(releaseInstance)

		// then
		require.NoError(t, err)
		gotAsMap := make(map[string]interface{})
		require.NoError(t, yaml.Unmarshal([]byte(got), &gotAsMap)) //use for equality check (avoids whitespace diffs)

		expected, err := os.ReadFile(filepath.Join(chartDir, "configmap-expected.yaml"))
		require.NoError(t, err)
		expectedAsMap := make(map[string]interface{})
		require.NoError(t, yaml.Unmarshal(expected, &expectedAsMap)) //use for equality check (avoids whitespace diffs)

		require.Equal(t, expectedAsMap, gotAsMap)
	})
}

func Test_RenderManifestAsUnStructured(t *testing.T) {
	logger, err := testutils.NewTestLogger()
	require.NoError(t, err)
	sugaredLogger := logger.Sugar()

	t.Run("Should render the template as UnStructured", func(t *testing.T) {
		// given
		helm, err := NewHelmRenderer(chartDir, sugaredLogger)
		require.NoError(t, err)

		releaseInstance := &ReleaseInstance{
			Name:      testChartName,
			Namespace: "test",
			Configuration: map[string]interface{}{
				"config.key2": "value2 from override",
				"showKey2":    true,
			},
		}

		expected, err := os.ReadFile(filepath.Join(chartDir, "configmap-expected.yaml"))
		require.NoError(t, err)
		expectedAsMap := make(map[string]interface{})
		require.NoError(t, yaml.Unmarshal(expected, &expectedAsMap)) //use for equality check (avoids whitespace diffs)
		unstructuredObj := unstructured.Unstructured{
			Object: expectedAsMap,
		}

		expectedManifest := ManifestResources{
			Items: []*unstructured.Unstructured{
				&unstructuredObj,
			},
		}

		// when
		gotManifest, err := helm.RenderManifestAsUnStructured(releaseInstance)

		// then
		require.NoError(t, err)
		require.Equal(t, &expectedManifest, gotManifest)
	})
}

func loadHelmChart(t *testing.T) *chart.Chart {
	helmChart, err := loader.Load(chartDir)
	require.NoError(t, err)
	return helmChart
}
