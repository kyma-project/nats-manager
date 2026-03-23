package nats

import (
	"context"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"go.uber.org/zap"
	kapierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	shootInfoConfigMapName        = "shoot-info"
	shootInfoConfigMapNamespace   = "kube-system"
	shootInfoConfigMapKeyProvider = "provider"
)

// syncCloudProvider reads the cloud provider from the Gardener shoot-info ConfigMap and
// stores it in the NATS status. On non-Gardener clusters the field is cleared.
func (r *Reconciler) syncCloudProvider(ctx context.Context, nats *nmapiv1alpha1.NATS, log *zap.SugaredLogger) {
	cm, err := r.kubeClient.GetConfigMap(ctx, shootInfoConfigMapName, shootInfoConfigMapNamespace)
	if err != nil {
		if kapierrors.IsNotFound(err) {
			log.Info("shoot-info ConfigMap not found; assuming non-Gardener cluster, clearing cloud provider")
			nats.Status.CloudProvider = ""
			return
		}
		// For any other error we log it but don't fail reconciliation – we simply leave the
		// previous value untouched so a transient API error doesn't wipe the cached provider.
		log.Warnw("failed to read shoot-info ConfigMap; keeping existing cloud provider value", "error", err)
		return
	}

	nats.Status.CloudProvider = cm.Data[shootInfoConfigMapKeyProvider]
	log.Infow("cloud provider read from shoot-info", "provider", nats.Status.CloudProvider)
}
