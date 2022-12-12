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

package workload

import (
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1alpha1"
)

// Constants for well known error codes and defaults. These are exposed on the
// package and documented here so that they appear in the godoc. These also
// need to be documented in the CRD
const (
	DefaultProxyImage = "gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.0.0-preview.4"

	// DefaultFirstPort is the first port number chose for an instance listener by the
	// proxy.
	DefaultFirstPort int32 = 5000

	// DefaultHealthCheckPort is the used by the proxy to expose prometheus
	// and kubernetes health checks.
	DefaultHealthCheckPort int32 = 9801
)

var l = logf.Log.WithName("internal.workload")

// Updater holds global state used while reconciling workloads.
type Updater struct {
}

// NewUpdater creates a new instance of Updater with a supplier
// that loads the default proxy impage from the public docker registry
func NewUpdater() *Updater {
	return &Updater{}
}

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

func (e *ConfigError) add(errorCode, description string, p *cloudsqlapi.AuthProxyWorkload) {
	e.details = append(e.details,
		ConfigErrorDetail{
			WorkloadKind:       e.workloadKind,
			WorkloadName:       e.workloadName,
			WorkloadNamespace:  e.workloadNamespace,
			AuthProxyNamespace: p.GetNamespace(),
			AuthProxyName:      p.GetName(),
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
		"memory": resource.MustParse("2Gi"),
	},
}

// ConfigurePodProxies finds all AuthProxyWorkload resources matching this workload and then
// updates the workload's containers. This does not save the updated workload.
func (u *Updater) FindMatchingAuthProxyWorkloads(pl *cloudsqlapi.AuthProxyWorkloadList, wl *PodWorkload, owners []Workload) []*cloudsqlapi.AuthProxyWorkload {

	// if a wl has an owner, then ignore it.
	wls := u.filterMatchingInstances(pl, wl.Object())
	for _, owner := range owners {
		wls = append(wls, u.filterMatchingInstances(pl, owner.Object())...)
	}

	// remove duplicates from wls by Name
	m := map[string]*cloudsqlapi.AuthProxyWorkload{}
	for _, w := range wls {
		m[w.GetNamespace()+"/"+w.GetName()] = w
	}
	wls = make([]*cloudsqlapi.AuthProxyWorkload, 0, len(m))
	for _, w := range m {
		wls = append(wls, w)
	}
	// if this was updated return matching DBInstances
	return wls
}

// filterMatchingInstances returns a list of AuthProxyWorkload whose selectors match
// the workload.
func (u *Updater) filterMatchingInstances(pl *cloudsqlapi.AuthProxyWorkloadList, wl client.Object) []*cloudsqlapi.AuthProxyWorkload {
	matchingAuthProxyWorkloads := make([]*cloudsqlapi.AuthProxyWorkload, 0, len(pl.Items))
	for i := range pl.Items {
		p := &pl.Items[i]
		if workloadMatches(wl, p.Spec.Workload, p.Namespace) {
			// if this is pending deletion, exclude it.
			if !p.ObjectMeta.DeletionTimestamp.IsZero() {
				continue
			}

			matchingAuthProxyWorkloads = append(matchingAuthProxyWorkloads, p)
			// need to update wl
			l.Info("Found matching wl",
				"wl", wl.GetNamespace()+"/"+wl.GetName(),
				"wlSelector", p.Spec.Workload,
				"AuthProxyWorkload", p.Namespace+"/"+p.Name)
		}
	}
	return matchingAuthProxyWorkloads
}

// ConfigureWorkload applies the proxy containers from all of the
// instances listed in matchingAuthProxyWorkloads to the workload
func (u *Updater) ConfigureWorkload(wl *PodWorkload, matches []*cloudsqlapi.AuthProxyWorkload) error {
	state := updateState{
		updater:    u,
		nextDBPort: DefaultFirstPort,
		err: ConfigError{
			workloadKind:      wl.Object().GetObjectKind().GroupVersionKind(),
			workloadName:      wl.Object().GetName(),
			workloadNamespace: wl.Object().GetNamespace(),
		},
	}
	return state.update(wl, matches)
}

type managedEnvVar struct {
	Instance             dbInstance        `json:"dbInstance"`
	OperatorManagedValue corev1.EnvVar     `json:"operatorManagedValue"`
	OriginalValues       map[string]string `json:"originalValues,omitempty"`
}

type managedPort struct {
	Instance       dbInstance       `json:"dbInstance"`
	OriginalValues map[string]int32 `json:"originalValues,omitempty"`
	Port           int32            `json:"port,omitempty"`
}

type managedVolume struct {
	Volume      corev1.Volume      `json:"volume"`
	VolumeMount corev1.VolumeMount `json:"volumeMount"`
	Instance    dbInstance         `json:"dbInstance"`
}

type dbInstance struct {
	AuthProxyWorkload types.NamespacedName `json:"authProxyWorkload"`
	ConnectionString  string               `json:"connectionString"`
}

func dbInst(namespace, name, connectionString string) dbInstance {
	return dbInstance{
		AuthProxyWorkload: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
		ConnectionString: connectionString,
	}
}

// updateState holds internal state while a particular workload being configured
// with one or more DBInstances.
type updateState struct {
	err        ConfigError
	mods       workloadMods
	nextDBPort int32
	updater    *Updater
}

// workloadMods holds all modifications to this workload done by the operator so
// so that it can be undone later.
type workloadMods struct {
	DBInstances  []*dbInstance    `json:"dbInstances"`
	EnvVars      []*managedEnvVar `json:"envVars"`
	VolumeMounts []*managedVolume `json:"volumeMounts"`
	Ports        []*managedPort   `json:"ports"`
}

func (s *updateState) addVolumeMount(p *cloudsqlapi.AuthProxyWorkload, is *cloudsqlapi.InstanceSpec, m corev1.VolumeMount, v corev1.Volume) {
	key := dbInst(p.Namespace, p.Name, is.ConnectionString)
	vol := &managedVolume{
		Instance:    key,
		Volume:      v,
		VolumeMount: m,
	}

	for i, mount := range s.mods.VolumeMounts {
		if mount.Instance == key {
			s.mods.VolumeMounts[i] = vol
			return
		}
	}
	s.mods.VolumeMounts = append(s.mods.VolumeMounts, vol)
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

	// Does a managedPort already exist for this workload+instance?
	var proxyPort *managedPort
	for _, mp := range s.mods.Ports {
		if mp.Instance.AuthProxyWorkload == n && mp.Instance.ConnectionString == is.ConnectionString {
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
		for s.isPortInUse(s.nextDBPort) {
			s.nextDBPort++
		}
		port = s.nextDBPort
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
			Instance:       dbInst(n.Namespace, n.Name, connectionString),
			Port:           p,
			OriginalValues: map[string]int32{},
		}
		s.mods.Ports = append(s.mods.Ports, mp)
	}
	if containerName != "" && !strings.HasPrefix(containerName, ContainerPrefix) {
		mp.OriginalValues[containerName] = p
	}

}

// addWorkloadEnvVar adds or replaces the envVar based on its Name, returning the old and new values
func (s *updateState) addWorkloadEnvVar(p *cloudsqlapi.AuthProxyWorkload, is *cloudsqlapi.InstanceSpec, ev corev1.EnvVar) {
	for i := 0; i < len(s.mods.EnvVars); i++ {
		if s.mods.EnvVars[i].OperatorManagedValue.Name == ev.Name {
			old := s.mods.EnvVars[i].OperatorManagedValue
			s.mods.EnvVars[i].OperatorManagedValue = ev
			if old.Value != ev.Value {
				s.addError(cloudsqlapi.ErrorCodeEnvConflict,
					fmt.Sprintf("environment variable named %s already exists", ev.Name), p)
			}
			return
		}
	}
	s.mods.EnvVars = append(s.mods.EnvVars, &managedEnvVar{
		Instance:             dbInst(p.Namespace, p.Name, is.ConnectionString),
		OriginalValues:       map[string]string{},
		OperatorManagedValue: ev,
	})
}

func (s *updateState) initState(pl []*cloudsqlapi.AuthProxyWorkload) {
	// Reset the mods.DBInstances to the list of pl being
	// applied right now.
	s.mods.DBInstances = make([]*dbInstance, 0, len(pl))
	for _, wl := range pl {
		for _, instance := range wl.Spec.Instances {
			s.mods.DBInstances = append(s.mods.DBInstances,
				&dbInstance{
					AuthProxyWorkload: types.NamespacedName{
						Namespace: wl.Namespace,
						Name:      wl.Name,
					},
					ConnectionString: instance.ConnectionString,
				})
		}
	}

}

// update Reconciles the state of a workload, applying the matching DBInstances
// and removing any out-of-date configuration related to deleted DBInstances
func (s *updateState) update(wl *PodWorkload, matches []*cloudsqlapi.AuthProxyWorkload) error {

	s.initState(matches)
	podSpec := wl.PodSpec()
	containers := podSpec.Containers

	var nonAuthProxyContainers []corev1.Container
	for i := 0; i < len(containers); i++ {
		if !strings.HasPrefix(containers[i].Name, ContainerPrefix) {
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
	for i := range matches {
		inst := matches[i]

		newContainer := corev1.Container{}
		s.updateContainer(inst, wl, &newContainer)
		containers = append(containers, newContainer)
	}

	podSpec.Containers = containers

	for i := range podSpec.Containers {
		c := &podSpec.Containers[i]
		s.updateContainerEnv(c)
		for _, mount := range s.mods.VolumeMounts {
			c.VolumeMounts = append(c.VolumeMounts, mount.VolumeMount)
		}
	}
	for _, mount := range s.mods.VolumeMounts {
		podSpec.Volumes = append(podSpec.Volumes, mount.Volume)
	}

	// only return ConfigError if there were reported
	// errors during processing.
	if len(s.err.details) > 0 {
		return &s.err
	}

	wl.SetPodSpec(podSpec)

	return nil
}

// updateContainer Creates or updates the proxy container in the workload's PodSpec
func (s *updateState) updateContainer(p *cloudsqlapi.AuthProxyWorkload, wl Workload, c *corev1.Container) {
	l.Info("Updating wl {{wl}}, no update needed.", "name", client.ObjectKeyFromObject(wl.Object()))

	// if the c was fully overridden, just use that c.
	if p.Spec.AuthProxyContainer != nil && p.Spec.AuthProxyContainer.Container != nil {
		p.Spec.AuthProxyContainer.Container.DeepCopyInto(c)
		c.Name = ContainerName(p)
		return
	}

	// Build the c
	var cliArgs []string

	// always enable http port healthchecks on 0.0.0.0 and structured logs
	healthcheckPort := s.addHealthCheck(p, c)
	cliArgs = append(cliArgs,
		fmt.Sprintf("--http-port=%d", healthcheckPort),
		"--http-address=0.0.0.0",
		"--health-check",
		"--structured-logs")

	c.Name = ContainerName(p)
	c.ImagePullPolicy = "IfNotPresent"

	cliArgs = s.applyContainerSpec(p, c, cliArgs)
	cliArgs = s.applyTelemetrySpec(p, cliArgs)
	cliArgs = s.applyAuthenticationSpec(p, c, cliArgs)

	// Instances
	for i := range p.Spec.Instances {
		inst := &p.Spec.Instances[i]
		params := map[string]string{}

		// if it is a TCP socket
		if inst.SocketType == "tcp" ||
			(inst.SocketType == "" && inst.UnixSocketPath == "") {
			port := s.useInstancePort(p, inst)
			params["port"] = fmt.Sprint(port)
			if inst.HostEnvName != "" {
				s.addWorkloadEnvVar(p, inst, corev1.EnvVar{
					Name:  inst.HostEnvName,
					Value: "127.0.0.1",
				})
			}
			if inst.PortEnvName != "" {
				s.addWorkloadEnvVar(p, inst, corev1.EnvVar{
					Name:  inst.PortEnvName,
					Value: fmt.Sprint(port),
				})
			}
		} else {
			// else if it is a unix socket
			params["unix-socket"] = inst.UnixSocketPath
			mountName := VolumeName(p, inst, "unix")
			s.addVolumeMount(p, inst,
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
				s.addWorkloadEnvVar(p, inst, corev1.EnvVar{
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
	c.Args = cliArgs
}

// applyContainerSpec applies settings from cloudsqlapi.AuthProxyContainerSpec
// to the container
func (s *updateState) applyContainerSpec(p *cloudsqlapi.AuthProxyWorkload, c *corev1.Container, cliArgs []string) []string {
	c.Image = s.defaultProxyImage()
	c.Resources = defaultContainerResources

	if p.Spec.AuthProxyContainer == nil {
		return cliArgs
	}

	// Fuse
	if p.Spec.AuthProxyContainer.FUSEDir != "" || p.Spec.AuthProxyContainer.FUSETempDir != "" {
		s.addError(cloudsqlapi.ErrorCodeFUSENotSupported, "the FUSE filesystem is not yet supported", p)
		// if FUSE is used, we need to use the 'buster' or 'alpine' image.
	}

	if p.Spec.AuthProxyContainer.Image != "" {
		c.Image = p.Spec.AuthProxyContainer.Image
	}

	if p.Spec.AuthProxyContainer.Resources != nil {
		c.Resources = *p.Spec.AuthProxyContainer.Resources.DeepCopy()
	}

	if p.Spec.AuthProxyContainer.SQLAdminAPIEndpoint != "" {
		cliArgs = append(cliArgs, "--sqladmin-api-endpoint="+p.Spec.AuthProxyContainer.SQLAdminAPIEndpoint)
	}
	if p.Spec.AuthProxyContainer.MaxConnections != nil &&
		*p.Spec.AuthProxyContainer.MaxConnections != 0 {
		cliArgs = append(cliArgs, fmt.Sprintf("--max-connections=%d", *p.Spec.AuthProxyContainer.MaxConnections))
	}
	if p.Spec.AuthProxyContainer.MaxSigtermDelay != nil &&
		*p.Spec.AuthProxyContainer.MaxSigtermDelay != 0 {
		cliArgs = append(cliArgs, fmt.Sprintf("--max-sigterm-delay=%d", *p.Spec.AuthProxyContainer.MaxSigtermDelay))
	}

	return cliArgs
}

// applyTelemetrySpec applies settings from cloudsqlapi.TelemetrySpec
// to the container
func (s *updateState) applyTelemetrySpec(p *cloudsqlapi.AuthProxyWorkload, cliArgs []string) []string {
	if p.Spec.AuthProxyContainer == nil || p.Spec.AuthProxyContainer.Telemetry == nil {
		return cliArgs
	}
	tel := p.Spec.AuthProxyContainer.Telemetry

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

		for j := 0; j < len(c.Env); j++ {
			if operatorEnv.Name == c.Env[j].Name {
				found = true
				c.Env[j] = operatorEnv
			}
		}
		if !found {
			c.Env = append(c.Env, operatorEnv)
		}
	}

}

// addHealthCheck adds the health check declaration to this workload.
func (s *updateState) addHealthCheck(p *cloudsqlapi.AuthProxyWorkload, c *corev1.Container) int32 {
	var port int32

	cs := p.Spec.AuthProxyContainer
	// if the TelemetrySpec.HTTPPort is explicitly set
	if cs != nil && cs.Telemetry != nil && cs.Telemetry.HTTPPort != nil {
		port = *cs.Telemetry.HTTPPort
		if s.isPortInUse(port) {
			s.addError(cloudsqlapi.ErrorCodePortConflict,
				fmt.Sprintf("telemetry httpPort %d is already in use", port), p)
		}
	} else {
		port = DefaultHealthCheckPort
		for s.isPortInUse(port) {
			port++
		}
	}

	c.StartupProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
			Port: intstr.IntOrString{IntVal: port},
			Path: "/startup",
		}},
		PeriodSeconds: 30,
	}
	c.ReadinessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
			Port: intstr.IntOrString{IntVal: port},
			Path: "/readiness",
		}},
		PeriodSeconds: 30,
	}
	c.LivenessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
			Port: intstr.IntOrString{IntVal: port},
			Path: "/liveness",
		}},
		PeriodSeconds: 30,
	}
	return port
}

func (s *updateState) addError(errorCode, description string, p *cloudsqlapi.AuthProxyWorkload) {
	s.err.add(errorCode, description, p)
}

func (s *updateState) applyAuthenticationSpec(proxy *cloudsqlapi.AuthProxyWorkload, _ *corev1.Container, args []string) []string {
	if proxy.Spec.Authentication == nil {
		return args
	}
	// Authentication needs end-to-end test in place before we can check
	// that it is implemented correctly.
	// --credentials-file
	return args
}

func (s *updateState) defaultProxyImage() string {
	return DefaultProxyImage
}
