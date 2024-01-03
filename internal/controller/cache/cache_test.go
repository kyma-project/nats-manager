package cache

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	natslabels "github.com/kyma-project/nats-manager/pkg/labels"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
				natslabels.KeyManagedBy: natslabels.ValueNATSManager,
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
					&appsv1.Deployment{}:                     selector,
					&appsv1.StatefulSet{}:                    selector,
					&corev1.ServiceAccount{}:                 selector,
					&corev1.Secret{}:                         selector,
					&corev1.Service{}:                        selector,
					&corev1.ConfigMap{}:                      selector,
					&rbacv1.ClusterRole{}:                    selector,
					&rbacv1.ClusterRoleBinding{}:             selector,
					&autoscalingv1.HorizontalPodAutoscaler{}: selector,
					&policyv1.PodDisruptionBudget{}:          selector,
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
					&appsv1.Deployment{}:                     selector,
					&appsv1.StatefulSet{}:                    selector,
					&corev1.ServiceAccount{}:                 selector,
					&corev1.Secret{}:                         selector,
					&corev1.Service{}:                        selector,
					&corev1.ConfigMap{}:                      selector,
					&rbacv1.ClusterRole{}:                    selector,
					&rbacv1.ClusterRoleBinding{}:             selector,
					&autoscalingv1.HorizontalPodAutoscaler{}: selector,
					&policyv1.PodDisruptionBudget{}:          selector,
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
		if obj, ok := k.(*appsv1.Deployment); ok {
			key := keyOf(obj)
			typeMap[key] = v
		}
		if obj, ok := k.(*corev1.ServiceAccount); ok {
			key := keyOf(obj)
			typeMap[key] = v
		}
		if obj, ok := k.(*rbacv1.ClusterRole); ok {
			key := keyOf(obj)
			typeMap[key] = v
		}
		if obj, ok := k.(*rbacv1.ClusterRoleBinding); ok {
			key := keyOf(obj)
			typeMap[key] = v
		}
		if obj, ok := k.(*autoscalingv1.HorizontalPodAutoscaler); ok {
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
