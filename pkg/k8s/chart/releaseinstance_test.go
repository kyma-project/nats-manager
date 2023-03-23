package chart

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_GetConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("Test chart configuration processing", func(t *testing.T) {
		releaseInstance := NewReleaseInstance("main", "unittest-kyma",
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
		}`), &expected) //use marshaller for convenience instead building a nested map by code
		require.NoError(t, err)

		got, err := releaseInstance.GetConfiguration()
		require.NoError(t, err)

		require.Equal(t, expected, got)
	})
}

func Test_convertToNestedMap(t *testing.T) {
	t.Parallel()

	t.Run("Convert dot-notated configuration keys to a nested map", func(t *testing.T) {
		releaseInstance := NewReleaseInstance("main", "unittest-kyma", nil)

		got := releaseInstance.convertToNestedMap("this.is.a.test", "the test value")
		expected := make(map[string]interface{})
		err := json.Unmarshal([]byte(`{
			"this":{
				"is":{
					"a":{
						"test":"the test value"
					}
				}
			}
		}`), &expected) //use marshaller for convenience instead building a nested map by code
		require.NoError(t, err)

		require.Equal(t, expected, got)
	})
}
