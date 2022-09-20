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

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ConditionUpToDate indicates whether the reconciliation loop
// has properly processed the latest generation of an AuthProxyInstance
const ConditionUpToDate = "UpToDate"

// AuthProxyWorkloadSpec defines the desired state of AuthProxyWorkload
type AuthProxyWorkloadSpec struct {
	// Workload selects the workload to
	// +kubebuilder:validation:Required
	Workload WorkloadSelectorSpec `json:"workloadSelector,required"`

	// Authentication describes how to authenticate the Auth Proxy container to Google Cloud
	// +kubebuilder:validation:Optional
	Authentication *AuthenticationSpec `json:"authentication,omitempty"`

	// AuthProxyContainer describes the resources and config for the Auth Proxy container
	// +kubebuilder:validation:Optional
	AuthProxyContainer *AuthProxyContainerSpec `json:"authProxyContainer,omitempty"`

	// Instances lists the Cloud SQL instances to connect
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Instances []InstanceSpec `json:"instances,required"`
}

// WorkloadSelectorSpec describes which workloads should be configured with this
// proxy configuration. To be valid, WorkloadSelectorSpec must specify Kind
// and either Name or Selector.
type WorkloadSelectorSpec struct {
	// Kind specifies what kind of workload
	// Supported kinds: Deployment, StatefulSet, Pod, DaemonSet, Job, CronJob
	// Example: "Deployment" "Deployment.v1" or "Deployment.v1.apps".
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=\w+(\.\w+)*
	Kind string `json:"kind,required"`

	// Namespace specifies namespace in which to select the resource.
	// Optional, defaults to the namespace of the AuthProxyWorkload resource.
	// All or Wildcard namespaces are not supported.
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`

	// Name specifies the name of the resource to select.
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`

	// Selector selects resources using labels. See "Label selectors" in the kubernetes docs
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	// +kubebuilder:validation:Optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
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

// AuthenticationSpec describes how the proxy should get its Google Cloud identity
// to authenticate to the Google Cloud api. The proxy can get its Google Cloud
// identity in one of two ways:
//
//  1. Using the Google Cloud metadata server, in which case the AuthenticationSpec
//     would set the GCloudAuth field to true. e.g. `{gcloudAuth:true}`
//  2. Using a IAM credential key file stored in a kubernetes secret, in which
//     case the AuthenticationSpec would set CredentialFileSecret and CredentialFileKey.
//     e.g. `{credentialFileSecret: "default/gcloud-cred", credentialFileKey="gcloud.json"}`
type AuthenticationSpec struct {
	// GCloudAuth true when we should use the Google Cloud metadata server to authenticate.
	// This sets the Cloud SQL Proxy container's CLI argument `--gcloud-auth`
	// +kubebuilder:validation:Optional
	GCloudAuth bool `json:"gcloudAuth,omitempty"`

	// CredentialsFileSecret the "name" or "namespace/name" for the secret.
	// This sets the Cloud SQL Proxy container's CLI argument `--credentials-file`
	// +kubebuilder:validation:Optional
	CredentialsFileSecret string `json:"credentialsFileSecret,omitempty"`

	// CredentialsFileKey The key within the kubernetes secret containing the credentials file.
	// This sets the Cloud SQL Proxy container's CLI argument `--credentials-file`
	// +kubebuilder:validation:Optional
	CredentialsFileKey string `json:"credentialsFileKey,omitempty"`
}

// AuthProxyContainerSpec specifies configuration for the proxy container.
type AuthProxyContainerSpec struct {
	// FUSEDir is the path where the FUSE volume will be mounted.
	// This sets the proxy container's CLI argument `--fuse` and
	// will mount the FUSE volume at this path on all containers in the workload.
	// +kubebuilder:validation:Optional
	FUSEDir string `json:"fuseDir,omitempty"`

	// FUSETempDir is the path for the temp dir for Unix sockets created with FUSE.
	// This sets the proxy container's CLI argument `--fuse-tmp-dir` and
	// will mount the FUSE temp volume at this path on all containers in the workload.
	// +kubebuilder:validation:Optional
	FUSETempDir string `json:"fuseTempDir,omitempty"`

	// Resources specifies the resources required for the proxy pod.
	// +kubebuilder:validation:Optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`

	// MaxConnections limits the number of connections. Default value is no limit.
	// This sets the proxy container's CLI argument `--max-connections`
	// +kubebuilder:validation:Optional
	MaxConnections *int64 `json:"maxConnections,omitempty"`

	// MaxSigtermDelay is the maximum number of seconds to wait for connections to close after receiving a TERM signal.
	// This sets the proxy container's CLI argument `--max-sigterm-delay` and
	// configures `terminationGracePeriodSeconds` on the workload's PodSpec.
	// +kubebuilder:validation:Optional
	MaxSigtermDelay *int64 `json:"maxSigtermDelay,omitempty"`

	// Image is the URL to the proxy image. Optional, by default the operator
	// will use the latest known compatible proxy image.
	// +kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`

	// Telemetry specifies how the proxy should expose telemetry.
	// Optional, by default
	// +kubebuilder:validation:Optional
	Telemetry *TelemetrySpec `json:"telemetry,omitempty"`

	// SQLAdminAPIEndpoint is a debugging parameter that when specified will
	// change the Google Cloud api endpoint used by the proxy.
	// +kubebuilder:validation:Optional
	SQLAdminAPIEndpoint string `json:"sqlAdminAPIEndpoint,omitempty"`

	// Container is debugging parameter that when specified will override the
	// proxy container with a completely custom Container spec.
	// +kubebuilder:validation:Optional
	Container *v1.Container `json:"container,omitempty"`
}

// TelemetrySpec specifies how the proxy container will expose telemetry.
type TelemetrySpec struct {
	// QuotaProject Specifies the project to use for Cloud SQL Admin API quota tracking.
	// The IAM principal must have the "serviceusage.services.use" permission
	// for the given project. See https://cloud.google.com/service-usage/docs/overview and
	// https://cloud.google.com/storage/docs/requester-pays
	// This sets the proxy container's CLI argument `--quota-project`
	// +kubebuilder:validation:Optional
	QuotaProject string `json:"quotaProject,omitempty"`

	// Prometheus Enables Prometheus HTTP endpoint /metrics on localhost
	// This sets the proxy container's CLI argument `--prometheus`
	// +kubebuilder:validation:Optional
	Prometheus *bool `json:"prometheus,omitempty"`

	// PrometheusNamespace is used the provided Prometheus namespace for metrics
	// This sets the proxy container's CLI argument `--prometheus-namespace`
	// +kubebuilder:validation:Optional
	PrometheusNamespace *string `json:"prometheusNamespace,omitempty"`

	// TelemetryProject enables Cloud Monitoring and Cloud Trace with the provided project ID.
	// This sets the proxy container's CLI argument `--telemetry-project`
	// +kubebuilder:validation:Optional
	TelemetryProject *string `json:"telemetryProject,omitempty"`

	// TelemetryPrefix is the prefix for Cloud Monitoring metrics.
	// This sets the proxy container's CLI argument `--telemetry-prefix`
	// +kubebuilder:validation:Optional
	TelemetryPrefix *string `json:"telemetryPrefix,omitempty"`

	// TelemetrySampleRate is the Cloud Trace sample rate. A smaller number means more traces.
	// This sets the proxy container's CLI argument `--telemetry-sample-rate`
	// +kubebuilder:validation:Optional
	TelemetrySampleRate *int `json:"telemetrySampleRate,omitempty"`

	// HTTPPort the port for Prometheus and health check server.
	// This sets the proxy container's CLI argument `--http-port`
	// +kubebuilder:validation:Optional
	HTTPPort *int32 `json:"httpPort,omitempty"`

	// DisableTraces disables Cloud Trace integration (used with telemetryProject)
	// This sets the proxy container's CLI argument `--disable-traces`
	// +kubebuilder:validation:Optional
	DisableTraces *bool `json:"disableTraces,omitempty"`

	// DisableMetrics disables Cloud Monitoring integration (used with telemetryProject)
	// This sets the proxy container's CLI argument `--disable-metrics`
	// +kubebuilder:validation:Optional
	DisableMetrics *bool `json:"disableMetrics,omitempty"`
}

// InstanceSpec describes the configuration for how the proxy should expose
// a Cloud SQL database instance to a workload. The simplest possible configuration
// declares just the connection string and the port number or unix socket.
//
// For example, for a TCP port:
//
//	{ "connectionString":"my-project:us-central1:my-db-server", "port":5000 }
//
// or for a unix socket:
//
//	{ "connectionString":"my-project:us-central1:my-db-server",
//	  "unixSocketPath" : "/mnt/db/my-db-server" }
//
// You may allow the operator to choose a non-conflicting TCP port or unix socket
// instead of explicitly setting the port or socket path. This may be easier to
// manage when workload needs to connect to many databases.
//
// For example, for a TCP port:
//
//	{ "connectionString":"my-project:us-central1:my-db-server",
//	  "portEnvName":"MY_DB_SERVER_PORT"
//	  "hostEnvName":"MY_DB_SERVER_HOST"
//	 }
//
// will set environment variables MY_DB_SERVER_PORT MY_DB_SERVER_HOST with the
// value of the TCP port and hostname. Then, the application can read these values
// to connect to the database through the proxy.
//
// or for a unix socket:
//
//	{ "connectionString":"my-project:us-central1:my-db-server",
//	  "unixSocketPathEnvName" : "MY_DB_SERVER_SOCKET_DIR" }
//
// will set environment variables MY_DB_SERVER_SOCKET_DIR with the
// value of the unix socket path. Then, the application can read this value
// to connect to the database through the proxy.
type InstanceSpec struct {

	// ConnectionString is the Cloud SQL instance.
	// +kubebuilder:validation:Required
	ConnectionString string `json:"connectionString,omitempty"`

	// SocketType declares what type of socket to create for this database. Allowed
	// values: "tcp" or "unix"
	// +kubebuilder:validation:Enum=tcp;unix
	// +kubebuilder:validation:Optional
	SocketType string `json:"socketType,omitempty"`

	// Port sets the tcp port for this instance. Optional, if not set, a value will
	// be automatically assigned by the operator and set as an environment variable
	// on all containers in the workload named according to PortEnvName. The operator will choose
	// a port so that it does not conflict with other ports on the workload.
	// +kubebuilder:validation:Optional
	Port *int32 `json:"port,omitempty"`

	// UnixSocketPath is the directory to mount the unix socket for this instance.
	// When set, this directory will be mounted on all containers in the workload.
	// +kubebuilder:validation:Optional
	UnixSocketPath string `json:"unixSocketPath,omitempty"`

	// AutoIAMAuthN Enables IAM Authentication for this instance. Optional, default
	// false.
	// +kubebuilder:validation:Optional
	AutoIAMAuthN *bool `json:"autoIAMAuthN,omitempty"`

	// PrivateIP Enable connection to the Cloud SQL instance's private ip for this instance.
	// Optional, default false.
	// +kubebuilder:validation:Optional
	PrivateIP *bool `json:"privateIP,omitempty"`

	// PortEnvName is name of the environment variable containing this instance's tcp port.
	// Optional, when set this environment variable will be added to all containers in the workload.
	// +kubebuilder:validation:Optional
	PortEnvName string `json:"portEnvName,omitempty"`

	// HostEnvName The name of the environment variable containing this instances tcp hostname
	// Optional, when set this environment variable will be added to all containers in the workload.
	// +kubebuilder:validation:Optional
	HostEnvName string `json:"hostEnvName,omitempty"`

	// UnixSocketPathEnvName the name of the environment variable containing the unix socket path
	// Optional, when set this environment variable will be added to all containers in the workload.
	// +kubebuilder:validation:Optional
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
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// WorkloadStatus presents the observed status of individual workloads that match
	// this AuthProxyWorkload resource.
	WorkloadStatus []WorkloadStatus `json:"WorkloadStatus,omitempty"`
}

// WorkloadStatus presents the status for how this AuthProxyWorkload resource
// was applied to a specific workload.
type WorkloadStatus struct {

	// Kind Version Namespace Name identify the specific workload.
	// +kubebuilder:validation:Enum=Pod;Deployment;StatefulSet;DaemonSet;Job;CronJob
	Kind      string `json:"kind,omitempty,"`
	Version   string `json:"version,omitempty,"`
	Namespace string `json:"namespace,omitempty,"`
	Name      string `json:"name,omitempty,"`

	// Conditions show the status of the AuthProxyWorkload resource on this
	// matching workload.
	//
	// The "UpToDate" condition indicates that the proxy was successfully
	// applied to all matching workloads. See ConditionUpToDate.
	Conditions []metav1.Condition `json:"conditions"`
}

// AuthProxyWorkload declares how a Cloud SQL Proxy container should be applied
// to a matching set of workloads, and shows the status of those proxy containers.
// This is the Schema for the authproxyworkloads API.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type AuthProxyWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuthProxyWorkloadSpec   `json:"spec,omitempty"`
	Status AuthProxyWorkloadStatus `json:"status,omitempty"`
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
