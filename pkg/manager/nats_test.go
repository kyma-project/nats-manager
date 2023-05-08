package manager

import (
	"context"
	"errors"
	"testing"

	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	chartmocks "github.com/kyma-project/nats-manager/pkg/k8s/chart/mocks"
	k8smocks "github.com/kyma-project/nats-manager/pkg/k8s/mocks"
	"github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func Test_GenerateNATSResources(t *testing.T) {
	t.Parallel()

	givenNATSCR := testutils.NewNATSCR()

	// define test cases
	testCases := []struct {
		name         string
		givenOptions []Option
		wantOwnerRef bool
	}{
		{
			name:         "should work with empty options",
			givenOptions: []Option{},
			wantOwnerRef: false,
		},
		{
			name: "should apply the provided options",
			givenOptions: []Option{
				WithOwnerReference(*givenNATSCR),
			},
			wantOwnerRef: true,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			releaseInstance := chart.NewReleaseInstance("test", "test", false, map[string]interface{}{})
			sugaredLogger, err := testutils.NewSugaredLogger()
			require.NoError(t, err)

			manifestResources := &chart.ManifestResources{
				Items: []*unstructured.Unstructured{
					testutils.NewNATSStatefulSetUnStruct(),
				},
			}

			mockHelmRenderer := chartmocks.NewRenderer(t)
			mockHelmRenderer.On("RenderManifestAsUnstructured",
				releaseInstance).Return(manifestResources, nil).Once()

			manager := NewNATSManger(k8smocks.NewClient(t), mockHelmRenderer, sugaredLogger)

			// when
			gotManifests, err := manager.GenerateNATSResources(releaseInstance, tc.givenOptions...)

			// then
			require.NoError(t, err)
			require.Len(t, gotManifests.Items, len(manifestResources.Items))
			if tc.wantOwnerRef {
				unstructuredObj := gotManifests.Items[0]
				require.NotNil(t, unstructuredObj.Object["metadata"])
				metadata, ok := unstructuredObj.Object["metadata"].(map[string]interface{})
				require.True(t, ok)
				require.NotNil(t, metadata["ownerReferences"])
				require.Len(t, metadata["ownerReferences"], 1)
				// match values of owner reference
				ownerReferences, ok := metadata["ownerReferences"].([]map[string]interface{})
				require.True(t, ok)
				require.Equal(t, givenNATSCR.Kind, ownerReferences[0]["kind"])
				require.Equal(t, givenNATSCR.APIVersion, ownerReferences[0]["apiVersion"])
				require.Equal(t, givenNATSCR.Name, ownerReferences[0]["name"])
				require.Equal(t, givenNATSCR.UID, ownerReferences[0]["uid"])
				require.Equal(t, true, ownerReferences[0]["blockOwnerDeletion"])
			}
			// check if all the required mock methods were called.
			mockHelmRenderer.AssertExpectations(t)
		})
	}
}

func Test_DeployInstance(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name      string
		wantError error
	}{
		{
			name:      "should deploy each resource successfully",
			wantError: nil,
		},
		{
			name:      "should fail when k8s fails to deploy resource",
			wantError: errors.New("failed to deploy"),
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			sugaredLogger, err := testutils.NewSugaredLogger()
			require.NoError(t, err)

			releaseInstance := chart.NewReleaseInstance("test", "test",
				false, map[string]interface{}{})
			releaseInstance.SetRenderedManifests(chart.ManifestResources{
				Items: []*unstructured.Unstructured{
					testutils.NewNATSStatefulSetUnStruct(),
					testutils.NewNATSStatefulSetUnStruct(),
					testutils.NewNATSStatefulSetUnStruct(),
				},
			})

			mockKubeClient := k8smocks.NewClient(t)
			if tc.wantError != nil {
				mockKubeClient.On("PatchApply",
					mock.Anything, mock.Anything).Return(tc.wantError)
			} else {
				// should have being called for each ManifestResources.Item
				mockKubeClient.On("PatchApply",
					mock.Anything, mock.Anything).Return(nil).Times(
					len(releaseInstance.RenderedManifests.Items))
			}

			manager := NewNATSManger(mockKubeClient, chartmocks.NewRenderer(t), sugaredLogger)

			// when
			err = manager.DeployInstance(context.Background(), releaseInstance)

			// then
			if tc.wantError != nil {
				require.Error(t, err)
				require.Equal(t, tc.wantError, err)
			} else {
				require.NoError(t, err)
				mockKubeClient.AssertExpectations(t)
			}
		})
	}
}

func Test_DeleteInstance(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name      string
		wantError error
	}{
		{
			name:      "should delete each resource successfully",
			wantError: nil,
		},
		{
			name:      "should fail when k8s fails to delete resource",
			wantError: errors.New("failed to delete"),
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// given
			sugaredLogger, err := testutils.NewSugaredLogger()
			require.NoError(t, err)

			releaseInstance := chart.NewReleaseInstance("test", "test",
				false, map[string]interface{}{})
			releaseInstance.SetRenderedManifests(chart.ManifestResources{
				Items: []*unstructured.Unstructured{
					testutils.NewNATSStatefulSetUnStruct(),
					testutils.NewNATSStatefulSetUnStruct(),
					testutils.NewNATSStatefulSetUnStruct(),
				},
			})

			mockKubeClient := k8smocks.NewClient(t)
			if tc.wantError != nil {
				mockKubeClient.On("Delete",
					mock.Anything, mock.Anything).Return(tc.wantError)
			} else {
				// should have being called for each ManifestResources.Item
				mockKubeClient.On("Delete",
					mock.Anything, mock.Anything).Return(nil).Times(
					len(releaseInstance.RenderedManifests.Items))
			}

			manager := NewNATSManger(mockKubeClient, chartmocks.NewRenderer(t), sugaredLogger)

			// when
			err = manager.DeleteInstance(context.Background(), releaseInstance)

			// then
			if tc.wantError != nil {
				require.Error(t, err)
				require.Equal(t, tc.wantError, err)
			} else {
				require.NoError(t, err)
				mockKubeClient.AssertExpectations(t)
			}
		})
	}
}

func Test_IsNATSStatefulSetReady(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name             string
		givenStatefulSet *unstructured.Unstructured
		wantError        error
		wantIsReady      bool
	}{
		{
			name:             "should return error if no StatefulSet exists in manifests",
			givenStatefulSet: nil,
			wantError:        errors.New("NATS StatefulSet not found in manifests"),
		},
		{
			name: "should return not ready when CurrentReplicas is not as needed",
			givenStatefulSet: testutils.NewNATSStatefulSetUnStruct(
				testutils.WithName("test1"),
				testutils.WithNamespace("test1"),
				testutils.WithSpecReplicas(3),
				testutils.WithStatefulSetStatusCurrentReplicas(1),
				testutils.WithStatefulSetStatusUpdatedReplicas(3),
				testutils.WithStatefulSetStatusReadyReplicas(3),
			),
			wantIsReady: false,
		},
		{
			name: "should return not ready when UpdatedReplicas is not as needed",
			givenStatefulSet: testutils.NewNATSStatefulSetUnStruct(
				testutils.WithName("test1"),
				testutils.WithNamespace("test1"),
				testutils.WithSpecReplicas(3),
				testutils.WithStatefulSetStatusCurrentReplicas(3),
				testutils.WithStatefulSetStatusUpdatedReplicas(1),
				testutils.WithStatefulSetStatusReadyReplicas(3),
			),
			wantIsReady: false,
		},
		{
			name: "should return not ready when ReadyReplicas is not as needed",
			givenStatefulSet: testutils.NewNATSStatefulSetUnStruct(
				testutils.WithName("test1"),
				testutils.WithNamespace("test1"),
				testutils.WithSpecReplicas(3),
				testutils.WithStatefulSetStatusCurrentReplicas(3),
				testutils.WithStatefulSetStatusUpdatedReplicas(3),
				testutils.WithStatefulSetStatusReadyReplicas(1),
			),
			wantIsReady: false,
		},
		{
			name: "should return ready when all replicas are available",
			givenStatefulSet: testutils.NewNATSStatefulSetUnStruct(
				testutils.WithName("test1"),
				testutils.WithNamespace("test1"),
				testutils.WithSpecReplicas(3),
				testutils.WithStatefulSetStatusCurrentReplicas(3),
				testutils.WithStatefulSetStatusUpdatedReplicas(3),
				testutils.WithStatefulSetStatusReadyReplicas(3),
			),
			wantIsReady: true,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			sugaredLogger, err := testutils.NewSugaredLogger()
			require.NoError(t, err)
			// mock for k8s kube client
			mockKubeClient := k8smocks.NewClient(t)

			var items []*unstructured.Unstructured
			if tc.givenStatefulSet != nil {
				items = []*unstructured.Unstructured{
					tc.givenStatefulSet,
				}

				var stsStructObject appsv1.StatefulSet
				err = runtime.DefaultUnstructuredConverter.FromUnstructured(
					tc.givenStatefulSet.UnstructuredContent(), &stsStructObject)
				require.NoError(t, err)

				// set method in mock
				mockKubeClient.On("GetStatefulSet",
					mock.Anything, tc.givenStatefulSet.GetName(), tc.givenStatefulSet.GetNamespace(),
				).Return(&stsStructObject, nil).Once()
			}

			releaseInstance := chart.NewReleaseInstance("test", "test",
				false, map[string]interface{}{})
			releaseInstance.SetRenderedManifests(chart.ManifestResources{
				Items: items,
			})

			manager := NewNATSManger(mockKubeClient, chartmocks.NewRenderer(t), sugaredLogger)

			// when
			isReady, err := manager.IsNATSStatefulSetReady(context.Background(), releaseInstance)

			// then
			if tc.wantError != nil {
				require.Error(t, err)
				require.Equal(t, tc.wantError, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.wantIsReady, isReady)
				mockKubeClient.AssertExpectations(t)
			}
		})
	}
}
