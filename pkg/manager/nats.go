package manager

import (
	"context"
	"fmt"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/pkg/k8s"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	"go.uber.org/zap"
)

type NatsConfig struct {
	ClusterSize int
}

// Perform a compile time check.
var _ Manager = &NATSManager{}

//go:generate mockery --name=Manager --outpkg=mocks --case=underscore
type Manager interface {
	GenerateNATSResources(*chart.ReleaseInstance, ...Option) (*chart.ManifestResources, error)
	DeployInstance(context.Context, *chart.ReleaseInstance) error
	DeleteInstance(context.Context, *chart.ReleaseInstance) error
	IsNATSStatefulSetReady(context.Context, *chart.ReleaseInstance) (bool, error)
	GenerateOverrides(*natsv1alpha1.NATSSpec, bool, bool) map[string]interface{}
}

type NATSManager struct {
	kubeClient    k8s.Client
	chartRenderer chart.Renderer
	logger        *zap.SugaredLogger
}

func NewNATSManger(kubeClient k8s.Client, chartRenderer chart.Renderer, logger *zap.SugaredLogger) Manager {
	return NATSManager{
		kubeClient:    kubeClient,
		chartRenderer: chartRenderer,
		logger:        logger,
	}
}

func (m NATSManager) GenerateNATSResources(instance *chart.ReleaseInstance,
	opts ...Option) (*chart.ManifestResources, error) {
	manifests, err := m.chartRenderer.RenderManifestAsUnstructured(instance)
	if err == nil {
		// apply options
		for _, obj := range manifests.Items {
			for _, opt := range opts {
				err = opt(obj)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return manifests, err
}

func (m NATSManager) DeployInstance(ctx context.Context, instance *chart.ReleaseInstance) error {
	for _, object := range instance.RenderedManifests.Items {
		if err := m.kubeClient.PatchApply(ctx, object); err != nil {
			return err
		}
	}
	return nil
}

func (m NATSManager) DeleteInstance(ctx context.Context, instance *chart.ReleaseInstance) error {
	for _, object := range instance.RenderedManifests.Items {
		if err := m.kubeClient.Delete(ctx, object); err != nil {
			return err
		}
	}
	return nil
}

func (m NATSManager) IsNATSStatefulSetReady(ctx context.Context, instance *chart.ReleaseInstance) (bool, error) {
	// get statefulSets from rendered manifests
	statefulSets := instance.GetStatefulSets()
	if len(statefulSets) == 0 {
		return false, fmt.Errorf("NATS StatefulSet not found in manifests")
	}

	// fetch statefulSets from cluster and check if they are ready
	for _, sts := range statefulSets {
		currentSts, err := m.kubeClient.GetStatefulSet(ctx, sts.GetName(), sts.GetNamespace())
		if err != nil {
			return false, err
		}
		if *currentSts.Spec.Replicas != currentSts.Status.CurrentReplicas ||
			*currentSts.Spec.Replicas != currentSts.Status.UpdatedReplicas ||
			*currentSts.Spec.Replicas != currentSts.Status.ReadyReplicas {
			return false, nil
		}
	}

	return true, nil
}
