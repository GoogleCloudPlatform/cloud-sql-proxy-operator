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

// AuthProxyWorkloadSpec defines the desired state of AuthProxyWorkload
type AuthProxyWorkloadSpec struct {
	// Workload selects the workload to
	//+kubebuilder:validation:Required
	Workload WorkloadSelectorSpec `json:"workloadSelector"`

	// AuthProxyContainer describes the resources and config for the Auth Proxy container
	//+kubebuilder:validation:Optional
	AuthProxyContainer *AuthProxyContainerSpec `json:"authProxyContainer,omitempty"`

	// Instances lists the Cloud SQL instances to connect
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:MinItems=1
	Instances []InstanceSpec `json:"instances"`
}

// WorkloadSelectorSpec describes which workloads should be configured with this
// proxy configuration. To be valid, WorkloadSelectorSpec must specify Kind
// and either Name or Selector.
type WorkloadSelectorSpec struct {
	// Selector selects resources using labels. See "Label selectors" in the kubernetes docs
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

// AuthProxyContainerSpec specifies configuration for the proxy container.
type AuthProxyContainerSpec struct {

	// Container is debugging parameter that when specified will override the
	// proxy container with a completely custom Container spec.
	//+kubebuilder:validation:Optional
	Container *v1.Container `json:"container,omitempty"`

	// Resources specifies the resources required for the proxy pod.
	//+kubebuilder:validation:Optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`

	// Telemetry specifies how the proxy should expose telemetry.
	// Optional, by default
	//+kubebuilder:validation:Optional
	Telemetry *TelemetrySpec `json:"telemetry,omitempty"`

	// MaxConnections limits the number of connections. Default value is no limit.
	// This sets the proxy container's CLI argument `--max-connections`
	//+kubebuilder:validation:Optional
	MaxConnections *int64 `json:"maxConnections,omitempty"`

	// MaxSigtermDelay is the maximum number of seconds to wait for connections to close after receiving a TERM signal.
	// This sets the proxy container's CLI argument `--max-sigterm-delay` and
	// configures `terminationGracePeriodSeconds` on the workload's PodSpec.
	//+kubebuilder:validation:Optional
	MaxSigtermDelay *int64 `json:"maxSigtermDelay,omitempty"`

	// SQLAdminAPIEndpoint is a debugging parameter that when specified will
	// change the Google Cloud api endpoint used by the proxy.
	//+kubebuilder:validation:Optional
	SQLAdminAPIEndpoint string `json:"sqlAdminAPIEndpoint,omitempty"`

	// Image is the URL to the proxy image. Optional, by default the operator
	// will use the latest known compatible proxy image.
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

// TelemetrySpec specifies how the proxy container will expose telemetry.
type TelemetrySpec struct {
	// HTTPPort the port for Prometheus and health check server.
	// This sets the proxy container's CLI argument `--http-port`
	//+kubebuilder:validation:Optional
	HTTPPort *int32 `json:"httpPort,omitempty"`

	// AdminPort the port for the /quitquitquit endpoint.
	// This sets the proxy container's CLI argument `--admin-port`
	//+kubebuilder:validation:Optional
	AdminPort *int32 `json:"adminPort,omitempty"`
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
	//+kubebuilder:validation:Required
	ConnectionString string `json:"connectionString,omitempty"`

	// Port sets the tcp port for this instance. Optional, if not set, a value will
	// be automatically assigned by the operator and set as an environment variable
	// on all containers in the workload named according to PortEnvName. The operator will choose
	// a port so that it does not conflict with other ports on the workload.
	//+kubebuilder:validation:Optional
	Port *int32 `json:"port,omitempty"`

	// AutoIAMAuthN Enables IAM Authentication for this instance. Optional, default
	// false.
	//+kubebuilder:validation:Optional
	AutoIAMAuthN *bool `json:"autoIAMAuthN,omitempty"`

	// PrivateIP Enable connection to the Cloud SQL instance's private ip for this instance.
	// Optional, default false.
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
