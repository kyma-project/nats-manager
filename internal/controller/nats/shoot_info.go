package nats

import (
	"context"

	"go.uber.org/zap"
	kapierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	shootInfoConfigMapName        = "shoot-info"
	shootInfoConfigMapNamespace   = "kube-system"
	shootInfoConfigMapKeyProvider = "provider"
)

// syncCloudProvider reads the cloud provider from the Gardener shoot-info ConfigMap and caches it.
// Since shoot-info never changes after cluster provisioning, the API call is made at most once.
// On transient errors the cache is left as nil so the next reconciliation retries.
func (r *Reconciler) syncCloudProvider(ctx context.Context, log *zap.SugaredLogger) {
	if r.cloudProvider != nil {
		return
	}

	cm, err := r.kubeClient.GetConfigMap(ctx, shootInfoConfigMapName, shootInfoConfigMapNamespace)
	if err != nil {
		if kapierrors.IsNotFound(err) {
			log.Info("shoot-info ConfigMap not found; assuming non-Gardener cluster")
			provider := ""
			r.cloudProvider = &provider
			return
		}
		// Transient error — leave cache as nil so the next reconciliation retries.
		log.Warnw("failed to read shoot-info ConfigMap; will retry on next reconciliation", "error", err)
		return
	}

	provider := cm.Data[shootInfoConfigMapKeyProvider]
	r.cloudProvider = &provider
	log.Infow("cloud provider resolved from shoot-info", "provider", provider)
}
