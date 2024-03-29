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

var errPatchNotAllowed = errors.New("apply patches are not supported in the fake client")

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
