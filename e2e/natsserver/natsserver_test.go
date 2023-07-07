package natsserver_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/kyma-project/nats-manager/e2e/common"
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

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

func Test_NATSHealth(t *testing.T) {
	ports := [3]int{8222, 8223, 8224}
	err := Retry(attempts, interval, logger, func() error {
		// For all Pods, lets get the status from the `/healthz` endpoint end check if it is `{"status":"ok"}`.
		for _, port := range ports {
			actual, checkErr := getHealthz(port)
			if checkErr != nil {
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

func Test_MemSize(t *testing.T) {
	varz, err := getVarz(8222)
	require.NoError(t, err)

	logger.Debug(fmt.Sprintf("pure %v", varz.JetStream.Config.MaxMemory))
	logger.Debug(fmt.Sprintf("Humanize + uint64 %v", humanize.IBytes(uint64(varz.JetStream.Config.MaxMemory))))
	t.Fail()
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
