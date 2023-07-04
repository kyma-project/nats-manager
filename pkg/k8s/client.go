package k8s

import (
	"context"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	k8sclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Perform a compile time check.
var _ Client = &KubeClient{}

//go:generate mockery --name=Client --outpkg=mocks --case=underscore
type Client interface {
	PatchApply(context.Context, *unstructured.Unstructured) error
	GetStatefulSet(context.Context, string, string) (*appsv1.StatefulSet, error)
	Delete(context.Context, *unstructured.Unstructured) error
	GetSecret(context.Context, string, string) (*apiv1.Secret, error)
	GetCRD(context.Context, string) (*apiextensionsv1.CustomResourceDefinition, error)
	DestinationRuleCRDExists(context.Context) (bool, error)
	DeletePVCsWithLabel(context.Context, string, string, string) error
}

type KubeClient struct {
	client       client.Client
	clientset    k8sclientset.Interface
	fieldManager string
}

func NewKubeClient(client client.Client, clientset k8sclientset.Interface, fieldManager string) Client {
	return &KubeClient{
		client:       client,
		clientset:    clientset,
		fieldManager: fieldManager,
	}
}

func (c *KubeClient) PatchApply(ctx context.Context, object *unstructured.Unstructured) error {
	return c.client.Patch(ctx, object, client.Apply, &client.PatchOptions{
		Force:        pointer.Bool(true),
		FieldManager: c.fieldManager,
	})
}

func (c *KubeClient) Delete(ctx context.Context, object *unstructured.Unstructured) error {
	return client.IgnoreNotFound(c.client.Delete(ctx, object))
}

func (c *KubeClient) GetStatefulSet(ctx context.Context, name, namespace string) (*appsv1.StatefulSet, error) {
	nn := k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &appsv1.StatefulSet{}
	if err := c.client.Get(ctx, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *KubeClient) GetSecret(ctx context.Context, name, namespace string) (*apiv1.Secret, error) {
	nn := k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &apiv1.Secret{}
	if err := c.client.Get(ctx, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *KubeClient) GetCRD(ctx context.Context, name string) (*apiextensionsv1.CustomResourceDefinition, error) {
	return c.clientset.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
}

func (c *KubeClient) DestinationRuleCRDExists(ctx context.Context) (bool, error) {
	_, err := c.GetCRD(ctx, DestinationRuleCrdName)
	if err != nil {
		return false, client.IgnoreNotFound(err)
	}
	return true, nil
}

func (c *KubeClient) DeletePVCsWithLabel(ctx context.Context, labelSelector string,
	mustHaveNamePrefix, namespace string) error {
	// create a new labels.Selector object for the label selector
	selector, err := labels.Parse(labelSelector)
	if err != nil {
		return err
	}

	pvcList := &apiv1.PersistentVolumeClaimList{}
	if err = c.client.List(ctx, pvcList, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: selector,
	}); err != nil {
		return err
	}

	// if there are no PVCs in the list, do nothing.
	if len(pvcList.Items) == 0 {
		return nil
	}

	// delete each PVC in the list.
	for i := range pvcList.Items {
		pvc := pvcList.Items[i]
		// pvc.Name string starts with "eventing-"
		if strings.HasPrefix(pvc.Name, mustHaveNamePrefix) {
			err = c.client.Delete(ctx, &pvc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
