package k8s

import (
	"context"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Perform a compile time check.
var _ Client = &KubeClient{}

type Client interface {
	PatchApply(context.Context, *unstructured.Unstructured) error
}

type KubeClient struct {
	client client.Client
	fieldManager string
}

func NewKubeClient(client client.Client, fieldManager string) Client {
	return &KubeClient{
		client:    client,
		fieldManager: fieldManager,
	}
}

func (c *KubeClient) PatchApply(ctx context.Context, object *unstructured.Unstructured) error {
	return c.client.Patch(ctx, object, client.Apply, &client.PatchOptions{
		Force:        pointer.Bool(true),
		FieldManager: c.fieldManager,
	})
}
