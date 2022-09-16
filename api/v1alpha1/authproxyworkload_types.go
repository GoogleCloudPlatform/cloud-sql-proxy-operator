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

// ConditionUpToDate indicates whether the reconcilliation loop
// has properly processed the latest generation of an AuthProxyInstance
const ConditionUpToDate = "UpToDate"

// AuthProxyWorkloadSpec defines the desired state of AuthProxyWorkload
type AuthProxyWorkloadSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Workload selects the workload to
	// +kubebuilder:validation:Required
	Workload WorkloadSelectorSpec `json:"workloadSelector,required"`

	// Authentication describes how to authenticate the proxy container to Google Cloud
	// +kubebuilder:validation:Optional
	Authentication *AuthenticationSpec `json:"authentication,omitempty"`

	// ProxyContainer describes the resources and config for the proxy sidecar container
	// +kubebuilder:validation:Optional
	ProxyContainer *ProxyContainerSpec `json:"proxyContainer,omitempty"`

	// Instances lists the Cloud SQL instances to connect
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Instances []InstanceSpec `json:"instances,required"`
}

// WorkloadSelectorSpec describes which workloads should be configured with this
// proxy configuration.
type WorkloadSelectorSpec struct {
	// Kind the kind of workload where the auth proxy should be configured.
	// Supported kinds: Deployment, StatefulSet, Pod, DaemonSet, Job, CronJob
	// Use a string in the format. Example: "Deployment" "Deployment.v1" or "Deployment.v1.apps".
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=\w+(\.\w+)*
	Kind string `json:"kind,required"`

	// Namespace which namespace in which to select the resource kind Kind in the namespace.
	// Optional, defaults to the namespace of the AuthProxyWorkload.
	// All or Wildcard namespaces are not supported.
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`

	// Name selects a resource of kind Kind by name
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`

	// Selector selects resources by of kind Kind  for
	// +kubebuilder:validation:Optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

// LabelsSelector returns (selector, error) based on the value of the
// Selector field. If the Selector field is nil, returns an empty selector
// which will match all labels.
func (s *WorkloadSelectorSpec) LabelsSelector() (labels.Selector, error) {
	if s.Selector == nil {
		return labels.NewSelector(), nil
	} else {
		return metav1.LabelSelectorAsSelector(s.Selector)
	}
}

// AuthenticationSpec describes how the proxy shoudl authenticate with
// the google cloud api. May be one of the following:
// gcloud auth: `{gcloudAuth:true}`
// or
// kubernetes secret: `{credentialFileSecret: "default/gcloud-cred", credentialFileKey="gcloud.json"}`
type AuthenticationSpec struct {
	// GCloudAuth true when we should use the gcloud metadata server to authenticate
	// +kubebuilder:validation:Optional
	GCloudAuth bool `json:"gcloudAuth,omitempty"`

	// CredentialsFileSecret the "name" or "namespace/name" for the secret
	// +kubebuilder:validation:Optional
	CredentialsFileSecret string `json:"credentialsFileSecret,omitempty"`

	// CredentialsFileKey The key within the kubernetes secret containing the credentials file.
	// +kubebuilder:validation:Optional
	CredentialsFileKey string `json:"credentialsFileKey,omitempty"`
}

// ProxyContainerSpec configuration for the proxy container
type ProxyContainerSpec struct {
	// Resources the resources required for the proxy pod
	// +kubebuilder:validation:Optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`

	// Image The image
	// +kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`

	// SQLAdminApiEndpoint debugging parameter to change the api endpoint
	// +kubebuilder:validation:Optional
	SQLAdminApiEndpoint string `json:"sqlAdminApiEndpoint,omitempty"`

	// Container debugging parameter to fully override the proxy container
	// +kubebuilder:validation:Optional
	Container *v1.Container `json:"container,omitempty"`

	// +kubebuilder:validation:Optional
	Telemetry *TelemetrySpec `json:"telemetry,omitempty"`
}

// TelemetrySpec telemetry configuration for the proxy
type TelemetrySpec struct {
	// +kubebuilder:validation:Optional
	PrometheusNamespace *string `json:"prometheusNamespace,omitempty"`
	// +kubebuilder:validation:Optional
	TelemetryPrefix *string `json:"telemetryPrefix,omitempty"`
	// +kubebuilder:validation:Optional
	TelemetryProject *string `json:"telemetryProject,omitempty"`
	// +kubebuilder:validation:Optional
	TelemetrySampleRate *int `json:"telemetrySampleRate,omitempty"`
	// +kubebuilder:validation:Optional
	HttpPort *int32 `json:"httpPort,omitempty"`
	// +kubebuilder:validation:Optional
	DisableTraces *bool `json:"disableTraces,omitempty"`
	// +kubebuilder:validation:Optional
	DisableMetrics *bool `json:"disableMetrics,omitempty"`
	// +kubebuilder:validation:Optional
	Prometheus *bool `json:"prometheus,omitempty"`
}

// SocketType enum of socket types available on InstanceSpec
type SocketType string

const (
	SocketTypeTCP  SocketType = "tcp"
	SocketTypeUnix SocketType = "unix"
)

// InstanceSpec describes how to connect to a proxy instance
type InstanceSpec struct {
	// +kubebuilder:validation:Required
	// ConnectionString the Cloud SQL instance to connect
	ConnectionString string `json:"connectionString,omitempty"`

	// SocketType enum of {"tcp","unix"} declares what type of socket to
	// open for this database.
	// +kubebuilder:validation:Enum=tcp;unix
	// +kubebuilder:validation:Optional
	SocketType SocketType `json:"socketType,omitempty"`

	// Port Set the tcp port for this instance
	// +kubebuilder:validation:Optional
	Port *int32 `json:"port,omitempty"`
	// AutoIamAuthn Enable IAM Authentication for this instance
	// +kubebuilder:validation:Optional
	AutoIAMAuthN *bool `json:"autoIAMAuthN,omitempty"`
	// PrivateIP Enable connection to the Cloud SQL instance's private ip for this instance
	// +kubebuilder:validation:Optional
	PrivateIP *bool `json:"privateIP,omitempty"`
	// UnixSocketPath Use this directory to hold the unix socket for this instance
	// +kubebuilder:validation:Optional
	UnixSocketPath string `json:"unixSocketPath,omitempty"`
	// FusePath Use this directory as the fuse volume for this instance
	// +kubebuilder:validation:Optional
	FusePath string `json:"fusePath,omitempty"`

	// PortEnvName The name of the environment variable containing this instances tcp port
	// +kubebuilder:validation:Optional
	PortEnvName string `json:"portEnvName,omitempty"`
	// HostEnvName The name of the environment variable containing this instances tcp hostname
	// +kubebuilder:validation:Optional
	HostEnvName string `json:"hostEnvName,omitempty"`
	// UnixSocketPathEnvName the name of the environment variable containing the unix socket path
	// +kubebuilder:validation:Optional
	UnixSocketPathEnvName string `json:"unixSocketPathEnvName,omitempty"`
	// FuseVolumePathEnvName the name of the environment variable containing fuse volume path
	// +kubebuilder:validation:Optional
	FuseVolumePathEnvName string `json:"fuseVolumePathEnvName,omitempty"`
}

// AuthProxyWorkloadStatus defines the observed state of AuthProxyWorkload
type AuthProxyWorkloadStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Conditions for the state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	WorkloadStatus []WorkloadStatus `json:"WorkloadStatus,omitempty"`
}

// WorkloadStatus holds the status for the application of this proxy config to
// a matching workload.
type WorkloadStatus struct {
	// +kubebuilder:validation:Enum=Pod;Deployment;StatefulSet;DaemonSet;Job;CronJob
	Kind      string `json:"kind,omitempty,"`
	Version   string `json:"version,omitempty,"`
	Namespace string `json:"namespace,omitempty,"`
	Name      string `json:"name,omitempty,"`

	// Conditions for the state of the workload status
	Conditions []metav1.Condition `json:"conditions"`
}

// AuthProxyWorkload is the Schema for the authproxyworkloads API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type AuthProxyWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuthProxyWorkloadSpec   `json:"spec,omitempty"`
	Status AuthProxyWorkloadStatus `json:"status,omitempty"`
}

// AuthProxyWorkloadList contains a list of AuthProxyWorkload
// +kubebuilder:object:root=true
type AuthProxyWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuthProxyWorkload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AuthProxyWorkload{}, &AuthProxyWorkloadList{})
}
