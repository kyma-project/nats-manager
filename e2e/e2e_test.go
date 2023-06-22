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

	"github.com/avast/retry-go/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	kymaSystem   = "kyma-system"
	eventingNats = "eventing-nats"
)

const (
	timeout  = 3 * time.Minute
	interval = 10 * time.Second
)

const (
	attempts = 30
	delay    = 10 * time.Second
)

var clientset *kubernetes.Clientset

func Retry[T any](timeout, interval time.Duration, fn func() (*T, error)) (*T, error) {
	var err error
	var obj *T
	for start := time.Now(); time.Since(start) <= timeout; {
		obj, err = fn()
		if err == nil {
			return obj, err
		}
		time.Sleep(interval)
	}
	return obj, err
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

// Test_namespace_was_created simply tries to get the namespace on the cluster.
func Test_namespace_was_created(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	ns, err := Retry(timeout, interval, func() (*v1.Namespace, error) {
		return clientset.CoreV1().Namespaces().Get(ctx, kymaSystem, metav1.GetOptions{})
	},
	)
	println(ns.GetName())
	require.NoError(t, err)
}

func Test_podsHealthy(t *testing.T) {
	t.Parallel()

	// Get the StatefulSet.
	ctx := context.TODO()
	sts, err := Retry(timeout, interval, func() (*appsv1.StatefulSet, error) {
		return clientset.AppsV1().StatefulSets(kymaSystem).Get(ctx, eventingNats, metav1.GetOptions{})
	})
	require.NoError(t, err)

	// Get the pods via labels.
	var pods *v1.PodList
	listOptions := metav1.ListOptions{
		LabelSelector: "nats_cluster=eventing-nats",
	}
	err = retry.Do(
		func() error {
			pods, err = clientset.CoreV1().Pods(kymaSystem).List(ctx, listOptions)
			if err != nil {
				return err
			}
			if len(pods.Items) != int(*sts.Spec.Replicas) {
				return fmt.Errorf("Error fetching Pods; expexted %v but is %v", sts.Spec.Replicas, len(pods.Items))
			}
			return nil
		},
		retry.Delay(delay),
		retry.Attempts(attempts),
	)
	require.NoError(t, err)

	// Check if all Pods are ready (the status.conditions array has an entry with .type="Ready" and the
	// .status="True").
	for _, pod := range pods.Items {
		fmt.Printf("\n the pod %s ", pod.GetName())
		for _, cond := range pod.Status.Conditions {
			if cond.Type == "Ready" {
				expected := "True"
				actual := fmt.Sprintf("%v", cond.Status)
				assert.Equal(t, expected, actual)
			}
		}
	}
}
