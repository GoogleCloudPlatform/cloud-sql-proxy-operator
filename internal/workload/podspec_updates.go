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
	"path"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	cloudsqlapi "github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1"
)

// Constants for well known error codes and defaults. These are exposed on the
// package and documented here so that they appear in the godoc. These also
// need to be documented in the CRD
const (
	// DefaultProxyImage is the latest version of the proxy as of the release
	// of this operator. This is managed as a dependency. We update this constant
	// when the Cloud SQL Auth Proxy releases a new version.
	DefaultProxyImage = "gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.1.1"

	// DefaultAlloyDBProxyImage is the latest version of the proxy as of the release
	// of this operator. This is managed as a dependency. We update this constant
	// when the AlloyDB Auth Proxy releases a new version.
	DefaultAlloyDBProxyImage = "gcr.io/alloydb-connectors/alloydb-auth-proxy:1.2.1"

	// DefaultFirstPort is the first port number chose for an instance listener by the
	// proxy.
	DefaultFirstPort int32 = 5000

	// DefaultHealthCheckPort is the used by the proxy to expose prometheus
	// and kubernetes health checks.
	DefaultHealthCheckPort int32 = 9801

	// DefaultAdminPort is the used by the proxy to expose prometheus
	// and kubernetes health checks.
	DefaultAdminPort int32 = 9802
)

var l = logf.Log.WithName("internal.workload")

// PodAnnotation returns the annotation (key, value) that should be added to
// pods that are configured with this AuthProxyWorkload resource. This takes
// into account whether the AuthProxyWorkload exists or was recently deleted.
// The defaultProxyImage is part of the annotation value.
func PodAnnotation(r *cloudsqlapi.AuthProxyWorkload, defaultProxyImage string) (string, string) {
	img := defaultProxyImage
	if r.Spec.AuthProxyContainer != nil && r.Spec.AuthProxyContainer.Image != "" {
		img = ""
	}
	k := fmt.Sprintf("%s/%s", cloudsqlapi.AnnotationPrefix, r.Name)
	v := fmt.Sprintf("%d,%s", r.Generation, img)
	// if r was deleted, use a different value
	if !r.GetDeletionTimestamp().IsZero() {
		v = fmt.Sprintf("%d-deleted-%s,%s", r.Generation, r.GetDeletionTimestamp().Format(time.RFC3339), img)
	}

	return k, v
}

// PodAnnotation returns the annotation (key, value) that should be added to
// pods that are configured with this AuthProxyWorkload resource. This takes
// into account whether the AuthProxyWorkload exists or was recently deleted.
func (u *Updater) PodAnnotation(r *cloudsqlapi.AuthProxyWorkload) (string, string) {
	return PodAnnotation(r, u.defaultProxyImage)
}

// Updater holds global state used while reconciling workloads.
type Updater struct {
	// userAgent is the userAgent of the operator
	userAgent string

	// defaultProxyImage is the current default proxy image for the operator
	defaultProxyImage string
}

// NewUpdater creates a new instance of Updater with a supplier
// that loads the default proxy impage from the public docker registry
func NewUpdater(userAgent string, defaultProxyImage string) *Updater {
	return &Updater{userAgent: userAgent, defaultProxyImage: defaultProxyImage}
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
	Instance             proxyInstanceID `json:"proxyInstanceID"`
	ContainerName        string          `json:"containerName"`
	OperatorManagedValue corev1.EnvVar   `json:"operatorManagedValue"`
}

type managedPort struct {
	Instance proxyInstanceID `json:"proxyInstanceID"`
	Port     int32           `json:"port,omitempty"`
}

type managedVolume struct {
	Volume      corev1.Volume      `json:"volume"`
	VolumeMount corev1.VolumeMount `json:"volumeMount"`
	Instance    proxyInstanceID    `json:"proxyInstanceID"`
}

// proxyInstanceID is an identifier for a proxy and/or specific proxy database
// instance that created the EnvVar or Port. When this is empty, means that the
// EnvVar or Port was created by the user, and is not associated with a proxy
type proxyInstanceID struct {
	AuthProxyWorkload types.NamespacedName `json:"authProxyWorkload"`
	ConnectionString  string               `json:"connectionString"`
	Type              string               `json:"type"`
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
	EnvVars      []*managedEnvVar `json:"envVars"`
	VolumeMounts []*managedVolume `json:"volumeMounts"`
	Ports        []*managedPort   `json:"ports"`
}

func (s *updateState) addWorkloadPort(p int32) {
	// This port is associated with the workload, not the proxy.
	// so this uses an empty proxyInstanceID{}
	s.addPort(p, proxyInstanceID{})
}

func (s *updateState) addProxyPort(port int32, p *cloudsqlapi.AuthProxyWorkload) {
	// This port is associated with the workload, not the proxy.
	// so this uses an empty proxyInstanceID{}
	s.addPort(port, proxyInstanceID{
		AuthProxyWorkload: types.NamespacedName{
			Namespace: p.Namespace,
			Name:      p.Name,
		},
	})
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

	s.addPort(port, proxyInstanceID{
		AuthProxyWorkload: types.NamespacedName{
			Name:      p.Name,
			Namespace: p.Namespace,
		},
		ConnectionString: is.ConnectionString,
	})

	return port
}

func (s *updateState) addPort(p int32, instance proxyInstanceID) {
	var mp *managedPort

	for i := 0; i < len(s.mods.Ports); i++ {
		if s.mods.Ports[i].Port == p {
			mp = s.mods.Ports[i]
		}
	}

	if mp == nil {
		mp = &managedPort{
			Instance: instance,
			Port:     p,
		}
		s.mods.Ports = append(s.mods.Ports, mp)
	}
}

// addWorkloadEnvVar adds or replaces the envVar based on its Name, returning the old and new values
func (s *updateState) addWorkloadEnvVar(p *cloudsqlapi.AuthProxyWorkload, is *cloudsqlapi.InstanceSpec, ev corev1.EnvVar) {
	s.addEnvVar(p, managedEnvVar{
		Instance: proxyInstanceID{
			AuthProxyWorkload: types.NamespacedName{
				Namespace: p.Namespace,
				Name:      p.Name,
			},
			ConnectionString: is.ConnectionString,
		},
		OperatorManagedValue: ev,
	})
}
func (s *updateState) addEnvVar(p *cloudsqlapi.AuthProxyWorkload, v managedEnvVar) {
	for i := 0; i < len(s.mods.EnvVars); i++ {
		oldEnv := s.mods.EnvVars[i]
		// if the values don't match and either one is global, or its set twice
		if isEnvVarConflict(oldEnv, v) {
			s.addError(cloudsqlapi.ErrorCodeEnvConflict,
				fmt.Sprintf("environment variable named %s is set more than once",
					oldEnv.OperatorManagedValue.Name),
				p)
			return
		}
	}

	s.mods.EnvVars = append(s.mods.EnvVars, &v)
}

func isEnvVarConflict(oldEnv *managedEnvVar, v managedEnvVar) bool {
	// it's a different name, no conflict
	if oldEnv.OperatorManagedValue.Name != v.OperatorManagedValue.Name {
		return false
	}

	// if the envvar is intended for a different container
	if oldEnv.ContainerName != v.ContainerName && oldEnv.ContainerName != "" && v.ContainerName != "" {
		return false
	}

	// different value, therefore conflict
	return oldEnv.OperatorManagedValue.Value != v.OperatorManagedValue.Value
}

// update Reconciles the state of a workload, applying the matching DBInstances
// and removing any out-of-date configuration related to deleted DBInstances
func (s *updateState) update(wl *PodWorkload, matches []*cloudsqlapi.AuthProxyWorkload) error {

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
			s.addWorkloadPort(c.Ports[j].ContainerPort)
		}
	}

	// Copy the existing pod annotation map
	ann := map[string]string{}
	for k, v := range wl.PodTemplateAnnotations() {
		ann[k] = v
	}

	// add all new containers and update existing containers
	for i := range matches {
		inst := matches[i]

		// cloudsql proxy container
		if len(inst.Spec.Instances) > 0 {
			newContainer := corev1.Container{}
			ucs := &updateContainerState{
				us:        s,
				proxyType: "cloudsql",
				c:         &newContainer,
				p:         inst,
				cs:        inst.Spec.AuthProxyContainer,
				instances: inst.Spec.Instances,
				envPrefix: "CSQL_PROXY_",
			}
			ucs.init()
			containers = append(containers, newContainer)
		}

		// alloydb proxy container
		if len(inst.Spec.AlloyDBInstances) > 0 {
			newContainer := corev1.Container{}
			ucs := &updateContainerState{
				us:        s,
				proxyType: "alloydb",
				c:         &newContainer,
				p:         inst,
				cs:        inst.Spec.AlloyDBProxyContainer,
				instances: inst.Spec.AlloyDBInstances,
				envPrefix: "ALLOYDB_PROXY_",
			}
			ucs.init()
			containers = append(containers, newContainer)
		}

		// Add pod annotation for each instance
		k, v := s.updater.PodAnnotation(inst)
		ann[k] = v
	}

	podSpec.Containers = containers

	if len(ann) != 0 {
		wl.SetPodTemplateAnnotations(ann)
	}

	for i := range podSpec.Containers {
		c := &podSpec.Containers[i]
		s.updateContainerEnv(c)
		s.applyContainerVolumes(c)
	}
	s.applyVolumes(&podSpec)

	// only return ConfigError if there were reported
	// errors during processing.
	if len(s.err.details) > 0 {
		return &s.err
	}

	wl.SetPodSpec(podSpec)

	return nil
}

type updateContainerState struct {
	us            *updateState
	proxyType     string
	c             *corev1.Container
	p             *cloudsqlapi.AuthProxyWorkload
	cs            *cloudsqlapi.AuthProxyContainerSpec
	instances     []cloudsqlapi.InstanceSpec
	containerName string
	envPrefix     string
}

// updateContainer Creates or updates the proxy container in the workload's PodSpec
func (s *updateContainerState) init() {
	s.containerName = ContainerName(s.p, s.proxyType)

	// if the c was fully overridden, just use that c.
	if s.cs != nil && s.cs.Container != nil {
		s.cs.Container.DeepCopyInto(s.c)
		s.c.Name = s.containerName
		return
	}

	s.c.Name = s.containerName
	s.c.ImagePullPolicy = "IfNotPresent"

	// always enable http port healthchecks on 0.0.0.0 and structured logs
	s.addHealthCheck()
	s.applyTelemetrySpec()

	// enable the proxy's admin service
	s.addAdminServer()

	// add the user agent
	s.addProxyContainerEnvVar("USER_AGENT", s.us.updater.userAgent)

	// configure structured logs
	s.addProxyContainerEnvVar("STRUCTURED_LOGS", "true")

	s.applyContainerSpec()

	// Build the c
	var cliArgs []string

	// Instances
	for i := range s.instances {
		inst := &s.instances[i]
		params := map[string]string{}

		// if it is a TCP socket
		if inst.UnixSocketPath == "" {

			port := s.us.useInstancePort(s.p, inst)
			params["port"] = fmt.Sprint(port)
			if inst.HostEnvName != "" {
				s.us.addWorkloadEnvVar(s.p, inst, corev1.EnvVar{
					Name:  inst.HostEnvName,
					Value: "127.0.0.1",
				})
			}
			if inst.PortEnvName != "" {
				s.us.addWorkloadEnvVar(s.p, inst, corev1.EnvVar{
					Name:  inst.PortEnvName,
					Value: fmt.Sprint(port),
				})
			}
		} else {
			// else if it is a unix socket
			params["unix-socket-path"] = inst.UnixSocketPath
			mountName := VolumeName(s.p, inst, "unix")
			s.us.addVolumeMount(s.p, inst,
				corev1.VolumeMount{
					Name:      mountName,
					ReadOnly:  false,
					MountPath: path.Dir(inst.UnixSocketPath),
				},
				corev1.Volume{
					Name:         mountName,
					VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
				})

			if inst.UnixSocketPathEnvName != "" {
				s.us.addWorkloadEnvVar(s.p, inst, corev1.EnvVar{
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
	s.c.Args = cliArgs
}

func (s *updateContainerState) addProxyContainerEnvVar(k, v string) {
	s.us.addEnvVar(s.p, managedEnvVar{
		Instance: proxyInstanceID{
			AuthProxyWorkload: types.NamespacedName{
				Namespace: s.p.Namespace,
				Name:      s.p.Name,
			},
		},
		ContainerName:        s.containerName,
		OperatorManagedValue: corev1.EnvVar{Name: s.envPrefix + k, Value: v},
	})
}

// applyContainerSpec applies settings from cloudsqlapi.AuthProxyContainerSpec
// to the container
func (s *updateContainerState) applyContainerSpec() {
	if s.proxyType == "alloydb" {
		s.c.Image = DefaultAlloyDBProxyImage
	} else {
		s.c.Image = s.us.defaultProxyImage()
	}
	s.c.Resources = defaultContainerResources

	if s.cs == nil {
		return
	}

	if s.cs.Image != "" {
		s.c.Image = s.cs.Image
	}

	if s.cs.Resources != nil {
		s.c.Resources = *s.cs.Resources.DeepCopy()
	}

	if s.cs.SQLAdminAPIEndpoint != "" {
		s.addProxyContainerEnvVar("SQLADMIN_API_ENDPOINT", s.cs.SQLAdminAPIEndpoint)
	}
	if s.cs.MaxConnections != nil &&
		*s.cs.MaxConnections != 0 {
		s.addProxyContainerEnvVar("MAX_CONNECTIONS", fmt.Sprintf("%d", *s.cs.MaxConnections))
	}
	if s.cs.MaxSigtermDelay != nil &&
		*s.cs.MaxSigtermDelay != 0 {
		s.addProxyContainerEnvVar("MAX_SIGTERM_DELAY", fmt.Sprintf("%d", *s.cs.MaxSigtermDelay))
	}

	return
}

// applyTelemetrySpec applies settings from cloudsqlapi.TelemetrySpec
// to the container
func (s *updateContainerState) applyTelemetrySpec() {
	if s.cs == nil || s.cs.Telemetry == nil {
		return
	}
	tel := s.cs.Telemetry

	if tel.TelemetrySampleRate != nil {
		s.addProxyContainerEnvVar("TELEMETRY_SAMPLE_RATE", fmt.Sprintf("%d", *tel.TelemetrySampleRate))
	}
	if tel.DisableTraces != nil && *tel.DisableTraces {
		s.addProxyContainerEnvVar("DISABLE_TRACES", "true")
	}
	if tel.DisableMetrics != nil && *tel.DisableMetrics {
		s.addProxyContainerEnvVar("DISABLE_METRICS", "true")
	}
	if tel.PrometheusNamespace != nil || (tel.Prometheus != nil && *tel.Prometheus) {
		s.addProxyContainerEnvVar("PROMETHEUS", "true")
	}
	if tel.PrometheusNamespace != nil {
		s.addProxyContainerEnvVar("PROMETHEUS_NAMESPACE", *tel.PrometheusNamespace)
	}
	if tel.TelemetryProject != nil {
		s.addProxyContainerEnvVar("TELEMETRY_PROJECT", *tel.TelemetryProject)
	}
	if tel.TelemetryPrefix != nil {
		s.addProxyContainerEnvVar("TELEMETRY_PREFIX", *tel.TelemetryPrefix)
	}
	if tel.QuotaProject != nil {
		s.addProxyContainerEnvVar("QUOTA_PROJECT", *tel.QuotaProject)
	}

	return
}

// updateContainerEnv applies global container state to all containers
func (s *updateState) updateContainerEnv(c *corev1.Container) {
	for i := 0; i < len(s.mods.EnvVars); i++ {
		var found bool
		v := s.mods.EnvVars[i]
		operatorEnv := v.OperatorManagedValue

		// If this EnvVar is not for this container and not for all containers
		// don't add it to this container.
		if v.ContainerName != c.Name && v.ContainerName != "" {
			continue
		}

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
func (s *updateContainerState) addHealthCheck() {
	var portPtr *int32

	// if the TelemetrySpec.exists, get Port and Port values
	if s.cs != nil && s.cs.Telemetry != nil {
		if s.cs.Telemetry.HTTPPort != nil {
			portPtr = s.cs.Telemetry.HTTPPort
		}
	}

	port := s.us.usePort(portPtr, DefaultHealthCheckPort, s.p)

	s.c.StartupProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
			Port: intstr.IntOrString{IntVal: port},
			Path: "/startup",
		}},
		PeriodSeconds: 30,
	}
	s.c.ReadinessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
			Port: intstr.IntOrString{IntVal: port},
			Path: "/readiness",
		}},
		PeriodSeconds: 30,
	}
	s.c.LivenessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
			Port: intstr.IntOrString{IntVal: port},
			Path: "/liveness",
		}},
		PeriodSeconds: 30,
	}
	// Add a port that is associated with the proxy, but not a specific db instance
	s.us.addProxyPort(port, s.p)
	s.addProxyContainerEnvVar("HTTP_PORT", fmt.Sprintf("%d", port))
	s.addProxyContainerEnvVar("HTTP_ADDRESS", "0.0.0.0")
	s.addProxyContainerEnvVar("HEALTH_CHECK", "true")
}

func (s *updateContainerState) addAdminServer() {

	if s.cs == nil || s.cs.AdminServer == nil {
		return
	}

	a := s.cs.AdminServer
	s.us.addProxyPort(a.Port, s.p)
	s.addProxyContainerEnvVar("ADMIN_PORT", fmt.Sprintf("%d", a.Port))
	for _, name := range a.EnableAPIs {
		switch name {
		case "Debug":
			s.addProxyContainerEnvVar("DEBUG", "true")
		case "QuitQuitQuit":
			s.addProxyContainerEnvVar("QUITQUITQUIT", "true")
		}
	}

}

func (s *updateState) addVolumeMount(p *cloudsqlapi.AuthProxyWorkload, is *cloudsqlapi.InstanceSpec, m corev1.VolumeMount, v corev1.Volume) {
	key := proxyInstanceID{
		AuthProxyWorkload: types.NamespacedName{
			Namespace: p.Namespace,
			Name:      p.Name,
		},
		ConnectionString: is.ConnectionString,
	}
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
		if mount.VolumeMount.MountPath == vol.VolumeMount.MountPath {
			// avoid adding volume mounts with redundant MountPaths,
			// just the first one is enough.
			return
		}
	}
	s.mods.VolumeMounts = append(s.mods.VolumeMounts, vol)
}

// applyContainerVolumes applies all the VolumeMounts to this container.
func (s *updateState) applyContainerVolumes(c *corev1.Container) {
	nameAccessor := func(v corev1.VolumeMount) string {
		return v.Name
	}
	thingAccessor := func(v *managedVolume) corev1.VolumeMount {
		return v.VolumeMount
	}
	c.VolumeMounts = applyVolumeThings[corev1.VolumeMount](s, c.VolumeMounts, nameAccessor, thingAccessor)
}

// applyVolumes applies all volumes to this PodSpec.
func (s *updateState) applyVolumes(ps *corev1.PodSpec) {
	nameAccessor := func(v corev1.Volume) string {
		return v.Name
	}
	thingAccessor := func(v *managedVolume) corev1.Volume {
		return v.Volume
	}
	ps.Volumes = applyVolumeThings[corev1.Volume](s, ps.Volumes, nameAccessor, thingAccessor)
}

// applyVolumeThings modifies a slice of Volume/VolumeMount, to include all the
// shared volumes for the proxy container's unix sockets. This will replace
// an existing volume with the same name, or append a new volume to the slice.
func applyVolumeThings[T corev1.VolumeMount | corev1.Volume](
	s *updateState,
	newVols []T,
	nameAccessor func(T) string,
	thingAccessor func(*managedVolume) T) []T {

	// add or replace items for all new volume mounts
	for i := 0; i < len(s.mods.VolumeMounts); i++ {
		var found bool
		newVol := thingAccessor(s.mods.VolumeMounts[i])
		for j := 0; j < len(newVols); j++ {
			if nameAccessor(newVol) == nameAccessor(newVols[j]) {
				found = true
				newVols[j] = newVol
			}
		}
		if !found {
			newVols = append(newVols, newVol)
		}
	}
	return newVols
}

func (s *updateState) addError(errorCode, description string, p *cloudsqlapi.AuthProxyWorkload) {
	s.err.add(errorCode, description, p)
}

func (s *updateState) defaultProxyImage() string {
	return s.updater.defaultProxyImage
}

func (s *updateState) usePort(configValue *int32, defaultValue int32, p *cloudsqlapi.AuthProxyWorkload) int32 {
	if configValue != nil {
		s.addProxyPort(*configValue, p)
		return *configValue
	}

	port := defaultValue
	if configValue == nil {
		for s.isPortInUse(port) {
			port++
		}
	}
	s.addProxyPort(port, p)
	return port
}
