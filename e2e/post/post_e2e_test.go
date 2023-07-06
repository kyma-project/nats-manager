//go:build poste2e
// +build poste2e

package post

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	. "github.com/kyma-project/nats-manager/e2e/common"
	"github.com/kyma-project/nats-manager/e2e/fixtures"
	"github.com/kyma-project/nats-manager/testutils/retry"
)

// Consts for retries; the retry and the retryGet functions.
const (
	interval = 5 * time.Second
	attempts = 20
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

	// Create the NATS CR used for testing.
	ctx := context.TODO()
	err = retry.Do(attempts, interval, logger, func() error {
		errDel := k8sClient.Delete(ctx, fixtures.NATSCR())
		return errDel
	})
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

func Test_NoPodExists(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	err := retry.Do(attempts, interval, logger, func() error {
		pods, podErr := clientSet.CoreV1().Pods(fixtures.NamespaceName).List(ctx, fixtures.PodListOpts())
		if podErr != nil {
			return podErr
		}

		if l := len(pods.Items); l > 0 {
			return fmt.Errorf("expected to not find any pods but found %v", l)
		}

		return nil
	})
	require.NoError(t, err)
}

func Test_NoPVCExists(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	err := retry.Do(attempts, interval, logger, func() error {
		pvcs, pvcErr := clientSet.CoreV1().PersistentVolumeClaims(fixtures.NamespaceName).List(ctx, fixtures.PVCListOpts())
		if pvcErr != nil {
			return pvcErr
		}

		if l := len(pvcs.Items); l > 0 {
			return fmt.Errorf("expected to not find any PVCs but found %v", l)
		}

		return nil
	})
	require.NoError(t, err)
}
