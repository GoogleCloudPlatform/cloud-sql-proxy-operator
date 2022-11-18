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

// Package tests holds testcases for the end-to-end tests using Google Cloud
// Platform GKE and Cloud SQL.
package tests

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/controller"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/testhelpers"
	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	// logger is the test Logger used by the testintegration tests and server.
	logger = zap.New(zap.UseFlagOptions(&zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}))

	// These vars hold state initialized by SetupTests.
	c             client.Client
	infra         testInfra
	proxyImageURL string
	operatorURL   string
)

func params(t *testing.T, ns string) *testhelpers.TestCaseParams {
	return &testhelpers.TestCaseParams{
		T:                t,
		Client:           c,
		Namespace:        testhelpers.NewNamespaceName(ns),
		ConnectionString: infra.InstanceConnectionString,
		ProxyImageURL:    proxyImageURL,
		Ctx:              testContext(),
	}
}

var k8sClientSet *kubernetes.Clientset

// testContext returns a background context that includes appropriate Logging configuration.
func testContext() context.Context {
	return logr.NewContext(context.Background(), logger)
}

func loadValue(envVar, fromFile, defaultValue string) string {
	envValue, isset := os.LookupEnv(envVar)
	if isset && envValue != "" {
		return strings.Trim(envValue, "\r\n \t")
	}

	if fromFile == "" {
		return defaultValue
	}

	bytes, err := os.ReadFile(fromFile)
	if err != nil {
		return defaultValue
	}

	return strings.Trim(string(bytes), "\r\n \t")
}

func setupTests() (func(), error) {
	ctx, cancelFunc := context.WithCancel(testContext())

	// Cancel the context when teardown is called.
	teardownFunc := func() {
		logger.Info("Shuting down...")
		cancelFunc()
	}

	// Read e2e test configuration
	proxyImageURL = loadValue("PROXY_IMAGE_URL", "../bin/last-proxy-image-url.txt", "gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.0.0-preview.2")
	operatorURL = loadValue("OPERATOR_IMAGE_URL", "../bin/last-gcloud-operator-url.txt", "operator:latest")
	testInfraPath := loadValue("TEST_INFRA_JSON", "", "../bin/testinfra.json")
	ti, err := loadTestInfra(testInfraPath)
	if err != nil {
		kubeconfig := "../../bin/e2e-kubeconfig.yaml"
		if envKubeConfig, isset := os.LookupEnv("KUBECONFIG"); isset {
			kubeconfig = envKubeConfig
		}
		ti.Kubeconfig = kubeconfig
		ti.DB = "db"
		ti.InstanceConnectionString = "proj:region:inst"
		logger.Info("Test infrastructure not set. Using defaults",
			"instance", ti.InstanceConnectionString,
			"db", ti.DB,
			"kubeconfig", ti.Kubeconfig)
	}
	infra = ti

	// Build the kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", ti.Kubeconfig)
	if err != nil {
		return teardownFunc, fmt.Errorf("unable to build kubernetes client for config %s, %v", ti.Kubeconfig, err)
	}
	config.RateLimiter = nil
	k8sClientSet, err = kubernetes.NewForConfig(config)
	if err != nil {
		return teardownFunc, fmt.Errorf("unable to setup e2e kubernetes client  %v", err)
	}
	s := scheme.Scheme
	controller.InitScheme(s)
	c, err = client.New(config, client.Options{Scheme: s})
	if err != nil {
		return teardownFunc, fmt.Errorf("Unable to initialize kubernetes client %{v}", err)
	}
	if c == nil {
		return teardownFunc, fmt.Errorf("Kubernetes client was empty after initialization %v", err)
	}

	// Check that the e2e k8s cluster is the operator that was last built from
	// this working directory.
	d, err := waitForCorrectOperatorPods(ctx, err)

	if err != nil {
		return teardownFunc, fmt.Errorf("unable to find manager deployment %v", err)
	}

	// Start the goroutines to tail the logs from the operator deployment. This
	// prints the operator output in line with the test output so it's easier
	// for the developer to follow.
	podList, err := testhelpers.ListPods(ctx, c, d.GetNamespace(), d.Spec.Selector)
	if err != nil {
		return teardownFunc, fmt.Errorf("unable to find manager deployment %v", err)
	}
	tailPods(ctx, podList)

	logger.Info("Setup complete. K8s cluster is running.")

	return teardownFunc, nil
}

func waitForCorrectOperatorPods(ctx context.Context, err error) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	managerDeploymentKey := client.ObjectKey{Namespace: "cloud-sql-proxy-operator-system", Name: "cloud-sql-proxy-operator-controller-manager"}
	err = testhelpers.RetryUntilSuccess(&testSetupLogger{Logger: logger}, 5, 5*time.Second, func() error {
		// Fetch the deployment
		err := c.Get(ctx, managerDeploymentKey, deployment)
		if err != nil {
			return err
		}

		// If it isn't ready, return error and retry
		if deployment.Status.ReadyReplicas == 0 {
			return errors.New("deployment exists but 0 replicas are ready")
		}

		// Check that pods are running the right version
		// of the image. Sometimes deployments can take some time to roll out, and pods
		// will be running a different version.
		pods, err := testhelpers.ListPods(ctx, c, deployment.GetNamespace(), deployment.Spec.Selector)
		if err != nil {
			return fmt.Errorf("can't list manager deployment pods, %v", err)
		}
		for i := 0; i < len(pods.Items); i++ {
			var notReady bool
			for j := 0; j < len(pods.Items[i].Status.Conditions); j++ {
				if pods.Items[i].Status.Conditions[j].Type == "Ready" && pods.Items[i].Status.Conditions[j].Status == "False" {
					notReady = true
				}
			}
			if notReady {
				continue
			}
			for j := 0; j < len(pods.Items[i].Spec.Containers); j++ {
				c := pods.Items[i].Spec.Containers[j]
				if c.Name == "manager" {
					if !strings.Contains(c.Image, operatorURL) {
						return fmt.Errorf("Pod image was %s, expected %s", c.Image, operatorURL)
					}
				}
			}
		}

		return nil // OK to continue.
	})
	return deployment, err
}

func tailPods(ctx context.Context, podlist *corev1.PodList) {
	for _, item := range podlist.Items {
		go tailPod(ctx, item)
	}
}

func tailPod(ctx context.Context, pod corev1.Pod) {

	var since int64 = 10
	var rs, err = k8sClientSet.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container:    "manager",
		Follow:       true,
		SinceSeconds: &since,
		Timestamps:   true,
	}).Stream(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer rs.Close()

	sc := bufio.NewScanner(rs)

	for sc.Scan() {
		txt := sc.Text()
		if i := strings.Index(txt, "ERROR"); i > -1 {
			logger.Error(fmt.Errorf("OPER: %s", txt[i+6:]), "", "pod", pod.Name)
		} else if i := strings.Index(txt, "INFO"); i > -1 {
			logger.Info(fmt.Sprintf("OPER: %s", txt[i+4:]), "pod", pod.Name)
		} else if i := strings.Index(txt, "DEBUG"); i > -1 {
			logger.Info(fmt.Sprintf("OPER-dbg: %s", txt[i+5:]), "pod", pod.Name)
		} else {
			logger.Info(fmt.Sprintf("OPER: %s", txt), "pod", pod.Name)
		}
	}

}

// testSetupLoger implements TestLogger
type testSetupLogger struct {
	logr.Logger
}

func (l *testSetupLogger) Logf(format string, args ...interface{}) {
	l.Info(fmt.Sprintf(format, args...))
}
func (l *testSetupLogger) Helper() {}

type testInfra struct {
	InstanceConnectionString string `json:"instance,omitempty"`
	DB                       string `json:"db,omitempty"`
	RootPassword             string `json:"rootPassword,omitempty"`
	Kubeconfig               string `json:"kubeconfig,omitempty"`
}

func loadTestInfra(testInfraJSON string) (testInfra, error) {
	result := testInfra{}
	bytes, err := os.ReadFile(testInfraJSON)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
