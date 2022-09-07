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

package internal

import (
	"fmt"
	"sort"
	"strings"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/api/v1alpha1"
	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/names"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ErrorCodePortConflict occurs when an explicit port assignment for a workload
// is in conflict with a port assignment from the pod or another workload.
const ErrorCodePortConflict = "PortConflict"
const ErrorCodeEnvConflict = "EnvVarConflict"
const ErrorCodeFUSENotSupported = "FUSENotSupported"
const DefaultFirstPort int32 = 5000
const DefaultHealthCheckPort int32 = 9801

type ConfigError struct {
	workloadKind      schema.GroupVersionKind
	workloadName      string
	workloadNamespace string

	details []ConfigErrorDetail
}

func (c *ConfigError) DetailedErrors() []ConfigErrorDetail {
	return c.details
}

func (err *ConfigError) Error() string {
	return fmt.Sprintf("found %d configuration errors on workload %s %s/%s: %v",
		len(err.details),
		err.workloadKind.String(),
		err.workloadNamespace,
		err.workloadName,
		err.details)
}

func (err *ConfigError) add(errorCode, description string, proxy *cloudsqlapi.AuthProxyWorkload) {
	err.details = append(err.details,
		ConfigErrorDetail{
			WorkloadKind:       err.workloadKind,
			WorkloadName:       err.workloadName,
			WorkloadNamespace:  err.workloadNamespace,
			AuthProxyNamespace: proxy.GetNamespace(),
			AuthProxyName:      proxy.GetName(),
			ErrorCode:          errorCode,
			Description:        description,
		})
}

type ConfigErrorDetail struct {
	ErrorCode          string
	Description        string
	AuthProxyName      string
	AuthProxyNamespace string

	WorkloadKind      schema.GroupVersionKind
	WorkloadName      string
	WorkloadNamespace string
}

func (err *ConfigErrorDetail) Error() string {
	return fmt.Sprintf("error %s %s while applying AuthProxyWorkload %s/%s to workload  %s %s/%s",
		err.ErrorCode,
		err.Description,
		err.AuthProxyNamespace,
		err.AuthProxyName,
		err.WorkloadKind.String(),
		err.WorkloadNamespace,
		err.WorkloadName)

}

type updateState struct {
	err             ConfigError
	portsInUse      []int32
	workloadEnvVars []corev1.EnvVar
	nextDbPort      int32
	volumeMounts    []corev1.VolumeMount
	volumes         []corev1.Volume
}

func (state *updateState) addVolumeMount(mnt corev1.VolumeMount, vol corev1.Volume) {
	state.volumeMounts = append(state.volumeMounts, mnt)
	state.volumes = append(state.volumes, vol)
}

// addInUsePort Adds a new port to the list of ports in use
func (state *updateState) addInUsePort(port int32) {
	state.portsInUse = append(state.portsInUse, port)
}

// isPortInUse checks if the port is in use
func (state *updateState) isPortInUse(port int32) bool {
	for i := 0; i < len(state.portsInUse); i++ {
		if port == state.portsInUse[i] {
			return true
		}
	}
	return false
}

func (state *updateState) useNextDbPort() int32 {
	for state.isPortInUse(state.nextDbPort) {
		state.nextDbPort++
	}
	state.addInUsePort(state.nextDbPort)
	return state.nextDbPort
}

// addWorkloadEnvVar adds or replaces the envVar based on its Name, returning the old and new values
func (state *updateState) addWorkloadEnvVar(envVar corev1.EnvVar, proxy *cloudsqlapi.AuthProxyWorkload) {
	for i := 0; i < len(state.workloadEnvVars); i++ {
		if state.workloadEnvVars[i].Name == envVar.Name {
			old := state.workloadEnvVars[i]
			state.workloadEnvVars[i] = envVar
			if old.Value != envVar.Value {
				state.addError(ErrorCodeEnvConflict,
					fmt.Sprintf("environment variable named %s already exists", envVar.Name), proxy)
			}
			return
		}
	}
	state.workloadEnvVars = append(state.workloadEnvVars, envVar)
	return
}

// UpdateWorkloadContainers Applies the proxy containers from all of the instances listed in matchingAuthProxyWorkloads to the workload
func UpdateWorkloadContainers(workload Workload, matchingAuthProxyWorkloads []*cloudsqlapi.AuthProxyWorkload) (bool, *ConfigError) {
	state := updateState{
		nextDbPort: DefaultFirstPort,
		err: ConfigError{
			workloadKind:      workload.GetObject().GetObjectKind().GroupVersionKind(),
			workloadName:      workload.GetObject().GetName(),
			workloadNamespace: workload.GetObject().GetNamespace(),
		},
	}
	return state.update(workload, matchingAuthProxyWorkloads)
}

func (state *updateState) update(workload Workload, matchingAuthProxyWorkloads []*cloudsqlapi.AuthProxyWorkload) (bool, *ConfigError) {

	podSpec := workload.GetPodSpec()
	containers := podSpec.Containers
	updated := false

	//TODO maybe replace with regular for loop style go
	nonProxyContainers := fnFilter[corev1.Container](containers,
		func(t corev1.Container) bool {
			return strings.Index(t.Name, names.ContainerPrefix) != 0
		})

	state.portsInUse = fnFlatMap[corev1.Container, int32](
		nonProxyContainers,
		func(t corev1.Container) []int32 {
			return fnMap[corev1.ContainerPort, int32](t.Ports, func(t corev1.ContainerPort) int32 {
				return t.ContainerPort
			})
		})

	// add all new containers and update existing containers
	for i, _ := range matchingAuthProxyWorkloads {
		inst := matchingAuthProxyWorkloads[i]

		var instContainer *corev1.Container

		for j, _ := range containers {
			container := &containers[j]
			if container.Name == names.ContainerName(inst) {
				instContainer = container
				break
			}
		}
		if instContainer == nil {
			newContainer := corev1.Container{}
			state.UpdateContainer(inst, workload, &newContainer)
			containers = append(containers, newContainer)
			updated = true
		} else {
			updated = state.UpdateContainer(inst, workload, instContainer)
		}
	}

	// remove all csql containers that don't relate to one of the matchingAuthProxyWorkloads
	var filteredContainers []corev1.Container

	for j, _ := range containers {
		container := &containers[j]
		if strings.HasPrefix(container.Name, names.ContainerPrefix) {
			found := false
			for i, _ := range matchingAuthProxyWorkloads {
				if names.ContainerName(matchingAuthProxyWorkloads[i]) == container.Name {
					found = true
					break
				}
			}
			if found {
				filteredContainers = append(filteredContainers, *container)
			} else {
				// we're removing a container that doesn't match an csqlWorkload
				updated = true
			}
		} else {
			filteredContainers = append(filteredContainers, *container)
		}
	}
	podSpec.Containers = filteredContainers

	for i, _ := range podSpec.Containers {
		state.applyCommonContainerConfig(&podSpec.Containers[i])
	}
	state.applyVolumes(&podSpec)

	workload.SetPodSpec(podSpec)

	// only return ConfigError if there were reported
	// errors during processing.
	var err *ConfigError
	if len(state.err.details) > 0 {
		err = &state.err
	} else {
		err = nil
	}
	return updated, err
}

// UpdateContainer Creates or updates the proxy container in the workload's PodSpec
func (state *updateState) UpdateContainer(proxy *cloudsqlapi.AuthProxyWorkload, workload Workload, container *corev1.Container) bool {
	doUpdate, status := MarkWorkloadUpdated(proxy, workload)

	if doUpdate {
		var cliArgs []string

		if proxy.Spec.ProxyContainer != nil && proxy.Spec.ProxyContainer.Container != nil {
			proxy.Spec.ProxyContainer.Container.DeepCopyInto(container)
			container.Name = names.ContainerName(proxy)
		} else {
			container.Name = names.ContainerName(proxy)
			container.ImagePullPolicy = "IfNotPresent"

			if proxy.Spec.ProxyContainer != nil && proxy.Spec.ProxyContainer.Image != "" {
				container.Image = proxy.Spec.ProxyContainer.Image
			} else {
				container.Image = DefaultContainerImage
			}

			if proxy.Spec.ProxyContainer != nil && proxy.Spec.ProxyContainer.Resources != nil {
				container.Resources = *proxy.Spec.ProxyContainer.Resources.DeepCopy()
			} else {
				container.Resources = defaultContainerResources
			}

			// always enable http port healthchecks
			cliArgs = append(cliArgs, fmt.Sprintf("--http-port=%d", state.addHealthCheck(proxy)))
			cliArgs = append(cliArgs, "--health-check")
			cliArgs = append(cliArgs, "--structured-logs")

			if proxy.Spec.ProxyContainer != nil && proxy.Spec.ProxyContainer.SQLAdminApiEndpoint != "" {
				cliArgs = append(cliArgs, "--sqladmin-api-endpoint="+proxy.Spec.ProxyContainer.SQLAdminApiEndpoint)
			}
			if proxy.Spec.ProxyContainer != nil && proxy.Spec.ProxyContainer.Telemetry != nil {
				tel := proxy.Spec.ProxyContainer.Telemetry
				if tel != nil {
					if tel.TelemetrySampleRate != nil {
						cliArgs = append(cliArgs, fmt.Sprintf("--telemetry-sample-rate=%d", *tel.TelemetrySampleRate))
					}
					if tel.DisableTraces != nil && *tel.DisableTraces {
						cliArgs = append(cliArgs, "--disable-traces")
					}
					if tel.DisableMetrics != nil && *tel.DisableMetrics {
						cliArgs = append(cliArgs, "--disable-metrics")
					}
					if tel.PrometheusNamespace != nil || (tel.Prometheus != nil && *tel.Prometheus) {
						cliArgs = append(cliArgs, "--prometheus")
					}
					if tel.PrometheusNamespace != nil {
						cliArgs = append(cliArgs, fmt.Sprintf("--prometheus-namespace=%s", *tel.PrometheusNamespace))
					}
					if tel.TelemetryProject != nil {
						cliArgs = append(cliArgs, fmt.Sprintf("--telemetry-project=%s", *tel.TelemetryProject))
					}
					if tel.TelemetryPrefix != nil {
						cliArgs = append(cliArgs, fmt.Sprintf("--telemetry-prefix=%s", *tel.TelemetryPrefix))
					}
				}
			}

			//TODO Authorization
			// --credentials-file

			for _, inst := range proxy.Spec.Instances {

				params := map[string]string{}

				if inst.SocketType == cloudsqlapi.SocketTypeTCP ||
					(inst.SocketType == "" && inst.UnixSocketPath == "") {
					var port int32
					if inst.Port == nil {
						port = state.useNextDbPort()
					} else {
						port = *inst.Port
						if state.isPortInUse(port) {
							state.addError(ErrorCodePortConflict,
								fmt.Sprintf("proxy port %d for instance %s is already in use",
									port, inst.ConnectionString), proxy)
						}
						state.addInUsePort(port)
					}
					params["port"] = fmt.Sprint(port)
					if inst.HostEnvName != "" {
						state.addWorkloadEnvVar(corev1.EnvVar{
							Name:  inst.HostEnvName,
							Value: "localhost",
						}, proxy)
					}
					if inst.PortEnvName != "" {
						state.addWorkloadEnvVar(corev1.EnvVar{
							Name:  inst.PortEnvName,
							Value: fmt.Sprint(port),
						}, proxy)
					}
				}

				if inst.SocketType == cloudsqlapi.SocketTypeUnix ||
					(inst.SocketType == "" && inst.UnixSocketPath != "") {
					params["unix-socket"] = inst.UnixSocketPath
					mountName := names.VolumeName(proxy, &inst, "unix")
					state.addVolumeMount(
						corev1.VolumeMount{
							Name:      mountName,
							ReadOnly:  false,
							MountPath: inst.UnixSocketPath,
						},
						corev1.Volume{
							Name:         mountName,
							VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
						})

					if inst.UnixSocketPathEnvName != "" {
						state.addWorkloadEnvVar(corev1.EnvVar{
							Name:  inst.UnixSocketPathEnvName,
							Value: inst.UnixSocketPath,
						}, proxy)
					}
				}

				if inst.AutoIamAuthn != nil {
					if *inst.AutoIamAuthn {
						params["auto-iam-authn"] = "true"
					} else {
						params["auto-iam-authn"] = "false"
					}
				}
				if inst.PrivateIp != nil {
					if *inst.PrivateIp {
						params["private-ip"] = "true"
					} else {
						params["private-ip"] = "false"
					}
				}
				if inst.FusePath != "" {
					state.addError(ErrorCodeFUSENotSupported, "the FUSE filesystem is not yet supported", proxy)

					//TODO fuse...
					// if FUSE is used, we need to use the 'buster' or 'alpine' image.
					// for the proxy container. We can't use the default distroless image.
					if inst.FuseVolumePathEnvName != "" {
						state.addWorkloadEnvVar(corev1.EnvVar{
							Name:  inst.FuseVolumePathEnvName,
							Value: inst.FusePath,
						}, proxy)
					}

				}

				instArgs := []string{}
				for k, v := range params {
					instArgs = append(instArgs, fmt.Sprintf("%s=%s", k, v))
				}

				// sort the param args to make testing easier. params will always be
				// in a stable order
				sort.Strings(instArgs)

				if len(instArgs) > 0 {
					cliArgs = append(cliArgs, fmt.Sprintf("%s?%s", inst.ConnectionString, strings.Join(instArgs, "&")))
				} else {
					cliArgs = append(cliArgs, inst.ConnectionString)
				}

			}
			container.Args = cliArgs
		}
	}
	l.Info("Updating workload ", "name", workload.GetObject().GetName(),
		"doUpdate", doUpdate,
		"status", status)

	return doUpdate
}

// updateContainerEnv applies global container state to all containers
func (state *updateState) updateContainerEnv(c *corev1.Container) {
	for i := 0; i < len(state.workloadEnvVars); i++ {
		c.Env = appendOrReplace(c.Env, state.workloadEnvVars[i],
			func(a, b *corev1.EnvVar) bool { return a.Name == b.Name })
	}
}

// applyContainerVolumes applies global container state to all containers
func (state *updateState) applyContainerVolumes(c *corev1.Container) {
	for i := 0; i < len(state.volumeMounts); i++ {
		c.VolumeMounts = appendOrReplace(c.VolumeMounts, state.volumeMounts[i],
			func(a, b *corev1.VolumeMount) bool { return a.Name == b.Name })
	}
}

// applyVolumes applies global container state to all containers
func (state *updateState) applyVolumes(spec *corev1.PodSpec) {
	for i := 0; i < len(state.volumeMounts); i++ {
		spec.Volumes = appendOrReplace(spec.Volumes, state.volumes[i],
			func(a, b *corev1.Volume) bool { return a.Name == b.Name })
	}
}

func (state *updateState) applyCommonContainerConfig(c *corev1.Container) {
	state.updateContainerEnv(c)
	state.applyContainerVolumes(c)
}

func (state *updateState) addHealthCheck(csqlWorkload *cloudsqlapi.AuthProxyWorkload) int32 {
	var port int32
	if csqlWorkload.Spec.ProxyContainer != nil &&
		csqlWorkload.Spec.ProxyContainer.Telemetry != nil &&
		csqlWorkload.Spec.ProxyContainer.Telemetry.HttpPort != nil {
		port = *csqlWorkload.Spec.ProxyContainer.Telemetry.HttpPort
		if state.isPortInUse(port) {
			state.addError(ErrorCodePortConflict,
				fmt.Sprintf("telemetry httpPort %d is already in use", port), csqlWorkload)
		}
	} else {
		port = DefaultHealthCheckPort
		for state.isPortInUse(port) {
			port++
		}
		state.addInUsePort(port)
	}
	//TODO add healthcheck to podspec

	return port
}

func (state *updateState) addError(errorCode string, description string, proxy *cloudsqlapi.AuthProxyWorkload) {
	state.err.add(errorCode, description, proxy)
}

func appendOrReplace[T any](list []T, value T, isEqual func(*T, *T) bool) []T {
	for i := 0; i < len(list); i++ {
		if isEqual(&list[i], &value) {
			list[i] = value
			return list
		}
	}
	list = append(list, value)
	return list
}

func fnFilter[T any](list []T, test func(T) bool) []T {
	result := make([]T, 0, len(list))
	for i := 0; i < len(list); i++ {
		if test(list[i]) {
			result = append(result, list[i])
		}
	}
	return result
}

func fnMap[T any, V any](list []T, apply func(T) V) []V {
	result := make([]V, len(list))
	for i := 0; i < len(list); i++ {
		result[i] = apply(list[i])
	}
	return result
}
func fnFlatMap[T any, V any](list []T, apply func(T) []V) []V {
	result := make([]V, 0, len(list))
	for i := 0; i < len(list); i++ {
		oneResult := apply(list[i])
		for j := 0; j < len(oneResult); j++ {
			result = append(result, oneResult[j])
		}
	}
	return result
}
