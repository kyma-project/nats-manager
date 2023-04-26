package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_conditionEquals(t *testing.T) {
	testCases := []struct {
		name            string
		condition1      metav1.Condition
		condition2      metav1.Condition
		wantEqualStatus bool
	}{
		{
			name: "should not be equal if the types are the same but the status is different",
			condition1: metav1.Condition{
				Type: string(ConditionAvailable), Status: metav1.ConditionTrue,
			},

			condition2: metav1.Condition{
				Type: string(ConditionAvailable), Status: metav1.ConditionUnknown,
			},
			wantEqualStatus: false,
		},
		{
			name: "should not be equal if the types are different but the status is the same",
			condition1: metav1.Condition{
				Type: string(ConditionAvailable), Status: metav1.ConditionTrue,
			},

			condition2: metav1.Condition{
				Type: string(ConditionStatefulSet), Status: metav1.ConditionTrue,
			},
			wantEqualStatus: false,
		},
		{
			name: "should not be equal if the message fields are different",
			condition1: metav1.Condition{
				Type: string(ConditionAvailable), Status: metav1.ConditionTrue, Message: "",
			},

			condition2: metav1.Condition{
				Type: string(ConditionAvailable), Status: metav1.ConditionTrue, Message: "some message",
			},
			wantEqualStatus: false,
		},
		{
			name: "should not be equal if the reason fields are different",
			condition1: metav1.Condition{
				Type:   string(ConditionAvailable),
				Status: metav1.ConditionTrue,
				Reason: string(ConditionReasonProcessing),
			},

			condition2: metav1.Condition{
				Type:   string(ConditionAvailable),
				Status: metav1.ConditionTrue,
				Reason: string(ConditionReasonProcessingError),
			},
			wantEqualStatus: false,
		},
		{
			name: "should be equal if all the fields are the same",
			condition1: metav1.Condition{
				Type:    string(ConditionAvailable),
				Status:  metav1.ConditionFalse,
				Reason:  string(ConditionReasonProcessing),
				Message: "nats is not ready",
			},
			condition2: metav1.Condition{
				Type:    string(ConditionAvailable),
				Status:  metav1.ConditionFalse,
				Reason:  string(ConditionReasonProcessing),
				Message: "nats is not ready",
			},
			wantEqualStatus: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			want := tc.wantEqualStatus
			actual := ConditionEquals(tc.condition1, tc.condition2)
			if want != actual {
				t.Errorf("The conditions are not equal, want: %v but got: %v", want, actual)
			}
		})
	}
}

func Test_conditionsEquals(t *testing.T) {
	testCases := []struct {
		name            string
		conditionsSet1  []metav1.Condition
		conditionsSet2  []metav1.Condition
		wantEqualStatus bool
	}{
		{
			name: "should not be equal if the number of conditions are not equal",
			conditionsSet1: []metav1.Condition{
				{Type: string(ConditionAvailable), Status: metav1.ConditionTrue},
			},
			conditionsSet2:  []metav1.Condition{},
			wantEqualStatus: false,
		},
		{
			name: "should be equal if the conditions are the same",
			conditionsSet1: []metav1.Condition{
				{Type: string(ConditionAvailable), Status: metav1.ConditionTrue},
				{Type: string(ConditionStatefulSet), Status: metav1.ConditionTrue},
			},
			conditionsSet2: []metav1.Condition{
				{Type: string(ConditionAvailable), Status: metav1.ConditionTrue},
				{Type: string(ConditionStatefulSet), Status: metav1.ConditionTrue},
			},
			wantEqualStatus: true,
		},
		{
			name: "should not be equal if the condition types are different",
			conditionsSet1: []metav1.Condition{
				{Type: string(ConditionAvailable), Status: metav1.ConditionTrue},
			},
			conditionsSet2: []metav1.Condition{
				{Type: string(ConditionStatefulSet), Status: metav1.ConditionTrue},
			},
			wantEqualStatus: false,
		},
		{
			name: "should not be equal if the condition types are the same but the status is different",
			conditionsSet1: []metav1.Condition{
				{Type: string(ConditionAvailable), Status: metav1.ConditionTrue},
			},
			conditionsSet2: []metav1.Condition{
				{Type: string(ConditionAvailable), Status: metav1.ConditionFalse},
			},
			wantEqualStatus: false,
		},
		{
			name: "should not be equal if the condition types are different but the status is the same",
			conditionsSet1: []metav1.Condition{
				{Type: string(ConditionAvailable), Status: metav1.ConditionTrue},
			},
			conditionsSet2: []metav1.Condition{
				{Type: string(ConditionStatefulSet), Status: metav1.ConditionTrue},
			},
			wantEqualStatus: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			want := tc.wantEqualStatus
			actual := ConditionsEquals(tc.conditionsSet1, tc.conditionsSet2)
			if actual != want {
				t.Errorf("The list of conditions are not equal, want: %v but got: %v", want, actual)
			}
		})
	}
}

func Test_IsValidResourceQuantity(t *testing.T) {
	validQuatity := k8sresource.MustParse("256Mi")

	testCases := []struct {
		name          string
		givenQuantity *k8sresource.Quantity
		wantResult    bool
	}{
		{
			name:          "should return false when quantity is nil",
			givenQuantity: nil,
			wantResult:    false,
		},
		{
			name:          "should return true when quantity is valid",
			givenQuantity: &validQuatity,
			wantResult:    true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantResult, IsValidResourceQuantity(tc.givenQuantity))
		})
	}
}
