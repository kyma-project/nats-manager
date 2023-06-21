//go:build e2e
// +build e2e

package e2e_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const kymaSystem = "kyma-system"
const eventingNats = "eventing-nats"

func Test_podsHealthy(t *testing.T) {
	userHomeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	kubeConfigPath := filepath.Join(userHomeDir, ".kube", "config")
	fmt.Printf("Using kubeconfig: %s\n", kubeConfigPath)

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	require.NoError(t, err)

	clientSet, err := kubernetes.NewForConfig(kubeConfig)
	require.NoError(t, err)

	ctx := context.TODO()
	ns, err := clientSet.CoreV1().Namespaces().Get(ctx, kymaSystem, metav1.GetOptions{})
	require.NoError(t, err)
	fmt.Printf("found namespace: '%s'", ns.GetName())

	// Get the StatefulSet.
	sts, err := clientSet.AppsV1().StatefulSets(kymaSystem).Get(ctx, eventingNats, metav1.GetOptions{})
	require.NoError(t, err)

	// Get the pods via labels.
	listOptions := metav1.ListOptions{
		LabelSelector: "nats_cluster=eventing-nats",
	}
	pods, err := clientSet.CoreV1().Pods(kymaSystem).List(ctx, listOptions)
	require.NoError(t, err)

	fmt.Printf("\n the sts has `replicas=%v` and there are %v pods \n", int(*sts.Spec.Replicas), len(pods.Items))
	assert.Equal(t, int(*sts.Spec.Replicas), len(pods.Items))

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