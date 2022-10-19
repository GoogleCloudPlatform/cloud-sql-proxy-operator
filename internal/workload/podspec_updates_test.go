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
	"testing"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1alpha1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/workload"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var u = workload.NewUpdater()

func deploymentWorkload() *workload.DeploymentWorkload {
	return &workload.DeploymentWorkload{Deployment: &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "busybox", Labels: map[string]string{"app": "hello"}},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "busybox", Image: "busybox"}},
				},
			},
		},
	}}
}

func simpleAuthProxy(name, connectionString string) *v1alpha1.AuthProxyWorkload {
	return authProxyWorkload(name, []v1alpha1.InstanceSpec{{
		ConnectionString: connectionString,
	}})
}

func authProxyWorkload(name string, instances []v1alpha1.InstanceSpec) *v1alpha1.AuthProxyWorkload {
	return authProxyWorkloadFromSpec(name, v1alpha1.AuthProxyWorkloadSpec{
		Workload: v1alpha1.WorkloadSelectorSpec{
			Kind: "Deployment",
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "hello"},
			},
		},
		Instances: instances,
	})
}
func authProxyWorkloadFromSpec(name string, spec v1alpha1.AuthProxyWorkloadSpec) *v1alpha1.AuthProxyWorkload {
	proxy := &v1alpha1.AuthProxyWorkload{
		TypeMeta:   metav1.TypeMeta{Kind: "AuthProxyWorkload", APIVersion: v1alpha1.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Generation: 1},
		Spec:       spec,
	}
	proxy.Spec.Workload = v1alpha1.WorkloadSelectorSpec{
		Kind: "Deployment",
		Selector: &metav1.LabelSelector{
			MatchLabels:      map[string]string{"app": "hello"},
			MatchExpressions: nil,
		},
	}

	return proxy
}

// markWorkloadNeedsUpdate When an AuthProxyWorkload changes, matching workloads get
// marked with an annotation indicating that it needs to be updated. This function adds
// the appropriate "needs update" annotation to the workload wl for each of the
// AuthProxyWorkload in proxies.
func markWorkloadNeedsUpdate(wl *workload.DeploymentWorkload, proxies ...*v1alpha1.AuthProxyWorkload) []*v1alpha1.AuthProxyWorkload {
	for i := 0; i < len(proxies); i++ {
		u.MarkWorkloadNeedsUpdate(proxies[i], wl)
	}
	return proxies
}

func findContainer(wl *workload.DeploymentWorkload, name string) (corev1.Container, error) {
	for i := range wl.Deployment.Spec.Template.Spec.Containers {
		c := &wl.Deployment.Spec.Template.Spec.Containers[i]
		if c.Name == name {
			return *c, nil
		}
	}
	return corev1.Container{}, fmt.Errorf("no container found with name %s", name)
}

func findEnvVar(wl *workload.DeploymentWorkload, containerName, envName string) (corev1.EnvVar, error) {
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

func hasArg(wl *workload.DeploymentWorkload, containerName, argValue string) (bool, error) {
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

func logPodSpec(t *testing.T, wl *workload.DeploymentWorkload) {
	podSpecYaml, err := yaml.Marshal(wl.Deployment.Spec.Template.Spec)
	if err != nil {
		t.Errorf("unexpected error while marshaling PodSpec to yaml, %v", err)
	}
	t.Logf("PodSpec: %s", string(podSpecYaml))
}

func TestUpdateWorkload(t *testing.T) {
	var (
		wantsName                      = "instance1"
		wantsPort                int32 = 8080
		wantContainerName              = "csql-default-" + wantsName
		wantsInstanceName              = "project:server:db"
		wantsUpdatedInstanceName       = "project:server:newdb"
		wantsInstanceArg               = fmt.Sprintf("%s?port=%d", wantsInstanceName, wantsPort)
		wantsUpdatedInstanceArg        = fmt.Sprintf("%s?port=%d", wantsUpdatedInstanceName, wantsPort)
	)
	var err error

	// Create a deployment
	wl := deploymentWorkload()

	// ensure that the deployment only has one container before
	// updating the deployment.
	if len(wl.Deployment.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Deployment.Spec.Template.Spec.Containers))
	}

	// Create a AuthProxyWorkload that matches the deployment
	proxy := simpleAuthProxy(wantsName, wantsInstanceName)
	proxy.Spec.Instances[0].Port = ptr(wantsPort)
	proxies := markWorkloadNeedsUpdate(wl, proxy)

	// Update the container with new markWorkloadNeedsUpdate
	_, err = u.UpdateWorkloadContainers(wl, proxies)
	if err != nil {
		t.Fatal(err)
	}

	// test that there are now 2 containers
	if want, got := 2, len(wl.Deployment.Spec.Template.Spec.Containers); want != got {
		t.Fatalf("got %v want %v, number of deployment containers", got, want)
	}

	t.Logf("Containers: {%v}", wl.Deployment.Spec.Template.Spec.Containers)

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

	// Now change the spec
	proxies[0].Spec.Instances[0].ConnectionString = wantsUpdatedInstanceName
	proxies[0].ObjectMeta.Generation = 2
	// update the containers again with the new instance name

	// Indicate that the workload needs an update
	u.MarkWorkloadNeedsUpdate(proxies[0], wl)

	// Perform the update
	_, err = u.UpdateWorkloadContainers(wl, proxies)
	if err != nil {
		t.Fatal(err)
	}

	// test that there are still 2 containers
	if want, got := 2, len(wl.Deployment.Spec.Template.Spec.Containers); want != got {
		t.Fatalf("got %v want %v, number of deployment containers", got, want)
	}

	// test that the container has the proper name following the conventions
	foundContainer, err = findContainer(wl, wantContainerName)
	if err != nil {
		t.Fatal(err)
	}

	// test that the container args have the expected args
	if gotArg, err := hasArg(wl, wantContainerName, wantsUpdatedInstanceArg); err != nil || !gotArg {
		t.Errorf("wants connection string arg %v but it was not present in proxy container args %v",
			wantsInstanceArg, foundContainer.Args)
	}

	// now try with an empty workload list, which should remove the container
	_, err = u.UpdateWorkloadContainers(wl, []*v1alpha1.AuthProxyWorkload{})
	if err != nil {
		t.Fatal(err)
	}

	// test that there is now only 1 container
	if len(wl.Deployment.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Deployment.Spec.Template.Spec.Containers))
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
			"DB_HOST": "localhost",
			"DB_PORT": strconv.Itoa(int(wantsPort)),
		}
	)

	// Create a deployment
	wl := deploymentWorkload()
	wl.Deployment.Spec.Template.Spec.Containers[0].Ports =
		[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}

	// Create a AuthProxyWorkload that matches the deployment
	csqls := []*v1alpha1.AuthProxyWorkload{
		authProxyWorkload("instance1", []v1alpha1.InstanceSpec{{
			ConnectionString: wantsInstanceName,
			Port:             &wantsPort,
			PortEnvName:      "DB_PORT",
			HostEnvName:      "DB_HOST",
		}}),
	}

	// ensure that the new container does not exist
	if len(wl.Deployment.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Deployment.Spec.Template.Spec.Containers))
	}

	// Indicate that the workload needs an update
	u.MarkWorkloadNeedsUpdate(csqls[0], wl)

	// update the containers
	_, err := u.UpdateWorkloadContainers(wl, csqls)
	if err != nil {
		t.Fatal(err)
	}

	// ensure that the new container does not exist
	if len(wl.Deployment.Spec.Template.Spec.Containers) != 2 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Deployment.Spec.Template.Spec.Containers))
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
	var wantsInstanceName = "project:server:db"
	var wantsPort = int32(5000)
	var wantContainerArgs = []string{
		fmt.Sprintf("%s?port=%d", wantsInstanceName, wantsPort),
	}
	var wantWorkloadEnv = map[string]string{
		"DB_HOST": "localhost",
		"DB_PORT": strconv.Itoa(int(wantsPort)),
	}

	// Create a deployment
	wl := deploymentWorkload()
	wl.Deployment.Spec.Template.Spec.Containers[0].Ports =
		[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}

	// Create a AuthProxyWorkload that matches the deployment
	csqls := []*v1alpha1.AuthProxyWorkload{
		authProxyWorkload("instance1", []v1alpha1.InstanceSpec{{
			ConnectionString: wantsInstanceName,
			PortEnvName:      "DB_PORT",
			HostEnvName:      "DB_HOST",
		}}),
	}

	// ensure that the new container does not exist
	if len(wl.Deployment.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Deployment.Spec.Template.Spec.Containers))
	}

	// Indicate that the workload needs an update
	u.MarkWorkloadNeedsUpdate(csqls[0], wl)

	// update the containers
	_, err := u.UpdateWorkloadContainers(wl, csqls)
	if err != nil {
		t.Fatal(err)
	}

	// ensure that the new container does not exist
	if len(wl.Deployment.Spec.Template.Spec.Containers) != 2 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Deployment.Spec.Template.Spec.Containers))
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

func TestWorkloadUnixVolume(t *testing.T) {
	var wantsInstanceName = "project:server:db"
	var wantsUnixDir = "/mnt/db"
	var wantContainerArgs = []string{
		fmt.Sprintf("%s?unix-socket=%s", wantsInstanceName, wantsUnixDir),
	}
	var wantWorkloadEnv = map[string]string{
		"DB_SOCKET_PATH": wantsUnixDir,
	}

	// Create a deployment
	wl := deploymentWorkload()
	wl.Deployment.Spec.Template.Spec.Containers[0].Ports =
		[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}

	// Create a AuthProxyWorkload that matches the deployment
	csqls := []*v1alpha1.AuthProxyWorkload{
		authProxyWorkload("instance1", []v1alpha1.InstanceSpec{{
			ConnectionString:      wantsInstanceName,
			UnixSocketPath:        wantsUnixDir,
			UnixSocketPathEnvName: "DB_SOCKET_PATH",
		}}),
	}

	// Indicate that the workload needs an update
	u.MarkWorkloadNeedsUpdate(csqls[0], wl)

	// update the containers
	_, err := u.UpdateWorkloadContainers(wl, csqls)
	if err != nil {
		t.Fatal(err)
	}

	// ensure that the new container exists
	if len(wl.Deployment.Spec.Template.Spec.Containers) != 2 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Deployment.Spec.Template.Spec.Containers))
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
	if want, got := 1, len(wl.Deployment.Spec.Template.Spec.Volumes); want != got {
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
	if want, got := wantsUnixDir, busyboxContainer.VolumeMounts[0].MountPath; want != got {
		t.Fatalf("got %v, wants %v. Busybox Container.VolumeMounts.MountPath", got, want)
	}
	if want, got := wl.Deployment.Spec.Template.Spec.Volumes[0].Name, busyboxContainer.VolumeMounts[0].Name; want != got {
		t.Fatalf("got %v, wants %v. Busybox Container.VolumeMounts.MountPath", got, want)
	}

}

func TestContainerImageChanged(t *testing.T) {
	var wantsInstanceName = "project:server:db"
	var wantImage = "custom-image:latest"

	// Create a deployment
	wl := deploymentWorkload()
	wl.Deployment.Spec.Template.Spec.Containers[0].Ports =
		[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}

	// Create a AuthProxyWorkload that matches the deployment
	csqls := []*v1alpha1.AuthProxyWorkload{
		simpleAuthProxy("instance1", wantsInstanceName),
	}
	csqls[0].Spec.AuthProxyContainer = &v1alpha1.AuthProxyContainerSpec{Image: wantImage}

	// Indicate that the workload needs an update
	u.MarkWorkloadNeedsUpdate(csqls[0], wl)

	// update the containers
	_, err := u.UpdateWorkloadContainers(wl, csqls)
	if err != nil {
		t.Fatal(err)
	}

	// ensure that the new container exists
	if len(wl.Deployment.Spec.Template.Spec.Containers) != 2 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Deployment.Spec.Template.Spec.Containers))
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

func TestContainerReplaced(t *testing.T) {
	var wantsInstanceName = "project:server:db"
	var wantContainer = &corev1.Container{
		Name: "sample", Image: "debian:latest", Command: []string{"/bin/bash"},
	}

	// Create a deployment
	wl := deploymentWorkload()
	wl.Deployment.Spec.Template.Spec.Containers[0].Ports =
		[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}

	// Create a AuthProxyWorkload that matches the deployment
	csqls := []*v1alpha1.AuthProxyWorkload{simpleAuthProxy("instance1", wantsInstanceName)}
	csqls[0].Spec.AuthProxyContainer = &v1alpha1.AuthProxyContainerSpec{Container: wantContainer}

	// Indicate that the workload needs an update
	u.MarkWorkloadNeedsUpdate(csqls[0], wl)

	// update the containers
	_, err := u.UpdateWorkloadContainers(wl, csqls)
	if err != nil {
		t.Fatal(err)
	}

	// ensure that the new container exists
	if len(wl.Deployment.Spec.Template.Spec.Containers) != 2 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Deployment.Spec.Template.Spec.Containers))
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

func ptr[T int | int32 | int64 | string](i T) *T {
	return &i
}

func TestProxyCLIArgs(t *testing.T) {
	type testParam struct {
		desc                 string
		proxySpec            v1alpha1.AuthProxyWorkloadSpec
		wantProxyArgContains []string
		wantErrorCodes       []string
	}
	wantTrue := true
	wantFalse := false

	var wantPort int32 = 5000

	var testcases = []testParam{
		{
			desc: "default cli config",
			proxySpec: v1alpha1.AuthProxyWorkloadSpec{
				Instances: []v1alpha1.InstanceSpec{{
					ConnectionString: "hello:world:db",
					Port:             &wantPort,
					PortEnvName:      "DB_PORT",
				}},
			},
			wantProxyArgContains: []string{
				"--structured-logs",
				"--health-check",
				fmt.Sprintf("--http-port=%d", workload.DefaultHealthCheckPort),
			},
		},
		{
			desc: "port explicitly set",
			proxySpec: v1alpha1.AuthProxyWorkloadSpec{
				Instances: []v1alpha1.InstanceSpec{{
					ConnectionString: "hello:world:db",
					Port:             &wantPort,
					PortEnvName:      "DB_PORT",
				}},
			},
			wantProxyArgContains: []string{"hello:world:db?port=5000"},
		},
		{
			desc: "fuse not supported error",
			proxySpec: v1alpha1.AuthProxyWorkloadSpec{
				Instances: []v1alpha1.InstanceSpec{{
					ConnectionString: "hello:world:db",
				}},
				AuthProxyContainer: &v1alpha1.AuthProxyContainerSpec{
					FUSEDir: "/fuse/db",
				},
			},
			wantProxyArgContains: []string{"hello:world:db?port=5000"},
			wantErrorCodes:       []string{v1alpha1.ErrorCodeFUSENotSupported},
		},
		{
			desc: "port implicitly set and increments",
			proxySpec: v1alpha1.AuthProxyWorkloadSpec{
				Instances: []v1alpha1.InstanceSpec{{
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
			proxySpec: v1alpha1.AuthProxyWorkloadSpec{
				Instances: []v1alpha1.InstanceSpec{{
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
			wantErrorCodes: []string{v1alpha1.ErrorCodeEnvConflict},
		},
		{
			desc: "auto-iam-authn set",
			proxySpec: v1alpha1.AuthProxyWorkloadSpec{
				Instances: []v1alpha1.InstanceSpec{{
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
			proxySpec: v1alpha1.AuthProxyWorkloadSpec{
				Instances: []v1alpha1.InstanceSpec{{
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
			desc: "telemetry flags",
			proxySpec: v1alpha1.AuthProxyWorkloadSpec{
				AuthProxyContainer: &v1alpha1.AuthProxyContainerSpec{
					SQLAdminAPIEndpoint: "https://example.com",
					Telemetry: &v1alpha1.TelemetrySpec{
						PrometheusNamespace: ptr("hello"),
						TelemetryPrefix:     ptr("telprefix"),
						TelemetryProject:    ptr("telproject"),
						TelemetrySampleRate: ptr(200),
						HTTPPort:            ptr(int32(9091)),
						DisableTraces:       &wantTrue,
						DisableMetrics:      &wantTrue,
						Prometheus:          &wantTrue,
						QuotaProject:        ptr("qp"),
					},
					MaxConnections:  ptr(int64(10)),
					MaxSigtermDelay: ptr(int64(20)),
				},
				Instances: []v1alpha1.InstanceSpec{{
					ConnectionString: "hello:world:one",
				}},
			},
			wantProxyArgContains: []string{
				fmt.Sprintf("hello:world:one?port=%d", workload.DefaultFirstPort),
				"--sqladmin-api-endpoint=https://example.com",
				"--telemetry-sample-rate=200",
				"--prometheus-namespace=hello",
				"--telemetry-project=telproject",
				"--telemetry-prefix=telprefix",
				"--http-port=9091",
				"--health-check",
				"--disable-traces",
				"--disable-metrics",
				"--prometheus",
				"--quota-project=qp",
				"--max-connections=10",
				"--max-sigterm-delay=20",
			},
		},
		{
			desc: "port conflict with other instance causes error",
			proxySpec: v1alpha1.AuthProxyWorkloadSpec{
				Instances: []v1alpha1.InstanceSpec{{
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
			wantErrorCodes: []string{v1alpha1.ErrorCodePortConflict},
		},
		{
			desc: "port conflict with workload container",
			proxySpec: v1alpha1.AuthProxyWorkloadSpec{
				Instances: []v1alpha1.InstanceSpec{{
					ConnectionString: "hello:world:one",
					PortEnvName:      "DB_PORT_1",
					Port:             ptr(int32(8080)),
				}},
			},
			wantProxyArgContains: []string{
				fmt.Sprintf("hello:world:one?port=%d", 8080)},
			wantErrorCodes: []string{v1alpha1.ErrorCodePortConflict},
		},
	}

	for i := 0; i < len(testcases); i++ {
		tc := &testcases[i]
		t.Run(tc.desc, func(t *testing.T) {

			// Create a deployment
			wl := &workload.DeploymentWorkload{Deployment: &appsv1.Deployment{
				TypeMeta:   metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "busybox", Labels: map[string]string{"app": "hello"}},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "busybox", Image: "busybox",
								Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}}},
						},
					},
				},
			}}

			// Create a AuthProxyWorkload that matches the deployment
			csqls := []*v1alpha1.AuthProxyWorkload{authProxyWorkloadFromSpec("instance1", tc.proxySpec)}

			// Indicate that the workload needs an update
			u.MarkWorkloadNeedsUpdate(csqls[0], wl)

			// update the containers
			_, updateErr := u.UpdateWorkloadContainers(wl, csqls)

			if len(tc.wantErrorCodes) > 0 {
				assertErrorCodeContains(t, updateErr, tc.wantErrorCodes)
				return
			}

			// ensure that the new container exists
			if len(wl.Deployment.Spec.Template.Spec.Containers) != 2 {
				t.Fatalf("got %v, wants 2. deployment containers length", len(wl.Deployment.Spec.Template.Spec.Containers))
			}

			// test that the instancename matches the new expected instance name.
			csqlContainer, err := findContainer(wl, fmt.Sprintf("csql-default-%s", csqls[0].GetName()))
			if err != nil {
				t.Fatal(err)
			}

			// test that port cli args are set correctly
			assertContainerArgsContains(t, csqlContainer.Args, tc.wantProxyArgContains)

		})
	}

}

func TestProperCleanupOfEnvAndVolumes(t *testing.T) {
	var (
		wantsInstanceName  = "project:server:db"
		wantsUnixDir       = "/mnt/db"
		wantsInstanceName2 = "project:server:db2"
		wantsPort          = int32(5000)
		wantContainerArgs  = []string{
			fmt.Sprintf("%s?unix-socket=%s", wantsInstanceName, wantsUnixDir),
			fmt.Sprintf("%s?port=%d", wantsInstanceName2, wantsPort),
		}
		wantWorkloadEnv = map[string]string{
			"DB_SOCKET_PATH": wantsUnixDir,
			"DB_PORT":        "5000",
		}
	)

	// Create a deployment
	wl := deploymentWorkload()
	wl.Deployment.Spec.Template.Spec.Containers[0].Ports =
		[]corev1.ContainerPort{{Name: "http", ContainerPort: 8080}}
	wl.Deployment.Spec.Template.Spec.Containers[0].Env =
		[]corev1.EnvVar{{Name: "DB_PORT", Value: "not set"}}

	wl.Deployment.Spec.Template.Spec.Containers[0].VolumeMounts =
		[]corev1.VolumeMount{{Name: "TempVol", MountPath: "/tmp/testvol"}}
	wl.Deployment.Spec.Template.Spec.Volumes =
		[]corev1.Volume{{Name: "TempVol", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}

	// Create a AuthProxyWorkload that matches the deployment
	csqls := []*v1alpha1.AuthProxyWorkload{{
		TypeMeta:   metav1.TypeMeta{Kind: "AuthProxyWorkload", APIVersion: v1alpha1.GroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Name: "instance1", Namespace: "default", Generation: 1},
		Spec: v1alpha1.AuthProxyWorkloadSpec{
			Workload: v1alpha1.WorkloadSelectorSpec{
				Kind: "Deployment",
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "hello"},
				},
			},
			Instances: []v1alpha1.InstanceSpec{{
				ConnectionString:      wantsInstanceName,
				UnixSocketPath:        wantsUnixDir,
				UnixSocketPathEnvName: "DB_SOCKET_PATH",
			},
				{
					ConnectionString: wantsInstanceName2,
					PortEnvName:      "DB_PORT",
				}},
		},
		Status: v1alpha1.AuthProxyWorkloadStatus{},
	}}

	// Indicate that the workload needs an update
	u.MarkWorkloadNeedsUpdate(csqls[0], wl)

	// update the containers
	_, err := u.UpdateWorkloadContainers(wl, csqls)
	if err != nil {
		t.Fatal(err)
		return
	}
	// do it again to make sure its idempotent
	csqls[0].SetGeneration(csqls[0].GetGeneration() + 1)
	u.MarkWorkloadNeedsUpdate(csqls[0], wl)
	_, err = u.UpdateWorkloadContainers(wl, csqls)
	if err != nil {
		t.Fatal(err)
		return
	}
	// ensure that the new container exists
	if len(wl.Deployment.Spec.Template.Spec.Containers) != 2 {
		t.Fatalf("got %v, wants 1. deployment containers length", len(wl.Deployment.Spec.Template.Spec.Containers))
	}

	// test that the instancename matches the new expected instance name.
	csqlContainer, err := findContainer(wl, fmt.Sprintf("csql-%s-%s", csqls[0].GetNamespace(), csqls[0].GetName()))
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
	if want, got := 2, len(wl.Deployment.Spec.Template.Spec.Volumes); want != got {
		t.Fatalf("got %v, wants %v. PodSpec.Volumes", got, want)
	}

	// test that Volume mount exists on busybox
	busyboxContainer, err := findContainer(wl, "busybox")
	if err != nil {
		t.Fatal(err)
	}
	if want, got := 2, len(busyboxContainer.VolumeMounts); want != got {
		t.Fatalf("got %v, wants %v. Busybox Container.VolumeMounts", got, want)
	}
	if want, got := wantsUnixDir, busyboxContainer.VolumeMounts[1].MountPath; want != got {
		t.Fatalf("got %v, wants %v. Busybox Container.VolumeMounts.MountPath", got, want)
	}
	if want, got := wl.Deployment.Spec.Template.Spec.Volumes[1].Name, busyboxContainer.VolumeMounts[1].Name; want != got {
		t.Fatalf("got %v, wants %v. Busybox Container.VolumeMounts.MountPath", got, want)
	}

	// Update again with an empty list
	_, err = u.UpdateWorkloadContainers(wl, []*v1alpha1.AuthProxyWorkload{})
	if err != nil {
		t.Fatal(err)
	}

	// Test that the workload was properly cleaned up
	busyboxContainer, _ = findContainer(wl, "busybox")

	// Test that added workload vars were removed
	_, err = findEnvVar(wl, "busybox", "DB_SOCKET_PATH")
	if err == nil {
		t.Errorf("got EnvVar named %v, wants no EnvVar", "DB_SOCKET_PATH")
	}

	// Test that replaced workload vars were restored
	val, err := findEnvVar(wl, "busybox", "DB_PORT")
	if err != nil {
		t.Errorf("got missing EnvVar named %v, wants value for EnvVar", "DB_PORT")
	}
	if val.Value != "not set" {
		t.Errorf("got EnvVar value %v=%v, wants %v", "DB_PORT", val.Value, "not set")
	}

	// Test that the VolumeMounts were removed
	if want, got := 1, len(busyboxContainer.VolumeMounts); want != got {
		t.Errorf("wants %d VolumeMounts, got %d", want, got)
	}

	// Test that the Volumes were removed
	if want, got := 1, len(wl.Deployment.Spec.Template.Spec.Volumes); want != got {
		t.Errorf("wants %d Volumes, got %d", want, got)
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
