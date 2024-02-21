package cache

import (
	kappsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	kcorev1 "k8s.io/api/core/v1"
	kapipolicyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	natslabels "github.com/kyma-project/nats-manager/pkg/labels"
)

// New returns a cache with the cache-options applied, generade form the rest-config.
func New(config *rest.Config, options cache.Options) (cache.Cache, error) {
	return cache.New(config, applySelectors(options))
}

func applySelectors(options cache.Options) cache.Options {
	// The only objects we allow are the ones with the 'managed-by: nats-manager' label applied.
	managedByNATS := fromLabelSelector(natslabels.SelectorManagedByNATS())

	// Apply the label selector to all relevant objects.
	options.ByObject = map[client.Object]cache.ByObject{
		&kappsv1.Deployment{}:                    managedByNATS,
		&kappsv1.StatefulSet{}:                   managedByNATS,
		&kcorev1.ServiceAccount{}:                managedByNATS,
		&kcorev1.Secret{}:                        managedByNATS,
		&kcorev1.Service{}:                       managedByNATS,
		&kcorev1.ConfigMap{}:                     managedByNATS,
		&rbacv1.ClusterRole{}:                    managedByNATS,
		&rbacv1.ClusterRoleBinding{}:             managedByNATS,
		&autoscalingv1.HorizontalPodAutoscaler{}: managedByNATS,
		&kapipolicyv1.PodDisruptionBudget{}:      managedByNATS,
	}
	return options
}

func fromLabelSelector(selector labels.Selector) cache.ByObject {
	return cache.ByObject{Label: selector}
}
