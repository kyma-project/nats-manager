package labels

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/labels"
)

func TestSelectorInstanceNATS(t *testing.T) {
	// arrange
	wantedSelector := labels.SelectorFromSet(map[string]string{"app.kubernetes.io/instance": "nats-manager"})

	// act
	actualSelector := SelectorInstanceNATS()

	// assert
	if !reflect.DeepEqual(wantedSelector, actualSelector) {
		t.Errorf("Expected %v, but got %v", wantedSelector, actualSelector)
	}
}

func TestSelectorCreatedByNATS(t *testing.T) {
	// arrange
	wantedSelector := labels.SelectorFromSet(map[string]string{"app.kubernetes.io/created-by": "nats-manager"})

	// act
	actualSelector := SelectorCreatedByNATS()

	// assert
	if !reflect.DeepEqual(wantedSelector, actualSelector) {
		t.Errorf("Expected %v, but got %v", wantedSelector, actualSelector)
	}
}

func TestSelectorManagedByNATS(t *testing.T) {
	// arrange
	wantedSelector := labels.SelectorFromSet(map[string]string{"app.kubernetes.io/managed-by": "nats-manager"})

	// act
	actualSelector := SelectorManagedByNATS()

	// assert
	if !reflect.DeepEqual(wantedSelector, actualSelector) {
		t.Errorf("Expected %v, but got %v", wantedSelector, actualSelector)
	}
}
