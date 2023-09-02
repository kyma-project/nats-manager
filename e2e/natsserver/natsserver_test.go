//go:build e2e
// +build e2e

// Package natsserver_test is part of the end-to-end-tests. This package contains tests that check the
// internal of the nats servers.
// To run the tests a Kubernetes cluster and a nats-cr need to be available and configured. Further, the ports of the
// NATS-server Pods need to be forwarded. For this reason, the tests are seperated via the `e2e` buildtags. For more
// information please consult the `readme.md`.
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
	"k8s.io/apimachinery/pkg/api/resource"

	. "github.com/kyma-project/nats-manager/e2e/common"
	. "github.com/kyma-project/nats-manager/e2e/common/fixtures"
)

const (
	interval = 2 * time.Second
	attempts = 60
)

var logger *zap.Logger

// TestMain runs before all the other test functions. It sets up all the resources that are shared between the different
// test functions. It will then run the tests and finally shuts everything down.
func TestMain(m *testing.M) {
	var err error
	logger, err = SetupLogger()
	if err != nil {
		logger.Fatal(err.Error())
	}

	// Run the tests and exit.
	code := m.Run()
	os.Exit(code)
}

func Test_NATSHealth(t *testing.T) {
	wantStatus := "ok"

	ports := [3]int{8222, 8223, 8224}
	err := Retry(attempts, interval, func() error {
		// For all Pods, let's get the status from the `/healthz` endpoint and check
		// if the response is `{"status":"ok"}`.
		for _, port := range ports {
			actualStatus, checkErr := getHealthz(port)
			if checkErr != nil {
				logger.Warn("error while requesting healthz; is port-forwarding operational?")
				return checkErr
			}
			if actualStatus != wantStatus {
				return fmt.Errorf("health `status` schould be `%s`, but is `%s`", wantStatus, actualStatus)
			}
			return nil
		}
		return nil
	})
	require.NoError(t, err)
}

func Test_Varz(t *testing.T) {
	// To make the wanted MemStorageSize and FileStorageSize comparable to what we will find on the NATS-Server, we need
	// to transform it to the same unit; bytes.
	wm := resource.MustParse(MemStorageSize)
	wantMem := wm.Value()
	wf := resource.MustParse(FileStorageSize)
	wantStore := wf.Value()

	// Let's get the config of NATS from the `/varz` endpoint.
	err := Retry(attempts, interval, func() error {
		varz, varzErr := getVarz(8222)
		if varzErr != nil {
			logger.Warn("error while requesting varz; is port-forwarding operational?")
			return varzErr
		}

		actualStore := varz.JetStream.Config.MaxStore
		if wantStore != actualStore {
			return fmt.Errorf(
				"wanted 'MaxStore' to be '%s' but was '%s'",
				humanize.IBytes(uint64(wantStore)),
				humanize.IBytes(uint64(actualStore)),
			)
		}

		actualMem := varz.JetStream.Config.MaxMemory
		if wantMem != actualMem {
			return fmt.Errorf(
				"wanted 'MaxMemory' to be '%s' but was '%s'",
				humanize.IBytes(uint64(wantMem)),
				humanize.IBytes(uint64(actualMem)),
			)
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
