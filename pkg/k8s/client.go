package k8s

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jellydator/ttlcache/v3"
	kappsv1 "k8s.io/api/apps/v1"
	kcorev1 "k8s.io/api/core/v1"
	kapiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kapiextclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Perform a compile time check.
var _ Client = &KubeClient{}

//go:generate go run github.com/vektra/mockery/v2 --name=Client --outpkg=mocks --case=underscore
type Client interface {
	PatchApply(context.Context, *unstructured.Unstructured) error
	GetStatefulSet(context.Context, string, string) (*kappsv1.StatefulSet, error)
	Delete(context.Context, *unstructured.Unstructured) error
	GetSecret(context.Context, string, string) (*kcorev1.Secret, error)
	GetCRD(context.Context, string) (*kapiextv1.CustomResourceDefinition, error)
	DestinationRuleCRDExists(context.Context) (bool, error)
	DeletePVCsWithLabel(context.Context, string, string, string) error
	GetNodeZone(context.Context, string) (string, error)
	GetPodsByLabels(context.Context, string, map[string]string) (*kcorev1.PodList, error)
	GetNumberOfAvailabilityZonesUsedByPods(context.Context, string, map[string]string) (int, error)
}

const (
	nodesZoneTTL     = 10 * time.Hour
	nodeZoneLabelKey = "topology.kubernetes.io/zone"
)

var ErrNodeZoneLabelMissing = errors.New("zone label missing")

type KubeClient struct {
	client         client.Client
	clientset      kapiextclientset.Interface
	fieldManager   string
	nodesZoneCache *ttlcache.Cache[string, string]
}

func NewKubeClient(client client.Client, clientset kapiextclientset.Interface, fieldManager string) Client {
	// initialize the cache for the nodes zone information.
	nodesZoneCache := ttlcache.New[string, string](
		ttlcache.WithTTL[string, string](nodesZoneTTL),
		ttlcache.WithDisableTouchOnHit[string, string](),
	)

	return &KubeClient{
		client:         client,
		clientset:      clientset,
		fieldManager:   fieldManager,
		nodesZoneCache: nodesZoneCache,
	}
}

func (c *KubeClient) PatchApply(ctx context.Context, object *unstructured.Unstructured) error {
	return c.client.Patch(ctx, object, client.Apply, &client.PatchOptions{
		Force:        ptr.To(true),
		FieldManager: c.fieldManager,
	})
}

func (c *KubeClient) Delete(ctx context.Context, object *unstructured.Unstructured) error {
	return client.IgnoreNotFound(c.client.Delete(ctx, object))
}

func (c *KubeClient) GetStatefulSet(ctx context.Context, name, namespace string) (*kappsv1.StatefulSet, error) {
	nn := ktypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &kappsv1.StatefulSet{}
	if err := c.client.Get(ctx, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *KubeClient) GetSecret(ctx context.Context, name, namespace string) (*kcorev1.Secret, error) {
	nn := ktypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &kcorev1.Secret{}
	if err := c.client.Get(ctx, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *KubeClient) GetCRD(ctx context.Context, name string) (*kapiextv1.CustomResourceDefinition, error) {
	return c.clientset.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, kmetav1.GetOptions{})
}

func (c *KubeClient) DestinationRuleCRDExists(ctx context.Context) (bool, error) {
	_, err := c.GetCRD(ctx, DestinationRuleCrdName)
	if err != nil {
		return false, client.IgnoreNotFound(err)
	}
	return true, nil
}

func (c *KubeClient) DeletePVCsWithLabel(ctx context.Context, labelSelector string,
	mustHaveNamePrefix, namespace string,
) error {
	// create a new labels.Selector object for the label selector
	selector, err := labels.Parse(labelSelector)
	if err != nil {
		return err
	}

	pvcList := &kcorev1.PersistentVolumeClaimList{}
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

func (c *KubeClient) GetNode(ctx context.Context, name string) (*kcorev1.Node, error) {
	node := &kcorev1.Node{}
	if err := c.client.Get(ctx, ktypes.NamespacedName{Name: name}, node); err != nil {
		return nil, err
	}
	return node, nil
}

// GetNodeZone returns the zone information of the node.
// It caches the zone information of the node for a certain duration.
func (c *KubeClient) GetNodeZone(ctx context.Context, name string) (string, error) {
	// delete expired entries from the cache.
	c.nodesZoneCache.DeleteExpired()

	// check if the zone information is already in the cache.
	if c.nodesZoneCache.Has(name) {
		item := c.nodesZoneCache.Get(name)
		if item.Value() != "" {
			return item.Value(), nil
		}
	}

	// get the node from kubernetes.
	node, err := c.GetNode(ctx, name)
	if err != nil {
		return "", err
	}

	// extract the zone information.
	zone, ok := node.Labels[nodeZoneLabelKey]
	if !ok || zone == "" {
		return "", fmt.Errorf("%w : label: %s, node: %s", ErrNodeZoneLabelMissing, nodeZoneLabelKey, name)
	}

	// set the zone information in the cache.
	c.nodesZoneCache.Set(name, zone, nodesZoneTTL)

	// return the zone information.
	return zone, nil
}

func (c *KubeClient) GetPodsByLabels(ctx context.Context, namespace string,
	matchLabels map[string]string,
) (*kcorev1.PodList, error) {
	podList := &kcorev1.PodList{}
	err := c.client.List(ctx, podList, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.Set(matchLabels).AsSelector(),
	})
	if err != nil {
		return nil, err
	}
	return podList, nil
}

func (c *KubeClient) GetNumberOfAvailabilityZonesUsedByPods(ctx context.Context,
	namespace string, matchLabels map[string]string,
) (int, error) {
	// get pods from cluster.
	podsList, err := c.GetPodsByLabels(ctx, namespace, matchLabels)
	if err != nil {
		return 0, err
	}

	// map to keep unique values of zones of pods.
	podZonesSet := map[string]bool{}

	// extract names of zones of pods.
	for _, pod := range podsList.Items {
		if pod.Spec.NodeName != "" {
			zone, err := c.GetNodeZone(ctx, pod.Spec.NodeName)
			if err != nil {
				return 0, err
			}

			// add to map.
			podZonesSet[zone] = true
		}
	}

	return len(podZonesSet), nil
}
