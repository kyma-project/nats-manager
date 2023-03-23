package manager

import (
	"context"
	"github.com/kyma-project/nats-manager/pkg/k8s"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
)

type NatsConfig struct {
	ClusterSize int
}

// Perform a compile time check.
var _ Manager = &NatsManager{}

type Manager interface {
	GenerateNATSResources(*chart.ReleaseInstance, ...Option) (*chart.ManifestResources, error)
	DeployInstance(context.Context, *chart.ReleaseInstance) error
	DeleteInstance(context.Context, *chart.ManifestResources) error
}

type NatsManager struct {
	kubeClient k8s.Client
	chartRenderer  chart.Renderer
}

func NewNATSManger(kubeClient k8s.Client, chartRenderer  chart.Renderer) Manager {
	return NatsManager{
		kubeClient:    kubeClient,
		chartRenderer: chartRenderer,
	}
}

func (m NatsManager) GenerateNATSResources(instance *chart.ReleaseInstance, opts ...Option) (*chart.ManifestResources, error) {
	manifests, err := m.chartRenderer.RenderManifestAsUnStructured(instance)
	if err == nil {
		// apply options
		for _, obj := range manifests.Items {
			for _, opt := range opts {
				opt(obj)
			}
		}
	}
	return manifests, err
}

func (m NatsManager) DeployInstance(ctx context.Context, instance *chart.ReleaseInstance) error {
	for _, object := range instance.RenderedManifests.Items {
		if err := m.kubeClient.PatchApply(ctx, object); err != nil {
			return err
		}
	}
	return nil
}

func (m NatsManager) DeleteInstance(_ context.Context, _ *chart.ManifestResources) error {
	// TODO
	return nil
}
