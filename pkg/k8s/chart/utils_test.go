package chart

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func Test_IsStatefulSetObject(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name        string
		givenObject unstructured.Unstructured
		wantResult  bool
	}{
		{
			name: "should return false when it is not StatefulSet",
			givenObject: unstructured.Unstructured{
				Object: map[string]any{
					"kind": "Deployment",
				},
			},
			wantResult: false,
		},
		{
			name: "should return true when it is StatefulSet",
			givenObject: unstructured.Unstructured{
				Object: map[string]any{
					"kind": "StatefulSet",
				},
			},
			wantResult: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.wantResult, IsStatefulSetObject(tc.givenObject))
		})
	}
}

func Test_ParseManifestStringToObjects(t *testing.T) {
	t.Parallel()
	t.Run("Should parse the template as object", func(t *testing.T) {
		t.Parallel()
		// given
		manifestString, err := os.ReadFile(filepath.Join(testChartDir, "configmap-expected.yaml"))
		require.NoError(t, err)

		unstructuredObj := unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]any{
					"name": "component-1",
				},
				"data": map[string]any{
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
