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

package e2e

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/controllers"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/test/helpers"
	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (

	// Log is the test Logger used by the integration tests and server.
	Log = zap.New(zap.UseFlagOptions(&zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}))
	Client           client.Client
	Infrastructure   TestInfra
	ProxyImageURL    string
	OperatorImageURL string
)

var k8sClientSet *kubernetes.Clientset

// TestContext returns a background context that includes appropriate Logging configuration.
func TestContext() context.Context {
	return logr.NewContext(context.Background(), Log)
}

func E2eTestSetup() (func(), error) {
	ctx, cancelFunc := context.WithCancel(TestContext())

	teardownFunc := func() {
		Log.Info("Shuting down...")
		cancelFunc()
	}

	scheme := scheme.Scheme
	controllers.InitScheme(scheme)
	var err error

	if proxyImageURL, isset := os.LookupEnv("PROXY_IMAGE_URL"); isset && proxyImageURL != "" {
		Log.Info(fmt.Sprintf("Proxy image from env PROXY_IMAGE_URL: %s", proxyImageURL))
		ProxyImageURL = strings.Trim(proxyImageURL, "\r\n \t")
	} else {
		bytes, err := os.ReadFile("../../bin/last-proxy-image-url.txt")
		if err != nil {
			ProxyImageURL = "cloudsql-proxy:latest" //TODO a valid public image url
			Log.Info(fmt.Sprintf("Proxy image from default: %s", ProxyImageURL))
		} else {
			ProxyImageURL = strings.Trim(string(bytes), "\r\n \t")
			Log.Info(fmt.Sprintf("Proxy image from build file: %s", ProxyImageURL))
		}
	}

	if proxyImageURL, isset := os.LookupEnv("OPERATOR_IMAGE_URL"); isset && proxyImageURL != "" {
		Log.Info(fmt.Sprintf("Operator image from env OPERATOR_IMAGE_URL: %s", proxyImageURL))
		OperatorImageURL = strings.Trim(proxyImageURL, "\r\n \t")
	} else {
		bytes, err := os.ReadFile("../../bin/latest-image.url")
		if err != nil {
			OperatorImageURL = "operator:latest" //TODO a valid public image url
			Log.Info(fmt.Sprintf("Proxy image from default: %s", ProxyImageURL))
		} else {
			OperatorImageURL = strings.Trim(string(bytes), "\r\n \t")
			Log.Info(fmt.Sprintf("Proxy image from build file: %s", ProxyImageURL))
		}
	}

	if testInfraJson, isset := os.LookupEnv("TEST_INFRA_JSON"); isset {
		Infrastructure, err = LoadTestInfra(testInfraJson)
		if err != nil {
			return teardownFunc, fmt.Errorf("unable load infrastructure configuration from %s, %v", testInfraJson, err)
		}
		Log.Info(fmt.Sprintf("Loaded test infrastructure from %s", testInfraJson),
			"instance", Infrastructure.InstanceConnectionString,
			"db", Infrastructure.Db,
			"kubeconfig", Infrastructure.Kubeconfig)
	} else {
		kubeconfig := "../../bin/kind-kubeconfig.yaml"
		if envKubeConfig, isset := os.LookupEnv("KUBECONFIG"); isset {
			kubeconfig = envKubeConfig
		}
		Infrastructure.Kubeconfig = kubeconfig
		Infrastructure.Db = "db"
		Infrastructure.InstanceConnectionString = "proj:region:inst"
		Log.Info("Test infrastructure not set. Using defaults",
			"instance", Infrastructure.InstanceConnectionString,
			"db", Infrastructure.Db,
			"kubeconfig", Infrastructure.Kubeconfig)
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", Infrastructure.Kubeconfig)
	if err != nil {
		return teardownFunc, fmt.Errorf("Unable to build kubernetes client for config %s, %v", Infrastructure.Kubeconfig, err)
	}

	config.RateLimiter = nil
	k8sClientSet, err = kubernetes.NewForConfig(config)
	if err != nil {
		return teardownFunc, fmt.Errorf("unable to setup e2e kubernetes client  %v", err)
	}
	Client, err = client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return teardownFunc, fmt.Errorf("Unable to initialize kubernetes client %{v}", err)
	}
	if Client == nil {
		return teardownFunc, fmt.Errorf("Kubernetes client was empty after initialization %v", err)
	}

	// make it safer to set up the manager with the correct version
	deployment := appsv1.Deployment{}
	managerDeploymentKey := client.ObjectKey{Namespace: "cloud-sql-proxy-operator-system", Name: "cloud-sql-proxy-operator-controller-manager"}

	err = helpers.RetryUntilSuccess(&testSetupLogger{Logger: Log}, 5, 5*time.Second, func() error {
		err := Client.Get(ctx, managerDeploymentKey, &deployment)
		if err != nil {
			return err
		}
		if deployment.Status.ReadyReplicas == 0 {
			return fmt.Errorf("deployment exists but 0 replicas are ready.")
		}
		// ensure cluster is running the most recently built image
		pods, err := ListDeploymentPods(ctx, managerDeploymentKey)
		if err != nil {
			return fmt.Errorf("can't list manager deployment pods, %v", err)
		}
		for i := 0; i < len(pods.Items); i++ {
			for j := 0; j < len(pods.Items[i].Spec.Containers); j++ {
				c := pods.Items[i].Spec.Containers[j]
				if c.Name == "manager" {
					if !strings.Contains(c.Image, OperatorImageURL) {
						return fmt.Errorf("Pod image was %s, expected %s", c.Image, OperatorImageURL)
					}
				}
			}
		}
		return nil
	})

	if Client == nil {
		return teardownFunc, fmt.Errorf("unable to find manager deployment %v", err)
	}

	podList, err := ListDeploymentPods(ctx, managerDeploymentKey)
	if err != nil {
		return teardownFunc, fmt.Errorf("unable to find manager deployment %v", err)
	}
	tailPods(ctx, podList)

	Log.Info("Setup complete. K8s cluster is running.")

	return teardownFunc, nil
}

func ListDeploymentPods(ctx context.Context, deploymentKey client.ObjectKey) (*corev1.PodList, error) {
	dep, err := k8sClientSet.AppsV1().Deployments(deploymentKey.Namespace).Get(ctx, deploymentKey.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to find manager deployment %v", err)
	}
	podList, err := k8sClientSet.CoreV1().Pods(deploymentKey.Namespace).List(ctx, v1.ListOptions{

		LabelSelector: metav1.FormatLabelSelector(dep.Spec.Selector),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to find manager deployment %v", err)
	}

	return podList, nil

}

func tailPods(ctx context.Context, podlist *corev1.PodList) {
	for _, item := range podlist.Items {
		tailPod(ctx, item)
	}
}

func tailPod(ctx context.Context, pod corev1.Pod) {

	go func() {
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
				Log.Error(fmt.Errorf("OPER: %s", txt[i+6:]), "", "pod", pod.Name)
			} else if i := strings.Index(txt, "INFO"); i > -1 {
				Log.Info(fmt.Sprintf("OPER: %s", txt[i+4:]), "pod", pod.Name)
			} else if i := strings.Index(txt, "DEBUG"); i > -1 {
				Log.Info(fmt.Sprintf("OPER-dbg: %s", txt[i+5:]), "pod", pod.Name)
			} else {
				Log.Info(fmt.Sprintf("OPER: %s", txt), "pod", pod.Name)
			}
		}
	}()

}

type testSetupLogger struct {
	logr.Logger
}

func (l *testSetupLogger) Logf(format string, args ...interface{}) {
	l.Info(fmt.Sprintf(format, args...))
}

type TestInfra struct {
	InstanceConnectionString string `json:"instance,omitempty"`
	Db                       string `json:"db,omitempty"`
	RootPassword             string `json:"rootPassword,omitempty"`
	Kubeconfig               string `json:"kubeconfig,omitempty"`
}

func LoadTestInfra(testInfraJson string) (TestInfra, error) {
	result := TestInfra{}
	bytes, err := os.ReadFile(testInfraJson)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
