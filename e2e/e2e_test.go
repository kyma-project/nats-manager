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
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	kymaSystem       = "kyma-system"
	eventingNats     = "eventing-nats"
	natsCLusterLabel = "nats_cluster=eventing-nats"
)

const (
	interval = 10 * time.Second
	attempts = 30
	delay    = 10 * time.Second
)

var clientset *kubernetes.Clientset

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

func TestMain(m *testing.M) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	kubeConfigPath := filepath.Join(userHomeDir, ".kube", "config")
	fmt.Printf("Using kubeconfig: %s\n", kubeConfigPath)

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		panic(err)
	}

	clientset, err = kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		panic(err)
	}
}

// Test_namespace_was_created tries to get the namespace from the cluster.
func Test_namespace_was_created(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	_, err := retryGet(attempts, interval, func() (*v1.Namespace, error) {
		return clientset.CoreV1().Namespaces().Get(ctx, kymaSystem, metav1.GetOptions{})
	})
	require.NoError(t, err)
}

func Test_PodsHealthy(t *testing.T) {
	t.Parallel()

	// Get the StatefulSet.
	ctx := context.TODO()
	sts, err := retryGet(attempts, interval, func() (*appsv1.StatefulSet, error) {
		return clientset.AppsV1().StatefulSets(kymaSystem).Get(ctx, eventingNats, metav1.GetOptions{})
	})
	require.NoError(t, err)

	err = retry(attempts, interval, func() error {
		// Get the NATS pods via labels.
		listOptions := metav1.ListOptions{LabelSelector: natsCLusterLabel}
		var pods *v1.PodList
		pods, err = clientset.CoreV1().Pods(kymaSystem).List(ctx, listOptions)
		if err != nil {
			return err
		}

		// The number of Pods must be equal to the number of Replicas in the StatefulSet.
		if int32(len(pods.Items)) != *sts.Spec.Replicas {
			return fmt.Errorf("Error while fetching pods; wanted %v Pods but got %v", sts.Spec.Replicas, pods.Items)
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
						return fmt.Errorf("Pod %s has 'Ready' conditon '%s' but wanted 'True'.", pod.GetName(), actual)
					}
				}
			}
			if !foundReadyCondition {
				return fmt.Errorf("Could not find 'Ready' condition for Pod %s", pod.GetName())
			}
		}

		return nil
	})
	require.NoError(t, err)
}
