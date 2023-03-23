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

package controller

import (
	"context"
	"fmt"
	"github.com/kyma-project/nats-manager/pkg/k8s"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	"github.com/kyma-project/nats-manager/pkg/manager"
	"k8s.io/client-go/tools/record"

	"github.com/go-logr/logr"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	natsFinalizerName = "nats.operator.kyma-project.io/finalizer"
	namespace = "kyma-system"
	controllerName = "nats-manager"
)

// NatsReconciler reconciles a Nats object.
type NatsReconciler struct {
	client.Client
	kubeClient k8s.Client
	chartRenderer  chart.Renderer
	Scheme          *runtime.Scheme
	recorder        record.EventRecorder
	logger          logr.Logger
	natsManager manager.Manager
}

func NewNatsReconciler(client client.Client, chartRenderer chart.Renderer, scheme *runtime.Scheme, logger logr.Logger,
	recorder record.EventRecorder, natsManager manager.Manager) *NatsReconciler {

	kubeClient := k8s.NewKubeClient(client, controllerName)

	return &NatsReconciler{
		Client:          client,
		kubeClient: kubeClient,
		chartRenderer:   chartRenderer,
		Scheme:          scheme,
		recorder:        recorder,
		logger:          logger,
		natsManager: natsManager,
	}
}

//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=nats,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=nats/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=nats/finalizers,verbs=update

func (r *NatsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger.Info("Reconciliation triggered")
	// fetch latest subscription object
	currentNats := &natsv1alpha1.Nats{}
	if err := r.Get(ctx, req.NamespacedName, currentNats); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// copy the object, so we don't modify the source object
	nats := currentNats.DeepCopy()

	// check if nats is in deletion state
	if isInDeletion(nats) {
		return r.handleNatsDeletion(ctx, nats)
	}

	// handle reconciliation
	return r.handleNatsReconcile(ctx, nats)
}

func (r *NatsReconciler) generateNatsResources(nats *natsv1alpha1.Nats, instance *chart.ReleaseInstance) error {
	// generate Nats resources from chart
	natsResources, err := r.natsManager.GenerateNATSResources(
		instance,
		manager.WithOwnerReference(*nats), // add owner references to all resources
	)
	if err != nil {
		return err
	}

	// update manifests in instance
	instance.SetRenderedManifests(*natsResources)
	return nil
}

func (r *NatsReconciler) handleNatsReconcile(ctx context.Context, nats *natsv1alpha1.Nats) (ctrl.Result, error) {
	if err := r.addFinalizer(ctx, nats); err != nil {
		return ctrl.Result{}, err
	}

	// Check if istio is enabled in cluster

	// Init a release instance
	instance := &chart.ReleaseInstance{
		Name:      nats.Name,
		Namespace: namespace,
		// @TODO: Provide the overrides in component.Configuration
	}

	if err := r.generateNatsResources(nats, instance); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.natsManager.DeployInstance(ctx, instance); err != nil {
		deployErr := fmt.Errorf("failed to deploy NATS: %w", err)
		nats.UpdateStateFromErr(natsv1alpha1.StateError, natsv1alpha1.ConditionReasonDeployError, deployErr)
		if err = r.Status().Update(ctx, nats); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, deployErr
	}

	// Sync CR status
	nats.UpdateStateReady(natsv1alpha1.StateReady, natsv1alpha1.ConditionReasonDeployed, "NATS is deployed")
	if err := r.Status().Update(ctx, nats); err != nil {
		return ctrl.Result{}, err
	}

	r.logger.Info("Reconciliation successful")
	return ctrl.Result{}, nil
}

func (r *NatsReconciler) handleNatsDeletion(ctx context.Context, nats *natsv1alpha1.Nats) (ctrl.Result, error) {
	// skip deletion if the finalizer is not in the resource
	if !controllerutil.ContainsFinalizer(nats, natsFinalizerName) {
		return ctrl.Result{}, nil
	}

	r.logger.Info("Deleting the NATS")
	nats.UpdateStateDeletion(natsv1alpha1.StateDeleting, natsv1alpha1.ConditionReasonDeletion, "NATS is being deleted")
	var err error
	if err = r.Status().Update(ctx, nats); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.natsManager.DeleteInstance(ctx, nil); err != nil {
		deletionErr := fmt.Errorf("failed to delete NATS: %w", err)
		nats.UpdateStateFromErr(natsv1alpha1.StateError, natsv1alpha1.ConditionReasonDeletionError, deletionErr)
		if err = r.Status().Update(ctx, nats); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, deletionErr
	}

	controllerutil.RemoveFinalizer(nats, natsFinalizerName)
	if err := r.Update(ctx, nats); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *NatsReconciler) addFinalizer(ctx context.Context, nats *natsv1alpha1.Nats) error {
	// do add finalizer if already in deletion
	if !nats.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil
	}

	if !controllerutil.ContainsFinalizer(nats, natsFinalizerName) {
		controllerutil.AddFinalizer(nats, natsFinalizerName)
		if err := r.Update(ctx, nats); err != nil {
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NatsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&natsv1alpha1.Nats{}).
		Complete(r)
}

//// SetupWithManager sets up the controller with the Manager.
//func (r *NatsReconciler) SetupWithManager(mgr ctrl.Manager) error {
//	return ctrl.NewControllerManagedBy(mgr).
//		For(&natsv1alpha1.Nats{}).
//		WithEventFilter(
//			predicate.Or(
//				predicate.GenerationChangedPredicate{},
//				predicate.LabelChangedPredicate{},
//				predicate.AnnotationChangedPredicate{})).
//		Complete(r)
//}
