//go:build e2e
// +build e2e

package e2e_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
	"github.com/kyma-project/nats-manager/e2e/fixtures"
)

// Const for retries; the retry and the retryGet functions.
const (
	interval = 5 * time.Second
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
	setupLogging()

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

	// Create the Namespace used for testing.
	ctx := context.TODO()
	err = retry(attempts, interval, func() error {
		_, nsErr := clientSet.CoreV1().Namespaces().Create(ctx, fixtures.Namespace(), metav1.CreateOptions{})
		if nsErr == nil || k8serrors.IsAlreadyExists(nsErr) {
			return nil
		}
		return nsErr
	})
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// Create the NATS CR used for testing.
	err = retry(attempts, interval, func() error {
		return k8sClient.Create(ctx, fixtures.NATSCR())
	})
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

// Test_Pods checks if the number of Pods is the same as defined in the NATS CR and that all Pods have the resources,
// that .
func Test_PodResources(t *testing.T) {
	t.Parallel()

	// Get the NATS CR. It will tell us how many Pods we should expect and what the resources should be configured to.
	ctx := context.TODO()
	nats, err := retryGet(attempts, interval, func() (*natsv1alpha1.NATS, error) {
		return getNATSCR(ctx, fixtures.CRName, fixtures.NamespaceName)
	})
	require.NoError(t, err)

	// Get the NATS Pods and test them.
	listOptions := metav1.ListOptions{LabelSelector: fixtures.PodLabel}
	err = retry(attempts, interval, func() error {
		// Get the NATS Pods via labels.
		var pods *v1.PodList
		pods, err = clientSet.CoreV1().Pods(fixtures.NamespaceName).List(ctx, listOptions)
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
				if !(container.Name == fixtures.ContainerName) {
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
	natsCR, err := retryGet(attempts, interval,
		func() (*natsv1alpha1.NATS, error) {
			return getNATSCR(ctx, fixtures.CRName, fixtures.NamespaceName)
		})
	require.NoError(t, err)

	// Get the NATS Pods and test them.
	listOptions := metav1.ListOptions{LabelSelector: fixtures.PodLabel}
	err = retry(attempts, interval, func() error {
		var pods *v1.PodList
		// Get the NATS Pods via labels.
		pods, err = clientSet.CoreV1().Pods(fixtures.NamespaceName).List(ctx, listOptions)
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
	natsCR, err := retryGet(attempts, interval, func() (*natsv1alpha1.NATS, error) {
		return getNATSCR(ctx, fixtures.CRName, fixtures.NamespaceName)
	})
	require.NoError(t, err)

	// Get the PersistentVolumeClaims, PVCs, and test them.
	var pvcs *v1.PersistentVolumeClaimList
	listOpt := metav1.ListOptions{LabelSelector: fixtures.PVCLabel}
	err = retry(attempts, interval, func() error {
		// Get PVCs via a label.
		pvcs, err = retryGet(attempts, interval, func() (*v1.PersistentVolumeClaimList, error) {
			return clientSet.CoreV1().PersistentVolumeClaims(fixtures.NamespaceName).List(ctx, listOpt)
		})
		if err != nil {
			return err
		}

		// Check if the amount of PVCs is equal to the spec.cluster.size in the NATS CR. We do this in the retry,
		// because it may take some time for all PVCs to be there.
		want, actual := natsCR.Spec.Cluster.Size, len(pvcs.Items)
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
		require.True(t, size.Equal(natsCR.Spec.FileStorage.Size))
	}
}

func Test_NATSServer(t *testing.T) {
	t.Parallel()

	// We need a context that can be canceled, if we work with port-forwarding
	ctx, cancel := context.WithCancel(context.TODO())

	// Get the NATS CR.
	_, err := retryGet(attempts, interval,
		func() (*natsv1alpha1.NATS, error) {
			return getNATSCR(ctx, fixtures.CRName, fixtures.NamespaceName)
		})
	require.NoError(t, err)

	// Get one of the Pods.
	var pod *v1.Pod
	pod, err = retryGet(attempts, interval, func() (*v1.Pod, error) {
		listOpts := metav1.ListOptions{LabelSelector: fixtures.PodLabel}
		pods, podErr := clientSet.CoreV1().Pods(fixtures.NamespaceName).List(ctx, listOpts)
		if podErr != nil {
			return nil, err
		}

		if len(pods.Items) == 0 {
			return nil, fmt.Errorf("could not find pod")
		}

		return &pods.Items[0], nil
	})

	// Forwarding the port is so easy.
	_, err = portForward(ctx, *pod, "4222")
	require.NoError(t, err)

	// Close the port-forward.
	cancel()
}

func setupLogging() {
	logLevel := os.Getenv("E2E_LOG_LEVEL")

	var level zapcore.Level
	switch logLevel {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	default:
		// level = zap.ErrorLevel
		// todo
		level = zap.DebugLevel
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	var err error
	logger, err = config.Build()
	if err != nil {
		panic(err)
	}
}

func retry(attempts int, interval time.Duration, fn func() error) error {
	ticker := time.NewTicker(interval)
	var err error
	for {
		select {
		case <-ticker.C:
			attempts -= 1
			err = fn()
			if err != nil {
				logger.Warn(fmt.Sprintf("error while retrying: %s", err.Error()))
			}
			if err == nil || attempts == 0 {
				return err
			}
			logger.Warn(fmt.Sprintf("retrying with %v attempts left", attempts))
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
			if err != nil {
				logger.Warn(fmt.Sprintf("error while retrying: %s", err.Error()))
			}
			if err == nil || attempts == 0 {
				return obj, err
			}
			logger.Warn(fmt.Sprintf("retrying with %v attempts left", attempts))
		}
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

func connectToNATSServer() (*nats.Conn, error) {
	nc, err := nats.Connect("nats://nats:4222")
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return nc, nil
}

func getNATSServerInfo(c *nats.Conn) error {
	nc, err := connectToNATSServer()
	if err != nil {
		return err
	}

	id := "" // todo

	subj := fmt.Sprintf("$SYS.REQ.SERVER.%s.VARZ", id)
	body := []byte("{}")

	if len(id) != 56 || strings.ToUpper(id) != id {
		subj = "$SYS.REQ.SERVER.PING.VARZ"
		opts := server.VarzEventOptions{EventFilterOptions: server.EventFilterOptions{Name: id}}
		body, err = json.Marshal(opts)
		if err != nil {
			return err
		}
	}

	resp, err := nc.Request(subj, body, interval)
	if err != nil {
		return fmt.Errorf(
			"no results received, ensure the account used has system privileges and appropriate permissions",
		)
	}

	reqresp := map[string]json.RawMessage{}
	err = json.Unmarshal(resp.Data, &reqresp)
	if err != nil {
		return err
	}

	data, ok := reqresp["data"]
	if !ok {
		return fmt.Errorf("no data received in response: %#v", reqresp)
	}

	varz := &server.Varz{}
	err = json.Unmarshal(data, varz)
	if err != nil {
		return err
	}

	// Todo

	return nil
}

// The following section is all about the port forward. It was borrowed from a much smarter person:
// https://microcumul.us/blog/k8s-port-forwarding/

// portForward allows to forward a port. Pass context that can be canceled, so the port-forwarding can be closed,
// once it is no longer needed.
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
