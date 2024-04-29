package k8s

import (
	"context"
	"errors"
	"testing"

	"github.com/kyma-project/nats-manager/testutils"
	"github.com/stretchr/testify/require"
	kcorev1 "k8s.io/api/core/v1"
	kapiextclientsetfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	kapierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ktypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testFieldManager = "nats-manager"

var (
	errPatchNotAllowed = errors.New("apply patches are not supported in the fake client")
	errNotFound        = errors.New("not found")
)

func Test_GetStatefulSet(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name              string
		givenStatefulSet  *unstructured.Unstructured
		wantNotFoundError bool
	}{
		{
			name:              "should return not found error when StatefulSet is missing in k8s",
			givenStatefulSet:  testutils.NewNATSStatefulSetUnStruct(),
			wantNotFoundError: true,
		},
		{
			name:              "should return correct StatefulSet from k8s",
			givenStatefulSet:  testutils.NewNATSStatefulSetUnStruct(),
			wantNotFoundError: false,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			fakeClientBuilder := fake.NewClientBuilder()

			var objs []client.Object
			if !tc.wantNotFoundError {
				objs = append(objs, tc.givenStatefulSet)
			}
			fakeClient := fakeClientBuilder.WithObjects(objs...).Build()

			kubeClient := NewKubeClient(fakeClient, nil, testFieldManager)

			// when
			gotSTS, err := kubeClient.GetStatefulSet(context.Background(),
				tc.givenStatefulSet.GetName(), tc.givenStatefulSet.GetNamespace())

			// then
			if tc.wantNotFoundError {
				require.Error(t, err)
				require.True(t, kapierrors.IsNotFound(err))
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.givenStatefulSet.GetName(), gotSTS.Name)
				require.Equal(t, tc.givenStatefulSet.GetNamespace(), gotSTS.Namespace)
			}
		})
	}
}

func Test_GetSecret(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name              string
		givenSecret       *unstructured.Unstructured
		wantNotFoundError bool
	}{
		{
			name:              "should return not found error when Secret is missing in k8s",
			givenSecret:       testutils.NewSecretUnStruct(),
			wantNotFoundError: true,
		},
		{
			name:              "should return correct Secret from k8s",
			givenSecret:       testutils.NewSecretUnStruct(),
			wantNotFoundError: false,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			var objs []client.Object
			if !tc.wantNotFoundError {
				objs = append(objs, tc.givenSecret)
			}
			fakeClientBuilder := fake.NewClientBuilder()
			fakeClient := fakeClientBuilder.WithObjects(objs...).Build()
			kubeClient := NewKubeClient(fakeClient, nil, testFieldManager)

			// when
			gotSecret, err := kubeClient.GetSecret(context.Background(),
				tc.givenSecret.GetName(), tc.givenSecret.GetNamespace())

			// then
			if tc.wantNotFoundError {
				require.Error(t, err)
				require.True(t, kapierrors.IsNotFound(err))
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.givenSecret.GetName(), gotSecret.Name)
				require.Equal(t, tc.givenSecret.GetNamespace(), gotSecret.Namespace)
			}
		})
	}
}

func Test_Delete(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name                    string
		givenStatefulSet        *unstructured.Unstructured
		givenStatefulSetCreated bool
	}{
		{
			name:                    "should delete existing resource from k8s",
			givenStatefulSet:        testutils.NewNATSStatefulSetUnStruct(),
			givenStatefulSetCreated: true,
		},
		{
			name:                    "should delete non-existing resource from k8s",
			givenStatefulSet:        testutils.NewNATSStatefulSetUnStruct(),
			givenStatefulSetCreated: false,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			var objs []client.Object
			if !tc.givenStatefulSetCreated {
				objs = append(objs, tc.givenStatefulSet)
			}
			fakeClientBuilder := fake.NewClientBuilder()
			fakeClient := fakeClientBuilder.WithObjects(objs...).Build()
			kubeClient := NewKubeClient(fakeClient, nil, testFieldManager)

			// when
			err := kubeClient.Delete(context.Background(), tc.givenStatefulSet)

			// then
			require.NoError(t, err)
			// check that it should not exist on k8s.
			gotSTS, err := kubeClient.GetStatefulSet(context.Background(),
				tc.givenStatefulSet.GetName(), tc.givenStatefulSet.GetNamespace())
			require.Error(t, err)
			require.True(t, kapierrors.IsNotFound(err))
			require.Nil(t, gotSTS)
		})
	}
}

func Test_PatchApply(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name                   string
		givenStatefulSet       *unstructured.Unstructured
		givenUpdateStatefulSet *unstructured.Unstructured
	}{
		{
			name: "should update resource when exists in k8s",
			givenStatefulSet: testutils.NewNATSStatefulSetUnStruct(
				testutils.WithSpecReplicas(1),
			),
			givenUpdateStatefulSet: testutils.NewNATSStatefulSetUnStruct(
				testutils.WithSpecReplicas(3),
			),
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			var objs []client.Object
			if tc.givenStatefulSet != nil {
				objs = append(objs, tc.givenStatefulSet)
			}
			fakeClientBuilder := fake.NewClientBuilder()
			fakeClient := fakeClientBuilder.WithObjects(objs...).Build()
			kubeClient := NewKubeClient(fakeClient, nil, testFieldManager)

			// when
			err := kubeClient.PatchApply(context.Background(), tc.givenUpdateStatefulSet)

			// then
			// NOTE: The kubeClient.PatchApply is not supported in the fake client.
			// (https://github.com/kubernetes/kubernetes/issues/115598)
			// So in unit test we only check that the client.Patch with client.Apply
			// is called or not.
			// The real behaviour will be tested in integration tests with envTest pkg.
			require.ErrorContains(t, err, errPatchNotAllowed.Error())
		})
	}
}

func Test_GetCRD(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name              string
		givenCRDName      string
		wantNotFoundError bool
	}{
		{
			name:              "should return not found error when CRD is missing in k8s",
			givenCRDName:      DestinationRuleCrdName,
			wantNotFoundError: false,
		},
		{
			name:              "should return correct CRD from k8s",
			givenCRDName:      "non-existing",
			wantNotFoundError: true,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			sampleCRD := testutils.NewDestinationRuleCRD()
			var objs []runtime.Object
			if !tc.wantNotFoundError {
				objs = append(objs, sampleCRD)
			}

			fakeClientSet := kapiextclientsetfake.NewSimpleClientset(objs...)
			kubeClient := NewKubeClient(nil, fakeClientSet, testFieldManager)

			// when
			gotCRD, err := kubeClient.GetCRD(context.Background(), tc.givenCRDName)

			// then
			if tc.wantNotFoundError {
				require.Error(t, err)
				require.True(t, kapierrors.IsNotFound(err))
			} else {
				require.NoError(t, err)
				require.Equal(t, sampleCRD.GetName(), gotCRD.Name)
			}
		})
	}
}

func Test_DestinationRuleCRDExists(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name       string
		wantResult bool
	}{
		{
			name:       "should return false when CRD is missing in k8s",
			wantResult: false,
		},
		{
			name:       "should return true when CRD exists in k8s",
			wantResult: true,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			sampleCRD := testutils.NewDestinationRuleCRD()
			var objs []runtime.Object
			if tc.wantResult {
				objs = append(objs, sampleCRD)
			}

			fakeClientSet := kapiextclientsetfake.NewSimpleClientset(objs...)
			kubeClient := NewKubeClient(nil, fakeClientSet, testFieldManager)

			// when
			gotResult, err := kubeClient.DestinationRuleCRDExists(context.Background())

			// then
			require.NoError(t, err)
			require.Equal(t, tc.wantResult, gotResult)
		})
	}
}

func Test_DeletePVCsWithLabel(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		mustHaveNamePrefix string
		labelSelector      string
		namespace          string
		givenPVC           *kcorev1.PersistentVolumeClaim
		wantNotFoundErr    bool
	}{
		{
			name:               "should delete PVCs with matching label and name prefix",
			mustHaveNamePrefix: "my",
			labelSelector:      "app=myapp",
			namespace:          "mynamespace",
			givenPVC:           testutils.NewPVC("mypvc", "mynamespace", map[string]string{"app": "myapp"}),
			wantNotFoundErr:    true,
		},
		{
			name:     "should do nothing if no PVC exists",
			givenPVC: nil,
		},
		{
			name:          "should not delete PVCs with non-matching label",
			labelSelector: "app=myapp",
			namespace:     "mynamespace",
			givenPVC:      testutils.NewPVC("mypvc", "mynamespace", map[string]string{"app": "notmyapp"}),
		},
		{
			name:          "should not delete PVCs in different namespace",
			labelSelector: "app=myapp",
			namespace:     "mynamespace",
			givenPVC:      testutils.NewPVC("mypvc", "othernamespace", map[string]string{"app": "myapp"}),
		},
		{
			name:          "should not delete PVCs if none match label",
			labelSelector: "app=myapp",
			namespace:     "mynamespace",
			givenPVC:      testutils.NewPVC("mypvc", "mynamespace", map[string]string{"app": "notmyapp"}),
		},
		{
			name:               "should not delete PVCs if mustHaveNamePrefix is not matched",
			labelSelector:      "app=myapp",
			mustHaveNamePrefix: "app=notmy",
			namespace:          "mynamespace",
			givenPVC:           testutils.NewPVC("mypvc", "mynamespace", map[string]string{"app": "myapp"}),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			var objs []client.Object
			if tc.givenPVC != nil {
				objs = append(objs, tc.givenPVC)
			}
			fakeClientBuilder := fake.NewClientBuilder()
			fakeClient := fakeClientBuilder.WithObjects(objs...).Build()
			kubeClient := NewKubeClient(fakeClient, nil, testFieldManager)

			// when
			err := kubeClient.DeletePVCsWithLabel(context.Background(), tc.labelSelector, tc.mustHaveNamePrefix, tc.namespace)

			// then
			require.NoError(t, err)
			// no need to execute following checks if no PVCs were given
			if tc.givenPVC == nil {
				return
			}
			// check that the PVCs were deleted
			err = fakeClient.Get(context.Background(),
				ktypes.NamespacedName{Name: tc.givenPVC.Name, Namespace: tc.givenPVC.Namespace}, tc.givenPVC)
			if tc.wantNotFoundErr {
				require.True(t, kapierrors.IsNotFound(err))
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_GetNode(t *testing.T) {
	t.Parallel()

	// define test cases
	testCases := []struct {
		name              string
		givenNode         *unstructured.Unstructured
		wantNotFoundError bool
	}{
		{
			name:              "should return not found error when Node is missing in k8s",
			givenNode:         testutils.NewNodeUnStruct(),
			wantNotFoundError: true,
		},
		{
			name:              "should return correct Node from k8s",
			givenNode:         testutils.NewNodeUnStruct(),
			wantNotFoundError: false,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			fakeClientBuilder := fake.NewClientBuilder()

			var objs []client.Object
			if !tc.wantNotFoundError {
				objs = append(objs, tc.givenNode)
			}
			fakeClient := fakeClientBuilder.WithObjects(objs...).Build()

			kubeClient := NewKubeClient(fakeClient, nil, testFieldManager)

			// when
			gotNode, err := kubeClient.GetNode(context.Background(), tc.givenNode.GetName())

			// then
			if tc.wantNotFoundError {
				require.Error(t, err)
				require.True(t, kapierrors.IsNotFound(err))
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.givenNode.GetName(), gotNode.Name)
			}
		})
	}
}

func Test_GetNodeZone(t *testing.T) {
	t.Parallel()

	givenLabels := map[string]string{nodeZoneLabelKey: "east-us-1"}

	// define test cases
	testCases := []struct {
		name               string
		givenNode          *unstructured.Unstructured
		givenNodeExists    bool
		givenExistsInCache bool
		wantZone           string
		wantError          error
	}{
		{
			name:            "should return not found error when Node is missing in k8s",
			givenNode:       testutils.NewNodeUnStruct(),
			givenNodeExists: false,
			wantError:       errNotFound,
		},
		{
			name:            "should return zone label missing error when Node do have the zone label",
			givenNode:       testutils.NewNodeUnStruct(), // zone label is not set.
			givenNodeExists: true,
			wantError:       ErrNodeZoneLabelMissing,
		},
		{
			name:               "should return correct Node Zone from k8s when cache is empty",
			givenNode:          testutils.NewNodeUnStruct(testutils.WithLabels(givenLabels)),
			givenNodeExists:    true,
			givenExistsInCache: false,
			wantZone:           givenLabels[nodeZoneLabelKey],
		},
		{
			name:               "should return correct Node Zone from cache",
			givenNode:          testutils.NewNodeUnStruct(), // zone label is not set, so the value should come from cache.
			givenNodeExists:    false,
			givenExistsInCache: true,
			wantZone:           givenLabels[nodeZoneLabelKey],
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			fakeClientBuilder := fake.NewClientBuilder()

			var objs []client.Object
			if tc.givenNodeExists {
				objs = append(objs, tc.givenNode)
			}
			fakeClient := fakeClientBuilder.WithObjects(objs...).Build()

			kubeClient := NewKubeClient(fakeClient, nil, testFieldManager)

			if tc.givenExistsInCache {
				kcStruct, ok := kubeClient.(*KubeClient)
				require.True(t, ok)
				kcStruct.nodesZoneCache[tc.givenNode.GetName()] = givenLabels[nodeZoneLabelKey]
			}

			// when
			gotNodeZone, err := kubeClient.GetNodeZone(context.Background(), tc.givenNode.GetName())

			// then
			if tc.wantError != nil {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantError.Error())
				return
			}

			require.NoError(t, err)
			require.Equal(t, givenLabels[nodeZoneLabelKey], gotNodeZone)

			// check cache entry.
			kcStruct, ok := kubeClient.(*KubeClient)
			require.True(t, ok)
			gotValue, ok := kcStruct.nodesZoneCache[tc.givenNode.GetName()]
			require.True(t, ok)
			require.Equal(t, tc.wantZone, gotValue)
		})
	}
}

func Test_GetPodsByLabels(t *testing.T) {
	t.Parallel()

	// given
	givenNamespace := "test-namespace1"
	givenLabels := map[string]string{"app.kubernetes.io/managed-by": "nats-manager"}

	givenPods := []client.Object{
		testutils.NewPodUnStruct(
			testutils.WithName("pod1"),
			testutils.WithNamespace(givenNamespace),
			testutils.WithLabels(givenLabels)),
		testutils.NewPodUnStruct(
			testutils.WithName("pod2"),
			testutils.WithNamespace(givenNamespace),
			testutils.WithLabels(givenLabels)),
		testutils.NewPodUnStruct(
			testutils.WithName("pod3"),
			testutils.WithNamespace(givenNamespace),
			// no labels in pod3.
		),
	}

	fakeClientBuilder := fake.NewClientBuilder()
	fakeClient := fakeClientBuilder.WithObjects(givenPods...).Build()
	kubeClient := NewKubeClient(fakeClient, nil, testFieldManager)

	// when
	gotPodList, err := kubeClient.GetPodsByLabels(context.Background(), givenNamespace, givenLabels)

	// then
	require.NoError(t, err)
	// should return only the pods with the given labels.
	require.Len(t, gotPodList.Items, 2)
}

func Test_GetNumberOfAvailabilityZonesUsedByPods(t *testing.T) {
	t.Parallel()

	givenNamespace := "test-namespace1"
	givenPodLabels := map[string]string{"app.kubernetes.io/managed-by": "nats-manager"}

	givenNodes := []client.Object{
		testutils.NewNodeUnStruct(
			testutils.WithName("node1"),
			testutils.WithLabels(map[string]string{nodeZoneLabelKey: "east-us-1"}),
		),
		testutils.NewNodeUnStruct(
			testutils.WithName("node2"),
			testutils.WithLabels(map[string]string{nodeZoneLabelKey: "east-us-2"}),
		),
		testutils.NewNodeUnStruct(
			testutils.WithName("node3"),
			testutils.WithLabels(map[string]string{nodeZoneLabelKey: "east-us-3"}),
		),
	}

	// define test cases
	testCases := []struct {
		name           string
		givenPods      []client.Object
		wantZonesCount int
	}{
		{
			name: "should return 3 when all 3 pods are in different zones",
			givenPods: []client.Object{
				testutils.NewPodUnStruct(
					testutils.WithName("pod1"),
					testutils.WithNamespace(givenNamespace),
					testutils.WithLabels(givenPodLabels),
					testutils.WithSpecNodeName("node1"),
				),
				testutils.NewPodUnStruct(
					testutils.WithName("pod2"),
					testutils.WithNamespace(givenNamespace),
					testutils.WithLabels(givenPodLabels),
					testutils.WithSpecNodeName("node2"),
				),
				testutils.NewPodUnStruct(
					testutils.WithName("pod3"),
					testutils.WithNamespace(givenNamespace),
					testutils.WithLabels(givenPodLabels),
					testutils.WithSpecNodeName("node3"),
				),
			},
			wantZonesCount: 3,
		},
		{
			name: "should return 1 when all 3 pods are in same zone",
			givenPods: []client.Object{
				testutils.NewPodUnStruct(
					testutils.WithName("pod1"),
					testutils.WithNamespace(givenNamespace),
					testutils.WithLabels(givenPodLabels),
					testutils.WithSpecNodeName("node1"),
				),
				testutils.NewPodUnStruct(
					testutils.WithName("pod2"),
					testutils.WithNamespace(givenNamespace),
					testutils.WithLabels(givenPodLabels),
					testutils.WithSpecNodeName("node1"),
				),
				testutils.NewPodUnStruct(
					testutils.WithName("pod3"),
					testutils.WithNamespace(givenNamespace),
					testutils.WithLabels(givenPodLabels),
					testutils.WithSpecNodeName("node1"),
				),
			},
			wantZonesCount: 1,
		},
		{
			name: "should return 2 when 2 of 3 pods are in same zone",
			givenPods: []client.Object{
				testutils.NewPodUnStruct(
					testutils.WithName("pod1"),
					testutils.WithNamespace(givenNamespace),
					testutils.WithLabels(givenPodLabels),
					testutils.WithSpecNodeName("node1"),
				),
				testutils.NewPodUnStruct(
					testutils.WithName("pod2"),
					testutils.WithNamespace(givenNamespace),
					testutils.WithLabels(givenPodLabels),
					testutils.WithSpecNodeName("node1"),
				),
				testutils.NewPodUnStruct(
					testutils.WithName("pod3"),
					testutils.WithNamespace(givenNamespace),
					testutils.WithLabels(givenPodLabels),
					testutils.WithSpecNodeName("node3"),
				),
			},
			wantZonesCount: 2,
		},
	}

	// run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// given
			fakeClientBuilder := fake.NewClientBuilder()

			var objs []client.Object
			objs = append(objs, givenNodes...)
			objs = append(objs, tc.givenPods...)
			fakeClient := fakeClientBuilder.WithObjects(objs...).Build()

			kubeClient := NewKubeClient(fakeClient, nil, testFieldManager)

			// when
			gotZonesCount, err := kubeClient.GetNumberOfAvailabilityZonesUsedByPods(context.Background(),
				givenNamespace, givenPodLabels)

			// then
			require.NoError(t, err)
			require.Equal(t, tc.wantZonesCount, gotZonesCount)
		})
	}
}
