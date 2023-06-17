package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const kymaSystem = "kyma-system"
const eventingNats = "eventing-nats"

func main() {
	// Get the kubeconfig.
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	// Extract the config from the kubeconfig.
	config, err := kubeconfig.ClientConfig()
	if err != nil {
		panic(err)
	}

	// Get a clientSet to connect to K8s.
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	ctx := context.TODO()
	sts, err := clientSet.AppsV1().StatefulSets(kymaSystem).Get(ctx, eventingNats, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	// Get the pods via a label selector.
	listOptions := metav1.ListOptions{
		LabelSelector: "nats_cluster=eventing-nats",
	}
	pods, err := clientSet.CoreV1().Pods(kymaSystem).List(ctx, listOptions)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n the sts has `replicas=%v` and there are %v pods \n", int(*sts.Spec.Replicas), len(pods.Items))

	// Check that all Pods are Ready (the status.conditions array has an entry with .type="Ready" and the
	// .status="True")
	for _, pod := range pods.Items {
		fmt.Printf("\n the pod %s ", pod.GetName())
		for _, cond := range pod.Status.Conditions {
			if cond.Type == "Ready" {
				fmt.Printf("has the condition of type %s ", cond.Type)
				fmt.Printf("and the status %s \n", cond.Status)
			}
		}
	}
}
