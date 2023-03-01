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
	"github.com/go-logr/logr"
	"github.com/kyma-project/nats-manager/pkg/provisioner"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logger "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	eventingv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const natsFinalizerName = "nats.kyma-project.io/finalizer"

// NatsReconciler reconciles a Nats object
type NatsReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	NatsProvisioner provisioner.Provisioner
	log             logr.Logger
}

//+kubebuilder:rbac:groups=eventing.kyma-project.io,resources=nats,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eventing.kyma-project.io,resources=nats/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=eventing.kyma-project.io,resources=nats/finalizers,verbs=update

func (r *NatsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = logger.FromContext(ctx)
	r.log.Info("Reconciling...")
	var nats eventingv1alpha1.Nats
	if err := r.Get(ctx, req.NamespacedName, &nats); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if nats.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&nats, natsFinalizerName) {
			controllerutil.AddFinalizer(&nats, natsFinalizerName)
			if err := r.Update(ctx, &nats); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(&nats, natsFinalizerName) {
			if err := r.deleteNats(ctx, &nats); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&nats, natsFinalizerName)
			if err := r.Update(ctx, &nats); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if err := r.deployNats(ctx, &nats); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NatsReconciler) deployNats(ctx context.Context, nats *eventingv1alpha1.Nats) error {
	r.log.Info("Deploying NATS ...")
	nats.UpdateStateProcessing(eventingv1alpha1.StateReady, eventingv1alpha1.ConditionReasonDeploying, "NATS is being deployed")
	var err error
	if err = r.Status().Update(ctx, nats); err != nil {
		return err
	}

	natsConfig := provisioner.NatsConfig{
		ClusterSize: nats.Spec.Cluster.Size,
	}
	err = r.NatsProvisioner.Deploy(natsConfig)
	if err != nil {
		deployErr := fmt.Errorf("failed to deploy NATS %v", err)
		nats.UpdateStateFromErr(eventingv1alpha1.StateProcessing, eventingv1alpha1.ConditionReasonDeployError, deployErr)
		return deployErr
	}

	nats.UpdateStateReady(eventingv1alpha1.StateReady, eventingv1alpha1.ConditionReasonDeployed, "NATS is deployed")
	if err = r.Status().Update(ctx, nats); err != nil {
		return err
	}
	return nil
}

func (r *NatsReconciler) deleteNats(ctx context.Context, nats *eventingv1alpha1.Nats) error {
	r.log.Info("Deleting the NATS")
	nats.UpdateStateDeletion(eventingv1alpha1.StateDeleted, eventingv1alpha1.ConditionReasonDeletion, "NATS is being deleted")
	var err error
	if err = r.Status().Update(ctx, nats); err != nil {
		return err
	}

	err = r.NatsProvisioner.Delete()
	if err != nil {
		deletionErr := fmt.Errorf("failed to delete NATS: %v", err)
		nats.UpdateStateFromErr(eventingv1alpha1.StateError, eventingv1alpha1.ConditionReasonDeletionError, deletionErr)
		return deletionErr
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NatsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&eventingv1alpha1.Nats{}).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.LabelChangedPredicate{},
				predicate.AnnotationChangedPredicate{})).
		Complete(r)
}
