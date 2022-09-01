// Copyright 2022 Google LLC
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

package integration

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	cloudsqlv1alpha1 "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/controllers"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/test/helpers"
	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var testEnv *envtest.Environment
var serverContext context.Context
var cancel context.CancelFunc
var log logr.Logger
var k8sClient client.Client

func runSetupEnvtest() (string, error) {
	// bin/setup-envtest use 1.24.1 -p path
	cmd := exec.Command("../../bin/setup-envtest", "use", "1.24.1", "-p", "path")
	path, err := cmd.Output()

	if err != nil {
		out, _ := cmd.CombinedOutput()
		log.Error(err, "Unable to run setup-envtest", "output", string(out))
		return "", err
	}

	if err != nil {
		return "", err
	}
	return string(path), nil
}

func EnvTestSetup(m *testing.M) (func(), error) {
	var err error

	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.ISO8601TimeEncoder,
	}
	log = zap.New(zap.UseFlagOptions(&opts))

	ctrl.SetLogger(log)

	// if the KUBEBUILDER_ASSETS env var is not set, then run setup-envtest
	// and set it according.
	kubebuilderAssets, exists := os.LookupEnv("KUBEBUILDER_ASSETS")
	if !exists {
		kubebuilderAssets, err = runSetupEnvtest()
		if err != nil {
			return nil, fmt.Errorf("Unable to run setup-envtest %v", err)
		}
	}

	log.Info("Starting up kubebuilder EnvTest")
	serverContext, cancel = context.WithCancel(logr.NewContext(context.TODO(), log))

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false,
		BinaryAssetsDirectory: kubebuilderAssets,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "config", "webhook")},
		},
	}

	teardownFunc := func() {
		err := testEnv.Stop()
		if err != nil {
			log.Error(err, "Unable to stop envtest environment %v")
		}
	}

	cfg, err := testEnv.Start()
	if err != nil {
		return nil, fmt.Errorf("enable to start kuberenetes envtest %v", err)
	}

	scheme := scheme.Scheme
	controllers.InitScheme(scheme)

	err = cloudsqlv1alpha1.AddToScheme(scheme)
	if err != nil {
		return teardownFunc, fmt.Errorf("unable to start kuberenetes envtest %v", err)
	}

	err = admissionv1beta1.AddToScheme(scheme)
	if err != nil {
		return teardownFunc, fmt.Errorf("unable to start kuberenetes envtest %v", err)
	}

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return teardownFunc, fmt.Errorf("unable to start kuberenetes envtest %v", err)
	}
	if k8sClient == nil {
		return teardownFunc, fmt.Errorf("unable to start kuberenetes envtest %v", err)
	}

	// start webhook server using Manager
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme,
		Host:               webhookInstallOptions.LocalServingHost,
		Port:               webhookInstallOptions.LocalServingPort,
		CertDir:            webhookInstallOptions.LocalServingCertDir,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	if err != nil {
		return teardownFunc, fmt.Errorf("unable to start kuberenetes envtest %v", err)
	}

	err = controllers.SetupManagers(mgr)
	if err != nil {
		return teardownFunc, fmt.Errorf("unable to start kuberenetes envtest %v", err)
	}

	go func() {
		log.Info("Starting controller manager.")
		err = mgr.Start(serverContext)
		if err != nil {
			log.Info("Starting manager failed.")
		} else {
			log.Info("Started controller exited normally.")
		}
	}()

	// wait for the controller manager webhook server to get ready
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)

	err = helpers.RetryUntilSuccess(&helpers.TestSetupLogger{Logger: log}, 5, 2*time.Second, func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	})

	log.Info("Setup complete. Webhook server started.")

	return teardownFunc, nil
}
