/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nats

import (
	"context"
	"fmt"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/pkg/k8s"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	"github.com/kyma-project/nats-manager/pkg/manager"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

const (
	NATSFinalizerName   = "nats.operator.kyma-project.io/finalizer"
	ControllerName      = "nats-manager"
	ManagedByLabelKey   = "app.kubernetes.io/managed-by"
	ManagedByLabelValue = ControllerName
)

// Reconciler reconciles a Nats object.
type Reconciler struct {
	client.Client
	controller.Controller
	kubeClient                  k8s.Client
	chartRenderer               chart.Renderer
	Scheme                      *runtime.Scheme
	recorder                    record.EventRecorder
	logger                      *zap.SugaredLogger
	NATSManager                 manager.Manager
	destinationRuleWatchStarted bool
}

func NewReconciler(
	client client.Client,
	kubeClient k8s.Client,
	chartRenderer chart.Renderer,
	scheme *runtime.Scheme,
	logger *zap.SugaredLogger,
	recorder record.EventRecorder,
	natsManager manager.Manager,
) *Reconciler {
	return &Reconciler{
		Client:                      client,
		kubeClient:                  kubeClient,
		chartRenderer:               chartRenderer,
		Scheme:                      scheme,
		recorder:                    recorder,
		logger:                      logger,
		NATSManager:                 natsManager,
		destinationRuleWatchStarted: false,
		Controller:                  nil,
	}
}

//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apps/v1",resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=nats,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=nats/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=nats/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger.Info("Reconciliation triggered")
	// fetch latest subscription object
	currentNats := &natsv1alpha1.Nats{}
	if err := r.Get(ctx, req.NamespacedName, currentNats); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// copy the object, so we don't modify the source object
	nats := currentNats.DeepCopy()

	// logger with nats details
	log := r.loggerWithNATS(nats)

	// check if nats is in deletion state
	if nats.IsInDeletion() {
		return r.handleNATSDeletion(ctx, nats, log)
	}

	// handle reconciliation
	return r.handleNATSReconcile(ctx, nats, log)
}

// generateNatsResources renders the NATS chart with provided overrides.
// It puts results into ReleaseInstance.
func (r *Reconciler) generateNatsResources(nats *natsv1alpha1.Nats, instance *chart.ReleaseInstance) error {
	// generate Nats resources from chart
	natsResources, err := r.NATSManager.GenerateNATSResources(
		instance,
		manager.WithOwnerReference(*nats), // add owner references to all resources
		manager.WithLabel(ManagedByLabelKey, ManagedByLabelValue),
	)
	if err != nil {
		return err
	}

	// update manifests in instance
	instance.SetRenderedManifests(*natsResources)
	return nil
}

// initNATSInstance initializes a new NATS release instance based on NATS CR.
func (r *Reconciler) initNATSInstance(ctx context.Context, nats *natsv1alpha1.Nats,
	log *zap.SugaredLogger) (*chart.ReleaseInstance, error) {
	// Check if istio is enabled in cluster
	istioExists, err := r.kubeClient.DestinationRuleCRDExists(ctx)
	if err != nil {
		return nil, err
	}
	log.Infof("Istio enabled on cluster: %t", istioExists)

	// Init a release instance
	instance := &chart.ReleaseInstance{
		Name:         nats.Name,
		Namespace:    nats.Namespace,
		IstioEnabled: istioExists,
	}

	accountSecretName := fmt.Sprintf("%s-secret", nats.Name)
	// Check if secret exists then make sure the password is same
	accountSecret, err := r.kubeClient.GetSecret(ctx, accountSecretName, nats.Namespace)
	if err != nil && !errors.IsNotFound(err) {
		log.Errorf("Failed to fetch secret: %s", accountSecretName)
		log.Error(err)
		return nil, err
	}
	log.Infof("NATS account secret (name: %s) exists: %t", accountSecretName, accountSecret != nil)

	// @TODO: Provide the overrides in component.Configuration
	instance.Configuration = map[string]interface{}{
		"istio.enabled":       istioExists,
		"auth.rotatePassword": accountSecret == nil, // do not recreate secret if it exists
	}

	if err = r.generateNatsResources(nats, instance); err != nil {
		return nil, err
	}

	return instance, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error
	r.Controller, err = ctrl.NewControllerManagedBy(mgr).
		For(&natsv1alpha1.Nats{}).
		Owns(&appsv1.StatefulSet{}). // watch for StatefulSets.
		Owns(&apiv1.Service{}).      // watch for services.
		Owns(&apiv1.ConfigMap{}).    // watch for ConfigMaps.
		Owns(&apiv1.Secret{}).       // watch for Secrets.
		Build(r)

	return err
}

// loggerWithNATS returns a logger with the given NATS CR details.
func (r *Reconciler) loggerWithNATS(nats *natsv1alpha1.Nats) *zap.SugaredLogger {
	return r.logger.With(
		"kind", nats.GetObjectKind().GroupVersionKind().Kind,
		"version", nats.GetGeneration(),
		"namespace", nats.GetNamespace(),
		"name", nats.GetName(),
	)
}
