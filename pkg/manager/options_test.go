package manager

import (
	"github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func Test_WithOwnerReference(t *testing.T) {
	t.Parallel()

	t.Run("Should set owner reference of unstructured k8s object", func(t *testing.T) {
		t.Parallel()

		// given
		natsCR := v1alpha1.Nats{
			// Name, UUID, Kind, APIVersion
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1alpha1",
				Kind:       "Nats",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-object1",
				Namespace: "test-ns1",
				UID:       "1234-5678-1234-5678",
			},
		}
		unstructuredObj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]interface{}{
					"name": "component-1",
				},
			},
		}

		// when
		optionFunc := WithOwnerReference(natsCR)
		err := optionFunc(unstructuredObj)

		// then
		require.NoError(t, err)
		require.NotNil(t, unstructuredObj.Object["metadata"])
		metadata := unstructuredObj.Object["metadata"].(map[string]interface{})
		require.NotNil(t, metadata["ownerReferences"])
		require.Len(t, metadata["ownerReferences"], 1)
		// match values of owner reference
		ownerReferences := metadata["ownerReferences"].([]map[string]interface{})
		require.Equal(t, natsCR.Kind, ownerReferences[0]["kind"])
		require.Equal(t, natsCR.APIVersion, ownerReferences[0]["apiVersion"])
		require.Equal(t, natsCR.Name, ownerReferences[0]["name"])
		require.Equal(t, natsCR.UID, ownerReferences[0]["uid"])
		require.Equal(t, true, ownerReferences[0]["blockOwnerDeletion"])
	})
}
