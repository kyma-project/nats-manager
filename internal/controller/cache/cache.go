package cache

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/nats-manager/pkg/label"
)

// New returns a cache with the cache-options applied, generade form the rest-config.
func New(config *rest.Config, options cache.Options) (cache.Cache, error) {
	return cache.New(config, applySelectors(options))
}

func applySelectors(options cache.Options) cache.Options {
	// The only objects we allow are the ones with the 'created-by: nats-manager' label applied.
	createdByNATSManager := fromLabelSelector(label.SelectorCreatedByNATS())

	// Apply the label selector to all relevant objects.
	options.ByObject = map[client.Object]cache.ByObject{
		&appsv1.Deployment{}:                     createdByNATSManager,
		&autoscalingv1.HorizontalPodAutoscaler{}: createdByNATSManager,
		&corev1.ServiceAccount{}:                 createdByNATSManager,
		&rbacv1.ClusterRole{}:                    createdByNATSManager,
		&rbacv1.ClusterRoleBinding{}:             createdByNATSManager,
	}
	return options
}

func fromLabelSelector(selector labels.Selector) cache.ByObject {
	return cache.ByObject{Label: selector}
}
