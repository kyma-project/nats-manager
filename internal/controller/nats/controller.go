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

	"sigs.k8s.io/controller-runtime/pkg/predicate"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/pkg/k8s"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	"github.com/kyma-project/nats-manager/pkg/manager"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NATSFinalizerName = "nats.operator.kyma-project.io/finalizer"
)

// Reconciler reconciles a Nats object.
type Reconciler struct {
	client.Client
	kubeClient    k8s.Client
	chartRenderer chart.Renderer
	Scheme        *runtime.Scheme
	recorder      record.EventRecorder
	logger        *zap.SugaredLogger
	NATSManager   manager.Manager
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
		Client:        client,
		kubeClient:    kubeClient,
		chartRenderer: chartRenderer,
		Scheme:        scheme,
		recorder:      recorder,
		logger:        logger,
		NATSManager:   natsManager,
	}
}

//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=nats,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=nats/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=nats/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger.Info("Reconciliation triggered")
	// fetch latest subscription object
	currentNats := &natsv1alpha1.NATS{}
	if err := r.Get(ctx, req.NamespacedName, currentNats); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// copy the object, so we don't modify the source object
	nats := currentNats.DeepCopy()

	// check if nats is in deletion state
	if nats.IsInDeletion() {
		return ctrl.Result{}, nil
	}

	// handle reconciliation
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&natsv1alpha1.NATS{}).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.LabelChangedPredicate{},
				predicate.AnnotationChangedPredicate{})).
		Complete(r)
}
