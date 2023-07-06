//go:build e2e
// +build e2e

package e2e_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	. "github.com/kyma-project/nats-manager/e2e/common"
	. "github.com/kyma-project/nats-manager/e2e/fixtures"
	"github.com/kyma-project/nats-manager/testutils/retry"
)

// Consts for retries; the retry and the retryGet functions.
const (
	interval = 2 * time.Second
	attempts = 60
)

// kubeConfig will not only be needed to set up the clientSet and the k8sClient, but also to forward the ports of Pods.
var kubeConfig *rest.Config //nolint:gochecknoglobals // This will only be accessible in e2e tests.

// clientSet is what is used to access K8s build-in resources like Pods, Namespaces and so on.
var clientSet *kubernetes.Clientset //nolint:gochecknoglobals // This will only be accessible in e2e tests.

// k8sClient is what is used to access the NATS CR.
var k8sClient client.Client //nolint:gochecknoglobals // This will only be accessible in e2e tests.

var logger *zap.Logger

// TestMain runs before all the other test functions. It sets up all the resources that are shared between the different
// test functions. It will then run the tests and finally shuts everything down.
func TestMain(m *testing.M) {
	l, err := SetupLogger()
	if err != nil {
		panic(err)
	}
	logger = l

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}
	kubeConfigPath := filepath.Join(userHomeDir, ".kube", "config")

	kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// Set up the clientSet that is used to access regular K8s objects.
	clientSet, err = kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// We need to add the NATS CRD to the scheme, so we can create a client that can access NATS objects.
	err = natsv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// Set up the k8s client, so we can access NATS CR-objects.
	// +kubebuilder:scaffold:scheme
	k8sClient, err = client.New(kubeConfig, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	ctx := context.TODO()
	// Create the Namespace used for testing.
	err = retry.Do(attempts, interval, logger, func() error {
		return client.IgnoreAlreadyExists(k8sClient.Create(ctx, Namespace()))
	})
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// Create the NATS CR used for testing.
	err = retry.Do(attempts, interval, logger, func() error {
		errNATS := k8sClient.Create(ctx, NATSCR())
		if k8serrors.IsAlreadyExists(errNATS) {
			logger.Warn(
				"error while creating NATS CR, resource already exist; test will continue with existing NATS CR",
			)
			return nil
		}
		return errNATS
	})
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

// Test_CR checks if the CR in the cluster is equal to what we created.
func Test_CR(t *testing.T) {
	want := NATSCR()

	ctx := context.TODO()
	actual, err := retry.Get(attempts, interval, logger, func() (*natsv1alpha1.NATS, error) {
		return getNATSCR(ctx, want.Name, want.Namespace)
	})
	require.NoError(t, err)

	require.True(t,
		reflect.DeepEqual(want.Spec, actual.Spec),
		fmt.Sprintf("wanted spec.cluster to be \n\t%v\n but got \n\t%v", want.Spec, actual.Spec),
	)
}

// Test_Pods checks if the number of Pods is the same as defined in the NATS CR and that all Pods have the resources,
// that .
func Test_Pods(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	// Get the NATS Pods and test them.
	err := retry.Do(attempts, interval, logger, func() error {
		// Get the NATS Pods via labels.
		pods, err := clientSet.CoreV1().Pods(NamespaceName).List(ctx, PodListOpts())
		if err != nil {
			return err
		}

		// The number of Pods must be equal NATS.spec.cluster.size. We check this in the retry, because it may take
		// some time for all Pods to be there.
		if len(pods.Items) != NATSCR().Spec.Cluster.Size {
			return fmt.Errorf(
				"error while fetching Pods; wanted %v Pods but got %v",
				NATSCR().Spec.Cluster.Size,
				pods.Items,
			)
		}

		// Go through all Pods, find the natsCR container in each and compare its Resources with what is defined in
		// the NATS CR.
		foundContainers := 0
		for _, pod := range pods.Items {
			for _, container := range pod.Spec.Containers {
				if !(container.Name == ContainerName) {
					continue
				}
				foundContainers += 1
				if !reflect.DeepEqual(NATSCR().Spec.Resources, container.Resources) {
					return fmt.Errorf(
						"error when checking pod %s resources:\n\twanted: %s\n\tgot: %s",
						pod.GetName(),
						NATSCR().Spec.Resources.String(),
						container.Resources.String(),
					)
				}
			}
		}
		if foundContainers != NATSCR().Spec.Cluster.Size {
			return fmt.Errorf(
				"error while fethching 'natsCR' Containers: expected %v but found %v",
				NATSCR().Spec.Cluster.Size,
				foundContainers,
			)
		}

		// Everything is fine.
		return nil
	})
	require.NoError(t, err)
}

// Test_Pods checks if the number of Pods is the same as defined in the NATS CR and that all Pods are ready.
func Test_PodsHealth(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	// Get the NATS CR. It will tell us how many Pods we should expect.
	natsCR, err := retry.Get(attempts, interval, logger, func() (*natsv1alpha1.NATS, error) {
		return getNATSCR(ctx, CRName, NamespaceName)
	})
	require.NoError(t, err)

	// Get the NATS Pods and test them.
	err = retry.Do(attempts, interval, logger, func() error {
		var pods *v1.PodList
		// Get the NATS Pods via labels.
		pods, err = clientSet.CoreV1().Pods(NamespaceName).List(ctx, PodListOpts())
		if err != nil {
			return err
		}

		// The number of Pods must be equal NATS.spec.cluster.size. We check this in the retry, because it may take
		// some time for all Pods to be there.
		if len(pods.Items) != natsCR.Spec.Cluster.Size {
			return fmt.Errorf(
				"Error while fetching pods; wanted %v Pods but got %v", natsCR.Spec.Cluster.Size, pods.Items,
			)
		}

		// Check if all Pods are ready (the status.conditions array has an entry with .type="Ready" and .status="True").
		for _, pod := range pods.Items {
			foundReadyCondition := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type != "Ready" {
					continue
				}
				foundReadyCondition = true
				if cond.Status != "True" {
					return fmt.Errorf(
						"Pod %s has 'Ready' conditon '%s' but wanted 'True'.", pod.GetName(), cond.Status,
					)
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

	// Get the NATS CR. It will tell us how many PVCs we should expect and what their size should be.
	ctx := context.TODO()
	// Get the PersistentVolumeClaims, PVCs, and test them.
	var pvcs *v1.PersistentVolumeClaimList
	err := retry.Do(attempts, interval, logger, func() error {
		// Get PVCs via a label.
		var err error
		pvcs, err = retry.Get(attempts, interval, logger, func() (*v1.PersistentVolumeClaimList, error) {
			return clientSet.CoreV1().PersistentVolumeClaims(NamespaceName).List(ctx, PVCListOpts())
		})
		if err != nil {
			return err
		}

		// Check if the amount of PVCs is equal to the spec.cluster.size in the NATS CR. We do this in the retry,
		// because it may take some time for all PVCs to be there.
		want, actual := NATSCR().Spec.Cluster.Size, len(pvcs.Items)
		if want != actual {
			return fmt.Errorf("error while fetching PVSs; wanted %v PVCs but got %v", want, actual)
		}

		// Everything is fine.
		return nil
	})
	require.NoError(t, err)

	// Compare the PVC's sizes with the definition in the CRD.
	for _, pvc := range pvcs.Items {
		size := pvc.Spec.Resources.Requests[v1.ResourceStorage]
		require.True(t, size.Equal(NATSCR().Spec.FileStorage.Size))
	}
}

func getNATSCR(ctx context.Context, name, namespace string) (*natsv1alpha1.NATS, error) {
	var natsCR natsv1alpha1.NATS
	err := k8sClient.Get(ctx, k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &natsCR)
	return &natsCR, err
}
