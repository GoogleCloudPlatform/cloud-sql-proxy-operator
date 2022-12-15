# API Reference

## Packages
- [cloudsql.cloud.google.com/v1alpha1](#cloudsqlcloudgooglecomv1alpha1)


## cloudsql.cloud.google.com/v1alpha1

Package v1alpha1 contains API Schema definitions for the cloudsql v1alpha1 API group:
the custom resource AuthProxyWorkload version v1alpha1
This follows the kubebuilder pattern for defining custom resources.


### Resource Types
- [AuthProxyWorkload](#authproxyworkload)



#### AuthProxyContainerSpec



AuthProxyContainerSpec specifies configuration for the proxy container.

_Appears in:_
- [AuthProxyWorkloadSpec](#authproxyworkloadspec)

| Field | Description |
| --- | --- |
| `container` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#container-v1-core)_ | Container is debugging parameter that when specified will override the proxy container with a completely custom Container spec. |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#resourcerequirements-v1-core)_ | Resources specifies the resources required for the proxy pod. |
| `maxConnections` _integer_ | MaxConnections limits the number of connections. Default value is no limit. This sets the proxy container's CLI argument `--max-connections` |
| `maxSigtermDelay` _integer_ | MaxSigtermDelay is the maximum number of seconds to wait for connections to close after receiving a TERM signal. This sets the proxy container's CLI argument `--max-sigterm-delay` and configures `terminationGracePeriodSeconds` on the workload's PodSpec. |
| `sqlAdminAPIEndpoint` _string_ | SQLAdminAPIEndpoint is a debugging parameter that when specified will change the Google Cloud api endpoint used by the proxy. |
| `image` _string_ | Image is the URL to the proxy image. Optional, by default the operator will use the latest known compatible proxy image. |


#### AuthProxyWorkload



AuthProxyWorkload declares how a Cloud SQL Proxy container should be applied to a matching set of workloads, and shows the status of those proxy containers. This is the Schema for the authproxyworkloads API.



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `cloudsql.cloud.google.com/v1alpha1`
| `kind` _string_ | `AuthProxyWorkload`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[AuthProxyWorkloadSpec](#authproxyworkloadspec)_ |  |


#### AuthProxyWorkloadSpec



AuthProxyWorkloadSpec defines the desired state of AuthProxyWorkload

_Appears in:_
- [AuthProxyWorkload](#authproxyworkload)

| Field | Description |
| --- | --- |
| `workloadSelector` _[WorkloadSelectorSpec](#workloadselectorspec)_ | Workload selects the workload to |
| `authProxyContainer` _[AuthProxyContainerSpec](#authproxycontainerspec)_ | AuthProxyContainer describes the resources and config for the Auth Proxy container |
| `instances` _[InstanceSpec](#instancespec) array_ | Instances lists the Cloud SQL instances to connect |


#### InstanceSpec



InstanceSpec describes the configuration for how the proxy should expose a Cloud SQL database instance to a workload. The simplest possible configuration declares just the connection string and the port number or unix socket. 
 For example, for a TCP port: 
 	{ "connectionString":"my-project:us-central1:my-db-server", "port":5000 } 
 or for a unix socket: 
 	{ "connectionString":"my-project:us-central1:my-db-server", 	  "unixSocketPath" : "/mnt/db/my-db-server" } 
 You may allow the operator to choose a non-conflicting TCP port or unix socket instead of explicitly setting the port or socket path. This may be easier to manage when workload needs to connect to many databases. 
 For example, for a TCP port: 
 	{ "connectionString":"my-project:us-central1:my-db-server", 	  "portEnvName":"MY_DB_SERVER_PORT" 	  "hostEnvName":"MY_DB_SERVER_HOST" 	 } 
 will set environment variables MY_DB_SERVER_PORT MY_DB_SERVER_HOST with the value of the TCP port and hostname. Then, the application can read these values to connect to the database through the proxy. 
 or for a unix socket: 
 	{ "connectionString":"my-project:us-central1:my-db-server", 	  "unixSocketPathEnvName" : "MY_DB_SERVER_SOCKET_DIR" } 
 will set environment variables MY_DB_SERVER_SOCKET_DIR with the value of the unix socket path. Then, the application can read this value to connect to the database through the proxy.

_Appears in:_
- [AuthProxyWorkloadSpec](#authproxyworkloadspec)

| Field | Description |
| --- | --- |
| `connectionString` _string_ | ConnectionString is the Cloud SQL instance. |
| `port` _integer_ | Port sets the tcp port for this instance. Optional, if not set, a value will be automatically assigned by the operator and set as an environment variable on all containers in the workload named according to PortEnvName. The operator will choose a port so that it does not conflict with other ports on the workload. |
| `autoIAMAuthN` _boolean_ | AutoIAMAuthN Enables IAM Authentication for this instance. Optional, default false. |
| `privateIP` _boolean_ | PrivateIP Enable connection to the Cloud SQL instance's private ip for this instance. Optional, default false. |
| `portEnvName` _string_ | PortEnvName is name of the environment variable containing this instance's tcp port. Optional, when set this environment variable will be added to all containers in the workload. |
| `hostEnvName` _string_ | HostEnvName The name of the environment variable containing this instances tcp hostname Optional, when set this environment variable will be added to all containers in the workload. |


#### WorkloadSelectorSpec



WorkloadSelectorSpec describes which workloads should be configured with this proxy configuration. To be valid, WorkloadSelectorSpec must specify Kind and either Name or Selector.

_Appears in:_
- [AuthProxyWorkloadSpec](#authproxyworkloadspec)

| Field | Description |
| --- | --- |
| `selector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#labelselector-v1-meta)_ | Selector selects resources using labels. See "Label selectors" in the kubernetes docs https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors |
| `kind` _string_ | Kind specifies what kind of workload Supported kinds: Deployment, StatefulSet, Pod, ReplicaSet,DaemonSet, Job, CronJob Example: "Deployment" "Deployment.v1" or "Deployment.v1.apps". |
| `namespace` _string_ | Namespace specifies namespace in which to select the resource. Optional, defaults to the namespace of the AuthProxyWorkload resource. All or Wildcard namespaces are not supported. |
| `name` _string_ | Name specifies the name of the resource to select. |




