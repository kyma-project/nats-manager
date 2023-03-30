package manager

import (
	"context"
	"fmt"

	"github.com/kyma-project/nats-manager/pkg/k8s"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	"go.uber.org/zap"
)

type NatsConfig struct {
	ClusterSize int
}

// Perform a compile time check.
var _ Manager = &NatsManager{}

//go:generate mockery --name=Manager --outpkg=mocks --case=underscore
type Manager interface {
	GenerateNATSResources(*chart.ReleaseInstance, ...Option) (*chart.ManifestResources, error)
	DeployInstance(context.Context, *chart.ReleaseInstance) error
	DeleteInstance(context.Context, *chart.ReleaseInstance) error
	IsNatsStatefulSetReady(context.Context, *chart.ReleaseInstance) (bool, error)
}

type NatsManager struct {
	kubeClient    k8s.Client
	chartRenderer chart.Renderer
	logger        *zap.SugaredLogger
}

func NewNATSManger(kubeClient k8s.Client, chartRenderer chart.Renderer, logger *zap.SugaredLogger) Manager {
	return NatsManager{
		kubeClient:    kubeClient,
		chartRenderer: chartRenderer,
		logger:        logger,
	}
}

func (m NatsManager) GenerateNATSResources(instance *chart.ReleaseInstance,
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

func (m NatsManager) DeployInstance(ctx context.Context, instance *chart.ReleaseInstance) error {
	for _, object := range instance.RenderedManifests.Items {
		if err := m.kubeClient.PatchApply(ctx, object); err != nil {
			return err
		}
	}
	return nil
}

func (m NatsManager) DeleteInstance(ctx context.Context, instance *chart.ReleaseInstance) error {
	for _, object := range instance.RenderedManifests.Items {
		if err := m.kubeClient.Delete(ctx, object); err != nil {
			return err
		}
	}
	return nil
}

func (m NatsManager) IsNatsStatefulSetReady(ctx context.Context, instance *chart.ReleaseInstance) (bool, error) {
	// get statefulSets from rendered manifests
	statefulSets := instance.GetStatefulSets()
	if len(statefulSets) == 0 {
		return false, fmt.Errorf("NATS StatefulSet not found in manifests")
	}

	// fetch statefulSets from cluster and check if they are ready
	result := true
	for _, sts := range statefulSets {
		currentSts, err := m.kubeClient.GetStatefulSet(ctx, sts.GetName(), sts.GetNamespace())
		if err != nil {
			return false, err
		}
		if *currentSts.Spec.Replicas != currentSts.Status.AvailableReplicas ||
			*currentSts.Spec.Replicas != currentSts.Status.ReadyReplicas {
			result = false
		}
	}

	return result, nil
}
