package chart

import (
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os"
	"path/filepath"
	"testing"
)

func Test_ParseManifestStringToObjects(t *testing.T) {
	t.Run("Should parse the template as object", func(t *testing.T) {
		// given
		manifestString, err := os.ReadFile(filepath.Join(chartDir, "configmap-expected.yaml"))
		require.NoError(t, err)

		unstructuredObj := unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "component-1",
				},
				"data": map[string]interface{}{
					"key1": "value1 from values.yaml",
					"key2": "value2 from override",
				},
			},
		}

		expectedManifest := ManifestResources{
			Items: []*unstructured.Unstructured{
				&unstructuredObj,
			},
		}

		// when
		gotManifest, err := ParseManifestStringToObjects(string(manifestString))

		// then
		require.NoError(t, err)
		require.Equal(t, &expectedManifest, gotManifest)
	})
}
