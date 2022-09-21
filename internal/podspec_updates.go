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
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var l = logf.Log.WithName("internal.workload")

const DefaultContainerImage = "us-central1-docker.pkg.dev/csql-operator-test/test76e6d646e2caac1c458c/proxy-v2:latest" //TODO get the right name here.

// Constants for well known error codes and defaults. These are exposed on the
// package and documented here so that they appear in the godoc. These also
// need to be documented in the CRD
const (
	// ErrorCodePortConflict occurs when an explicit port assignment for a workload
	// is in conflict with a port assignment from the pod or another proxy container.
	ErrorCodePortConflict = "PortConflict"

	// ErrorCodeEnvConflict occurs when an the environment code does not work.
	ErrorCodeEnvConflict = "EnvVarConflict"

	// ErrorCodeEnvConflict occurs when any FUSE configuration is set, because fuse is not yet supported.
	ErrorCodeFUSENotSupported = "FUSENotSupported"

	// DefaultFirstPort is the first port number chose for an instance listener by the
	// proxy.
	DefaultFirstPort int32 = 5000

	// DefaultHealthCheckPort is the used by the proxy to expose prometheus
	// and kubernetes health checks.
	DefaultHealthCheckPort int32 = 9801
)

// ConfigError is an error with extra details about why an AuthProxyWorkload
// cannot be configured.
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

// ConfigErrorDetail is an error that contains details about specific kinds of errors that caused
// a AuthProxyWorkload to fail when being configured on a workload.
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

// defaultContainerResources used when the AuthProxyWorkload resource is not specified.
var defaultContainerResources = corev1.ResourceRequirements{
	Requests: corev1.ResourceList{
		"cpu":    resource.MustParse("1.0"),
		"memory": resource.MustParse("1Gi"),
	},
}

// ReconcileWorkload finds all AuthProxyWorkload resources matching this workload and then
// updates the workload's containers. This does not save the updated workload.
func ReconcileWorkload(instList cloudsqlapi.AuthProxyWorkloadList, workload Workload) (bool, []*cloudsqlapi.AuthProxyWorkload, *ConfigError) {
	// if a workload has an owner, then ignore it.
	if len(workload.Object().GetOwnerReferences()) > 0 {
		return false, []*cloudsqlapi.AuthProxyWorkload{}, nil
	}

	matchingAuthProxyWorkloads := filterMatchingInstances(instList, workload)
	updated, err := UpdateWorkloadContainers(workload, matchingAuthProxyWorkloads)
	if updated {
		return true, matchingAuthProxyWorkloads, err
	} else {
		return false, []*cloudsqlapi.AuthProxyWorkload{}, nil
	}

}

// filterMatchingInstances returns a list of AuthProxyWorkload whose selectors match
// the workload.
func filterMatchingInstances(wlList cloudsqlapi.AuthProxyWorkloadList, workload Workload) []*cloudsqlapi.AuthProxyWorkload {
	matchingAuthProxyWorkloads := make([]*cloudsqlapi.AuthProxyWorkload, 0, len(wlList.Items))
	for i, _ := range wlList.Items {
		csqlWorkload := &wlList.Items[i]
		if workloadMatches(workload, csqlWorkload.Spec.Workload, csqlWorkload.Namespace) {
			// need to update workload
			l.Info("Found matching workload", "workload", workload.Object().GetNamespace()+"/"+workload.Object().GetName(), "wlSelector", csqlWorkload.Spec.Workload, "AuthProxyWorkload", csqlWorkload.Namespace+"/"+csqlWorkload.Name)
			matchingAuthProxyWorkloads = append(matchingAuthProxyWorkloads, csqlWorkload)
		}
	}
	return matchingAuthProxyWorkloads
}

// WorkloadUpdateStatus describes when a workload was last updated
type WorkloadUpdateStatus struct {
	InstanceGeneration    string
	LastRequstGeneration  string
	RequestGeneration     string
	LastUpdatedGeneration string
	UpdatedGeneration     string
}

// MarkWorkloadNeedsUpdate Updates annotations on the workload indicating that it may need an update.
// returns true if the workload actually needs an update.
func MarkWorkloadNeedsUpdate(csqlWorkload *cloudsqlapi.AuthProxyWorkload, workload Workload) (bool, WorkloadUpdateStatus) {
	return updateWorkloadAnnotations(csqlWorkload, workload, false)
}

// MarkWorkloadUpdated Updates annotations on the workload indicating that it
// has been updated, returns true of any modifications were made to the workload.
// for the AuthProxyWorkload.
func MarkWorkloadUpdated(csqlWorkload *cloudsqlapi.AuthProxyWorkload, workload Workload) (bool, WorkloadUpdateStatus) {
	return updateWorkloadAnnotations(csqlWorkload, workload, true)
}

// updateWorkloadAnnotations adds annotations to the workload
// to track which generation of a AuthProxyWorkload needs to be applied, and which
// generation has been applied. The AuthProxyWorkload controller is responsible for
// tracking which version should be applied, The workload admission webhook is
// responsible for applying the AuthProxyWorkloads that apply to a workload
// when the workload is created or modified.
func updateWorkloadAnnotations(csqlWorkload *cloudsqlapi.AuthProxyWorkload, workload Workload, doingUpdate bool) (bool, WorkloadUpdateStatus) {
	s := WorkloadUpdateStatus{}
	doUpdate := false
	reqName := names.SafePrefixedName("csqlr-", csqlWorkload.Name)
	resultName := names.SafePrefixedName("csqlu-", csqlWorkload.Name)
	s.InstanceGeneration = fmt.Sprintf("%d", csqlWorkload.GetGeneration())

	ann := workload.Object().GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	s.LastRequstGeneration = ann[reqName]
	s.LastUpdatedGeneration = ann[resultName]

	if s.LastRequstGeneration != s.InstanceGeneration {
		ann[reqName] = s.InstanceGeneration
		if doingUpdate {
			ann[resultName] = s.InstanceGeneration
		}
		doUpdate = true
	}
	if s.LastUpdatedGeneration != s.InstanceGeneration {
		if doingUpdate {
			ann[resultName] = s.InstanceGeneration
		}
		doUpdate = true
	}
	workload.Object().SetAnnotations(ann)
	s.RequestGeneration = ann[reqName]
	s.UpdatedGeneration = ann[resultName]

	return doUpdate, s
}

// UpdateWorkloadContainers applies the proxy containers from all of the
// instances listed in matchingAuthProxyWorkloads to the workload
func UpdateWorkloadContainers(workload Workload, matchingAuthProxyWorkloads []*cloudsqlapi.AuthProxyWorkload) (bool, *ConfigError) {
	state := updateState{
		nextDbPort: DefaultFirstPort,
		err: ConfigError{
			workloadKind:      workload.Object().GetObjectKind().GroupVersionKind(),
			workloadName:      workload.Object().GetName(),
			workloadNamespace: workload.Object().GetNamespace(),
		},
	}
	return state.update(workload, matchingAuthProxyWorkloads)
}

// updateState holds internal state while a particular workload being configured
// with one or more AuthProxyWorkloads.
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

func (state *updateState) addInUsePort(port int32) {
	state.portsInUse = append(state.portsInUse, port)
}

// isPortInUse checks if the port is in use.
func (state *updateState) isPortInUse(port int32) bool {
	for i := 0; i < len(state.portsInUse); i++ {
		if port == state.portsInUse[i] {
			return true
		}
	}
	return false
}

// useNextDbPort consumes the next available db port, marking that port as "in-use."
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

// update Reconciles the state of a workload, applying the matching AuthProxyWorkloads
// and removing any out-of-date configuration related to deleted AuthProxyWorkloads
func (state *updateState) update(workload Workload, matchingAuthProxyWorkloads []*cloudsqlapi.AuthProxyWorkload) (bool, *ConfigError) {

	podSpec := workload.PodSpec()
	containers := podSpec.Containers
	updated := false

	var nonAuthProxyContainers []corev1.Container
	for i := 0; i < len(containers); i++ {
		if strings.Index(containers[i].Name, names.ContainerPrefix) != 0 {
			nonAuthProxyContainers = append(nonAuthProxyContainers, containers[i])
		}
	}

	for i := 0; i < len(nonAuthProxyContainers); i++ {
		c := nonAuthProxyContainers[i]
		for j := 0; j < len(c.Ports); j++ {
			state.portsInUse = append(state.portsInUse, c.Ports[j].ContainerPort)
		}
	}

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

	var removedContainers []corev1.Container

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
				removedContainers = append(removedContainers, *container)
			}
		} else {
			filteredContainers = append(filteredContainers, *container)
		}
	}
	podSpec.Containers = filteredContainers

	//TODO remove all EnvVar, Volume, and VolumeMount related to deleted
	// AuthProxyWorkload resources.

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

		if proxy.Spec.AuthProxyContainer != nil && proxy.Spec.AuthProxyContainer.Container != nil {
			proxy.Spec.AuthProxyContainer.Container.DeepCopyInto(container)
			container.Name = names.ContainerName(proxy)
		} else {
			container.Name = names.ContainerName(proxy)
			container.ImagePullPolicy = "IfNotPresent"

			if proxy.Spec.AuthProxyContainer != nil && proxy.Spec.AuthProxyContainer.FUSEDir != "" {
				state.addError(ErrorCodeFUSENotSupported, "the FUSE filesystem is not yet supported", proxy)

				//TODO fuse...
				// if FUSE is used, we need to use the 'buster' or 'alpine' image.
				// for the proxy container. We can't use the default distroless image.

			}
			if proxy.Spec.AuthProxyContainer != nil && proxy.Spec.AuthProxyContainer.FUSETempDir != "" {
				state.addError(ErrorCodeFUSENotSupported, "the FUSE filesystem is not yet supported", proxy)

				//TODO fuse...
				// if FUSE is used, we need to use the 'buster' or 'alpine' image.
				// for the proxy container. We can't use the default distroless image.

			}

			if proxy.Spec.AuthProxyContainer != nil && proxy.Spec.AuthProxyContainer.Image != "" {
				container.Image = proxy.Spec.AuthProxyContainer.Image
			} else {
				container.Image = DefaultContainerImage
			}

			if proxy.Spec.AuthProxyContainer != nil && proxy.Spec.AuthProxyContainer.Resources != nil {
				container.Resources = *proxy.Spec.AuthProxyContainer.Resources.DeepCopy()
			} else {
				container.Resources = defaultContainerResources
			}

			// always enable http port healthchecks
			cliArgs = append(cliArgs, fmt.Sprintf("--http-port=%d", state.addHealthCheck(proxy)))
			cliArgs = append(cliArgs, "--health-check")
			cliArgs = append(cliArgs, "--structured-logs")

			if proxy.Spec.AuthProxyContainer != nil && proxy.Spec.AuthProxyContainer.SQLAdminAPIEndpoint != "" {
				cliArgs = append(cliArgs, "--sqladmin-api-endpoint="+proxy.Spec.AuthProxyContainer.SQLAdminAPIEndpoint)
			}
			if proxy.Spec.AuthProxyContainer != nil && proxy.Spec.AuthProxyContainer.Telemetry != nil {
				tel := proxy.Spec.AuthProxyContainer.Telemetry
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

				if inst.SocketType == "tcp" ||
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

				if inst.SocketType == "unix" ||
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

				if inst.AutoIAMAuthN != nil {
					if *inst.AutoIAMAuthN {
						params["auto-iam-authn"] = "true"
					} else {
						params["auto-iam-authn"] = "false"
					}
				}
				if inst.PrivateIP != nil {
					if *inst.PrivateIP {
						params["private-ip"] = "true"
					} else {
						params["private-ip"] = "false"
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
	l.Info("Updating workload ", "name", workload.Object().GetName(),
		"doUpdate", doUpdate,
		"status", status)

	return doUpdate
}

// updateContainerEnv applies global container state to all containers
func (state *updateState) updateContainerEnv(c *corev1.Container) {
	for i := 0; i < len(state.workloadEnvVars); i++ {
		var found bool
		for j := 0; j < len(c.Env); j++ {
			if state.workloadEnvVars[i].Name == c.Env[j].Name {
				found = true
				c.Env[j] = state.workloadEnvVars[i]
			}
		}
		if !found {
			c.Env = append(c.Env, state.workloadEnvVars[i])
		}
	}
}

// applyContainerVolumes applies global container state to all containers
func (state *updateState) applyContainerVolumes(c *corev1.Container) {
	for i := 0; i < len(state.volumeMounts); i++ {
		var found bool
		for j := 0; j < len(c.VolumeMounts); j++ {
			if state.workloadEnvVars[i].Name == c.VolumeMounts[j].Name {
				found = true
				c.VolumeMounts[j] = state.volumeMounts[i]
			}
		}
		if !found {
			c.VolumeMounts = append(c.VolumeMounts, state.volumeMounts[i])
		}
	}
}

// applyVolumes applies global container state to all containers
func (state *updateState) applyVolumes(spec *corev1.PodSpec) {
	for i := 0; i < len(state.volumeMounts); i++ {
		var found bool
		for j := 0; j < len(spec.Volumes); j++ {
			if state.workloadEnvVars[i].Name == spec.Volumes[j].Name {
				found = true
				spec.Volumes[j] = state.volumes[i]
			}
		}
		if !found {
			spec.Volumes = append(spec.Volumes, state.volumes[i])
		}
	}
}

func (state *updateState) applyCommonContainerConfig(c *corev1.Container) {
	state.updateContainerEnv(c)
	state.applyContainerVolumes(c)
}

func (state *updateState) addHealthCheck(csqlWorkload *cloudsqlapi.AuthProxyWorkload) int32 {
	var port int32
	if csqlWorkload.Spec.AuthProxyContainer != nil &&
		csqlWorkload.Spec.AuthProxyContainer.Telemetry != nil &&
		csqlWorkload.Spec.AuthProxyContainer.Telemetry.HTTPPort != nil {
		port = *csqlWorkload.Spec.AuthProxyContainer.Telemetry.HTTPPort
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
