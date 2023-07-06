//go:build e2e
// +build e2e

package post

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
// todo maybe put this to the fixtures
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

	// Delete the NATS CR.
	ctx := context.TODO()
	err = retry.Do(attempts, interval, logger, func() error {
		errDel := k8sClient.Delete(ctx, NATSCR())
		if k8serrors.IsNotFound(errDel) {
			return nil
		}
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

func Test_NoPodsExists(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	err := retry.Do(attempts, interval, logger, func() error {
		pods, podErr := clientSet.CoreV1().Pods(NamespaceName).List(ctx, PodListOpts())
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

// Test_NoPVCsExists verifies that no PVC, that was created in the E2E test, sti
func Test_NoPVCsExists(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	err := retry.Do(attempts, interval, logger, func() error {
		pvcs, pvcErr := clientSet.CoreV1().PersistentVolumeClaims(NamespaceName).List(ctx, PVCListOpts())
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

func Test_NoSTSExists(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	err := retry.Do(attempts, interval, logger, func() error {
		// Try, if we still can get the STS.
		_, stsErr := clientSet.AppsV1().StatefulSets(NamespaceName).Get(ctx, STSName, v1.GetOptions{})
		// This is what we want here.
		if k8serrors.IsNotFound(stsErr) {
			return nil
		}
		// All other errors are unexpected here.
		if stsErr != nil {
			return stsErr
		}
		// If we still find and STS we will return an error.
		return errors.New("found sts, but wanted the sts to be deleted")
	})
	require.NoError(t, err)
}

func Test_NoNATSSecretExists(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	err := retry.Do(attempts, interval, logger, func() error {
		_, secErr := clientSet.CoreV1().Secrets(NamespaceName).Get(ctx, SecretName, v1.GetOptions{})
		// This is what we want here.
		if k8serrors.IsNotFound(secErr) {
			return nil
		}
		// All other errors are unexpected here.
		if secErr != nil {
			return secErr
		}
		// If we still find and STS we will return an error.
		return errors.New("found Secret, but wanted the sts to be deleted")
	})
	require.NoError(t, err)
}

func Test_NoNATSCRExists(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	err := retry.Do(attempts, interval, logger, func() error {
		_, crErr := getNATSCR(ctx, CRName, NamespaceName)
		// This is what we want here.
		if k8serrors.IsNotFound(crErr) {
			return nil
		}
		// All other errors are unexpected here.
		if crErr != nil {
			return crErr
		}
		// If we still find the CR we will return an error.
		return errors.New("found NATS CR, but wanted the NATS CR to be deleted")
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
