// Copyright 2022 Google LLC.
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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	// ErrorCodePortConflict occurs when an explicit port assignment for a workload
	// is in conflict with a port assignment from the pod or another proxy container.
	ErrorCodePortConflict = "PortConflict"

	// ErrorCodeEnvConflict occurs when an the environment code does not work.
	ErrorCodeEnvConflict = "EnvVarConflict"

	// AnnotationPrefix is used as the prefix for all annotations added to a domain object.
	// to hold metadata related to this operator.
	AnnotationPrefix = "cloudsql.cloud.google.com"

	// ConditionUpToDate indicates whether the reconciliation loop
	// has properly processed the latest generation of an AuthProxyInstance
	ConditionUpToDate = "UpToDate"

	// ReasonStartedReconcile relates to condition UpToDate, this reason is set
	// when the resource is not up to date because reconcile has started, but not
	// finished.
	ReasonStartedReconcile = "StartedReconcile"

	// ReasonFinishedReconcile relates to condition UpToDate, this reason is set
	// when the resource reconcile has finished running.
	ReasonFinishedReconcile = "FinishedReconcile"

	// ReasonWorkloadNeedsUpdate relates to condition UpToDate, this reason is set
	// when the resource reconcile found existing workloads related to this
	// AuthProxyWorkload resource that are not yet configured with an up-to-date
	// proxy configuration.
	ReasonWorkloadNeedsUpdate = "WorkloadNeedsUpdate"

	// ReasonNoWorkloadsFound relates to condition UpToDate, this reason is set
	// when there are no workloads related to this AuthProxyWorkload resource.
	ReasonNoWorkloadsFound = "NoWorkloadsFound"

	// ConditionWorkloadUpToDate indicates whether the reconciliation loop
	// has properly processed the latest generation of an AuthProxyInstance
	ConditionWorkloadUpToDate = "WorkloadUpToDate"

	// ReasonUpToDate relates to condition WorkloadUpToDate, this reason is set
	// when there are no workloads related to this AuthProxyWorkload resource.
	ReasonUpToDate = "UpToDate"

	// WorkloadStrategy is the RolloutStrategy value that indicates that
	// when the AuthProxyWorkload is updated or deleted, the changes should be
	// applied to affected workloads (Deployments, StatefulSets, etc.) following
	// the Strategy defined by that workload.
	// See: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#strategy
	WorkloadStrategy = "Workload"

	// NoneStrategy is the RolloutStrategy value that indicates that the.
	// when the AuthProxyWorkload is updated or deleted, no action should be taken
	// by the operator to update the affected workloads.
	NoneStrategy = "None"
)

// AuthProxyWorkload declares how a Cloud SQL Proxy container should be applied
// to a matching set of workloads, and shows the status of those proxy containers.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type AuthProxyWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuthProxyWorkloadSpec   `json:"spec,omitempty"`
	Status AuthProxyWorkloadStatus `json:"status,omitempty"`
}

// AuthProxyWorkloadSpec describes where and how to configure the proxy.
type AuthProxyWorkloadSpec struct {
	// Workload selects the workload where the proxy container will be added.
	//+kubebuilder:validation:Required
	Workload WorkloadSelectorSpec `json:"workloadSelector"`

	// Instances describes the Cloud SQL instances to configure on the proxy container.
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:MinItems=1
	Instances []InstanceSpec `json:"instances"`

	// AuthProxyContainer describes the resources and config for the Auth Proxy container.
	//+kubebuilder:validation:Optional
	AuthProxyContainer *AuthProxyContainerSpec `json:"authProxyContainer,omitempty"`
}

// WorkloadSelectorSpec describes which workloads should be configured with this
// proxy configuration. To be valid, WorkloadSelectorSpec must specify `kind`
// and either `name` or `selector`.
type WorkloadSelectorSpec struct {
	// Selector (optional) selects resources using labels. See "Label selectors" in the kubernetes docs
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	//+kubebuilder:validation:Optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// Kind specifies what kind of workload
	// Supported kinds: Deployment, StatefulSet, Pod, ReplicaSet,DaemonSet, Job, CronJob
	// Example: "Deployment" "Deployment.v1" or "Deployment.v1.apps".
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Pattern=\w+(\.\w+)*
	Kind string `json:"kind"`

	// Name specifies the name of the resource to select.
	//+kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`
}

// LabelsSelector converts the Selector field into a controller-runtime labels.Selector
// for convenient use in the controller. If the Selector field is nil, returns
// an empty selector which will match all labels.
func (s *WorkloadSelectorSpec) LabelsSelector() (labels.Selector, error) {
	if s.Selector == nil {
		return labels.NewSelector(), nil
	}

	return metav1.LabelSelectorAsSelector(s.Selector)
}

// AuthProxyContainerSpec describes how to configure global proxy configuration and
// kubernetes-specific container configuration.
type AuthProxyContainerSpec struct {

	// Container is debugging parameter that when specified will override the
	// proxy container with a completely custom Container spec.
	//+kubebuilder:validation:Optional
	Container *corev1.Container `json:"container,omitempty"`

	// Resources specifies the resources required for the proxy pod.
	//+kubebuilder:validation:Optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// Telemetry specifies how the proxy should expose telemetry.
	// Optional, by default
	//+kubebuilder:validation:Optional
	Telemetry *TelemetrySpec `json:"telemetry,omitempty"`

	// AdminServer specifies the config for the proxy's admin service which is
	// available to other containers in the same pod.
	AdminServer *AdminServerSpec `json:"adminServer,omitempty"`

	// Authentication specifies the config for how the proxy authenticates itself
	// to the Google Cloud API.
	Authentication *AuthenticationSpec `json:"authentication,omitempty"`

	// MaxConnections limits the number of connections. Default value is no limit.
	// This sets the proxy container's CLI argument `--max-connections`
	//+kubebuilder:validation:Optional
	//+kubebuilder:validation:Minimum=0
	MaxConnections *int64 `json:"maxConnections,omitempty"`

	// MaxSigtermDelay is the maximum number of seconds to wait for connections to
	// close after receiving a TERM signal. This sets the proxy container's
	// CLI argument `--max-sigterm-delay` and
	// configures `terminationGracePeriodSeconds` on the workload's PodSpec.
	//+kubebuilder:validation:Optional
	//+kubebuilder:validation:Minimum=0
	MaxSigtermDelay *int64 `json:"maxSigtermDelay,omitempty"`

	// SQLAdminAPIEndpoint is a debugging parameter that when specified will
	// change the Google Cloud api endpoint used by the proxy.
	//+kubebuilder:validation:Optional
	SQLAdminAPIEndpoint string `json:"sqlAdminAPIEndpoint,omitempty"`

	// Image is the URL to the proxy image. Optional, by default the operator
	// will use the latest Cloud SQL Auth Proxy version as of the release of the
	// operator.
	//
	// The operator ensures that all workloads configured with the default proxy
	// image are upgraded automatically to use to the latest released proxy image.
	//
	// When the customer upgrades the operator, the operator upgrades all
	// workloads using the default proxy image to the latest proxy image. The
	// change to the proxy container image is applied in accordance with
	// the RolloutStrategy.
	//
	//+kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`

	// RolloutStrategy indicates the strategy to use when rolling out changes to
	// the workloads affected by the results. When this is set to
	// `Workload`, changes to this resource will be automatically applied
	// to a running Deployment, StatefulSet, DaemonSet, or ReplicaSet in
	// accordance with the Strategy set on that workload. When this is set to
	// `None`, the operator will take no action to roll out changes to affected
	// workloads. `Workload` will be used by default if no value is set.
	// See: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#strategy
	//+kubebuilder:validation:Optional
	//+kubebuilder:validation:Enum=Workload;None
	//+kubebuilder:default=Workload
	RolloutStrategy string `json:"rolloutStrategy,omitempty"`
}

// AdminServerSpec specifies how to start the proxy's admin server:
// which port and whether to enable debugging or quitquitquit. It controls
// to the proxy's --admin-port, --debug, and --quitquitquit CLI flags.
type AdminServerSpec struct {

	// Port the port for the proxy's localhost-only admin server.
	// This sets the proxy container's CLI argument `--admin-port`
	//+kubebuilder:validation:required
	//+kubebuilder:validation:Minimum=1
	Port int32 `json:"port,omitempty"`

	// EnableAPIs specifies the list of admin APIs to enable. At least one
	// API must be enabled. Possible values:
	// - "Debug" will enable pprof debugging by setting the `--debug` cli flag.
	// - "QuitQuitQuit" will enable pprof debugging by setting the `--quitquitquit`
	//   cli flag.
	//+kubebuilder:validation:MinItems:=1
	EnableAPIs []string `json:"enableAPIs,omitempty"`
}

// AuthenticationSpec specifies how the proxy is authenticated with the
// Google Cloud SQL Admin API. This configures proxy's
// --impersonate-service-account flag.
type AuthenticationSpec struct {
	// ImpersonationChain is a list of one or more service
	// accounts. The first entry in the chain is the impersonation target. Any
	// additional service accounts after the target are delegates. The
	// roles/iam.serviceAccountTokenCreator must be configured for each account
	// that will be impersonated. This sets the --impersonate-service-account
	// flag on the proxy.
	ImpersonationChain []string `json:"impersonationChain,omitempty"`
}

// TelemetrySpec specifies how the proxy container will expose telemetry.
type TelemetrySpec struct {
	// QuotaProject Specifies the project to use for Cloud SQL Admin API quota tracking.
	// The IAM principal must have the "serviceusage.services.use" permission
	// for the given project. See https://cloud.google.com/service-usage/docs/overview and
	// https://cloud.google.com/storage/docs/requester-pays
	// This sets the proxy container's CLI argument `--quota-project`
	//+kubebuilder:validation:Optional
	QuotaProject *string `json:"quotaProject,omitempty"`

	// Prometheus Enables Prometheus HTTP endpoint /metrics on localhost
	// This sets the proxy container's CLI argument `--prometheus`
	//+kubebuilder:validation:Optional
	Prometheus *bool `json:"prometheus,omitempty"`

	// PrometheusNamespace is used the provided Prometheus namespace for metrics
	// This sets the proxy container's CLI argument `--prometheus-namespace`
	//+kubebuilder:validation:Optional
	PrometheusNamespace *string `json:"prometheusNamespace,omitempty"`

	// TelemetryProject enables Cloud Monitoring and Cloud Trace with the provided project ID.
	// This sets the proxy container's CLI argument `--telemetry-project`
	//+kubebuilder:validation:Optional
	TelemetryProject *string `json:"telemetryProject,omitempty"`

	// TelemetryPrefix is the prefix for Cloud Monitoring metrics.
	// This sets the proxy container's CLI argument `--telemetry-prefix`
	//+kubebuilder:validation:Optional
	TelemetryPrefix *string `json:"telemetryPrefix,omitempty"`

	// TelemetrySampleRate is the Cloud Trace sample rate. A smaller number means more traces.
	// This sets the proxy container's CLI argument `--telemetry-sample-rate`
	//+kubebuilder:validation:Optional
	TelemetrySampleRate *int `json:"telemetrySampleRate,omitempty"`

	// HTTPPort the port for Prometheus and health check server.
	// This sets the proxy container's CLI argument `--http-port`
	//+kubebuilder:validation:Optional
	HTTPPort *int32 `json:"httpPort,omitempty"`

	// DisableTraces disables Cloud Trace testintegration (used with telemetryProject)
	// This sets the proxy container's CLI argument `--disable-traces`
	//+kubebuilder:validation:Optional
	DisableTraces *bool `json:"disableTraces,omitempty"`
	// DisableMetrics disables Cloud Monitoring testintegration (used with telemetryProject)
	// This sets the proxy container's CLI argument `--disable-metrics`
	//+kubebuilder:validation:Optional
	DisableMetrics *bool `json:"disableMetrics,omitempty"`
}

// InstanceSpec describes the configuration for how the proxy should expose
// a Cloud SQL database instance to a workload.
//
// In the minimum recommended configuration, the operator will choose
// a non-conflicting TCP port and set environment
// variables MY_DB_SERVER_PORT MY_DB_SERVER_HOST with the value of the TCP port
// and hostname. The application can read these values to connect to the database
// through the proxy. For example:
//
//	`{
//			   "connectionString":"my-project:us-central1:my-db-server",
//			   "portEnvName":"MY_DB_SERVER_PORT"
//			   "hostEnvName":"MY_DB_SERVER_HOST"
//	}`
//
// If you want to assign a specific port number for a database, set the `port`
// field. For example:
//
//	`{ "connectionString":"my-project:us-central1:my-db-server", "port":5000 }`
type InstanceSpec struct {

	// ConnectionString is the connection string for the Cloud SQL Instance
	// in the format `project_id:region:instance_name`
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Pattern:="^([^:]+(:[^:]+)?):([^:]+):([^:]+)$"
	ConnectionString string `json:"connectionString,omitempty"`

	// Port (optional) sets the tcp port for this instance. If not set, a value will
	// be automatically assigned by the operator and set as an environment variable
	// on all containers in the workload named according to PortEnvName. The operator will choose
	// a port so that it does not conflict with other ports on the workload.
	//+kubebuilder:validation:Optional
	//+kubebuilder:validation:Minimum:=1
	Port *int32 `json:"port,omitempty"`

	// AutoIAMAuthN (optional) Enables IAM Authentication for this instance.
	// Default value is false.
	//+kubebuilder:validation:Optional
	AutoIAMAuthN *bool `json:"autoIAMAuthN,omitempty"`

	// PrivateIP (optional) Enable connection to the Cloud SQL instance's private ip for this instance.
	// Default value is false.
	//+kubebuilder:validation:Optional
	PrivateIP *bool `json:"privateIP,omitempty"`

	// PortEnvName is name of the environment variable containing this instance's tcp port.
	// Optional, when set this environment variable will be added to all containers in the workload.
	//+kubebuilder:validation:Optional
	PortEnvName string `json:"portEnvName,omitempty"`

	// HostEnvName The name of the environment variable containing this instances tcp hostname
	// Optional, when set this environment variable will be added to all containers in the workload.
	//+kubebuilder:validation:Optional
	HostEnvName string `json:"hostEnvName,omitempty"`

	// UnixSocketPath is the path to the unix socket where the proxy will listen
	// for connnections. This will be mounted to all containers in the pod.
	//+kubebuilder:validation:Optional
	UnixSocketPath string `json:"unixSocketPath,omitempty"`

	// UnixSocketPathEnvName is the environment variable containing the value of
	// UnixSocketPath.
	//+kubebuilder:validation:Optional
	UnixSocketPathEnvName string `json:"unixSocketPathEnvName,omitempty"`
}

// AuthProxyWorkloadStatus presents the observed state of AuthProxyWorkload using
// standard Kubernetes Conditions.
type AuthProxyWorkloadStatus struct {
	// Conditions show the overall status of the AuthProxyWorkload resource on all
	// matching workloads.
	//
	// The "UpToDate" condition indicates that the proxy was successfully
	// applied to all matching workloads. See ConditionUpToDate.
	Conditions []*metav1.Condition `json:"conditions,omitempty"`

	// WorkloadStatus presents the observed status of individual workloads that match
	// this AuthProxyWorkload resource.
	WorkloadStatus []*WorkloadStatus `json:"WorkloadStatus,omitempty"`
}

// WorkloadStatus presents the status for how this AuthProxyWorkload resource
// was applied to a specific workload.
type WorkloadStatus struct {

	// Kind Version Namespace Name identify the specific workload.
	//+kubebuilder:validation:Enum=Pod;Deployment;StatefulSet;ReplicaSet;DaemonSet;Job;CronJob
	Kind      string `json:"kind,omitempty,"`
	Version   string `json:"version,omitempty,"`
	Namespace string `json:"namespace,omitempty,"`
	Name      string `json:"name,omitempty,"`

	// Conditions show the status of the AuthProxyWorkload resource on this
	// matching workload.
	//
	// The "UpToDate" condition indicates that the proxy was successfully
	// applied to all matching workloads. See ConditionUpToDate.
	Conditions []*metav1.Condition `json:"conditions"`
}

// AuthProxyWorkloadList contains a list of AuthProxyWorkload and is part of the
// authproxyworkloads API.
// +kubebuilder:object:root=true
type AuthProxyWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuthProxyWorkload `json:"items"`
}

// init registers these resource definitions with the controller-runtime framework.
func init() {
	SchemeBuilder.Register(&AuthProxyWorkload{}, &AuthProxyWorkloadList{})
}
