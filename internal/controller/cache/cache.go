package cache

import (
	nmlabels "github.com/kyma-project/nats-manager/pkg/labels"
	kappsv1 "k8s.io/api/apps/v1"
	kautoscalingv1 "k8s.io/api/autoscaling/v1"
	kcorev1 "k8s.io/api/core/v1"
	kapipolicyv1 "k8s.io/api/policy/v1"
	krbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// New returns a cache with the cache-options applied, generade form the rest-config.
func New(config *rest.Config, options cache.Options) (cache.Cache, error) {
	return cache.New(config, applySelectors(options))
}

func applySelectors(options cache.Options) cache.Options {
	// The only objects we allow are the ones with the 'managed-by: nats-manager' label applied.
	managedByNATS := fromLabelSelector(nmlabels.SelectorManagedByNATS())

	// Apply the label selector to all relevant objects.
	options.ByObject = map[client.Object]cache.ByObject{
		&kappsv1.Deployment{}:                     managedByNATS,
		&kappsv1.StatefulSet{}:                    managedByNATS,
		&kcorev1.ServiceAccount{}:                 managedByNATS,
		&kcorev1.Secret{}:                         managedByNATS,
		&kcorev1.Service{}:                        managedByNATS,
		&kcorev1.ConfigMap{}:                      managedByNATS,
		&krbacv1.ClusterRole{}:                    managedByNATS,
		&krbacv1.ClusterRoleBinding{}:             managedByNATS,
		&kautoscalingv1.HorizontalPodAutoscaler{}: managedByNATS,
		&kapipolicyv1.PodDisruptionBudget{}:       managedByNATS,
		&kcorev1.Pod{}:                            managedByNATS,
	}
	return options
}

func fromLabelSelector(selector labels.Selector) cache.ByObject {
	return cache.ByObject{Label: selector}
}
