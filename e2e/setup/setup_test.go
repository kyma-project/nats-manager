//go:build e2e
// +build e2e

// Package setup_test is part of the end-to-end-tests. This package contains tests that evaluate the creation of a
// NATS-server CR and the creation of all correlated Kubernetes resources.
// To run the tests a Kubernetes cluster and a NATS-CR need to be available and configured. For this reason, the tests
// are seperated via the `e2e` buildtags. For more information please consult the `readme.md`.
package setup_test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	. "github.com/kyma-project/nats-manager/e2e/common"
	. "github.com/kyma-project/nats-manager/e2e/common/fixtures"
)

// Constants for retries.
const (
	interval = 3 * time.Second
	attempts = 120
)

// clientSet is what is used to access K8s build-in resources like Pods, Namespaces and so on.
var clientSet *kubernetes.Clientset //nolint:gochecknoglobals // This will only be accessible in e2e tests.

// k8sClient is what is used to access the NATS CR.
var k8sClient client.Client //nolint:gochecknoglobals // This will only be accessible in e2e tests.

var logger *zap.Logger

// TestMain runs before all the other test functions. It sets up all the resources that are shared between the different
// test functions. It will then run the tests and finally shuts everything down.
func TestMain(m *testing.M) {
	var err error
	logger, err = SetupLogger()
	if err != nil {
		logger.Fatal(err.Error())
	}

	clientSet, k8sClient, err = GetK8sClients()
	if err != nil {
		logger.Fatal(err.Error())
	}

	ctx := context.TODO()
	// Create the Namespace used for testing.
	err = Retry(attempts, interval, func() error {
		// It's fine if the Namespace already exists.
		return client.IgnoreAlreadyExists(k8sClient.Create(ctx, Namespace()))
	})
	if err != nil {
		logger.Fatal(err.Error())
	}

	// Wait for NATS-manager deployment to get ready.
	managerImage := ""
	if _, ok := os.LookupEnv("MANAGER_IMAGE"); ok {
		managerImage = os.Getenv("MANAGER_IMAGE")
	} else {
		logger.Warn(
			"ENV `MANAGER_IMAGE` is not set. Test will not verify if the " +
				"manager deployment image is correct or not.",
		)
	}
	if err := waitForNATSManagerDeploymentReady(managerImage); err != nil {
		logger.Fatal(err.Error())
	}

	// Create the NATS CR used for testing.
	err = Retry(attempts, interval, func() error {
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
		logger.Fatal(err.Error())
	}

	// wait for an interval for reconciliation to update status.
	time.Sleep(interval)

	// Wait for NATS CR to get ready.
	if err := waitForNATSCRReady(); err != nil {
		logger.Fatal(err.Error())
	}

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

// Test_CR checks if the CR in the cluster is equal to what we defined.
func Test_CR(t *testing.T) {
	want := NATSCR()

	ctx := context.TODO()
	// Get the NATS CR from the cluster.
	actual, err := RetryGet(attempts, interval, func() (*natsv1alpha1.NATS, error) {
		return getNATSCR(ctx, want.Name, want.Namespace)
	})
	require.NoError(t, err)

	require.True(t,
		reflect.DeepEqual(want.Spec, actual.Spec),
		fmt.Sprintf("wanted spec.cluster to be \n\t%v\n but got \n\t%v", want.Spec, actual.Spec),
	)
}

// Test_PriorityClass will get the PriorityClass name from the StatefulSet and checks if a PriorityClass with that
// name exists in the cluster.
func Test_PriorityClass(t *testing.T) {
	ctx := context.TODO()

	err := Retry(attempts, interval, func() error {
		sts, stsErr := clientSet.AppsV1().StatefulSets(NamespaceName).Get(ctx, STSName, metav1.GetOptions{})
		if stsErr != nil {
			return stsErr
		}

		pcName := sts.Spec.Template.Spec.PriorityClassName
		// todo remove this check after the next release.
		if len(pcName) < 1 {
			return nil
		}

		if pcName != PriorityClassName {
			return fmt.Errorf("PriorityClassName was expected to be %s but was %s", PriorityClassName, pcName)
		}

		_, pcErr := clientSet.SchedulingV1().PriorityClasses().Get(ctx, pcName, metav1.GetOptions{})
		return pcErr
	})

	require.NoError(t, err)
}

// Test_ConfigMap tests the ConfigMap that the NATS-Manger creates when we define a CR.
func Test_ConfigMap(t *testing.T) {
	ctx := context.TODO()

	err := Retry(attempts, interval, func() error {
		cm, cmErr := clientSet.CoreV1().ConfigMaps(NamespaceName).Get(ctx, CMName, metav1.GetOptions{})
		if cmErr != nil {
			return cmErr
		}

		cmMap := cmToMap(cm.Data["nats.conf"])

		if err := checkValueInCMMap(cmMap, "debug", True); err != nil {
			return err
		}

		if err := checkValueInCMMap(cmMap, "trace", True); err != nil {
			return err
		}

		// **********************
		// TODO: remove this section when NATS server 2.10.x is released.
		// `max_file` is changed to `max_file_store` in NATS 2.10.x.
		// `max_mem` is changed to `max_memory_store` in NATS 2.10.x.
		// To check the correct key in configMap,
		// fetch the NATS statefulSet and get the NATS server version.
		// And then based on the version, check the expected key.
		sts, stsErr := clientSet.AppsV1().StatefulSets(NamespaceName).Get(ctx, STSName, metav1.GetOptions{})
		if stsErr != nil {
			return stsErr
		}

		imageName := ""
		for _, c := range sts.Spec.Template.Spec.Containers {
			if c.Name == ContainerName {
				imageName = c.Image
			}
		}
		if strings.Contains(imageName, "2.9.") {
			if err := checkValueInCMMap(cmMap, "max_file", FileStorageSize); err != nil {
				return err
			}

			if err := checkValueInCMMap(cmMap, "max_mem", MemStorageSize); err != nil {
				return err
			}
			return nil
		}
		// **********************

		if err := checkValueInCMMap(cmMap, "max_file_store", FileStorageSize); err != nil {
			return err
		}

		if err := checkValueInCMMap(cmMap, "max_memory_store", MemStorageSize); err != nil {
			return err
		}

		return nil
	})

	require.NoError(t, err)
}

// Test_PodsResources checks if the number of Pods is the same as defined in the NATS CR and that all Pods have the resources,
// that are defined in the CRD.
func Test_PodsResources(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	// RetryGet the NATS Pods and test them.
	err := Retry(attempts, interval, func() error {
		// RetryGet the NATS Pods via labels.
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

// Test_PodsReady checks if the number of Pods is the same as defined in the NATS CR and that all Pods are ready.
func Test_PodsReady(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	// RetryGet the NATS CR. It will tell us how many Pods we should expect.
	natsCR, err := RetryGet(attempts, interval, func() (*natsv1alpha1.NATS, error) {
		return getNATSCR(ctx, CRName, NamespaceName)
	})
	require.NoError(t, err)

	// RetryGet the NATS Pods and test them.
	err = Retry(attempts, interval, func() error {
		var pods *v1.PodList
		// RetryGet the NATS Pods via labels.
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
						"Pod %s has 'Ready' conditon '%s' but wanted 'True'", pod.GetName(), cond.Status,
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

	// Get the PersistentVolumeClaims --PVCs-- and test them.
	ctx := context.TODO()
	var pvcs *v1.PersistentVolumeClaimList
	err := Retry(attempts, interval, func() error {
		// RetryGet PVCs via a label.
		var err error
		pvcs, err = RetryGet(attempts, interval, func() (*v1.PersistentVolumeClaimList, error) {
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

// Test_Secret tests if the Secret was created.
func Test_Secret(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	err := Retry(attempts, interval, func() error {
		_, secErr := clientSet.CoreV1().Secrets(NamespaceName).Get(ctx, SecretName, metav1.GetOptions{})
		if secErr != nil {
			return secErr
		}
		return nil
	})
	require.NoError(t, err)
}

func getNATSCR(ctx context.Context, name, namespace string) (*natsv1alpha1.NATS, error) {
	var natsCR natsv1alpha1.NATS
	err := k8sClient.Get(ctx, k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &natsCR)
	return &natsCR, err
}

func getDeployment(ctx context.Context, name, namespace string) (*appsv1.Deployment, error) {
	return clientSet.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
}

func cmToMap(cm string) map[string]string {
	lines := strings.Split(cm, "\n")

	cmMap := make(map[string]string)
	for _, line := range lines {
		l := strings.Split(line, ": ")
		if len(l) < 2 {
			continue
		}
		key := strings.TrimSpace(l[0])
		val := strings.TrimSpace(l[1])
		cmMap[key] = val
	}

	return cmMap
}

func checkValueInCMMap(cmm map[string]string, key, expectedValue string) error {
	val, ok := cmm[key]
	if !ok {
		return fmt.Errorf("could net get '%s' from Configmap", key)
	}

	if val != expectedValue {
		return fmt.Errorf("expected value for '%s' to be '%s' but was '%s'", key, expectedValue, val)
	}

	return nil
}

// Wait for NATS CR to get ready.
func waitForNATSCRReady() error {
	// RetryGet the NATS CR and test status.
	return Retry(attempts, interval, func() error {
		want := NATSCR()
		logger.Debug(fmt.Sprintf("waiting for NATS CR to get ready. "+
			"CR name: %s, namespace: %s", want.Name, want.Namespace))

		ctx := context.TODO()
		// Get the NATS CR from the cluster.
		gotNATSCR, err := RetryGet(attempts, interval, func() (*natsv1alpha1.NATS, error) {
			return getNATSCR(ctx, want.Name, want.Namespace)
		})
		if err != nil {
			return err
		}

		if gotNATSCR.Status.State != natsv1alpha1.StateReady {
			err := fmt.Errorf("waiting for NATS CR to get ready state")
			logger.Debug(err.Error())
			return err
		}

		// Everything is fine.
		logger.Debug(fmt.Sprintf("NATS CR is ready. "+
			"CR name: %s, namespace: %s", want.Name, want.Namespace))
		return nil
	})
}

// Wait for NATS-manager deployment to get ready with correct image.
func waitForNATSManagerDeploymentReady(image string) error {
	// RetryGet the NATS Manager and test status.
	return Retry(attempts, interval, func() error {
		logger.Debug(fmt.Sprintf("waiting for nats-manager deployment to get ready with image: %s", image))
		ctx := context.TODO()
		// Get the NATS-manager deployment from the cluster.
		gotDeployment, err := RetryGet(attempts, interval, func() (*appsv1.Deployment, error) {
			return getDeployment(ctx, ManagerDeploymentName, NamespaceName)
		})
		if err != nil {
			return err
		}

		// if image is provided, then check if the deployment has correct image.
		if image != "" && gotDeployment.Spec.Template.Spec.Containers[0].Image != image {
			err := fmt.Errorf("expected NATS-manager image to be: %s, but found: %s", image,
				gotDeployment.Spec.Template.Spec.Containers[0].Image,
			)
			logger.Debug(err.Error())
			return err
		}

		// check if the deployment is ready.
		if *gotDeployment.Spec.Replicas != gotDeployment.Status.UpdatedReplicas ||
			*gotDeployment.Spec.Replicas != gotDeployment.Status.ReadyReplicas ||
			*gotDeployment.Spec.Replicas != gotDeployment.Status.AvailableReplicas {
			err := fmt.Errorf("waiting for NATS-manager deployment to get ready")
			logger.Debug(err.Error())
			return err
		}

		// Everything is fine.
		logger.Debug(fmt.Sprintf("nats-manager deployment is ready with image: %s", image))
		return nil
	})
}
