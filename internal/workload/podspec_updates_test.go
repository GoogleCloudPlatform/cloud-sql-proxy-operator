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

package workload_test

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workload"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func podWorkload() *workload.PodWorkload {
	return &workload.PodWorkload{Pod: &corev1.Pod{
		TypeMeta:   metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "busybox", Labels: map[string]string{"app": "hello"}},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "busybox", Image: "busybox"}},
		},
	}}
}

func simpleAuthProxy(name, connectionString string) *cloudsqlapi.AuthProxyWorkload {
	return authProxyWorkload(name, []cloudsqlapi.InstanceSpec{{
		ConnectionString: connectionString,
	}})
}

func authProxyWorkload(name string, instances []cloudsqlapi.InstanceSpec) *cloudsqlapi.AuthProxyWorkload {
	return authProxyWorkloadFromSpec(name, cloudsqlapi.AuthProxyWorkloadSpec{
		Workload: cloudsqlapi.WorkloadSelectorSpec{
			Kind: "Deployment",
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "hello"},
			},
		},
		Instances: instances,
	})
}
func authProxyWorkloadFromSpec(name string, spec cloudsqlapi.AuthProxyWorkloadSpec) *cloudsqlapi.AuthProxyWorkload {
	proxy := &cloudsqlapi.AuthProxyWorkload{
		TypeMeta:   metav1.TypeMeta{Kind: "AuthProxyWorkload", APIVersion: cloudsqlapi.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Generation: 1},
		Spec:       spec,
	}
	proxy.Spec.Workload = cloudsqlapi.WorkloadSelectorSpec{
		Kind: "Deployment",
		Selector: &metav1.LabelSelector{
			MatchLabels:      map[string]string{"app": "hello"},
			MatchExpressions: nil,
		},
	}

	return proxy
}

func findContainer(wl *workload.PodWorkload, name string) (corev1.Container, error) {
	for i := range wl.Pod.Spec.Containers {
		c := &wl.Pod.Spec.Containers[i]
		if c.Name == name {
			return *c, nil
		}
	}
	return corev1.Container{}, fmt.Errorf("no container found with name %s", name)
}

func findEnvVar(wl *workload.PodWorkload, containerName, envName string) (corev1.EnvVar, error) {
	container, err := findContainer(wl, containerName)
	if err != nil {
		return corev1.EnvVar{}, err
	}
	for i := 0; i < len(container.Env); i++ {
		if container.Env[i].Name == envName {
			return container.Env[i], nil
		}
	}
	return corev1.EnvVar{}, fmt.Errorf("no envvar named %v on container %v", envName, containerName)
}

func hasArg(wl *workload.PodWorkload, containerName, argValue string) (bool, error) {
	container, err := findContainer(wl, containerName)
	if err != nil {
		return false, err
	}
	for i := 0; i < len(container.Command); i++ {
		if container.Command[i] == argValue {
			return true, nil
		}
	}
	for i := 0; i < len(container.Args); i++ {
		if container.Args[i] == argValue {
			return true, nil
		}
	}
	return false, nil
}

func logPodSpec(t *testing.T, wl *workload.PodWorkload) {
	podSpecYaml, err := yaml.Marshal(wl.Pod.Spec)
	if err != nil {
		t.Errorf("unexpected error while marshaling PodSpec to yaml, %v", err)
	}
	t.Logf("PodSpec: %s", string(podSpecYaml))
}

func configureProxies(u *workload.Updater, wl *workload.PodWorkload, proxies []*cloudsqlapi.AuthProxyWorkload) error {
	l := &cloudsqlapi.AuthProxyWorkloadList{Items: make([]cloudsqlapi.AuthProxyWorkload, len(proxies))}
	for i := 0; i < len(proxies); i++ {
		l.Items[i] = *proxies[i]
	}
	apws := u.FindMatchingAuthProxyWorkloads(l, wl, nil)
	err := u.ConfigureWorkload(wl, apws)
	return err
}

func TestUpdatePodWorkload(t *testing.T) {
	var (
		wantsName               = "instance1"
		wantsPort         int32 = 8080
		wantContainerName       = "csql-default-" + wantsName
		wantsInstanceName       = "project:server:db"
		wantsInstanceArg        = fmt.Sprintf("%s?port=%d", wantsInstanceName, wantsPort)
		u                       = workload.NewUpdater("cloud-sql-proxy-operator/dev", workload.DefaultProxyImage)
	)
	var err error

	// Create a pod
	wl := podWorkload()

	// ensure that the deployment only has one container before
	// updating the deployment.
	if len(wl.Pod.Spec.Containers) != 1 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Pod.Spec.Containers))
	}

	// Create a AuthProxyWorkload that matches the deployment
	proxy := simpleAuthProxy(wantsName, wantsInstanceName)
	proxy.Spec.Instances[0].Port = ptr(wantsPort)

	// Update the container with new markWorkloadNeedsUpdate
	err = configureProxies(u, wl, []*cloudsqlapi.AuthProxyWorkload{proxy})
	if err != nil {
		t.Fatal(err)
	}

	// test that there are now 2 containers
	if want, got := 2, len(wl.Pod.Spec.Containers); want != got {
		t.Fatalf("got %v want %v, number of deployment containers", got, want)
	}

	t.Logf("Containers: {%v}", wl.Pod.Spec.Containers)

	// test that the container has the proper name following the conventions
	foundContainer, err := findContainer(wl, wantContainerName)
	if err != nil {
		t.Fatal(err)
	}

	// test that the container args have the expected args
	if gotArg, err := hasArg(wl, wantContainerName, wantsInstanceArg); err != nil || !gotArg {
		t.Errorf("wants connection string arg %v but it was not present in proxy container args %v",
			wantsInstanceArg, foundContainer.Args)
	}

}

func TestUpdateWorkloadFixedPort(t *testing.T) {
	var (
		wantsInstanceName = "project:server:db"
		wantsPort         = int32(5555)
		wantContainerArgs = []string{
			fmt.Sprintf("%s?port=%d", wantsInstanceName, wantsPort),
		}
		wantWorkloadEnv = map[string]string{
			"DB_HOST": "127.0.0.1",
			"DB_PORT": strconv.Itoa(int(wantsPort)),
		}
		u = workload.NewUpdater("cloud-sql-proxy-operator/dev", workload.DefaultProxyImage)
	)

	// Create a pod
	wl := podWorkload()
	wl.Pod.Spec.Containers[0].Ports =
		[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}

	// Create a AuthProxyWorkload that matches the deployment
	csqls := []*cloudsqlapi.AuthProxyWorkload{
		authProxyWorkload("instance1", []cloudsqlapi.InstanceSpec{{
			ConnectionString: wantsInstanceName,
			Port:             &wantsPort,
			PortEnvName:      "DB_PORT",
			HostEnvName:      "DB_HOST",
		}}),
	}

	// ensure that the new container does not exist
	if len(wl.Pod.Spec.Containers) != 1 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Pod.Spec.Containers))
	}

	// update the containers
	err := configureProxies(u, wl, csqls)
	if err != nil {
		t.Fatal(err)
	}

	// ensure that the new container does not exist
	if len(wl.Pod.Spec.Containers) != 2 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Pod.Spec.Containers))
	}

	// test that the instancename matches the new expected instance name.
	csqlContainer, err := findContainer(wl, fmt.Sprintf("csql-default-%s", csqls[0].GetName()))
	if err != nil {
		t.Fatal(err)
	}

	// test that port cli args are set correctly
	assertContainerArgsContains(t, csqlContainer.Args, wantContainerArgs)

	// Test that workload has the right env vars
	for wantKey, wantValue := range wantWorkloadEnv {
		gotEnvVar, err := findEnvVar(wl, "busybox", wantKey)
		if err != nil {
			t.Error(err)
			logPodSpec(t, wl)
		} else if gotEnvVar.Value != wantValue {
			t.Errorf("got %v, wants %v workload env var %v", gotEnvVar, wantValue, wantKey)
		}
	}

}

func TestWorkloadNoPortSet(t *testing.T) {
	var (
		wantsInstanceName = "project:server:db"
		wantsPort         = int32(5000)
		wantContainerArgs = []string{
			fmt.Sprintf("%s?port=%d", wantsInstanceName, wantsPort),
		}
		wantWorkloadEnv = map[string]string{
			"DB_HOST": "127.0.0.1",
			"DB_PORT": strconv.Itoa(int(wantsPort)),
		}
	)
	u := workload.NewUpdater("cloud-sql-proxy-operator/dev", workload.DefaultProxyImage)

	// Create a pod
	wl := podWorkload()
	wl.Pod.Spec.Containers[0].Ports =
		[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}

	// Create a AuthProxyWorkload that matches the deployment
	csqls := []*cloudsqlapi.AuthProxyWorkload{
		authProxyWorkload("instance1", []cloudsqlapi.InstanceSpec{{
			ConnectionString: wantsInstanceName,
			PortEnvName:      "DB_PORT",
			HostEnvName:      "DB_HOST",
		}}),
	}

	// ensure that the new container does not exist
	if len(wl.Pod.Spec.Containers) != 1 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Pod.Spec.Containers))
	}

	// update the containers
	err := configureProxies(u, wl, csqls)
	if err != nil {
		t.Fatal(err)
	}

	// ensure that the new container does not exist
	if len(wl.Pod.Spec.Containers) != 2 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Pod.Spec.Containers))
	}

	// test that the instancename matches the new expected instance name.
	csqlContainer, err := findContainer(wl, fmt.Sprintf("csql-default-%s", csqls[0].GetName()))
	if err != nil {
		t.Fatal(err)
	}

	// test that port cli args are set correctly
	assertContainerArgsContains(t, csqlContainer.Args, wantContainerArgs)

	// Test that workload has the right env vars
	for wantKey, wantValue := range wantWorkloadEnv {
		gotEnvVar, err := findEnvVar(wl, "busybox", wantKey)
		if err != nil {
			t.Error(err)
			logPodSpec(t, wl)
		} else if gotEnvVar.Value != wantValue {
			t.Errorf("got %v, wants %v workload env var %v", gotEnvVar, wantValue, wantKey)
		}
	}

}

func TestContainerImageChanged(t *testing.T) {
	var (
		wantsInstanceName = "project:server:db"
		wantImage         = "custom-image:latest"
		u                 = workload.NewUpdater("cloud-sql-proxy-operator/dev", workload.DefaultProxyImage)
	)

	// Create a pod
	wl := podWorkload()
	wl.Pod.Spec.Containers[0].Ports =
		[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}

	// Create a AuthProxyWorkload that matches the deployment
	csqls := []*cloudsqlapi.AuthProxyWorkload{
		simpleAuthProxy("instance1", wantsInstanceName),
	}
	csqls[0].Spec.AuthProxyContainer = &cloudsqlapi.AuthProxyContainerSpec{Image: wantImage}

	// update the containers
	err := configureProxies(u, wl, csqls)
	if err != nil {
		t.Fatal(err)
	}

	// ensure that the new container exists
	if len(wl.Pod.Spec.Containers) != 2 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Pod.Spec.Containers))
	}

	// test that the instancename matches the new expected instance name.
	csqlContainer, err := findContainer(wl, fmt.Sprintf("csql-default-%s", csqls[0].GetName()))
	if err != nil {
		t.Fatal(err)
	}

	// test that image was set
	if csqlContainer.Image != wantImage {
		t.Errorf("got %v, want %v for proxy container image", csqlContainer.Image, wantImage)
	}

}

func TestContainerImageEmpty(t *testing.T) {
	var (
		wantsInstanceName = "project:server:db"
		wantImage         = workload.DefaultProxyImage
		u                 = workload.NewUpdater("cloud-sql-proxy-operator/dev", workload.DefaultProxyImage)
	)
	// Create a AuthProxyWorkload that matches the deployment

	// create an AuthProxyContainer that has a value, but Image is empty.
	p1 := simpleAuthProxy("instance1", wantsInstanceName)
	p1.Spec.AuthProxyContainer = &cloudsqlapi.AuthProxyContainerSpec{MaxConnections: ptr(int64(5))}

	// create an AuthProxyContainer where AuthProxyContainer is nil
	p2 := simpleAuthProxy("instance1", wantsInstanceName)
	p2.Spec.AuthProxyContainer = nil

	tests := []struct {
		name  string
		proxy *cloudsqlapi.AuthProxyWorkload
	}{
		{name: "Image is empty", proxy: p1},
		{name: "AuthProxyContainer is nil", proxy: p2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a pod
			wl := podWorkload()
			wl.Pod.Spec.Containers[0].Ports =
				[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}
			csqls := []*cloudsqlapi.AuthProxyWorkload{test.proxy}

			// update the containers
			err := configureProxies(u, wl, csqls)
			if err != nil {
				t.Fatal(err)
			}

			// ensure that the new container exists
			if len(wl.Pod.Spec.Containers) != 2 {
				t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Pod.Spec.Containers))
			}

			// test that the instancename matches the new expected instance name.
			csqlContainer, err := findContainer(wl, fmt.Sprintf("csql-default-%s", csqls[0].GetName()))
			if err != nil {
				t.Fatal(err)
			}

			// test that image was set
			if csqlContainer.Image != wantImage {
				t.Fatalf("got %v, want %v for proxy container image", csqlContainer.Image, wantImage)
			}

		})
	}
}

func TestContainerReplaced(t *testing.T) {
	var (
		wantsInstanceName = "project:server:db"
		wantContainer     = &corev1.Container{
			Name: "sample", Image: "debian:latest", Command: []string{"/bin/bash"},
		}
		u = workload.NewUpdater("cloud-sql-proxy-operator/dev", workload.DefaultProxyImage)
	)

	// Create a pod
	wl := podWorkload()
	wl.Pod.Spec.Containers[0].Ports =
		[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}

	// Create a AuthProxyWorkload that matches the deployment
	csqls := []*cloudsqlapi.AuthProxyWorkload{simpleAuthProxy("instance1", wantsInstanceName)}
	csqls[0].Spec.AuthProxyContainer = &cloudsqlapi.AuthProxyContainerSpec{Container: wantContainer}

	// update the containers
	err := configureProxies(u, wl, csqls)
	if err != nil {
		t.Fatal(err)
	}

	// ensure that the new container exists
	if len(wl.Pod.Spec.Containers) != 2 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Pod.Spec.Containers))
	}

	// test that the instancename matches the new expected instance name.
	csqlContainer, err := findContainer(wl, fmt.Sprintf("csql-default-%s", csqls[0].GetName()))
	if err != nil {
		t.Fatal(err)
	}

	// test that image was set
	if csqlContainer.Image != wantContainer.Image {
		t.Errorf("got %v, want %v for proxy container image", csqlContainer.Image, wantContainer.Image)
	}
	// test that image was set
	if !reflect.DeepEqual(csqlContainer.Command, wantContainer.Command) {
		t.Errorf("got %v, want %v for proxy container command", csqlContainer.Command, wantContainer.Command)
	}

}

func ptr[T int | int32 | int64 | string | bool](i T) *T {
	return &i
}

func TestResourcesFromSpec(t *testing.T) {
	var (
		wantsInstanceName = "project:server:db"
		wantResources     = &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				"cpu":    resource.MustParse("4.0"),
				"memory": resource.MustParse("4Gi"),
			},
		}

		u = workload.NewUpdater("cloud-sql-proxy-operator/dev", workload.DefaultProxyImage)
	)

	// Create a pod
	wl := podWorkload()
	wl.Pod.Spec.Containers[0].Ports =
		[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}

	// Create a AuthProxyWorkload that matches the deployment
	csqls := []*cloudsqlapi.AuthProxyWorkload{simpleAuthProxy("instance1", wantsInstanceName)}
	csqls[0].Spec.AuthProxyContainer = &cloudsqlapi.AuthProxyContainerSpec{Resources: wantResources}

	// update the containers
	err := configureProxies(u, wl, csqls)
	if err != nil {
		t.Fatal(err)
	}

	// ensure that the new container exists
	if len(wl.Pod.Spec.Containers) != 2 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Pod.Spec.Containers))
	}

	// test that the instancename matches the new expected instance name.
	csqlContainer, err := findContainer(wl, fmt.Sprintf("csql-default-%s", csqls[0].GetName()))
	if err != nil {
		t.Fatal(err)
	}

	// test that resources was set
	if !reflect.DeepEqual(csqlContainer.Resources.Requests, wantResources.Requests) {
		t.Errorf("got %v, want %v for proxy container command", csqlContainer.Resources.Requests, wantResources.Requests)
	}

}

func TestProxyCLIArgs(t *testing.T) {
	wantTrue := true
	wantFalse := false

	var wantPort int32 = 5000

	testcases := []struct {
		desc                 string
		proxySpec            cloudsqlapi.AuthProxyWorkloadSpec
		wantProxyArgContains []string
		wantErrorCodes       []string
		wantWorkloadEnv      map[string]string
		dontWantEnvSet       []string
	}{
		{
			desc: "default cli config",
			proxySpec: cloudsqlapi.AuthProxyWorkloadSpec{
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "hello:world:db",
					Port:             &wantPort,
					PortEnvName:      "DB_PORT",
				}},
			},
			wantWorkloadEnv: map[string]string{
				"CSQL_PROXY_STRUCTURED_LOGS":      "true",
				"CSQL_PROXY_HEALTH_CHECK":         "true",
				"CSQL_PROXY_QUITQUITQUIT":         "true",
				"CSQL_PROXY_EXIT_ZERO_ON_SIGTERM": "true",
				"CSQL_PROXY_HTTP_PORT":            fmt.Sprintf("%d", workload.DefaultHealthCheckPort),
				"CSQL_PROXY_HTTP_ADDRESS":         "0.0.0.0",
				"CSQL_PROXY_USER_AGENT":           "cloud-sql-proxy-operator/dev",
				"CSQL_PROXY_ADMIN_PORT":           fmt.Sprintf("%d", workload.DefaultAdminPort),
			},
		},
		{
			desc: "port explicitly set",
			proxySpec: cloudsqlapi.AuthProxyWorkloadSpec{
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "hello:world:db",
					Port:             &wantPort,
					PortEnvName:      "DB_PORT",
				}},
			},
			wantProxyArgContains: []string{"hello:world:db?port=5000"},
		},
		{
			desc: "port implicitly set and increments",
			proxySpec: cloudsqlapi.AuthProxyWorkloadSpec{
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "hello:world:one",
					PortEnvName:      "DB_PORT",
				},
					{
						ConnectionString: "hello:world:two",
						PortEnvName:      "DB_PORT_2",
					}},
			},
			wantProxyArgContains: []string{
				fmt.Sprintf("hello:world:one?port=%d", workload.DefaultFirstPort),
				fmt.Sprintf("hello:world:two?port=%d", workload.DefaultFirstPort+1)},
		},
		{
			desc: "env name conflict causes error",
			proxySpec: cloudsqlapi.AuthProxyWorkloadSpec{
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "hello:world:one",
					PortEnvName:      "DB_PORT",
				},
					{
						ConnectionString: "hello:world:two",
						PortEnvName:      "DB_PORT",
					}},
			},
			wantProxyArgContains: []string{
				fmt.Sprintf("hello:world:one?port=%d", workload.DefaultFirstPort),
				fmt.Sprintf("hello:world:two?port=%d", workload.DefaultFirstPort+1)},
			wantErrorCodes: []string{cloudsqlapi.ErrorCodeEnvConflict},
		},
		{
			desc: "auto-iam-authn set",
			proxySpec: cloudsqlapi.AuthProxyWorkloadSpec{
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "hello:world:one",
					PortEnvName:      "DB_PORT",
					AutoIAMAuthN:     &wantTrue,
				},
					{
						ConnectionString: "hello:world:two",
						PortEnvName:      "DB_PORT_2",
						AutoIAMAuthN:     &wantFalse,
					}},
			},
			wantProxyArgContains: []string{
				fmt.Sprintf("hello:world:one?auto-iam-authn=true&port=%d", workload.DefaultFirstPort),
				fmt.Sprintf("hello:world:two?auto-iam-authn=false&port=%d", workload.DefaultFirstPort+1)},
		},
		{
			desc: "private-ip set",
			proxySpec: cloudsqlapi.AuthProxyWorkloadSpec{
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "hello:world:one",
					PortEnvName:      "DB_PORT",
					PrivateIP:        &wantTrue,
				},
					{
						ConnectionString: "hello:world:two",
						PortEnvName:      "DB_PORT_2",
						PrivateIP:        &wantFalse,
					}},
			},
			wantProxyArgContains: []string{
				fmt.Sprintf("hello:world:one?port=%d&private-ip=true", workload.DefaultFirstPort),
				fmt.Sprintf("hello:world:two?port=%d&private-ip=false", workload.DefaultFirstPort+1)},
		},
		{
			desc: "psc set",
			proxySpec: cloudsqlapi.AuthProxyWorkloadSpec{
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "hello:world:one",
					PortEnvName:      "DB_PORT",
					PSC:              &wantTrue,
				},
					{
						ConnectionString: "hello:world:two",
						PortEnvName:      "DB_PORT_2",
						PSC:              &wantFalse,
					}},
			},
			wantProxyArgContains: []string{
				fmt.Sprintf("hello:world:one?port=%d&psc=true", workload.DefaultFirstPort),
				fmt.Sprintf("hello:world:two?port=%d&psc=false", workload.DefaultFirstPort+1)},
		},
		{
			desc: "global flags",
			proxySpec: cloudsqlapi.AuthProxyWorkloadSpec{
				AuthProxyContainer: &cloudsqlapi.AuthProxyContainerSpec{
					SQLAdminAPIEndpoint: "https://example.com",
					Telemetry: &cloudsqlapi.TelemetrySpec{
						TelemetryPrefix:     ptr("telprefix"),
						TelemetryProject:    ptr("telproject"),
						TelemetrySampleRate: ptr(200),
						HTTPPort:            ptr(int32(9092)),
						DisableTraces:       &wantTrue,
						DisableMetrics:      &wantTrue,
						Prometheus:          &wantTrue,
						PrometheusNamespace: ptr("hello"),
						QuotaProject:        ptr("qp"),
					},
					AdminServer: &cloudsqlapi.AdminServerSpec{
						EnableAPIs: []string{"Debug", "QuitQuitQuit"},
						Port:       int32(9091),
					},
					Authentication: &cloudsqlapi.AuthenticationSpec{
						ImpersonationChain: []string{"sv1@developer.gserviceaccount.com", "sv2@developer.gserviceaccount.com"},
					},
					MaxConnections:  ptr(int64(10)),
					MaxSigtermDelay: ptr(int64(20)),
					Quiet:           true,
					RefreshStrategy: "lazy",
				},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "hello:world:one",
					Port:             ptr(int32(5000)),
				}},
			},
			wantProxyArgContains: []string{
				fmt.Sprintf("hello:world:one?port=%d", workload.DefaultFirstPort),
			},
			wantWorkloadEnv: map[string]string{
				"CSQL_PROXY_SQLADMIN_API_ENDPOINT":       "https://example.com",
				"CSQL_PROXY_TELEMETRY_SAMPLE_RATE":       "200",
				"CSQL_PROXY_PROMETHEUS_NAMESPACE":        "hello",
				"CSQL_PROXY_TELEMETRY_PROJECT":           "telproject",
				"CSQL_PROXY_TELEMETRY_PREFIX":            "telprefix",
				"CSQL_PROXY_HTTP_PORT":                   "9092",
				"CSQL_PROXY_ADMIN_PORT":                  "9091",
				"CSQL_PROXY_DEBUG":                       "true",
				"CSQL_PROXY_QUITQUITQUIT":                "true",
				"CSQL_PROXY_HEALTH_CHECK":                "true",
				"CSQL_PROXY_DISABLE_TRACES":              "true",
				"CSQL_PROXY_DISABLE_METRICS":             "true",
				"CSQL_PROXY_PROMETHEUS":                  "true",
				"CSQL_PROXY_QUOTA_PROJECT":               "qp",
				"CSQL_PROXY_MAX_CONNECTIONS":             "10",
				"CSQL_PROXY_MAX_SIGTERM_DELAY":           "20s",
				"CSQL_PROXY_IMPERSONATE_SERVICE_ACCOUNT": "sv1@developer.gserviceaccount.com,sv2@developer.gserviceaccount.com",
				"CSQL_PROXY_QUIET":                       "true",
				"CSQL_PROXY_STRUCTURED_LOGS":             "true",
				"CSQL_PROXY_LAZY_REFRESH":                "true",
			},
		},
		{
			desc: "Default admin port enabled when AdminServerSpec is nil",
			proxySpec: cloudsqlapi.AuthProxyWorkloadSpec{
				AuthProxyContainer: &cloudsqlapi.AuthProxyContainerSpec{},
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "hello:world:one",
					Port:             ptr(int32(5000)),
				}},
			},
			wantProxyArgContains: []string{
				fmt.Sprintf("hello:world:one?port=%d", workload.DefaultFirstPort),
			},
			wantWorkloadEnv: map[string]string{
				"CSQL_PROXY_HEALTH_CHECK": "true",
				"CSQL_PROXY_ADMIN_PORT":   fmt.Sprintf("%d", workload.DefaultAdminPort),
			},
			dontWantEnvSet: []string{"CSQL_PROXY_DEBUG"},
		},
		{
			desc: "port conflict with other instance causes error",
			proxySpec: cloudsqlapi.AuthProxyWorkloadSpec{
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "hello:world:one",
					PortEnvName:      "DB_PORT_1",
					Port:             ptr(int32(8081)),
				},
					{
						ConnectionString: "hello:world:two",
						PortEnvName:      "DB_PORT_2",
						Port:             ptr(int32(8081)),
					}},
			},
			wantProxyArgContains: []string{
				fmt.Sprintf("hello:world:one?port=%d", 8081),
				fmt.Sprintf("hello:world:two?port=%d", 8081)},
			wantErrorCodes: []string{cloudsqlapi.ErrorCodePortConflict},
		},
		{
			desc: "port conflict with workload container",
			proxySpec: cloudsqlapi.AuthProxyWorkloadSpec{
				Instances: []cloudsqlapi.InstanceSpec{{
					ConnectionString: "hello:world:one",
					PortEnvName:      "DB_PORT_1",
					Port:             ptr(int32(8080)),
				}},
			},
			wantProxyArgContains: []string{
				fmt.Sprintf("hello:world:one?port=%d", 8080)},
			wantErrorCodes: []string{cloudsqlapi.ErrorCodePortConflict},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			u := workload.NewUpdater("cloud-sql-proxy-operator/dev", workload.DefaultProxyImage)

			// Create a pod
			wl := &workload.PodWorkload{Pod: &corev1.Pod{
				TypeMeta:   metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "busybox", Labels: map[string]string{"app": "hello"}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "busybox", Image: "busybox",
						Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}}},
				},
			}}

			// Create a AuthProxyWorkload that matches the deployment
			csqls := []*cloudsqlapi.AuthProxyWorkload{authProxyWorkloadFromSpec("instance1", tc.proxySpec)}

			// ensure valid
			_, err := csqls[0].ValidateCreate()
			if err != nil {
				t.Fatal("Invalid AuthProxyWorkload resource", err)
			}

			// update the containers
			updateErr := configureProxies(u, wl, csqls)

			if len(tc.wantErrorCodes) > 0 {
				assertErrorCodeContains(t, updateErr, tc.wantErrorCodes)
				return
			}

			// ensure that the new container exists
			if len(wl.Pod.Spec.Containers) != 2 {
				t.Fatalf("got %v, wants 2. deployment containers length", len(wl.Pod.Spec.Containers))
			}

			// test that the instancename matches the new expected instance name.
			csqlContainer, err := findContainer(wl, fmt.Sprintf("csql-default-%s", csqls[0].GetName()))
			if err != nil {
				t.Fatal(err)
			}

			// test that port cli args are set correctly
			assertContainerArgsContains(t, csqlContainer.Args, tc.wantProxyArgContains)

			// Test that workload has the right env vars
			for wantKey, wantValue := range tc.wantWorkloadEnv {
				gotEnvVar, err := findEnvVar(wl, csqlContainer.Name, wantKey)
				if err != nil {
					t.Error(err)
					continue
				}

				if gotEnvVar.Value != wantValue {
					t.Errorf("got %v, wants %v workload env var %v", gotEnvVar, wantValue, wantKey)
				}
			}
			for _, dontWantKey := range tc.dontWantEnvSet {
				gotEnvVar, err := findEnvVar(wl, csqlContainer.Name, dontWantKey)
				if err != nil {
					continue
				}
				t.Errorf("got env %v=%v, wants no env var set", dontWantKey, gotEnvVar)
			}
		})
	}

}

func assertErrorCodeContains(t *testing.T, gotErr error, wantErrors []string) {
	if gotErr == nil {
		if len(wantErrors) > 0 {
			t.Errorf("got missing errors, wants errors with codes %v", wantErrors)
		}
		return
	}
	gotError, ok := gotErr.(*workload.ConfigError)
	if !ok {
		t.Errorf("got an error %v, wants error of type *internal.ConfigError", gotErr)
		return
	}

	errs := gotError.DetailedErrors()

	for i := 0; i < len(wantErrors); i++ {
		wantArg := wantErrors[i]
		found := false
		for j := 0; j < len(errs) && !found; j++ {
			if wantArg == errs[j].ErrorCode {
				found = true
			}
		}
		if !found {
			t.Errorf("missing error, wants error with code %v, got error %v", wantArg, gotError)
		}
	}

	for i := 0; i < len(errs); i++ {
		gotErr := errs[i]
		found := false
		for j := 0; j < len(wantErrors) && !found; j++ {
			if gotErr.ErrorCode == wantErrors[j] {
				found = true
			}
		}
		if !found {
			t.Errorf("got unexpected error %v", gotErr)
		}
	}

}

func assertContainerArgsContains(t *testing.T, gotArgs, wantArgs []string) {
	for i := 0; i < len(wantArgs); i++ {
		wantArg := wantArgs[i]
		found := false
		for j := 0; j < len(gotArgs) && !found; j++ {
			if wantArg == gotArgs[j] {
				found = true
			}
		}
		if !found {
			t.Errorf("missing argument, wants argument %v, got arguments %v", wantArg, gotArgs)
		}
	}
}

func TestPodTemplateAnnotations(t *testing.T) {

	var (
		now = metav1.Now()

		wantAnnotations = map[string]string{
			"cloudsql.cloud.google.com/instance1": "1," + workload.DefaultProxyImage,
			"cloudsql.cloud.google.com/instance2": "2," + workload.DefaultProxyImage,
		}

		u = workload.NewUpdater("cloud-sql-proxy-operator/dev", workload.DefaultProxyImage)
	)

	// Create a pod
	wl := podWorkload()
	wl.Pod.Spec.Containers[0].Ports =
		[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}

	// Create a AuthProxyWorkload that matches the deployment
	csqls := []*cloudsqlapi.AuthProxyWorkload{
		simpleAuthProxy("instance1", "project:server:db"),
		simpleAuthProxy("instance2", "project:server2:db2"),
		simpleAuthProxy("instance3", "project:server3:db3")}

	csqls[0].ObjectMeta.Generation = 1
	csqls[1].ObjectMeta.Generation = 2
	csqls[2].ObjectMeta.Generation = 3
	csqls[2].ObjectMeta.DeletionTimestamp = &now

	// update the containers
	err := configureProxies(u, wl, csqls)
	if err != nil {
		t.Fatal(err)
	}

	// test that annotation was set properly
	if !reflect.DeepEqual(wl.PodTemplateAnnotations(), wantAnnotations) {
		t.Errorf("got %v, want %v for proxy container command", wl.PodTemplateAnnotations(), wantAnnotations)
	}

}

func TestTelemetryAddsTelemetryContainerPort(t *testing.T) {

	var u = workload.NewUpdater("cloud-sql-proxy-operator/dev", workload.DefaultProxyImage)

	// Create a pod
	wl := podWorkload()
	wl.Pod.Spec.Containers[0].Ports =
		[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}

	// Create a AuthProxyWorkload that matches the deployment
	csqls := []*cloudsqlapi.AuthProxyWorkload{
		simpleAuthProxy("instance1", "project:server:db"),
		simpleAuthProxy("instance2", "project:server2:db2"),
		simpleAuthProxy("instance3", "project:server3:db3"),
	}

	// explicitly configure the telemetry http port for test consistency.
	csqls[0].Spec.AuthProxyContainer = &cloudsqlapi.AuthProxyContainerSpec{
		Telemetry: &cloudsqlapi.TelemetrySpec{
			HTTPPort: ptr(workload.DefaultHealthCheckPort),
		},
	}
	csqls[1].Spec.AuthProxyContainer = &cloudsqlapi.AuthProxyContainerSpec{
		Telemetry: &cloudsqlapi.TelemetrySpec{
			HTTPPort: ptr(workload.DefaultHealthCheckPort + 1),
		},
	}
	csqls[2].Spec.AuthProxyContainer = &cloudsqlapi.AuthProxyContainerSpec{
		Telemetry: &cloudsqlapi.TelemetrySpec{
			HTTPPort: ptr(workload.DefaultHealthCheckPort + 2),
		},
	}

	// update the containers
	err := configureProxies(u, wl, csqls)
	if err != nil {
		t.Fatal(err)
	}

	var wantPorts = map[string]int32{
		workload.ContainerName(csqls[0]): workload.DefaultHealthCheckPort,
		workload.ContainerName(csqls[1]): workload.DefaultHealthCheckPort + 1,
		workload.ContainerName(csqls[2]): workload.DefaultHealthCheckPort + 2,
	}

	// test that containerPort values were set properly
	for name, wantPort := range wantPorts {
		var found bool
		for _, c := range wl.PodSpec().Containers {
			if c.Name == name {
				found = true
				if len(c.Ports) == 0 {
					t.Fatalf("want container wantPort for conatiner %s at wantPort %d, got no containerPort", name, wantPort)
				}
				if got := c.Ports[0].ContainerPort; got != wantPort {
					t.Errorf("want container wantPort for conatiner %s at wantPort %d, got wantPort = %d ", name, wantPort, got)
				}
				continue
			}
		}
		if !found {
			t.Fatalf("want container %s, got no container", name)
		}
	}

}

func TestQuitURLEnvVar(t *testing.T) {

	var (
		u = workload.NewUpdater("cloud-sql-proxy-operator/dev", workload.DefaultProxyImage)
	)

	// Create a pod
	wl := podWorkload()
	wl.Pod.Spec.Containers[0].Ports =
		[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}

	// Create a AuthProxyWorkload that matches the deployment
	csqls := []*cloudsqlapi.AuthProxyWorkload{
		simpleAuthProxy("instance1", "project:server:db"),
		simpleAuthProxy("instance2", "project:server2:db2"),
		simpleAuthProxy("instance3", "project:server3:db3")}

	var wantQuitURLSEnv = strings.Join(
		[]string{
			fmt.Sprintf("http://localhost:%d/quitquitquit", workload.DefaultAdminPort),
			fmt.Sprintf("http://localhost:%d/quitquitquit", workload.DefaultAdminPort+1),
			fmt.Sprintf("http://localhost:%d/quitquitquit", workload.DefaultAdminPort+2),
		},
		" ",
	)

	// update the containers
	err := configureProxies(u, wl, csqls)
	if err != nil {
		t.Fatal(err)
	}

	// test that envvar was set
	ev, err := findEnvVar(wl, "busybox", "CSQL_PROXY_QUIT_URLS")
	if err != nil {
		t.Fatal("can't find env var", err)
	}
	if ev.Value != wantQuitURLSEnv {
		t.Fatal("got", ev.Value, "want", wantQuitURLSEnv)
	}
}

func TestPreStopHook(t *testing.T) {

	var u = workload.NewUpdater("cloud-sql-proxy-operator/dev", workload.DefaultProxyImage)

	// Create a pod
	wl := podWorkload()
	wl.Pod.Spec.Containers[0].Ports =
		[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}

	// Create a AuthProxyWorkload that matches the deployment
	csqls := []*cloudsqlapi.AuthProxyWorkload{
		simpleAuthProxy("instance1", "project:server:db")}

	csqls[0].ObjectMeta.Generation = 1

	// update the containers
	err := configureProxies(u, wl, csqls)
	if err != nil {
		t.Fatal(err)
	}

	// test that prestop hook was set
	c, err := findContainer(wl, workload.ContainerName(csqls[0]))
	if err != nil {
		t.Fatal("can't find proxy container", err)
	}
	if c.Lifecycle.PreStop == nil || c.Lifecycle.PreStop.HTTPGet == nil {
		t.Fatal("got nil, want lifecycle.preStop.HTTPGet")
	}
	get := c.Lifecycle.PreStop.HTTPGet
	if get.Port.IntVal != workload.DefaultAdminPort {
		t.Error("got", get.Port, "want", workload.DefaultAdminPort)
	}
	if get.Path != "/quitquitquit" {
		t.Error("got", get.Path, "want", "/quitquitquit")
	}
	if get.Host != "localhost" {
		t.Error("got", get.Host, "want", "localhost")
	}
}

func TestPodAnnotation(t *testing.T) {
	now := metav1.Now()
	server := &cloudsqlapi.AuthProxyWorkload{ObjectMeta: metav1.ObjectMeta{Name: "instance1", Generation: 1}}
	deletedServer := &cloudsqlapi.AuthProxyWorkload{ObjectMeta: metav1.ObjectMeta{Name: "instance2", Generation: 2, DeletionTimestamp: &now}}

	var testcases = []struct {
		name  string
		r     *cloudsqlapi.AuthProxyWorkload
		wantK string
		wantV string
	}{
		{
			name:  "instance1",
			r:     server,
			wantK: "cloudsql.cloud.google.com/instance1",
			wantV: fmt.Sprintf("1,%s", workload.DefaultProxyImage),
		}, {
			name:  "instance2",
			r:     deletedServer,
			wantK: "cloudsql.cloud.google.com/instance2",
			wantV: fmt.Sprintf("2-deleted-%s,%s", now.Format(time.RFC3339), workload.DefaultProxyImage),
		},
	}

	for _, tc := range testcases {
		gotK, gotV := workload.PodAnnotation(tc.r, workload.DefaultProxyImage)
		if tc.wantK != gotK {
			t.Errorf("got %v, want %v for key", gotK, tc.wantK)
		}
		if tc.wantV != gotV {
			t.Errorf("got %v, want %v for value", gotV, tc.wantV)
		}
	}
}

func TestWorkloadUnixVolume(t *testing.T) {
	var (
		wantsInstanceName    = "project:server:db"
		wantsInstanceName2   = "project:server:db2"
		wantsUnixSocketPath  = "/mnt/db/server"
		wantsUnixSocketPath2 = "/mnt/db/server2"
		wantUnixMountDir     = "/mnt/db"
		wantContainerArgs    = []string{
			fmt.Sprintf("%s?unix-socket-path=%s", wantsInstanceName, wantsUnixSocketPath),
			fmt.Sprintf("%s?unix-socket-path=%s", wantsInstanceName2, wantsUnixSocketPath2),
		}
		wantWorkloadEnv = map[string]string{
			"DB_SOCKET_PATH": wantsUnixSocketPath,
		}
		u = workload.NewUpdater("authproxyworkload/dev", workload.DefaultProxyImage)
	)

	// Create a pod
	wl := podWorkload()
	wl.Pod.Spec.Containers[0].Ports =
		[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}

	// Create a AuthProxyWorkload that matches the deployment
	csqls := []*cloudsqlapi.AuthProxyWorkload{
		authProxyWorkload("instance1", []cloudsqlapi.InstanceSpec{{
			ConnectionString:      wantsInstanceName,
			UnixSocketPath:        wantsUnixSocketPath,
			UnixSocketPathEnvName: "DB_SOCKET_PATH",
		}, {
			ConnectionString:      wantsInstanceName2,
			UnixSocketPath:        wantsUnixSocketPath2,
			UnixSocketPathEnvName: "DB_SOCKET_PATH2",
		}}),
	}

	// update the containers
	err := configureProxies(u, wl, csqls)
	if err != nil {
		t.Fatal(err)
	}

	// ensure that the new container exists
	if len(wl.Pod.Spec.Containers) != 2 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Pod.Spec.Containers))
	}

	// test that the instancename matches the new expected instance name.
	csqlContainer, err := findContainer(wl, fmt.Sprintf("csql-default-%s", csqls[0].GetName()))
	if err != nil {
		t.Fatal(err)
	}

	// test that port cli args are set correctly
	assertContainerArgsContains(t, csqlContainer.Args, wantContainerArgs)

	// Test that workload has the right env vars
	for wantKey, wantValue := range wantWorkloadEnv {
		gotEnvVar, err := findEnvVar(wl, "busybox", wantKey)
		if err != nil {
			t.Error(err)
			logPodSpec(t, wl)
		} else if gotEnvVar.Value != wantValue {
			t.Errorf("got %v, wants %v workload env var %v", gotEnvVar, wantValue, wantKey)

		}
	}

	// test that Volume exists
	if want, got := 1, len(wl.Pod.Spec.Volumes); want != got {
		t.Fatalf("got %v, wants %v. PodSpec.Volumes", got, want)
	}

	// test that Volume mount exists on busybox
	busyboxContainer, err := findContainer(wl, "busybox")
	if err != nil {
		t.Fatal(err)
	}
	if want, got := 1, len(busyboxContainer.VolumeMounts); want != got {
		t.Fatalf("got %v, wants %v. Busybox Container.VolumeMounts", got, want)
	}
	if want, got := wantUnixMountDir, busyboxContainer.VolumeMounts[0].MountPath; want != got {
		t.Fatalf("got %v, wants %v. Busybox Container.VolumeMounts.MountPath", got, want)
	}
	if want, got := wl.Pod.Spec.Volumes[0].Name, busyboxContainer.VolumeMounts[0].Name; want != got {
		t.Fatalf("got %v, wants %v. Busybox Container.VolumeMounts.MountPath", got, want)
	}

}

func TestUpdater_CheckWorkloadContainers(t *testing.T) {
	var (
		wantsInstanceName = "project:server:db"
		u                 = workload.NewUpdater("cloud-sql-proxy-operator/dev", workload.DefaultProxyImage)
	)

	// Create a AuthProxyWorkloads to match the pods
	p1 := simpleAuthProxy("instance1", wantsInstanceName)
	p2 := simpleAuthProxy("instance2", wantsInstanceName)
	csqls := []*cloudsqlapi.AuthProxyWorkload{p1, p2}

	// Pod configuration says it is running, and has all proxy containers
	wlCfgRunning := podWorkload()
	err := configureProxies(u, wlCfgRunning, csqls)
	if err != nil {
		t.Fatal(err)
	}
	wlCfgRunning.Pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
		Name:  wlCfgRunning.Pod.Spec.Containers[0].Name,
		State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: metav1.Now()}},
	}}

	// Pod configuration says it is in error, and has all proxy containers
	wlCfgError := podWorkload()
	err = configureProxies(u, wlCfgError, csqls)
	if err != nil {
		t.Fatal(err)
	}
	wlCfgError.Pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
		Name:  wlCfgError.Pod.Spec.Containers[0].Name,
		State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 2}},
	}}

	// Pod configuration says it is waiting, and has all proxy containers
	wlCfgWait := podWorkload()
	err = configureProxies(u, wlCfgWait, csqls)
	if err != nil {
		t.Fatal(err)
	}
	wlCfgWait.Pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
		Name:  wlCfgWait.Pod.Spec.Containers[0].Name,
		State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
	}}

	// Pod configuration says it is running and is missing its proxy containers
	wlCfgMissingRunning := podWorkload()
	wlCfgMissingRunning.Pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
		Name:  wlCfgMissingRunning.Pod.Spec.Containers[0].Name,
		State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: metav1.Now()}},
	}}

	// Pod configuration says it is in error and is missing its proxy containers
	wlCfgMissingError := podWorkload()
	wlCfgMissingError.Pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
		Name:  wlCfgMissingError.Pod.Spec.Containers[0].Name,
		State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 2}},
	}}

	// Pod configuration says it is waiting is missing its proxy containers
	wlCfgMissingWait := podWorkload()
	wlCfgMissingWait.Pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
		Name:  wlCfgMissingWait.Pod.Spec.Containers[0].Name,
		State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
	}}

	// Pod configuration says it is in error and is missing one of the 2 sidecar
	// containers.
	wlCfgOutOfDateError := podWorkload()
	// Only configure 1 of the 2 expected AuthProxyWorkload sidecar containers
	err = configureProxies(u, wlCfgWait, []*cloudsqlapi.AuthProxyWorkload{p1})
	if err != nil {
		t.Fatal(err)
	}
	wlCfgOutOfDateError.Pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
		Name:  wlCfgOutOfDateError.Pod.Spec.Containers[0].Name,
		State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 2}},
	}}

	tests := []struct {
		name    string
		wl      *workload.PodWorkload
		wantErr bool
	}{
		{name: "Workload Configured, state running", wl: wlCfgRunning},
		{name: "Workload Configured, state terminated", wl: wlCfgError},
		{name: "Workload Configured, state waiting", wl: wlCfgWait},
		{name: "Config missing, state running", wl: wlCfgMissingRunning},
		{name: "Config missing, state terminated", wl: wlCfgMissingError, wantErr: true},
		{name: "Config missing, state waiting", wl: wlCfgMissingWait, wantErr: true},
		{name: "One container missing, state error", wl: wlCfgOutOfDateError, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// update the containers
			err := u.CheckWorkloadContainers(test.wl, csqls)

			if test.wantErr && err == nil {
				t.Fatal("want not nil, got nil. err")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("want nil, got %v. err", err)
			}

		})
	}
}
