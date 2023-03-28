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

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const natsFinalizerName = "nats.operator.kyma-project.io/finalizer"

// NatsReconciler reconciles a Nats object.
type NatsReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	NatsProvisioner provisioner.Provisioner
	log             logr.Logger
}

//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=nats,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=nats/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=nats/finalizers,verbs=update

func (r *NatsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = logger.FromContext(ctx)
	r.log.Info("Reconciling...")
	nats := &natsv1alpha1.Nats{}
	if err := r.Get(ctx, req.NamespacedName, nats); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.addFinalizer(ctx, nats); err != nil {
		return ctrl.Result{}, err
	}

	if !nats.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := r.deleteNats(ctx, nats); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if err := r.deployNats(ctx, nats); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NatsReconciler) deployNats(ctx context.Context, nats *natsv1alpha1.Nats) error {
	r.log.Info("Deploying NATS ...")
	nats.UpdateStateProcessing(natsv1alpha1.StateReady, natsv1alpha1.ConditionReasonDeploying, "NATS is being deployed")
	var err error
	if err = r.Status().Update(ctx, nats); err != nil {
		return err
	}

	natsConfig := provisioner.NatsConfig{
		ClusterSize: nats.Spec.Cluster.Size,
	}
	err = r.NatsProvisioner.Deploy(natsConfig)
	if err != nil {
		deployErr := fmt.Errorf("failed to deploy NATS: %w", err)
		nats.UpdateStateFromErr(natsv1alpha1.StateReady, natsv1alpha1.ConditionReasonDeployError, deployErr)
		if err = r.Status().Update(ctx, nats); err != nil {
			return err
		}
		return deployErr
	}

	nats.UpdateStateReady(natsv1alpha1.StateReady, natsv1alpha1.ConditionReasonDeployed, "NATS is deployed")
	return r.Status().Update(ctx, nats)
}

func (r *NatsReconciler) deleteNats(ctx context.Context, nats *natsv1alpha1.Nats) error {
	// skip deletion if the finalizer is not in the resource
	if !controllerutil.ContainsFinalizer(nats, natsFinalizerName) {
		return nil
	}
	r.log.Info("Deleting the NATS")
	nats.UpdateStateDeletion(natsv1alpha1.StateDeleted, natsv1alpha1.ConditionReasonDeletion, "NATS is being deleted")
	var err error
	if err = r.Status().Update(ctx, nats); err != nil {
		return err
	}

	if err := r.NatsProvisioner.Delete(); err != nil {
		deletionErr := fmt.Errorf("failed to delete NATS: %w", err)
		nats.UpdateStateFromErr(natsv1alpha1.StateError, natsv1alpha1.ConditionReasonDeletionError, deletionErr)
		if err = r.Status().Update(ctx, nats); err != nil {
			return err
		}
		return deletionErr
	}

	controllerutil.RemoveFinalizer(nats, natsFinalizerName)
	return r.Update(ctx, nats)
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
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.LabelChangedPredicate{},
				predicate.AnnotationChangedPredicate{})).
		Complete(r)
}
