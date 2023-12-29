package label_test

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/kyma-project/nats-manager/internal/label"
)

func TestSelectorInstanceNATS(t *testing.T) {
	// arrange
	wantedSelector := labels.SelectorFromSet(map[string]string{"app.kubernetes.io/instance": "nats-manager"})

	// act
	actualSelector := label.SelectorInstanceNATS()

	// assert
	if !reflect.DeepEqual(wantedSelector, actualSelector) {
		t.Errorf("Expected %v, but got %v", wantedSelector, actualSelector)
	}
}
