package chart

import (
	"fmt"

	"dario.cat/mergo"
	"github.com/kyma-project/nats-manager/pkg/file"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Perform a compile time check.
var _ Renderer = &HelmRenderer{}

type HelmRenderer struct {
	chartPath string
	logger    *zap.SugaredLogger
	helmChart *chart.Chart
}

func NewHelmRenderer(chartPath string, logger *zap.SugaredLogger) (Renderer, error) {
	if !file.DirExists(chartPath) {
		return nil, fmt.Errorf("HELM chart directory '%s' not found", chartPath) //nolint: goerr113 // reason: This is the only place where we use dynamic error message
	}

	// load chart into memory
	helmChart, err := loader.Load(chartPath)
	if err != nil {
		return nil, errors.Wrap(err, "loader failed to load helm chart")
	}

	return &HelmRenderer{
		chartPath: chartPath,
		helmChart: helmChart,
		logger:    logger,
	}, nil
}

// RenderManifestAsUnstructured of the given chart as unstructured objects.
func (c *HelmRenderer) RenderManifestAsUnstructured(releaseInstance *ReleaseInstance) (*ManifestResources, error) {
	manifests, err := c.RenderManifest(releaseInstance)
	if err != nil {
		return nil, err
	}

	return ParseManifestStringToObjects(manifests)
}

// RenderManifest of the given chart as string.
func (c *HelmRenderer) RenderManifest(releaseInstance *ReleaseInstance) (string, error) {
	config, err := c.overrideChartConfiguration(releaseInstance)
	if err != nil {
		return "", errors.Wrap(err, "failed to merge chart configuration")
	}

	tplAction, err := c.newTemplatingAction(releaseInstance)
	if err != nil {
		return "", errors.Wrap(err, "templating action failed")
	}

	helmRelease, err := tplAction.Run(c.helmChart, config)
	if err != nil || helmRelease == nil {
		return "", errors.Wrap(err,
			fmt.Sprintf("Failed to render HELM template for ReleaseInstance '%s'", releaseInstance.Name))
	}

	return helmRelease.Manifest, nil
}

func (c *HelmRenderer) newTemplatingAction(releaseInstance *ReleaseInstance) (*action.Install, error) {
	cfg, err := c.newActionConfig(releaseInstance.Namespace)
	if err != nil {
		return nil, err
	}

	tplAction := action.NewInstall(cfg)
	tplAction.ReleaseName = releaseInstance.Name
	tplAction.Namespace = releaseInstance.Namespace
	tplAction.Atomic = true
	tplAction.Wait = true
	tplAction.CreateNamespace = true
	tplAction.DryRun = true
	tplAction.Replace = true     // Skip the name check
	tplAction.IncludeCRDs = true // include CRDs in the templated output
	tplAction.ClientOnly = true  // if false, it will validate the manifests against the Kubernetes cluster

	return tplAction, nil
}

func (c *HelmRenderer) newActionConfig(namespace string) (*action.Configuration, error) {
	clientGetter := genericclioptions.NewConfigFlags(false)
	clientGetter.Namespace = &namespace
	cfg := new(action.Configuration)
	if err := cfg.Init(clientGetter, namespace, "secrets", c.logger.Debugf); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *HelmRenderer) getChartConfiguration() map[string]interface{} {
	return c.helmChart.Values
}

func (c *HelmRenderer) overrideChartConfiguration(releaseInstance *ReleaseInstance) (map[string]interface{}, error) {
	result := c.getChartConfiguration()
	releaseInstanceConfig, err := releaseInstance.GetConfiguration()
	if err != nil {
		return nil, err
	}

	if err := mergo.Merge(&result, releaseInstanceConfig, mergo.WithOverride); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to merge chart configurations with ReleaseInstance "+
			"configuration for ReleaseInstance '%s'", releaseInstance.Name))
	}

	return result, nil
}
