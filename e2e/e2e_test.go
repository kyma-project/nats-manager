//go:build e2e
// +build e2e

package e2e_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"sigs.k8s.io/controller-runtime/pkg/client"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/testutils"
)

var errNamespaceExists = errors.New(fmt.Sprint("namespaces \" \" already exists", e2eNamespace))

const (
	e2eNamespace          = "kyma-system"
	eventingNats          = "eventing-nats"
	natsCLusterLabel      = "nats_cluster=eventing-nats"
	nameNatsLabel         = "app.kubernetes.io/name=nats"
	instanceEventingLabel = "app.kubernetes.io/instance=eventing"
	containerName         = "nats"
)

// Const for retries; the retry and the retryGet functions.
const (
	interval = 10 * time.Second
	attempts = 30
)

// kubeConfig will not only be needed to set up the clientSet and the k8sClient, but also to forward the ports of Pods.
var kubeConfig *rest.Config //nolint:gochecknoglobals // This will only be accessible in e2e tests.

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

	kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
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

	// Set up namespace and nats cr.
	ctx := context.TODO()
	ns := testutils.NewNamespace(e2eNamespace)
	err = retry(attempts, interval, func() error {
		nsErr := k8sClient.Create(ctx, ns)
		// If the error is only, that the namespaces already exists, we are fine.
		if errors.Is(nsErr, errNamespaceExists) {
			return nil
		}
		return nsErr
	})
	if err != nil {
		panic(err)
	}

	// Create a NATS CR.
	natsCR := testutils.NewNATSCR(
		testutils.WithNATSCRName(eventingNats),
		testutils.WithNATSCRNamespace(e2eNamespace),
		testutils.WithNATSClusterSize(3),
		testutils.WithNATSFileStorage(natsv1alpha1.FileStorage{
			StorageClassName: "default",
			Size:             resource.MustParse("1Gi"),
		}),
		testutils.WithNATSMemStorage(natsv1alpha1.MemStorage{
			Enabled: false,
			Size:    resource.MustParse("20Mi"),
		}),
		testutils.WithNATSResources(corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"cpu":    resource.MustParse("20m"),
				"memory": resource.MustParse("64Mi"),
			},
			Requests: corev1.ResourceList{
				"cpu":    resource.MustParse("5m"),
				"memory": resource.MustParse("16Mi"),
			},
		}),
		testutils.WithNATSLogging(
			true,
			true,
		),
	)
	err = retry(attempts, interval, func() error {
		return k8sClient.Create(ctx, natsCR)
	})
	if err != nil {
		panic(err)
	}

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

// Test_CreateNATSCR create the namespace and the
func Test_CreateNATSCR(t *testing.T) {
	t.Parallel()

	// Create a Namespace.
	ctx := context.TODO()
	ns := testutils.NewNamespace(e2eNamespace)
	err := retry(attempts, interval, func() error {
		nsErr := k8sClient.Create(ctx, ns)
		// If the error is only, that the namespaces already exists, we are fine.
		if errors.Is(nsErr, errNamespaceExists) {
			return nil
		}
		return nsErr
	})
	// todo, question, will there be an err if this ns already exists?
	require.NoError(t, err)

	// Create a NATS CR.
	natsCR := testutils.NewNATSCR(
		testutils.WithNATSCRName(eventingNats),
		testutils.WithNATSCRNamespace(e2eNamespace),
		testutils.WithNATSClusterSize(3),
		testutils.WithNATSFileStorage(natsv1alpha1.FileStorage{
			StorageClassName: "default",
			Size:             resource.MustParse("1Gi"),
		}),
		testutils.WithNATSMemStorage(natsv1alpha1.MemStorage{
			Enabled: false,
			Size:    resource.MustParse("20Mi"),
		}),
		testutils.WithNATSResources(corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"cpu":    resource.MustParse("20m"),
				"memory": resource.MustParse("64Mi"),
			},
			Requests: corev1.ResourceList{
				"cpu":    resource.MustParse("5m"),
				"memory": resource.MustParse("16Mi"),
			},
		}),
		testutils.WithNATSLogging(
			true,
			true,
		),
	)
	err = retry(attempts, interval, func() error {
		return k8sClient.Create(ctx, natsCR)
	})
	require.NoError(t, err)
}

// Test_namespace_was_created tries to get the namespace from the cluster.
func Test_NamespaceWasCreated(t *testing.T) {
	t.Parallel()

	// We will not do anything with the Namespace, so will only try to get it.
	ctx := context.TODO()
	_, err := retryGet(attempts, interval, func() (*v1.Namespace, error) {
		return clientSet.CoreV1().Namespaces().Get(ctx, e2eNamespace, metav1.GetOptions{})
	})
	require.NoError(t, err)
}

// Test_Pods checks if the number of Pods is the same as defined in the NATS CR and that all Pods have the resources,
// that .
func Test_PodResources(t *testing.T) {
	t.Parallel()

	// Get the NATS CR. It will tell us how many Pods we should expect and what the resources should be configured to.
	ctx := context.TODO()
	nats, err := retryGet(attempts, interval, func() (*natsv1alpha1.NATS, error) {
		return getNATS(ctx, eventingNats, e2eNamespace)
	})
	require.NoError(t, err)

	// Get the NATS Pods and test them.
	listOptions := metav1.ListOptions{LabelSelector: natsCLusterLabel}
	err = retry(attempts, interval, func() error {
		// Get the NATS Pods via labels.
		var pods *v1.PodList
		pods, err = clientSet.CoreV1().Pods(e2eNamespace).List(ctx, listOptions)
		if err != nil {
			return err
		}

		// The number of Pods must be equal NATS.spec.cluster.size. We check this in the retry, because it may take
		// some time for all Pods to be there.
		if len(pods.Items) != nats.Spec.Cluster.Size {
			return fmt.Errorf(
				"error while fetching Pods; wanted %v Pods but got %v",
				nats.Spec.Cluster.Size,
				pods.Items,
			)
		}

		// Go through all Pods, find the nats container in each and compare its Resources with what is defined in
		// the NATS CR.
		foundContainers := 0
		for _, pod := range pods.Items {
			for _, container := range pod.Spec.Containers {
				if !(container.Name == containerName) {
					continue
				}
				foundContainers += 1
				if !reflect.DeepEqual(nats.Spec.Resources, container.Resources) {
					return fmt.Errorf(
						"error when checking pod %s resources:\n\twanted: %s\n\tgot: %s",
						pod.GetName(),
						nats.Spec.Resources.String(),
						container.Resources.String(),
					)
				}
			}
		}
		if foundContainers != nats.Spec.Cluster.Size {
			return fmt.Errorf(
				"error while fethching 'nats' Containers: expected %v but found %v",
				nats.Spec.Cluster.Size,
				foundContainers,
			)
		}

		// Everything is fine.
		return nil
	})
	require.NoError(t, err)
}

// Test_Pods checks if the number of Pods is the same as defined in the NATS CR and that all Pods are ready.
func Test_Pods_health(t *testing.T) {
	t.Parallel()

	// Get the NATS CR. It will tell us how many Pods we should expect.
	ctx := context.TODO()
	nats, err := retryGet(attempts, interval,
		func() (*natsv1alpha1.NATS, error) {
			return getNATS(ctx, eventingNats, e2eNamespace)
		})
	require.NoError(t, err)

	// Get the NATS Pods and test them.
	listOptions := metav1.ListOptions{LabelSelector: natsCLusterLabel}
	err = retry(attempts, interval, func() error {
		var pods *v1.PodList
		// Get the NATS Pods via labels.
		pods, err = clientSet.CoreV1().Pods(e2eNamespace).List(ctx, listOptions)
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
	nats, err := retryGet(attempts, interval, func() (*natsv1alpha1.NATS, error) {
		return getNATS(ctx, eventingNats, e2eNamespace)
	})
	require.NoError(t, err)

	// Get the PersistentVolumeClaims, PVCs, and test them.
	var pvcs *v1.PersistentVolumeClaimList
	listOpt := metav1.ListOptions{LabelSelector: nameNatsLabel}
	err = retry(attempts, interval, func() error {
		// Get PVCs via a label.
		pvcs, err = retryGet(attempts, interval, func() (*v1.PersistentVolumeClaimList, error) {
			return clientSet.CoreV1().PersistentVolumeClaims(e2eNamespace).List(ctx, listOpt)
		})
		if err != nil {
			return err
		}

		// Check if the amount of PVCs is equal to the spec.cluster.size in the NATS CR. We do this in the retry,
		// because it may take some time for all PVCs to be there.
		want, actual := nats.Spec.Cluster.Size, len(pvcs.Items)
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
		require.True(t, size.Equal(nats.Spec.FileStorage.Size))
	}
}

func Test_NATSServer(t *testing.T) {
	t.Parallel()

	// Get the NATS CR.
	ctx := context.TODO()
	_, err := retryGet(attempts, interval,
		func() (*natsv1alpha1.NATS, error) {
			return getNATS(ctx, eventingNats, e2eNamespace)
		})
	require.NoError(t, err)

	pod, err := retryGet(attempts, interval, func() (*v1.Pod, error) {
		listOptions := metav1.ListOptions{LabelSelector: natsCLusterLabel}
		pods, podErr := clientSet.CoreV1().Pods(e2eNamespace).List(ctx, listOptions)
		if podErr != nil {
			return nil, err
		}

		if len(pods.Items) == 0 {
			return nil, fmt.Errorf("could not find pod")
		}

		return &pods.Items[0], nil
	})

	pod.GetName()
	// todo add port forward
	// todo get info from nats server
	// todo close port forward
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

// the following section is all about the port forward. I borrowed it from a much smarter person:
// https://microcumul.us/blog/k8s-port-forwarding/

func portForward(ctx context.Context, pod corev1.Pod, port string) (net.Conn, error) {
	req := clientSet.RESTClient().
		Post().
		Prefix("api/v1").
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("error getting transport/upgrader from restconfig: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
	conn, _, err := dialer.Dial(portforward.PortForwardProtocolV1Name)
	if err != nil {
		return nil, fmt.Errorf("error dialing for conn %w", err)
	}

	headers := http.Header{}
	headers.Set(v1.StreamType, v1.StreamTypeError)
	headers.Set(v1.PortHeader, port)
	headers.Set(v1.PortForwardRequestIDHeader, "1")

	errorStream, err := conn.CreateStream(headers)
	if err != nil {
		return nil, fmt.Errorf("error creating err stream: %w", err)
	}
	// we're not writing to this stream
	errorStream.Close()

	headers.Set(v1.StreamType, v1.StreamTypeData)
	dataStream, err := conn.CreateStream(headers)
	if err != nil {
		return nil, fmt.Errorf("error creating data stream: %w", err)
	}

	fc := &fakeConn{
		parent: conn,
		port:   port,
		err:    errorStream,
		errch:  make(chan error),
		data:   dataStream,
		pod:    pod,
	}
	go fc.watchErr(ctx)

	return fc, nil
}

// This is a FakeAddr type used just in case anything asks for the net.Addr on
// either side of this "network connection." It's there for debug and helps to
// show that the source is memory and the destination is a k8s pod in a specific
// namespace. `Network` returns "memory" because it's in-memory rather than tcp/udp.
type fakeAddr string

func (f fakeAddr) Network() string {
	return "memory"
}
func (f fakeAddr) String() string {
	return string(f)
}

// FakeConn is the guts of our connection. Most of this code is for handling
// channels and the fact that two things may error, resulting in a problem for
// our callers.
type fakeConn struct {
	parent    httpstream.Connection
	data, err httpstream.Stream
	errch     chan error
	port      string
	pod       v1.Pod
}

func (f *fakeConn) watchErr(ctx context.Context) {
	// This should only return if an err comes back.
	bs, err := io.ReadAll(f.err)
	if err != nil {
		select {
		case <-ctx.Done():
		case f.errch <- fmt.Errorf("error during read: %w", err):
		}
	}
	if len(bs) > 0 {
		select {
		case <-ctx.Done():
		case f.errch <- fmt.Errorf("error during read: %s", string(bs)):
		}
	}
}

func (f *fakeConn) Read(b []byte) (n int, err error) {
	select {
	case err := <-f.errch:
		return 0, err
	default:
	}
	return f.data.Read(b)
}

func (f *fakeConn) Write(b []byte) (n int, err error) {
	select {
	case err := <-f.errch:
		return 0, err
	default:
	}
	return f.data.Write(b)
}

func (f *fakeConn) Close() error {
	var errs []error
	select {
	case err := <-f.errch:
		if err != nil {
			errs = append(errs, err)
		}
	default:
	}
	err := f.data.Close()
	if err != nil {
		errs = append(errs, err)
	}
	f.parent.RemoveStreams(f.data, f.err)
	err = f.parent.Close()
	if err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (f *fakeConn) LocalAddr() net.Addr {
	return fakeAddr("memory:" + f.port)
}

func (f *fakeConn) RemoteAddr() net.Addr {
	return fakeAddr(fmt.Sprintf("k8s/%s/%s:%s", f.pod.Namespace, f.pod.Name, f.port))
}

func (f *fakeConn) SetDeadline(t time.Time) error {
	f.parent.SetIdleTimeout(time.Until(t))
	return nil
}

func (f *fakeConn) SetReadDeadline(t time.Time) error {
	f.parent.SetIdleTimeout(time.Until(t))
	return nil
}

func (f *fakeConn) SetWriteDeadline(t time.Time) error {
	f.parent.SetIdleTimeout(time.Until(t))
	return nil
}
