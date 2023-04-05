package v1alpha1

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_NATSIsEqual(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name            string
		natsStatus1     NATSStatus
		natsStatus2     NATSStatus
		wantEqualStatus bool
	}{
		{
			name: "should not be equal if the conditions are not equal",
			natsStatus1: NATSStatus{
				Conditions: []metav1.Condition{
					{Type: string(ConditionAvailable), Status: metav1.ConditionTrue},
				},
				State: StateReady,
			},
			natsStatus2: NATSStatus{
				Conditions: []metav1.Condition{
					{Type: string(ConditionAvailable), Status: metav1.ConditionFalse},
				},
				State: StateReady,
			},
			wantEqualStatus: false,
		},
		{
			name: "should not be equal if the ready status is not equal",
			natsStatus1: NATSStatus{
				Conditions: []metav1.Condition{
					{Type: string(ConditionAvailable), Status: metav1.ConditionTrue},
				},
				State: StateReady,
			},
			natsStatus2: NATSStatus{
				Conditions: []metav1.Condition{
					{Type: string(ConditionAvailable), Status: metav1.ConditionTrue},
				},
				State: StateProcessing,
			},
			wantEqualStatus: false,
		},
		{
			name: "should be equal if all the fields are equal",
			natsStatus1: NATSStatus{
				Conditions: []metav1.Condition{
					{Type: string(ConditionAvailable), Status: metav1.ConditionTrue},
				},
				State: StateReady,
			},
			natsStatus2: NATSStatus{
				Conditions: []metav1.Condition{
					{Type: string(ConditionAvailable), Status: metav1.ConditionTrue},
				},
				State: StateReady,
			},
			wantEqualStatus: true,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotEqualStatus := tc.natsStatus1.IsEqual(tc.natsStatus2)
			require.Equal(t, tc.wantEqualStatus, gotEqualStatus)
		})
	}
}

func Test_FindCondition(t *testing.T) {
	currentTime := metav1.NewTime(time.Now())

	testCases := []struct {
		name              string
		givenConditions   []metav1.Condition
		findConditionType ConditionType
		wantCondition     *metav1.Condition
	}{
		{
			name: "should be able to find the present condition",
			givenConditions: []metav1.Condition{
				{
					Type:               string(ConditionAvailable),
					Status:             metav1.ConditionTrue,
					LastTransitionTime: currentTime,
				}, {
					Type:               string(ConditionStatefulSet),
					Status:             metav1.ConditionTrue,
					LastTransitionTime: currentTime,
				},
			},
			findConditionType: ConditionAvailable,
			wantCondition: &metav1.Condition{
				Type:               string(ConditionAvailable),
				Status:             metav1.ConditionTrue,
				LastTransitionTime: currentTime,
			},
		},
		{
			name: "should not be able to find the non-present condition",
			givenConditions: []metav1.Condition{
				{
					Type:               string(ConditionStatefulSet),
					Status:             metav1.ConditionTrue,
					LastTransitionTime: currentTime,
				},
			},
			findConditionType: ConditionAvailable,
			wantCondition:     nil,
		},
	}

	status := NATSStatus{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status.Conditions = tc.givenConditions
			gotCondition := status.FindCondition(tc.findConditionType)

			if !reflect.DeepEqual(tc.wantCondition, gotCondition) {
				t.Errorf("FindCondition failed, want: %v but got: %v", tc.wantCondition, gotCondition)
			}
		})
	}
}

func Test_UpdateConditionStatefulSet(t *testing.T) {
	t.Parallel()

	t.Run("should update the StatefulSet condition", func(t *testing.T) {
		t.Parallel()

		// given
		natsStatus1 := &NATSStatus{
			Conditions: []metav1.Condition{
				{
					Type:    string(ConditionStatefulSet),
					Status:  metav1.ConditionFalse,
					Reason:  "",
					Message: "",
				},
			},
			State: StateReady,
		}

		givenStatus := metav1.ConditionTrue
		givenReason := ConditionReasonProcessing
		givenMessage := "test123"

		// when
		natsStatus1.UpdateConditionStatefulSet(givenStatus, givenReason, givenMessage)

		// then
		gotCondition := natsStatus1.Conditions[0]
		require.Equal(t, string(ConditionStatefulSet), gotCondition.Type)
		require.Equal(t, givenStatus, gotCondition.Status)
		require.Equal(t, string(givenReason), gotCondition.Reason)
		require.Equal(t, givenMessage, gotCondition.Message)
	})
}

func Test_UpdateConditionAvailable(t *testing.T) {
	t.Parallel()

	t.Run("should update the Available condition", func(t *testing.T) {
		t.Parallel()

		// given
		natsStatus1 := &NATSStatus{
			Conditions: []metav1.Condition{
				{
					Type:    string(ConditionAvailable),
					Status:  metav1.ConditionFalse,
					Reason:  "",
					Message: "",
				},
			},
			State: StateReady,
		}

		givenStatus := metav1.ConditionTrue
		givenReason := ConditionReasonProcessing
		givenMessage := "test123"

		// when
		natsStatus1.UpdateConditionAvailable(givenStatus, givenReason, givenMessage)

		// then
		gotCondition := natsStatus1.Conditions[0]
		require.Equal(t, string(ConditionAvailable), gotCondition.Type)
		require.Equal(t, givenStatus, gotCondition.Status)
		require.Equal(t, string(givenReason), gotCondition.Reason)
		require.Equal(t, givenMessage, gotCondition.Message)
	})
}

func Test_SetStateReady(t *testing.T) {
	t.Parallel()

	t.Run("should update the state", func(t *testing.T) {
		t.Parallel()

		// given
		natsStatus1 := &NATSStatus{
			State: StateError,
		}

		// when
		natsStatus1.SetStateReady()

		// then
		require.Equal(t, StateReady, natsStatus1.State)
	})
}

func Test_SetStateProcessing(t *testing.T) {
	t.Parallel()

	t.Run("should update the state", func(t *testing.T) {
		t.Parallel()

		// given
		natsStatus1 := &NATSStatus{
			State: StateError,
		}

		// when
		natsStatus1.SetStateProcessing()

		// then
		require.Equal(t, StateProcessing, natsStatus1.State)
	})
}

func Test_SetStateError(t *testing.T) {
	t.Parallel()

	t.Run("should update the state", func(t *testing.T) {
		t.Parallel()

		// given
		natsStatus1 := &NATSStatus{
			State: StateProcessing,
		}

		// when
		natsStatus1.SetStateError()

		// then
		require.Equal(t, StateError, natsStatus1.State)
	})
}

func Test_SetStateDeleting(t *testing.T) {
	t.Parallel()

	t.Run("should update the state", func(t *testing.T) {
		t.Parallel()

		// given
		natsStatus1 := &NATSStatus{
			State: StateError,
		}

		// when
		natsStatus1.SetStateDeleting()

		// then
		require.Equal(t, StateDeleting, natsStatus1.State)
	})
}

func Test_SetStateStatefulSetWaiting(t *testing.T) {
	t.Parallel()

	t.Run("should update the condition", func(t *testing.T) {
		t.Parallel()

		currentTime := metav1.NewTime(time.Now())

		// given
		natsStatus1 := &NATSStatus{
			State: StateError,
		}

		expectedSTSCondition := &metav1.Condition{
			Type:               string(ConditionStatefulSet),
			Status:             metav1.ConditionFalse,
			Reason:             string(ConditionReasonStatefulSetPending),
			Message:            "Waiting",
			LastTransitionTime: currentTime,
		}

		expectedAvailableCondition := &metav1.Condition{
			Type:               string(ConditionAvailable),
			Status:             metav1.ConditionFalse,
			Reason:             string(ConditionReasonDeploying),
			Message:            "",
			LastTransitionTime: currentTime,
		}

		// when
		natsStatus1.SetStateStatefulSetWaiting()

		// then
		require.Equal(t, StateProcessing, natsStatus1.State)
		// compare ConditionStatefulSet
		stsCondition := natsStatus1.FindCondition(ConditionStatefulSet)
		require.NotNil(t, stsCondition)
		stsCondition.LastTransitionTime = currentTime
		require.Equal(t, expectedSTSCondition, stsCondition)

		// compare ConditionAvailable
		availableCondition := natsStatus1.FindCondition(ConditionAvailable)
		require.NotNil(t, availableCondition)
		availableCondition.LastTransitionTime = currentTime
		require.Equal(t, expectedAvailableCondition, availableCondition)
	})
}

func Test_Initialize(t *testing.T) {
	t.Parallel()

	t.Run("should update the condition", func(t *testing.T) {
		t.Parallel()

		currentTime := metav1.NewTime(time.Now())

		// given
		natsStatus1 := &NATSStatus{
			State: StateError,
		}

		expectedSTSCondition := &metav1.Condition{
			Type:               string(ConditionStatefulSet),
			Status:             metav1.ConditionFalse,
			Reason:             string(ConditionReasonProcessing),
			Message:            "",
			LastTransitionTime: currentTime,
		}

		expectedAvailableCondition := &metav1.Condition{
			Type:               string(ConditionAvailable),
			Status:             metav1.ConditionFalse,
			Reason:             string(ConditionReasonProcessing),
			Message:            "",
			LastTransitionTime: currentTime,
		}

		// when
		natsStatus1.Initialize()

		// then
		require.Equal(t, StateProcessing, natsStatus1.State)
		// compare ConditionStatefulSet
		stsCondition := natsStatus1.FindCondition(ConditionStatefulSet)
		require.NotNil(t, stsCondition)
		stsCondition.LastTransitionTime = currentTime
		require.Equal(t, expectedSTSCondition, stsCondition)

		// compare ConditionAvailable
		availableCondition := natsStatus1.FindCondition(ConditionAvailable)
		require.NotNil(t, availableCondition)
		availableCondition.LastTransitionTime = currentTime
		require.Equal(t, expectedAvailableCondition, availableCondition)
	})
}
