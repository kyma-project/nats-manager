//go:build e2e
// +build e2e

package natsserver_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

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

func Test_NATSHealz(t *testing.T) {
	ports := [3]int{8222, 8223, 8224}
	err := Retry(attempts, interval, logger, func() error {
		for _, port := range ports {
			checkErr := checkPodHealth(port)
			if checkErr != nil {
				return checkErr
			}
		}
		return nil
	})
	require.NoError(t, err)
}

func checkPodHealth(port int) error {
	url := fmt.Sprintf("http://localhost:%v", port)
	resp, err := http.Get(url)
	logger.Debug(fmt.Sprintf("trying to connect to %s", url))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result map[string]string
	jsonErr := json.Unmarshal(body, &result)
	if jsonErr != nil {
		return err
	}

	want := `{"status":"ok"}`
	if strings.Contains(string(body), want) {
		return nil
	}
	return fmt.Errorf("body did not contain %s, but %s", want, string(body))
}
