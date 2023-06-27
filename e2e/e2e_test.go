//go:build e2e
// +build e2e

package e2e_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
)

const (
	kymaSystem            = "kyma-system"
	eventingNats          = "eventing-nats"
	natsCLusterLabel      = "nats_cluster=eventing-nats"
	nameNatsLabel         = "app.kubernetes.io/name=nats"
	instanceEventingLabel = "app.kubernetes.io/instance=eventing"
)

const (
	interval = 10 * time.Second
	attempts = 30
)

// clientSet is what is used to access K8s build-in resources like Pods, Namespaces and so on.
var clientSet *kubernetes.Clientset //nolint:gochecknoglobals // This will only be accessible in e2e tests.

// k8sClient is what is used to access the NATS CR.
var k8sClient client.Client //nolint:gochecknoglobals // This will only be accessible in e2e tests.

// TestMain runs before all the other test functions. It sets up all the resources that are shared between the different
// test functions. It will then run the tests and finally shuts everything down.
func TestMain(m *testing.M) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	kubeConfigPath := filepath.Join(userHomeDir, ".kube", "config")

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		panic(err)
	}

	// Set up the clientSet that is used to access regular K8s objects.
	clientSet, err = kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	// We need to add the NATS CRD to the scheme, so we can create a client that can access NATS objects.
	err = natsv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}

	// Set up the k8s client, so we can access NATS CR-objects.
	// +kubebuilder:scaffold:scheme
	k8sClient, err = client.New(kubeConfig, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		panic(err)
	}

	// Run the tests.
	code := m.Run()

	os.Exit(code)
}

// Test_namespace_was_created tries to get the namespace from the cluster.
func Test_NamespaceWasCreated(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	_, err := retryGet(attempts, interval, func() (*v1.Namespace, error) {
		return clientSet.CoreV1().Namespaces().Get(ctx, kymaSystem, metav1.GetOptions{})
	})
	require.NoError(t, err)
}

// Test_Pods checks if the number of Pods is the same as defined in the NATS CR and that all Pods are ready.
func Test_Pods(t *testing.T) {
	t.Parallel()

	// Get the NATS CR. It will tell us how many Pods we should expect.
	ctx := context.TODO()
	nats, err := retryGet(attempts, interval,
		func() (*natsv1alpha1.NATS, error) {
			return getNATS(ctx, eventingNats, kymaSystem)
		})
	require.NoError(t, err)

	// Get the NATS Pods and test them.
	listOptions := metav1.ListOptions{LabelSelector: natsCLusterLabel}
	err = retry(attempts, interval, func() error {
		var pods *v1.PodList
		// Get the NATS Pods via labels.
		pods, err = clientSet.CoreV1().Pods(kymaSystem).List(ctx, listOptions)
		if err != nil {
			return err
		}

		// The number of Pods must be equal NATS.spec.cluster.size. We check this in the retry, because it may take
		// some time for all Pods to be there.
		if len(pods.Items) != nats.Spec.Cluster.Size {
			return fmt.Errorf(
				"Error while fetching pods; wanted %v Pods but got %v", nats.Spec.Cluster.Size, pods.Items,
			)
		}

		// Check if all Pods are ready (the status.conditions array has an entry with .type="Ready" and .status="True").
		for _, pod := range pods.Items {
			foundReadyCondition := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type == "Ready" {
					foundReadyCondition = true
					expected := "True"
					actual := fmt.Sprintf("%v", cond.Status)
					if expected != actual {
						return fmt.Errorf(
							"Pod %s has 'Ready' conditon '%s' but wanted 'True'.", pod.GetName(), actual,
						)
					}
				}
			}
			if !foundReadyCondition {
				return fmt.Errorf("Could not find 'Ready' condition for Pod %s", pod.GetName())
			}
		}

		// Everything is fine.
		return nil
	})
	require.NoError(t, err)
}

// Test PVCs will test if any PVCs can be found, if their number is equal to the NATS CR's `spec.cluster.size` and if
// they all have the right size, as defined in `spec.jetStream.fileStorage`.
func Test_PVCs(t *testing.T) {
	t.Parallel()

	// Get the NATS CR. It will tell us how many PVCs we should expect.
	ctx := context.TODO()
	nats, err := retryGet(attempts, interval,
		func() (*natsv1alpha1.NATS, error) {
			return getNATS(ctx, eventingNats, kymaSystem)
		})
	require.NoError(t, err)

	// Get the PersistentVolumeClaims, PVCs, and test them.
	var pvcs *v1.PersistentVolumeClaimList
	listOpt := metav1.ListOptions{LabelSelector: nameNatsLabel}
	err = retry(attempts, interval, func() error {
		// Get PVCs via a label.
		pvcs, err = retryGet(attempts, interval, func() (*v1.PersistentVolumeClaimList, error) {
			return clientSet.CoreV1().PersistentVolumeClaims(kymaSystem).List(ctx, listOpt)
		})
		if err != nil {
			return err
		}

		// Check if the amount of PVCs is equal to the spec.cluster.size in the NATS CR. We do this in the retry,
		// because it may take some time for all PVCs to be there.
		want, actual := nats.Spec.Cluster.Size, len(pvcs.Items)
		if want != actual {
			return fmt.Errorf("Error while fetching PVSs; wanted %v PVCs but got %v", want, actual)
		}

		// Everything is fine.
		return nil
	})
	require.NoError(t, err)

	// Compare the PVC's sizes with the definition in the CRD.
	for _, pvc := range pvcs.Items {
		size := pvc.Spec.Resources.Requests[v1.ResourceStorage]
		require.True(t, size.Equal(nats.Spec.FileStorage.Size))
	}
}

func retry(attempts int, interval time.Duration, fn func() error) error {
	ticker := time.NewTicker(interval)
	var err error
	for {
		select {
		case <-ticker.C:
			attempts -= 1
			err = fn()
			if err == nil || attempts == 0 {
				return err
			}
		}
	}
}

func retryGet[T any](attempts int, interval time.Duration, fn func() (*T, error)) (*T, error) {
	ticker := time.NewTicker(interval)
	var err error
	var obj *T
	for {
		select {
		case <-ticker.C:
			attempts -= 1
			obj, err = fn()
			if err == nil || attempts == 0 {
				return obj, err
			}
		}
	}
}

func getNATS(ctx context.Context, name, namespace string) (*natsv1alpha1.NATS, error) {
	var nats natsv1alpha1.NATS
	err := k8sClient.Get(ctx, k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &nats)
	return &nats, err
}
