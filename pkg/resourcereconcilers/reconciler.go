package resourcereconcilers

import (
	"context"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	logger          logr.Logger
	k8sClient client.Client
	FieldManager string
}

func NewReconciler(logger logr.Logger, client client.Client, fieldManager string) *Reconciler {
	return &Reconciler{
		logger: logger,
		k8sClient: client,
		FieldManager: fieldManager,
	}
}

func (c *Reconciler) PatchApply(ctx context.Context, object *unstructured.Unstructured) error {
	return c.k8sClient.Patch(ctx, object, client.Apply, &client.PatchOptions{
		Force:        pointer.Bool(true),
		FieldManager: "nats-manager",
	})
}

func (c *Reconciler) Delete() error {
	return nil
}
