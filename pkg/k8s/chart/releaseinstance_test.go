package chart

import (
	"encoding/json"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/stretchr/testify/require"
)

func Test_GetConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("Test chart configuration processing", func(t *testing.T) {
		t.Parallel()
		// given
		releaseInstance := NewReleaseInstance("main", "unittest-kyma", false,
			map[string]interface{}{
				"test.key1.subkey1": "test value 1",
				"test.key1.subkey2": "test value 2",
				"test.key2.subkey1": "test value 3",
				"test.key2.subkey2": "test value 4",
			})

		expected := make(map[string]interface{})
		err := json.Unmarshal([]byte(`{
			"test":{
				"key1":{
					"subkey1":"test value 1",
					"subkey2":"test value 2"
				},
				"key2":{
					"subkey1":"test value 3",
					"subkey2":"test value 4"
				}
			}
		}`), &expected) // use marshaller for convenience instead building a nested map by code
		require.NoError(t, err)

		// when
		got, err := releaseInstance.GetConfiguration()

		// then
		require.NoError(t, err)
		require.Equal(t, expected, got)
	})
}

func Test_SetRenderedManifests(t *testing.T) {
	t.Parallel()

	t.Run("Should set the rendered manifests", func(t *testing.T) {
		t.Parallel()
		// given
		releaseInstance := NewReleaseInstance("main", "unittest-kyma",
			false, map[string]interface{}{})

		sampleManifests := ManifestResources{
			Items: []*unstructured.Unstructured{
				{
					Object: map[string]interface{}{
						"name": "test",
					},
				},
			},
			Blobs: [][]byte{},
		}

		require.NotEqual(t, sampleManifests, releaseInstance.RenderedManifests)

		// when
		releaseInstance.SetRenderedManifests(sampleManifests)

		// then
		require.Equal(t, sampleManifests, releaseInstance.RenderedManifests)
	})
}

func Test_convertToNestedMap(t *testing.T) {
	t.Parallel()

	t.Run("Convert dot-notated configuration keys to a nested map", func(t *testing.T) {
		t.Parallel()
		// given
		releaseInstance := NewReleaseInstance("main", "unittest-kyma", false, nil)

		got, err := releaseInstance.convertToNestedMap("this.is.a.test", "the test value")
		require.NoError(t, err)
		expected := make(map[string]interface{})
		err = json.Unmarshal([]byte(`{
			"this":{
				"is":{
					"a":{
						"test":"the test value"
					}
				}
			}
		}`), &expected) // use marshaller for convenience instead building a nested map by code
		require.NoError(t, err)

		// when, then
		require.Equal(t, expected, got)
	})
}

func Test_GetStatefulSets(t *testing.T) {
	t.Parallel()

	sampleSTS := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "StatefulSet",
		},
	}

	sampleObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "Deployment",
		},
	}

	// define test cases
	testCases := []struct {
		name             string
		manifests        ManifestResources
		wantResultLength int
		wantResult       []*unstructured.Unstructured
	}{
		{
			name:             "should not find StatefulSet when items is empty",
			manifests:        ManifestResources{},
			wantResultLength: 0,
			wantResult:       []*unstructured.Unstructured(nil),
		},
		{
			name: "should not find StatefulSet when items does not contain statefulSet",
			manifests: ManifestResources{
				Items: []*unstructured.Unstructured{
					sampleObj,
				},
			},
			wantResultLength: 0,
			wantResult:       []*unstructured.Unstructured(nil),
		},
		{
			name: "should find StatefulSet when items does not contains statefulSet",
			manifests: ManifestResources{
				Items: []*unstructured.Unstructured{
					sampleObj,
					sampleSTS,
				},
			},
			wantResultLength: 1,
			wantResult: []*unstructured.Unstructured{
				sampleSTS,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			releaseInstance := NewReleaseInstance("main", "test",
				false, map[string]interface{}{})
			releaseInstance.SetRenderedManifests(tc.manifests)

			// when
			result := releaseInstance.GetStatefulSets()

			// then
			require.Len(t, result, tc.wantResultLength)
			require.Equal(t, tc.wantResult, result)
		})
	}
}
