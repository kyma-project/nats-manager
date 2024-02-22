package cache

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	nmlabels "github.com/kyma-project/nats-manager/pkg/labels"
	"github.com/stretchr/testify/require"
	kappsv1 "k8s.io/api/apps/v1"
	kautoscalingv1 "k8s.io/api/autoscaling/v1"
	kcorev1 "k8s.io/api/core/v1"
	kapipolicyv1 "k8s.io/api/policy/v1"
	krbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_applySelectors(t *testing.T) {
	// given
	syncPeriod := 30 * time.Second
	selector := cache.ByObject{
		Label: labels.SelectorFromSet(
			map[string]string{
				nmlabels.KeyManagedBy: nmlabels.ValueNATSManager,
			},
		),
	}

	type args struct {
		options cache.Options
	}
	testCases := []struct {
		name string
		args args
		want cache.Options
	}{
		{
			name: "should apply the correct selectors",
			args: args{
				options: cache.Options{},
			},
			want: cache.Options{
				ByObject: map[client.Object]cache.ByObject{
					&kappsv1.Deployment{}:                     selector,
					&kappsv1.StatefulSet{}:                    selector,
					&kcorev1.ServiceAccount{}:                 selector,
					&kcorev1.Secret{}:                         selector,
					&kcorev1.Service{}:                        selector,
					&kcorev1.ConfigMap{}:                      selector,
					&krbacv1.ClusterRole{}:                    selector,
					&krbacv1.ClusterRoleBinding{}:             selector,
					&kautoscalingv1.HorizontalPodAutoscaler{}: selector,
					&kapipolicyv1.PodDisruptionBudget{}:       selector,
				},
			},
		},
		{
			name: "should not remove existing options",
			args: args{
				options: cache.Options{
					SyncPeriod: &syncPeriod,
				},
			},
			want: cache.Options{
				SyncPeriod: &syncPeriod,
				ByObject: map[client.Object]cache.ByObject{
					&kappsv1.Deployment{}:                     selector,
					&kappsv1.StatefulSet{}:                    selector,
					&kcorev1.ServiceAccount{}:                 selector,
					&kcorev1.Secret{}:                         selector,
					&kcorev1.Service{}:                        selector,
					&kcorev1.ConfigMap{}:                      selector,
					&krbacv1.ClusterRole{}:                    selector,
					&krbacv1.ClusterRoleBinding{}:             selector,
					&kautoscalingv1.HorizontalPodAutoscaler{}: selector,
					&kapipolicyv1.PodDisruptionBudget{}:       selector,
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := applySelectors(tc.args.options)

			// then
			require.True(t, deepEqualOptions(tc.want, got))
		})
	}
}

func deepEqualOptions(a, b cache.Options) bool {
	// we only care about the ByObject comparison
	o := deepEqualByObject(a.ByObject, b.ByObject)
	s := a.SyncPeriod == b.SyncPeriod
	return o && s
}

func deepEqualByObject(a, b map[client.Object]cache.ByObject) bool {
	if len(a) != len(b) {
		return false
	}

	aTypeMap := make(map[string]cache.ByObject, len(a))
	bTypeMap := make(map[string]cache.ByObject, len(a))
	computeTypeMap(a, aTypeMap)
	computeTypeMap(b, bTypeMap)
	return reflect.DeepEqual(aTypeMap, bTypeMap)
}

func computeTypeMap(byObjectMap map[client.Object]cache.ByObject, typeMap map[string]cache.ByObject) {
	keyOf := func(i interface{}) string { return fmt.Sprintf(">>> %T", i) }
	for k, v := range byObjectMap {
		if obj, ok := k.(*kappsv1.Deployment); ok {
			key := keyOf(obj)
			typeMap[key] = v
		}
		if obj, ok := k.(*kcorev1.ServiceAccount); ok {
			key := keyOf(obj)
			typeMap[key] = v
		}
		if obj, ok := k.(*krbacv1.ClusterRole); ok {
			key := keyOf(obj)
			typeMap[key] = v
		}
		if obj, ok := k.(*krbacv1.ClusterRoleBinding); ok {
			key := keyOf(obj)
			typeMap[key] = v
		}
		if obj, ok := k.(*kautoscalingv1.HorizontalPodAutoscaler); ok {
			key := keyOf(obj)
			typeMap[key] = v
		}
	}
}

func Test_fromLabelSelector(t *testing.T) {
	// given
	type args struct {
		label labels.Selector
	}
	tests := []struct {
		name string
		args args
		want cache.ByObject
	}{
		{
			name: "should return the correct selector",
			args: args{
				label: labels.SelectorFromSet(map[string]string{"key": "value"}),
			},
			want: cache.ByObject{
				Label: labels.SelectorFromSet(map[string]string{"key": "value"}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			got := fromLabelSelector(tt.args.label)

			// then
			require.Equal(t, tt.want, got)
		})
	}
}
