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
	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const kubeVersion = "1.24.1"

var (
	testEnv *envtest.Environment

	// Log is the test logger used by the testintegration tests and server.
	Log = zap.New(zap.UseFlagOptions(&zap.Options{
		Development: true,
		TimeEncoder: zapcore.ISO8601TimeEncoder,
	}))

	// Client is the kubernetes client.
	Client client.Client
)

// TestContext returns a background context that includes appropriate logging configuration.
func TestContext() context.Context {
	return logr.NewContext(context.Background(), Log)
}

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
func EnvTestSetup() (func(), error) {
	var err error

	ctrl.SetLogger(Log)

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
	ctx, cancel := context.WithCancel(logr.NewContext(context.TODO(), Log))

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false,
		BinaryAssetsDirectory: kubebuilderAssets,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "config", "webhook")},
		},
	}

	teardownFunc := func() {
		cancel()
		err := testEnv.Stop()
		if err != nil {
			Log.Error(err, "unable to stop envtest environment %v")
		}
	}

	cfg, err := testEnv.Start()
	if err != nil {
		return nil, fmt.Errorf("unable to start kuberenetes envtest %v", err)
	}

	s := scheme.Scheme
	controller.InitScheme(s)

	Client, err = client.New(cfg, client.Options{Scheme: s})
	if err != nil {
		return teardownFunc, fmt.Errorf("unable to start kuberenetes envtest %v", err)
	}
	if Client == nil {
		return teardownFunc, fmt.Errorf("unable to start kuberenetes envtest %v", err)
	}

	// start webhook server using Manager
	o := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             s,
		Host:               o.LocalServingHost,
		Port:               o.LocalServingPort,
		CertDir:            o.LocalServingCertDir,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	if err != nil {
		return teardownFunc, fmt.Errorf("unable to start kuberenetes envtest %v", err)
	}

	err = controller.SetupManagers(mgr)
	if err != nil {
		return teardownFunc, fmt.Errorf("unable to start kuberenetes envtest %v", err)
	}

	go func() {
		Log.Info("Starting controller manager.")
		err = mgr.Start(ctx)
		if err != nil {
			Log.Info("Starting manager failed.")
			return
		}
		Log.Info("Started controller exited normally.")
	}()

	// wait for the controller manager webhook server to get ready
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", o.LocalServingHost, o.LocalServingPort)

	err = testhelpers.RetryUntilSuccess(10, time.Second, func() error {

		// whyNoLint:Ignore InsecureSkipVerify warning, this is only for local testing.
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true}) //nolint:gosec
		if err != nil {
			return err
		}

		conn.Close()

		return nil
	})

	Log.Info("Setup complete. Webhook server started.")

	return teardownFunc, nil
}
