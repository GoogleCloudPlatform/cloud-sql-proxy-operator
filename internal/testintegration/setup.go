// Copyright 2022 Google LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package testintegration test setup for running testintegration tests using the envtest
// kubebuilder package.
package testintegration

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/controller"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testhelpers"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workload"
	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const kubeVersion = "1.24.1"

var (
	// Log is the test logger used by the testintegration tests and server.
	Log = zap.New(zap.UseFlagOptions(&zap.Options{
		Development: true,
		TimeEncoder: zapcore.ISO8601TimeEncoder,
	}))
)

// TestContext returns a background context that includes appropriate logging configuration.
func TestContext() context.Context {
	return logr.NewContext(context.Background(), Log)
}

// runSetupEnvtest runs the setup-envtest tool to download the latest k8s
// binaries.
func runSetupEnvtest() (string, error) {
	cmd := exec.Command("../../bin/setup-envtest", "use", kubeVersion, "-p", "path")
	path, err := cmd.Output()

	if err != nil {
		out, outputErr := cmd.CombinedOutput()
		if outputErr != nil {
			Log.Error(err, "Unable to run setup-envtest or get output log", "output error", outputErr)
			return "", err
		}
		Log.Error(err, "Unable to run setup-envtest", "output", string(out), "err")
		return "", err
	}

	return string(path), nil
}

// EnvTestSetup sets up the envtest environment for a testing package.
// This is intended to be called from `func TestMain(m *testing.M)` so
// that the environment is configured before
func EnvTestSetup() (*EnvTestHarness, error) {
	var err error

	ctrl.SetLogger(Log)

	// Initialize the test environment

	// if the KUBEBUILDER_ASSETS env var is not set, then run setup-envtest
	// and set it according.
	kubebuilderAssets, exists := os.LookupEnv("KUBEBUILDER_ASSETS")
	if !exists {
		kubebuilderAssets, err = runSetupEnvtest()
		if err != nil {
			return nil, fmt.Errorf("unable to run setup-envtest %v", err)
		}
	}

	Log.Info("Starting up kubebuilder EnvTest")
	ctx, cancel := context.WithCancel(logr.NewContext(context.Background(), Log))

	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false,
		BinaryAssetsDirectory: kubebuilderAssets,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "config", "webhook")},
		},
	}

	// Start the testenv
	cfg, err := testEnv.Start()
	mh := &EnvTestHarness{
		testEnvCtx:    ctx,
		testEnv:       testEnv,
		testEnvCancel: cancel,
		cfg:           cfg,
	}
	if err != nil {
		return mh, fmt.Errorf("unable to start kuberenetes envtest %v", err)
	}

	// Initialize rest client configuration
	mh.s = scheme.Scheme
	controller.InitScheme(mh.s)
	cl, err := client.New(cfg, client.Options{Scheme: mh.s})
	if err != nil {
		return mh, fmt.Errorf("unable to to create client %v", err)
	}
	if cl == nil {
		return mh, fmt.Errorf("client not created")
	}
	mh.Client = cl

	// Start the controller-runtime manager
	err = mh.StartMgr(workload.DefaultProxyImage)
	if err != nil {
		return mh, fmt.Errorf("unable to start kuberenetes envtest %v", err)
	}

	return mh, nil
}

// EnvTestHarness enables integration tests to control the lifecycle of the
// operator's controller-manager.
type EnvTestHarness struct {

	// Mgr is the manager
	Mgr ctrl.Manager

	// Client is the kubernetes client.
	Client client.Client

	// The actual EnvTest environment
	testEnv *envtest.Environment

	// testEnvCancel is the context cancel function for the testEnv
	testEnvCancel context.CancelFunc

	// cancel is the context cancel function for the manager
	cancel context.CancelFunc

	// stopped channel is closed when the manager actually stops
	stopped chan int

	// ctx is the manager's context
	ctx context.Context

	// cfg is the client configuration from envtest
	cfg *rest.Config

	// s is the client scheme
	s          *runtime.Scheme
	testEnvCtx context.Context
}

// Teardown closes the TestEnv environment at the end of the testcase.
func (h *EnvTestHarness) Teardown() {
	if h.testEnvCancel != nil {
		h.testEnvCancel()
	}
	if h.testEnv != nil {
		err := h.testEnv.Stop()
		if err != nil {
			Log.Error(err, "unable to stop envtest environment %v")
		}
	}
}

// StopMgr stops the controller manager and waits for it to exit, returning an
// error if the controller manager does not stop within 1 minute.
func (h *EnvTestHarness) StopMgr() error {
	if h.cancel != nil {
		h.cancel()
	}
	h.cancel = nil
	select {
	case <-h.stopped:
		return nil
	case <-time.After(1 * time.Minute):
		return fmt.Errorf("manager did not stop after 1 minute")
	}
}

// StartMgr starts up the manager, configuring it with the proxyImage.
func (h *EnvTestHarness) StartMgr(proxyImage string) error {
	h.ctx, h.cancel = context.WithCancel(h.testEnvCtx)

	// start webhook server using Manager
	o := &h.testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(h.cfg, ctrl.Options{
		Scheme:             h.s,
		Host:               o.LocalServingHost,
		Port:               o.LocalServingPort,
		CertDir:            o.LocalServingCertDir,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	if err != nil {
		return fmt.Errorf("unable to start manager %v", err)
	}
	h.Mgr = mgr

	err = controller.SetupManagers(mgr, "cloud-sql-proxy-operator/dev", proxyImage)
	if err != nil {

		// Run the
		return fmt.Errorf("unable to start kuberenetes envtest %v", err)
	}

	// Run the manager in a goroutine, close the channel when the manager exits.
	h.stopped = make(chan int)
	go func() {
		defer close(h.stopped)
		Log.Info("Starting controller manager.")
		err = mgr.Start(h.ctx)
		if err != nil {
			Log.Info("Starting manager failed.")
			return
		}
		Log.Info("Manager exited normally.")
	}()

	// Wait for the controller manager webhook server to get ready.
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", o.LocalServingHost, o.LocalServingPort)
	err = testhelpers.RetryUntilSuccess(10, time.Second, func() error {

		// whyNoLint:Ignore InsecureSkipVerify warning, this is only for local testing.
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort,
			&tls.Config{InsecureSkipVerify: true}) //nolint:gosec
		if err != nil {
			return err
		}

		conn.Close()

		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to connect to manager %v", err)
	}

	Log.Info("Setup complete. Manager started.")
	return nil
}
