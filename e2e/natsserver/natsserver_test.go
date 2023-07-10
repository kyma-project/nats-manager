//go:build e2e
// +build e2e

package natsserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/nats-manager/api/v1alpha1"
	. "github.com/kyma-project/nats-manager/e2e/common"
	. "github.com/kyma-project/nats-manager/e2e/common/fixtures"
)

const (
	interval = 2 * time.Second
	attempts = 60
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
		logger.Error(err.Error())
		panic(err)
	}

	clientSet, k8sClient, err = GetK8sClients()
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// Update the NATS CR, so we can test some testcases.
	ctx := context.TODO()
	err = Retry(attempts, interval, logger, func() error {
		cr := &v1alpha1.NATS{}
		key := client.ObjectKey{
			Namespace: NATSCR().GetNamespace(),
			Name:      NATSCR().GetName(),
		}
		natsErr := k8sClient.Get(ctx, key, cr)
		if natsErr != nil {
			return natsErr
		}

		// This was disabled for earlier test. Now we activate it, so we can test against the varz-endpoint and against
		// the configmap.
		cr.Spec.JetStream.MemStorage.Enabled = true

		natsErr = k8sClient.Update(ctx, cr)
		return natsErr
	})
	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

// Test_ConfigMap tests the ConfigMap that the NATS-Manger creates when we define a CR.
func Test_ConfigMap(t *testing.T) {
	ctx := context.TODO()

	err := Retry(attempts, interval, logger, func() error {
		cm, cmErr := clientSet.CoreV1().ConfigMaps(NamespaceName).Get(ctx, CMName, metav1.GetOptions{})
		if cmErr != nil {
			return cmErr
		}

		cmm := cmToMap(cm.Data["nats.conf"])

		if err := checkValueInCMMap(cmm, "max_file", FileStorageSize); err != nil {
			return err
		}

		if err := checkValueInCMMap(cmm, "max_mem", MemStorageSize); err != nil {
			return err
		}

		if err := checkValueInCMMap(cmm, "debug", True); err != nil {
			return err
		}

		if err := checkValueInCMMap(cmm, "trace", True); err != nil {
			return err
		}

		return nil
	})

	require.NoError(t, err)
}

func Test_NATSHealth(t *testing.T) {
	ports := [3]int{8222, 8223, 8224}
	err := Retry(attempts, interval, logger, func() error {
		// For all Pods, let's get the status from the `/healthz` endpoint and check
		// if the response is `{"status":"ok"}`.
		for _, port := range ports {
			actual, checkErr := getHealthz(port)
			if checkErr != nil {
				logger.Warn("error while requesting healthz; is port-forwarding operational?")
				return checkErr
			}
			if want := "ok"; actual != want {
				return fmt.Errorf("health `status` schould be `%s`, but is `%s`", want, actual)
			}
			return nil
		}
		return nil
	})
	require.NoError(t, err)
}

func Test_Varz(t *testing.T) {
	// Let's get the config of NATS from the `/varz` endpoint.
	err := Retry(attempts, interval, logger, func() error {
		varz, varzErr := getVarz(8222)
		if varzErr != nil {
			logger.Warn("error while requesting varz; is port-forwarding operational?")
			return varzErr
		}

		wantMaxMemory := MemStorageSize
		actualMaxMemory := humanize.IBytes(uint64(varz.JetStream.Config.MaxMemory))
		if wantMaxMemory != actualMaxMemory {
			return fmt.Errorf("wanted MaxMemory to be '%s' but was '%s'", wantMaxMemory, actualMaxMemory)
		}

		wantMaxStore := FileStorageSize
		actualMaxStore := humanize.IBytes(uint64(varz.JetStream.Config.MaxStore))
		if wantMaxStore != actualMaxStore {
			return fmt.Errorf("wanted MaxStory to be '%s' but was '%s'", wantMaxMemory, actualMaxMemory)
		}

		return nil
	})
	require.NoError(t, err)
}

func getHealthz(port int) (string, error) {
	url := fmt.Sprintf("http://localhost:%v/healthz", port)
	resp, err := http.Get(url)
	logger.Debug(fmt.Sprintf("trying to connect to %s", url))
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// The `/healthz` endpoint will return a very simple json that looks like `{"status":"ok"}`, so we will only pass
	// the value.
	var actual map[string]string
	jsonErr := json.Unmarshal(body, &actual)
	if jsonErr != nil {
		return "", err
	}
	return actual["status"], nil
}

func getVarz(port int) (*server.Varz, error) {
	url := fmt.Sprintf("http://localhost:%v/varz", port)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var varz server.Varz
	jsonErr := json.Unmarshal(body, &varz)
	if jsonErr != nil {
		return nil, err
	}
	return &varz, nil
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

func cmToMap(cm string) map[string]string {
	lines := strings.Split(cm, "\n")

	cmMap := make(map[string]string)
	for _, line := range lines {
		if strings.Contains(line, ": ") {
			l := strings.Split(line, ": ")
			if len(l) < 2 {
				continue
			}
			key := strings.TrimSpace(l[0])
			val := strings.TrimSpace(l[1])
			cmMap[key] = val
		}
	}

	return cmMap
}
