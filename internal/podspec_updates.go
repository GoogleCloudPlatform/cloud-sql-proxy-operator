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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var l = logf.Log.WithName("internal.workload")

// Constants for well known error codes and defaults. These are exposed on the
// package and documented here so that they appear in the godoc. These also
// need to be documented in the CRD
const (

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

func (e *ConfigError) DetailedErrors() []ConfigErrorDetail {
	return e.details
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("found %d configuration errors on workload %s %s/%s: %v",
		len(e.details),
		e.workloadKind.String(),
		e.workloadNamespace,
		e.workloadName,
		e.details)
}

func (e *ConfigError) add(errorCode, description string, proxy *cloudsqlapi.AuthProxyWorkload) {
	e.details = append(e.details,
		ConfigErrorDetail{
			WorkloadKind:       e.workloadKind,
			WorkloadName:       e.workloadName,
			WorkloadNamespace:  e.workloadNamespace,
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

func (e *ConfigErrorDetail) Error() string {
	return fmt.Sprintf("error %s %s while applying AuthProxyWorkload %s/%s to workload  %s %s/%s",
		e.ErrorCode,
		e.Description,
		e.AuthProxyNamespace,
		e.AuthProxyName,
		e.WorkloadKind.String(),
		e.WorkloadNamespace,
		e.WorkloadName)

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
	// if there was an error updating workloads, return the error
	if err != nil {
		return false, nil, err
	}

	// if this was not updated, then return nil and an empty array because
	// no AuthProxyWorkloads were applied
	if !updated {
		return updated, []*cloudsqlapi.AuthProxyWorkload{}, nil
	}

	// if this was updated return matching AuthProxyWorkloads
	return updated, matchingAuthProxyWorkloads, nil

}

// filterMatchingInstances returns a list of AuthProxyWorkload whose selectors match
// the workload.
func filterMatchingInstances(wl cloudsqlapi.AuthProxyWorkloadList, workload Workload) []*cloudsqlapi.AuthProxyWorkload {
	matchingAuthProxyWorkloads := make([]*cloudsqlapi.AuthProxyWorkload, 0, len(wl.Items))
	for i, _ := range wl.Items {
		csqlWorkload := &wl.Items[i]
		if workloadMatches(workload, csqlWorkload.Spec.Workload, csqlWorkload.Namespace) {
			// need to update workload
			l.Info("Found matching workload",
				"workload", workload.Object().GetNamespace()+"/"+workload.Object().GetName(),
				"wlSelector", csqlWorkload.Spec.Workload,
				"AuthProxyWorkload", csqlWorkload.Namespace+"/"+csqlWorkload.Name)
			matchingAuthProxyWorkloads = append(matchingAuthProxyWorkloads, csqlWorkload)
		}
	}
	return matchingAuthProxyWorkloads
}

// workloadUpdateStatus describes when a workload was last updated, mostly
// used to log errors
type workloadUpdateStatus struct {
	InstanceGeneration    string
	LastRequstGeneration  string
	RequestGeneration     string
	LastUpdatedGeneration string
	UpdatedGeneration     string
}

// MarkWorkloadNeedsUpdate Updates annotations on the workload indicating that it may need an update.
// returns true if the workload actually needs an update.
func MarkWorkloadNeedsUpdate(csqlWorkload *cloudsqlapi.AuthProxyWorkload, workload Workload) (bool, workloadUpdateStatus) {
	return updateWorkloadAnnotations(csqlWorkload, workload, false)
}

// MarkWorkloadUpdated Updates annotations on the workload indicating that it
// has been updated, returns true of any modifications were made to the workload.
// for the AuthProxyWorkload.
func MarkWorkloadUpdated(csqlWorkload *cloudsqlapi.AuthProxyWorkload, workload Workload) (bool, workloadUpdateStatus) {
	return updateWorkloadAnnotations(csqlWorkload, workload, true)
}

// updateWorkloadAnnotations adds annotations to the workload
// to track which generation of a AuthProxyWorkload needs to be applied, and which
// generation has been applied. The AuthProxyWorkload controller is responsible for
// tracking which version should be applied, The workload admission webhook is
// responsible for applying the AuthProxyWorkloads that apply to a workload
// when the workload is created or modified.
func updateWorkloadAnnotations(csqlWorkload *cloudsqlapi.AuthProxyWorkload, workload Workload, doingUpdate bool) (bool, workloadUpdateStatus) {
	var s workloadUpdateStatus
	var doUpdate bool
	reqName := names.SafePrefixedName("csqlr-", csqlWorkload.Namespace+"-"+csqlWorkload.Name)
	resultName := names.SafePrefixedName("csqlu-", csqlWorkload.Namespace+"-"+csqlWorkload.Name)
	s.InstanceGeneration = fmt.Sprintf("%d", csqlWorkload.GetGeneration())

	ann := workload.Object().GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	s.LastRequstGeneration = ann[reqName]
	s.LastUpdatedGeneration = ann[resultName]

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

type managedEnvVar struct {
	AuthProxyWorkload    types.NamespacedName `json:"AuthProxyWorkload"`
	ConnectionString     string               `json:"ConnectionString,omitempty"`
	OriginalValues       map[string]string    `json:"originalValues,omitempty"`
	OperatorManagedValue corev1.EnvVar        `json:"operatorManagedValue"`
}

type managedPort struct {
	AuthProxyWorkload types.NamespacedName `json:"authProxyWorkload"`
	ConnectionString  string               `json:"connectionString,omitempty"`
	Port              int32                `json:"port,omitempty"`
	OriginalValues    map[string]int32     `json:"originalValues,omitempty"`
}

type managedVolume struct {
	AuthProxyWorkload types.NamespacedName `json:"authProxyWorkload"`
	ConnectionString  string               `json:"connectionString"`
	Volume            corev1.Volume        `json:"volume"`
	VolumeMount       corev1.VolumeMount   `json:"volumeMount"`
}

// updateState holds internal state while a particular workload being configured
// with one or more AuthProxyWorkloads.
type updateState struct {
	err                   ConfigError
	oldMods               workloadMods
	mods                  workloadMods
	nextDbPort            int32
	removedContainerNames []string
}

// workloadMods holds all modifications to this workload done by the operator so
// so that it can be undone later.
type workloadMods struct {
	AuthProxyWorkloads []types.NamespacedName `json:"authProxyWorkloads"`
	EnvVars            []*managedEnvVar       `json:"envVars"`
	VolumeMounts       []*managedVolume       `json:"volumeMounts"`
	Ports              []*managedPort         `json:"ports"`
}

func (s *updateState) addVolumeMount(p *cloudsqlapi.AuthProxyWorkload, is *cloudsqlapi.InstanceSpec, m corev1.VolumeMount, v corev1.Volume) {
	s.mods.VolumeMounts = append(s.mods.VolumeMounts, &managedVolume{
		AuthProxyWorkload: types.NamespacedName{
			Namespace: p.Namespace,
			Name:      p.Name,
		},
		ConnectionString: is.ConnectionString,
		Volume:           v,
		VolumeMount:      m,
	})
}

func (s *updateState) addInUsePort(p int32, containerName string) {
	s.addPort(p, containerName, types.NamespacedName{}, "")
}

// isPortInUse checks if the port is in use.
func (s *updateState) isPortInUse(p int32) bool {
	for i := 0; i < len(s.mods.Ports); i++ {
		if p == s.mods.Ports[i].Port {
			return true
		}
	}
	return false
}

// useNextDbPort consumes the next available db port, marking that port as "in-use."
func (s *updateState) useInstancePort(p *cloudsqlapi.AuthProxyWorkload, is *cloudsqlapi.InstanceSpec) int32 {
	n := types.NamespacedName{
		Namespace: p.Namespace,
		Name:      p.Name,
	}

	//Does a managedPort already exist for this workload+instance?
	var proxyPort *managedPort
	for _, mp := range s.mods.Ports {
		if mp.AuthProxyWorkload == n && mp.ConnectionString == is.ConnectionString {
			proxyPort = mp
			break
		}
	}

	// Update the managedPort for this workload+instance
	if proxyPort != nil {
		if is.Port != nil && proxyPort.Port != *is.Port {
			if s.isPortInUse(*is.Port) {
				s.addError(cloudsqlapi.ErrorCodePortConflict,
					fmt.Sprintf("proxy port %d for instance %s is already in use",
						*is.Port, is.ConnectionString), p)
			}
			proxyPort.Port = *is.Port
		}
		return proxyPort.Port
	}

	// Since this is a new workload+instance, figure out the port number
	var port int32
	if is.Port != nil {
		port = *is.Port
	} else {
		for s.isPortInUse(s.nextDbPort) {
			s.nextDbPort++
		}
		port = s.nextDbPort
	}

	if s.isPortInUse(port) {
		s.addError(cloudsqlapi.ErrorCodePortConflict,
			fmt.Sprintf("proxy port %d for instance %s is already in use",
				port, is.ConnectionString), p)
	}

	s.addPort(port, "", n, is.ConnectionString)

	return port
}

func (s *updateState) addPort(p int32, containerName string, n types.NamespacedName, connectionString string) {
	var mp *managedPort

	for i := 0; i < len(s.mods.Ports); i++ {
		if s.mods.Ports[i].Port == p {
			mp = s.mods.Ports[i]
		}
	}

	if mp == nil {
		mp = &managedPort{
			AuthProxyWorkload: n,
			ConnectionString:  connectionString,
			Port:              p,
			OriginalValues:    map[string]int32{},
		}
		s.mods.Ports = append(s.mods.Ports, mp)
	}
	if containerName != "" && !strings.HasPrefix(containerName, names.ContainerPrefix) {
		mp.OriginalValues[containerName] = p
	}

}
func (s *updateState) useNextDbPort(p *cloudsqlapi.AuthProxyWorkload, is *cloudsqlapi.InstanceSpec) int32 {
	for s.isPortInUse(s.nextDbPort) {
		s.nextDbPort++
	}
	return s.nextDbPort
}

// addWorkloadEnvVar adds or replaces the envVar based on its Name, returning the old and new values
func (s *updateState) addWorkloadEnvVar(proxy *cloudsqlapi.AuthProxyWorkload, inst *cloudsqlapi.InstanceSpec, envVar corev1.EnvVar) {

	for i := 0; i < len(s.mods.EnvVars); i++ {
		if s.mods.EnvVars[i].OperatorManagedValue.Name == envVar.Name {
			old := s.mods.EnvVars[i].OperatorManagedValue
			s.mods.EnvVars[i].OperatorManagedValue = envVar
			if old.Value != envVar.Value {
				s.addError(cloudsqlapi.ErrorCodeEnvConflict,
					fmt.Sprintf("environment variable named %s already exists", envVar.Name), proxy)
			}
			return
		}
	}
	s.mods.EnvVars = append(s.mods.EnvVars, &managedEnvVar{
		AuthProxyWorkload: types.NamespacedName{
			Namespace: proxy.Namespace,
			Name:      proxy.Name,
		},
		ConnectionString:     inst.ConnectionString,
		OriginalValues:       map[string]string{},
		OperatorManagedValue: envVar,
	})
	return
}

// loadOldEnvVarState loads the state connecting EnvVar changes done by the
// AuthProxyWorkload workload webhook from an annotation on that workload. This
// enables changes to be checked and reverted when a AuthProxyWorkload is removed.
func (s *updateState) loadOldEnvVarState(wl Workload) {
	ann := wl.Object().GetAnnotations()
	if ann == nil {
		return
	}

	val, exists := ann["csql-env"]
	if !exists {
		return
	}

	err := json.Unmarshal([]byte(val), &s.oldMods)
	if err != nil {
		l.Info("unable to unmarshal old environment workload vars", "error", err)
	}
	err = json.Unmarshal([]byte(val), &s.mods)
	if err != nil {
		l.Info("unable to unmarshal old environment workload vars", "error", err)
	}
}

// saveEnvVarState saves the most recent state from updated workloads
func (s *updateState) saveEnvVarState(wl Workload) {
	ann := wl.Object().GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	bytes, err := json.Marshal(s.mods)
	if err != nil {
		l.Info("unable to marshal old environment workload vars, %v", err)
		return
	}
	ann["csql-env"] = string(bytes)
	wl.Object().SetAnnotations(ann)
}

// update Reconciles the state of a workload, applying the matching AuthProxyWorkloads
// and removing any out-of-date configuration related to deleted AuthProxyWorkloads
func (s *updateState) update(workload Workload, matchingAuthProxyWorkloads []*cloudsqlapi.AuthProxyWorkload) (bool, *ConfigError) {
	s.loadOldEnvVarState(workload)
	podSpec := workload.PodSpec()
	containers := podSpec.Containers
	var updated bool

	var nonAuthProxyContainers []corev1.Container
	for i := 0; i < len(containers); i++ {
		if !strings.HasPrefix(containers[i].Name, names.ContainerPrefix) {
			nonAuthProxyContainers = append(nonAuthProxyContainers, containers[i])
		}
	}

	for i := 0; i < len(nonAuthProxyContainers); i++ {
		c := nonAuthProxyContainers[i]
		for j := 0; j < len(c.Ports); j++ {
			s.addInUsePort(c.Ports[j].ContainerPort, c.Name)
		}
	}

	// add all new containers and update existing containers
	for i, _ := range matchingAuthProxyWorkloads {
		inst := matchingAuthProxyWorkloads[i]
		s.addAuthProxyWorkload(inst)
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
			s.UpdateContainer(inst, workload, &newContainer)
			containers = append(containers, newContainer)
			updated = true
		} else {
			updated = s.UpdateContainer(inst, workload, instContainer)
		}
	}

	// remove all csql containers that don't relate to one of the matchingAuthProxyWorkloads
	var filteredContainers []corev1.Container
	var removedContainerNames []string

	for j, _ := range containers {
		container := &containers[j]
		if !strings.HasPrefix(container.Name, names.ContainerPrefix) {
			filteredContainers = append(filteredContainers, *container)
			continue
		}

		var found bool
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
			removedContainerNames = append(removedContainerNames, container.Name)
		}
	}
	s.removedContainerNames = removedContainerNames

	podSpec.Containers = filteredContainers

	for i, _ := range podSpec.Containers {
		s.updateContainerEnv(&podSpec.Containers[i])
		s.applyContainerVolumes(&podSpec.Containers[i])
	}
	s.applyVolumes(&podSpec)

	// only return ConfigError if there were reported
	// errors during processing.
	if len(s.err.details) > 0 {
		var err *ConfigError
		err = &s.err
		return updated, err
	}

	if updated {
		workload.SetPodSpec(podSpec)
		s.saveEnvVarState(workload)
	}

	return updated, nil
}

// UpdateContainer Creates or updates the proxy container in the workload's PodSpec
func (s *updateState) UpdateContainer(proxy *cloudsqlapi.AuthProxyWorkload, workload Workload, container *corev1.Container) bool {
	doUpdate, status := MarkWorkloadUpdated(proxy, workload)

	if !doUpdate {
		l.Info("Skipping workload {{workload}}, no update needed.", "name", workload.Object().GetName(),
			"doUpdate", doUpdate,
			"status", status)
		return false
	}

	l.Info("Updating workload {{workload}}, no update needed.", "name", workload.Object().GetName(),
		"doUpdate", doUpdate,
		"status", status)

	// if the container was fully overridden, just use that container.
	if proxy.Spec.AuthProxyContainer != nil && proxy.Spec.AuthProxyContainer.Container != nil {
		proxy.Spec.AuthProxyContainer.Container.DeepCopyInto(container)
		container.Name = names.ContainerName(proxy)
		return doUpdate
	}

	// Build the container
	var cliArgs []string

	// always enable http port healthchecks on 0.0.0.0 and structured logs
	cliArgs = append(cliArgs, fmt.Sprintf("--http-port=%d", s.addHealthCheck(proxy)))
	cliArgs = append(cliArgs, "--http-address=0.0.0.0")
	cliArgs = append(cliArgs, "--health-check")
	cliArgs = append(cliArgs, "--structured-logs")

	container.Name = names.ContainerName(proxy)
	container.ImagePullPolicy = "IfNotPresent"

	cliArgs = s.applyContainerSpec(proxy, container, cliArgs)
	cliArgs = s.applyTelemetrySpec(proxy, cliArgs)
	cliArgs = s.applyAuthenticationSpec(proxy, container, cliArgs)

	// Instances
	for _, inst := range proxy.Spec.Instances {

		params := map[string]string{}

		// if it is a TCP socket
		if inst.SocketType == "tcp" ||
			(inst.SocketType == "" && inst.UnixSocketPath == "") {
			port := s.useInstancePort(proxy, &inst)
			params["port"] = fmt.Sprint(port)
			if inst.HostEnvName != "" {
				s.addWorkloadEnvVar(proxy, &inst, corev1.EnvVar{
					Name:  inst.HostEnvName,
					Value: "localhost",
				})
			}
			if inst.PortEnvName != "" {
				s.addWorkloadEnvVar(proxy, &inst, corev1.EnvVar{
					Name:  inst.PortEnvName,
					Value: fmt.Sprint(port),
				})
			}
		} else {
			// else if it is a unix socket
			params["unix-socket"] = inst.UnixSocketPath
			mountName := names.VolumeName(proxy, &inst, "unix")
			s.addVolumeMount(proxy, &inst,
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
				s.addWorkloadEnvVar(proxy, &inst, corev1.EnvVar{
					Name:  inst.UnixSocketPathEnvName,
					Value: inst.UnixSocketPath,
				})
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

		var instArgs []string
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

	return doUpdate
}

// applyContainerSpec applies settings from cloudsqlapi.AuthProxyContainerSpec
// to the container
func (s *updateState) applyContainerSpec(proxy *cloudsqlapi.AuthProxyWorkload, container *corev1.Container, cliArgs []string) []string {
	if proxy.Spec.AuthProxyContainer == nil {
		return cliArgs
	}

	// Fuse
	if proxy.Spec.AuthProxyContainer.FUSEDir != "" || proxy.Spec.AuthProxyContainer.FUSETempDir != "" {
		s.addError(cloudsqlapi.ErrorCodeFUSENotSupported, "the FUSE filesystem is not yet supported", proxy)

		//TODO fuse...
		// if FUSE is used, we need to use the 'buster' or 'alpine' image.

	}

	container.Image = s.defaultProxyImage()
	if proxy.Spec.AuthProxyContainer.Image != "" {
		container.Image = proxy.Spec.AuthProxyContainer.Image
	}

	container.Resources = defaultContainerResources
	if proxy.Spec.AuthProxyContainer.Resources != nil {
		container.Resources = *proxy.Spec.AuthProxyContainer.Resources.DeepCopy()
	}

	if proxy.Spec.AuthProxyContainer.SQLAdminAPIEndpoint != "" {
		cliArgs = append(cliArgs, "--sqladmin-api-endpoint="+proxy.Spec.AuthProxyContainer.SQLAdminAPIEndpoint)
	}
	if proxy.Spec.AuthProxyContainer.MaxConnections != nil &&
		*proxy.Spec.AuthProxyContainer.MaxConnections != 0 {
		cliArgs = append(cliArgs, fmt.Sprintf("--max-connections=%d", *proxy.Spec.AuthProxyContainer.MaxConnections))
	}
	if proxy.Spec.AuthProxyContainer.MaxSigtermDelay != nil &&
		*proxy.Spec.AuthProxyContainer.MaxSigtermDelay != 0 {
		cliArgs = append(cliArgs, fmt.Sprintf("--max-sigterm-delay=%d", *proxy.Spec.AuthProxyContainer.MaxSigtermDelay))
	}

	return cliArgs
}

// applyTelemetrySpec applies settings from cloudsqlapi.TelemetrySpec
// to the container
func (s *updateState) applyTelemetrySpec(proxy *cloudsqlapi.AuthProxyWorkload, cliArgs []string) []string {
	if proxy.Spec.AuthProxyContainer == nil || proxy.Spec.AuthProxyContainer.Telemetry == nil {
		return cliArgs
	}
	tel := proxy.Spec.AuthProxyContainer.Telemetry

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
	if tel.QuotaProject != nil {
		cliArgs = append(cliArgs, fmt.Sprintf("--quota-project=%s", *tel.QuotaProject))
	}

	return cliArgs
}

// updateContainerEnv applies global container state to all containers
func (s *updateState) updateContainerEnv(c *corev1.Container) {
	for i := 0; i < len(s.mods.EnvVars); i++ {
		var found bool
		operatorEnv := s.mods.EnvVars[i].OperatorManagedValue
		oldManagedEnv := s.oldManagedEnv(operatorEnv.Name)

		for j := 0; j < len(c.Env); j++ {
			if operatorEnv.Name == c.Env[j].Name {
				found = true

				if oldManagedEnv == nil {
					l.Info("Override env {{env}} on container {{container}} from {{old}} to {{new}}",
						"env", operatorEnv.Name,
						"container", c.Name,
						"old", c.Env[j].Value,
						"new", operatorEnv.Value)
					s.mods.EnvVars[i].OriginalValues[c.Name] = c.Env[j].Value
				}
				c.Env[j] = operatorEnv
			}
		}
		if !found {
			c.Env = append(c.Env, operatorEnv)
		}
	}

	// filter and restore csql env vars
	for i := 0; i < len(s.oldMods.EnvVars); i++ {
		oldEnvVar := s.oldMods.EnvVars[i]
		s.filterOldEnvVar(c, oldEnvVar)
	}
}

func (s *updateState) filterOldEnvVar(c *corev1.Container, oldEnvVar *managedEnvVar) {

	// Check if this env var belongs to a removed workload
	var workloadRemoved bool
	removedName := names.ContainerNameFromNamespacedName(oldEnvVar.AuthProxyWorkload)
	for j := 0; j < len(s.removedContainerNames); j++ {
		if s.removedContainerNames[j] == removedName {
			workloadRemoved = true
		}
	}
	if !workloadRemoved {
		return
	}

	// Check if this env var was replaced with a new one of the same name
	var newEnvVarWithSameName bool
	for j := 0; j < len(s.mods.EnvVars) && !newEnvVarWithSameName; j++ {
		mev := s.mods.EnvVars[j]
		if mev.OperatorManagedValue.Name == oldEnvVar.OperatorManagedValue.Name &&
			mev.AuthProxyWorkload != oldEnvVar.AuthProxyWorkload {
			newEnvVarWithSameName = true
		}
	}
	if newEnvVarWithSameName {
		return
	}

	// Check if the container has an env var with this name
	var containerEnv *corev1.EnvVar
	var index int
	for j := 0; j < len(c.Env) && containerEnv == nil; j++ {
		if oldEnvVar.OperatorManagedValue.Name == c.Env[j].Name {
			containerEnv = &c.Env[j]
			index = j
		}
	}
	if containerEnv == nil {
		return
	}

	// Restore the original value or remove the env var
	originalValue, hasOriginalValue := oldEnvVar.OriginalValues[c.Name]
	if hasOriginalValue {
		l.Info("Restored {{env}} to original value {{val}} on {{container}}.",
			"env", oldEnvVar.OperatorManagedValue.Name,
			"val", originalValue,
			"container", c.Name)
		// replace the original value
		containerEnv.Value = originalValue
	} else {
		// remove the element from the array
		l.Info("Removed {{env}} on {{container}}.",
			"env", oldEnvVar.OperatorManagedValue.Name,
			"container", c.Name)
		c.Env = append(c.Env[0:index], c.Env[index+1:]...)
	}

}

// applyContainerVolumes applies global container state to all containers
func (s *updateState) applyContainerVolumes(c *corev1.Container) {
	for i := 0; i < len(s.mods.VolumeMounts); i++ {
		var found bool
		for j := 0; j < len(c.VolumeMounts); j++ {
			if s.mods.VolumeMounts[i].VolumeMount.Name == c.VolumeMounts[j].Name {
				found = true
				c.VolumeMounts[j] = s.mods.VolumeMounts[i].VolumeMount
			}
		}
		if !found {
			c.VolumeMounts = append(c.VolumeMounts, s.mods.VolumeMounts[i].VolumeMount)
		}
	}
	// filter removed csql VolumeMounts
	var filteredMounts []corev1.VolumeMount
	for i := 0; i < len(c.VolumeMounts); i++ {
		var removed bool
		for j := 0; j < len(s.removedContainerNames); j++ {
			if strings.HasPrefix(c.VolumeMounts[i].Name, s.removedContainerNames[j]) {
				removed = true
			}
		}
		if !removed {
			filteredMounts = append(filteredMounts, c.VolumeMounts[i])
		}
	}
	c.VolumeMounts = filteredMounts
}

// applyVolumes applies global container state to all containers
func (s *updateState) applyVolumes(spec *corev1.PodSpec) {
	for i := 0; i < len(s.mods.VolumeMounts); i++ {
		var found bool
		for j := 0; j < len(spec.Volumes); j++ {
			if s.mods.VolumeMounts[i].Volume.Name == spec.Volumes[j].Name {
				found = true
				spec.Volumes[j] = s.mods.VolumeMounts[i].Volume
			}
		}
		if !found {
			spec.Volumes = append(spec.Volumes, s.mods.VolumeMounts[i].Volume)
		}
	}

	// filter removed csql volumes
	var filteredVolumes []corev1.Volume
	for i := 0; i < len(spec.Volumes); i++ {
		var removed bool
		for j := 0; j < len(s.removedContainerNames); j++ {
			if strings.HasPrefix(spec.Volumes[i].Name, s.removedContainerNames[j]) {
				removed = true
			}
		}
		if !removed {
			filteredVolumes = append(filteredVolumes, spec.Volumes[i])
		}
	}
	spec.Volumes = filteredVolumes
}

func (s *updateState) addHealthCheck(csqlWorkload *cloudsqlapi.AuthProxyWorkload) int32 {
	var port int32

	cs := csqlWorkload.Spec.AuthProxyContainer
	// if the TelemetrySpec.HTTPPort is explicitly set
	if cs != nil && cs.Telemetry != nil && cs.Telemetry.HTTPPort != nil {
		port = *cs.Telemetry.HTTPPort
		if s.isPortInUse(port) {
			s.addError(cloudsqlapi.ErrorCodePortConflict,
				fmt.Sprintf("telemetry httpPort %d is already in use", port), csqlWorkload)
		}
	} else {
		for port = DefaultHealthCheckPort; !s.isPortInUse(port); port++ {
			// start with DefaultHealthCheck and increment port until it is set to an unused port
		}
	}

	//TODO add healthcheck to podspec

	return port
}

func (s *updateState) addError(errorCode string, description string, proxy *cloudsqlapi.AuthProxyWorkload) {
	s.err.add(errorCode, description, proxy)
}

func (s *updateState) oldManagedEnv(name string) *managedEnvVar {
	for i := 0; i < len(s.oldMods.EnvVars); i++ {
		if s.oldMods.EnvVars[i].OperatorManagedValue.Name == name {
			return s.oldMods.EnvVars[i]
		}
	}
	return nil
}

func (s *updateState) applyAuthenticationSpec(proxy *cloudsqlapi.AuthProxyWorkload, container *corev1.Container, args []string) []string {
	if proxy.Spec.Authentication == nil {
		return args
	}

	//TODO Authentication needs end-to-end test in place before we can check
	// that it is implemented correctly.
	// --credentials-file
	return args
}

func (s *updateState) defaultProxyImage() string {
	//TODO look this up from the public registry
	return "us-central1-docker.pkg.dev/csql-operator-test/test76e6d646e2caac1c458c/proxy-v2:latest"
}

func (s *updateState) addAuthProxyWorkload(p *cloudsqlapi.AuthProxyWorkload) {
	s.mods.AuthProxyWorkloads = append(s.mods.AuthProxyWorkloads, types.NamespacedName{
		Namespace: p.Namespace,
		Name:      p.Name,
	})
}
