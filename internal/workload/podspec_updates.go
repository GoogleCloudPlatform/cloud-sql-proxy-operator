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
	DefaultProxyImage = "gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.11.4"

	// DefaultFirstPort is the first port number chose for an instance listener by the
	// proxy.
	DefaultFirstPort int32 = 5000

	// DefaultHealthCheckPort is the used by the proxy to expose prometheus
	// and kubernetes health checks.
	DefaultHealthCheckPort int32 = 9801

	// DefaultAdminPort is the used by the proxy to expose the quitquitquit
	// and debug api endpoints
	DefaultAdminPort int32 = 9091
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

	// starting with this pod, traverse the pod and its owners, and
	// fill wls with a list of workload resources that match an AuthProxyWorkload
	// in the pl.
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
		}
	}
	return matchingAuthProxyWorkloads
}

// CheckWorkloadContainers determines if a pod is configured incorrectly and
// therefore needs to be deleted. Pods must be (1) missing one or more proxy
// sidecar containers and (2) have a terminated container.
func (u *Updater) CheckWorkloadContainers(wl *PodWorkload, matches []*cloudsqlapi.AuthProxyWorkload) error {

	// Find the names of all AuthProxyWorkload resources that should have a
	// container on this pod, but there is no container.
	var missing []string
	for _, p := range matches {
		wantName := ContainerName(p)
		var found bool
		for _, c := range wl.PodSpec().Containers {
			if c.Name == wantName {
				found = true
			}
		}
		if !found {
			missing = append(missing, p.Name)
			break
		}
	}

	// If no containers are missing, then there is no error, return nil.
	if len(missing) == 0 {
		return nil
	}

	missingSidecars := strings.Join(missing, ", ")

	// Some proxy containers are missing. Are the remaining pod containers failing?
	for _, cs := range wl.Pod.Status.ContainerStatuses {
		if cs.State.Terminated != nil && cs.State.Terminated.Reason == "Error" {
			return fmt.Errorf("pod is in an error state and missing sidecar containers %v", missingSidecars)
		}
		if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
			return fmt.Errorf("pod is in a CrashLoopBackOff state and missing sidecar containers %v", missingSidecars)
		}
	}

	// Pod's other containers are not in an error state. Operator should not
	// interrupt running containers.
	return nil
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
	DBInstances  []*proxyInstanceID `json:"dbInstances"`
	EnvVars      []*managedEnvVar   `json:"envVars"`
	VolumeMounts []*managedVolume   `json:"volumeMounts"`
	Ports        []*managedPort     `json:"ports"`
	AdminPorts   []int32            `json:"adminPorts"`
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

func (s *updateState) addAdminPort(p int32) {
	s.mods.AdminPorts = append(s.mods.AdminPorts, p)
}

func (s *updateState) addQuitEnvVar() {
	urls := make([]string, len(s.mods.AdminPorts))
	for i := 0; i < len(s.mods.AdminPorts); i++ {
		urls[i] = fmt.Sprintf("http://localhost:%d/quitquitquit", s.mods.AdminPorts[i])
	}
	v := strings.Join(urls, " ")

	s.addEnvVar(nil, managedEnvVar{
		OperatorManagedValue: corev1.EnvVar{
			Name:  "CSQL_PROXY_QUIT_URLS",
			Value: v,
		}})
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

func (s *updateState) addProxyContainerEnvVar(p *cloudsqlapi.AuthProxyWorkload, k, v string) {
	s.addEnvVar(p, managedEnvVar{
		Instance: proxyInstanceID{
			AuthProxyWorkload: types.NamespacedName{
				Namespace: p.Namespace,
				Name:      p.Name,
			},
		},
		ContainerName:        ContainerName(p),
		OperatorManagedValue: corev1.EnvVar{Name: k, Value: v},
	})
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

func (s *updateState) initState(pl []*cloudsqlapi.AuthProxyWorkload) {
	// Reset the mods.DBInstances to the list of pl being
	// applied right now.
	s.mods.DBInstances = make([]*proxyInstanceID, 0, len(pl))
	for _, wl := range pl {
		for _, instance := range wl.Spec.Instances {
			s.mods.DBInstances = append(s.mods.DBInstances,
				&proxyInstanceID{
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

		newContainer := corev1.Container{}
		s.updateContainer(inst, &newContainer)
		containers = append(containers, newContainer)

		// Add pod annotation for each instance
		k, v := s.updater.PodAnnotation(inst)
		ann[k] = v
	}
	// Add the envvar containing the proxy quit urls to the workloads
	s.addQuitEnvVar()

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

// updateContainer Creates or updates the proxy container in the workload's PodSpec
func (s *updateState) updateContainer(p *cloudsqlapi.AuthProxyWorkload, c *corev1.Container) {
	// if the c was fully overridden, just use that c.
	if p.Spec.AuthProxyContainer != nil && p.Spec.AuthProxyContainer.Container != nil {
		p.Spec.AuthProxyContainer.Container.DeepCopyInto(c)
		c.Name = ContainerName(p)
		return
	}

	// always enable http port healthchecks on 0.0.0.0 and structured logs
	s.addHealthCheck(p, c)
	s.applyTelemetrySpec(p)

	// enable the proxy's admin service
	s.addAdminServer(p)

	// configure container authentication
	s.addAuthentication(p)

	// add the user agent
	s.addProxyContainerEnvVar(p, "CSQL_PROXY_USER_AGENT", s.updater.userAgent)

	// configure structured logs
	s.addProxyContainerEnvVar(p, "CSQL_PROXY_STRUCTURED_LOGS", "true")

	// configure quiet logs
	if p.Spec.AuthProxyContainer != nil && p.Spec.AuthProxyContainer.Quiet {
		s.addProxyContainerEnvVar(p, "CSQL_PROXY_QUIET", "true")
	}

	// configure lazy refresh
	if p.Spec.AuthProxyContainer != nil && p.Spec.AuthProxyContainer.RefreshStrategy == cloudsqlapi.RefreshStrategyLazy {
		s.addProxyContainerEnvVar(p, "CSQL_PROXY_LAZY_REFRESH", "true")
	}

	c.Name = ContainerName(p)
	c.ImagePullPolicy = "IfNotPresent"

	s.applyContainerSpec(p, c)

	// Build the c
	var cliArgs []string

	// Instances
	for i := range p.Spec.Instances {
		inst := &p.Spec.Instances[i]
		params := map[string]string{}

		// if it is a TCP socket
		if inst.UnixSocketPath == "" {

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
			params["unix-socket-path"] = inst.UnixSocketPath
			mountName := VolumeName(p, inst, "unix")
			s.addVolumeMount(p, inst,
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

		if inst.PSC != nil {
			if *inst.PSC {
				params["psc"] = "true"
			} else {
				params["psc"] = "false"
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
func (s *updateState) applyContainerSpec(p *cloudsqlapi.AuthProxyWorkload, c *corev1.Container) {
	t := true
	var f bool
	c.Image = s.defaultProxyImage()
	c.Resources = defaultContainerResources
	c.SecurityContext = &corev1.SecurityContext{
		// The default Cloud SQL Auth Proxy image runs as the
		// "nonroot" user and group (uid: 65532) by default.
		RunAsNonRoot: &t,
		// Use a read-only filesystem
		ReadOnlyRootFilesystem: &t,
		// Do not allow privilege escalation
		AllowPrivilegeEscalation: &f,
	}

	if p.Spec.AuthProxyContainer == nil {
		return
	}

	if p.Spec.AuthProxyContainer.Image != "" {
		c.Image = p.Spec.AuthProxyContainer.Image
	}

	if p.Spec.AuthProxyContainer.Resources != nil {
		c.Resources = *p.Spec.AuthProxyContainer.Resources.DeepCopy()
	}

	if p.Spec.AuthProxyContainer.SQLAdminAPIEndpoint != "" {
		s.addProxyContainerEnvVar(p, "CSQL_PROXY_SQLADMIN_API_ENDPOINT", p.Spec.AuthProxyContainer.SQLAdminAPIEndpoint)
	}
	if p.Spec.AuthProxyContainer.MaxConnections != nil &&
		*p.Spec.AuthProxyContainer.MaxConnections != 0 {
		s.addProxyContainerEnvVar(p, "CSQL_PROXY_MAX_CONNECTIONS", fmt.Sprintf("%d", *p.Spec.AuthProxyContainer.MaxConnections))
	}
	if p.Spec.AuthProxyContainer.MaxSigtermDelay != nil &&
		*p.Spec.AuthProxyContainer.MaxSigtermDelay != 0 {
		s.addProxyContainerEnvVar(p, "CSQL_PROXY_MAX_SIGTERM_DELAY", fmt.Sprintf("%ds", *p.Spec.AuthProxyContainer.MaxSigtermDelay))
	}

	return
}

// applyTelemetrySpec applies settings from cloudsqlapi.TelemetrySpec
// to the container
func (s *updateState) applyTelemetrySpec(p *cloudsqlapi.AuthProxyWorkload) {
	if p.Spec.AuthProxyContainer == nil || p.Spec.AuthProxyContainer.Telemetry == nil {
		return
	}
	tel := p.Spec.AuthProxyContainer.Telemetry

	if tel.TelemetrySampleRate != nil {
		s.addProxyContainerEnvVar(p, "CSQL_PROXY_TELEMETRY_SAMPLE_RATE", fmt.Sprintf("%d", *tel.TelemetrySampleRate))
	}
	if tel.DisableTraces != nil && *tel.DisableTraces {
		s.addProxyContainerEnvVar(p, "CSQL_PROXY_DISABLE_TRACES", "true")
	}
	if tel.DisableMetrics != nil && *tel.DisableMetrics {
		s.addProxyContainerEnvVar(p, "CSQL_PROXY_DISABLE_METRICS", "true")
	}
	if tel.PrometheusNamespace != nil || (tel.Prometheus != nil && *tel.Prometheus) {
		s.addProxyContainerEnvVar(p, "CSQL_PROXY_PROMETHEUS", "true")
	}
	if tel.PrometheusNamespace != nil {
		s.addProxyContainerEnvVar(p, "CSQL_PROXY_PROMETHEUS_NAMESPACE", *tel.PrometheusNamespace)
	}
	if tel.TelemetryProject != nil {
		s.addProxyContainerEnvVar(p, "CSQL_PROXY_TELEMETRY_PROJECT", *tel.TelemetryProject)
	}
	if tel.TelemetryPrefix != nil {
		s.addProxyContainerEnvVar(p, "CSQL_PROXY_TELEMETRY_PREFIX", *tel.TelemetryPrefix)
	}
	if tel.QuotaProject != nil {
		s.addProxyContainerEnvVar(p, "CSQL_PROXY_QUOTA_PROJECT", *tel.QuotaProject)
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
func (s *updateState) addHealthCheck(p *cloudsqlapi.AuthProxyWorkload, c *corev1.Container) int32 {
	var portPtr *int32
	var adminPortPtr *int32

	cs := p.Spec.AuthProxyContainer

	// if the TelemetrySpec.exists, get Port and Port values
	if cs != nil && cs.Telemetry != nil {
		if cs.Telemetry.HTTPPort != nil {
			portPtr = cs.Telemetry.HTTPPort
		}
	}

	port := s.usePort(portPtr, DefaultHealthCheckPort, p)

	c.StartupProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
			Port: intstr.IntOrString{IntVal: port},
			Path: "/startup",
		}},
		PeriodSeconds:    1,
		FailureThreshold: 60,
		TimeoutSeconds:   10,
	}
	c.LivenessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{
			Port: intstr.IntOrString{IntVal: port},
			Path: "/liveness",
		}},
		PeriodSeconds:    10,
		FailureThreshold: 3,
		TimeoutSeconds:   10,
	}

	// Add a port that is associated with the proxy, but not a specific db instance
	s.addProxyPort(port, p)
	s.addProxyContainerEnvVar(p, "CSQL_PROXY_HTTP_PORT", fmt.Sprintf("%d", port))
	s.addProxyContainerEnvVar(p, "CSQL_PROXY_HTTP_ADDRESS", "0.0.0.0")
	s.addProxyContainerEnvVar(p, "CSQL_PROXY_HEALTH_CHECK", "true")
	// For graceful exits as a sidecar, the proxy should exit with exit code 0
	// when it receives a SIGTERM.
	s.addProxyContainerEnvVar(p, "CSQL_PROXY_EXIT_ZERO_ON_SIGTERM", "true")

	// Add a containerPort declaration for the healthcheck & telemetry port
	c.Ports = append(c.Ports, corev1.ContainerPort{
		ContainerPort: port,
		Protocol:      corev1.ProtocolTCP,
	})

	// Also the operator will enable the /quitquitquit endpoint for graceful exit.
	// If the AdminServer.Port is set, use it, otherwise use the default
	// admin port.
	if cs != nil && cs.AdminServer != nil && cs.AdminServer.Port != 0 {
		adminPortPtr = &cs.AdminServer.Port
	}
	adminPort := s.usePort(adminPortPtr, DefaultAdminPort, p)
	s.addAdminPort(adminPort)
	s.addProxyContainerEnvVar(p, "CSQL_PROXY_QUITQUITQUIT", "true")
	s.addProxyContainerEnvVar(p, "CSQL_PROXY_ADMIN_PORT", fmt.Sprintf("%d", adminPort))

	// Configure the pre-stop hook for /quitquitquit
	c.Lifecycle = &corev1.Lifecycle{
		PreStop: &corev1.LifecycleHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Port: intstr.IntOrString{IntVal: adminPort},
				Path: "/quitquitquit",
				Host: "localhost",
			},
		},
	}
	return adminPort
}

func (s *updateState) addAdminServer(p *cloudsqlapi.AuthProxyWorkload) {

	if p.Spec.AuthProxyContainer == nil || p.Spec.AuthProxyContainer.AdminServer == nil {
		return
	}

	cs := p.Spec.AuthProxyContainer.AdminServer
	for _, name := range cs.EnableAPIs {
		switch name {
		case "Debug":
			s.addProxyContainerEnvVar(p, "CSQL_PROXY_DEBUG", "true")
		}
	}

}

func (s *updateState) addAuthentication(p *cloudsqlapi.AuthProxyWorkload) {
	if p.Spec.AuthProxyContainer == nil || p.Spec.AuthProxyContainer.Authentication == nil {
		return
	}
	as := p.Spec.AuthProxyContainer.Authentication
	if len(as.ImpersonationChain) > 0 {
		s.addProxyContainerEnvVar(p, "CSQL_PROXY_IMPERSONATE_SERVICE_ACCOUNT", strings.Join(as.ImpersonationChain, ","))
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
