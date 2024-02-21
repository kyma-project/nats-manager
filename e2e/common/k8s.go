package common

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
)

func GetK8sClients() (*kubernetes.Clientset, client.Client, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}
	kubeConfigPath := filepath.Join(userHomeDir, ".kube", "config")

	var kubeConfig *rest.Config
	kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, nil, err
	}

	// Set up the clientSet that is used to access regular K8s objects.
	var clientSet *kubernetes.Clientset
	clientSet, err = kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, nil, err
	}

	// We need to add the NATS CRD to the scheme, so we can create a client that can access NATS objects.
	err = nmapiv1alpha1.AddToScheme(kscheme.Scheme)
	if err != nil {
		return nil, nil, err
	}

	// Set up the k8s client, so we can access NATS CR-objects.
	// +kubebuilder:scaffold:scheme
	var k8sClient client.Client
	k8sClient, err = client.New(kubeConfig, client.Options{Scheme: kscheme.Scheme})
	if err != nil {
		return nil, nil, err
	}

	return clientSet, k8sClient, nil
}
