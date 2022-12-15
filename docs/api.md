# API Reference

## Packages
- [cloudsql.cloud.google.com/v1alpha1](#cloudsqlcloudgooglecomv1alpha1)


## cloudsql.cloud.google.com/v1alpha1

Package v1alpha1 contains the API Schema definitions for the
the custom resource AuthProxyWorkload version v1alpha1.


### Resource Types
- [AuthProxyWorkload](#authproxyworkload)



#### AuthProxyContainerSpec



AuthProxyContainerSpec describes how to configure global proxy configuration and kubernetes-specific container configuration.

_Appears in:_
- [AuthProxyWorkloadSpec](#authproxyworkloadspec)

| Field | Description |
| --- | --- |
| `maxConnections` _integer_ | MaxConnections limits the number of connections. Default value is no limit. This sets the proxy container's CLI argument `--max-connections` |
| `maxSigtermDelay` _integer_ | MaxSigtermDelay is the maximum number of seconds to wait for connections to close after receiving a TERM signal. This sets the proxy container's CLI argument `--max-sigterm-delay` and configures `terminationGracePeriodSeconds` on the workload's PodSpec. |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#resourcerequirements-v1-core)_ | Resources specifies the resources required for the proxy pod. |
| `image` _string_ | Image is the URL to the proxy image. Optional, by default the operator will use the latest known compatible proxy image. |
| `container` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#container-v1-core)_ | Container is debugging parameter that when specified will override the proxy container with a completely custom Container spec. |
| `sqlAdminAPIEndpoint` _string_ | SQLAdminAPIEndpoint is a debugging parameter that when specified will change the Google Cloud api endpoint used by the proxy. |


#### AuthProxyWorkload



AuthProxyWorkload declares how a Cloud SQL Proxy container should be applied to a matching set of workloads, and shows the status of those proxy containers.



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `cloudsql.cloud.google.com/v1alpha1`
| `kind` _string_ | `AuthProxyWorkload`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[AuthProxyWorkloadSpec](#authproxyworkloadspec)_ |  |


#### AuthProxyWorkloadSpec



AuthProxyWorkloadSpec describes where and how to configure the proxy.

_Appears in:_
- [AuthProxyWorkload](#authproxyworkload)

| Field | Description |
| --- | --- |
| `workloadSelector` _[WorkloadSelectorSpec](#workloadselectorspec)_ | Workload selects the workload where the proxy container will be added. |
| `instances` _[InstanceSpec](#instancespec) array_ | Instances describes the Cloud SQL instances to configure on the proxy container. |
| `authProxyContainer` _[AuthProxyContainerSpec](#authproxycontainerspec)_ | AuthProxyContainer describes the resources and config for the Auth Proxy container. |


#### InstanceSpec



InstanceSpec describes the configuration for how the proxy should expose a Cloud SQL database instance to a workload. 
 In the minimum recommended configuration, the operator will choose a non-conflicting TCP port and set environment variables MY_DB_SERVER_PORT MY_DB_SERVER_HOST with the value of the TCP port and hostname. The application can read these values to connect to the database through the proxy. For example: 
 	`{ 			   "connectionString":"my-project:us-central1:my-db-server", 			   "portEnvName":"MY_DB_SERVER_PORT" 			   "hostEnvName":"MY_DB_SERVER_HOST" 	}` 
 If you want to assign a specific port number for a database, set the `port` field. For example: 
 	`{ "connectionString":"my-project:us-central1:my-db-server", "port":5000 }`

_Appears in:_
- [AuthProxyWorkloadSpec](#authproxyworkloadspec)

| Field | Description |
| --- | --- |
| `connectionString` _string_ | ConnectionString is the connection string for the Cloud SQL Instance in the format `project_id:region:instance_name` |
| `portEnvName` _string_ | PortEnvName (optional) is name of the environment variable containing this instance's tcp port. When set this environment variable will be added to all containers in the workload. |
| `hostEnvName` _string_ | HostEnvName (optional) The name of the environment variable containing this instances tcp hostname. When set this environment variable will be added to all containers in the workload. |
| `port` _integer_ | Port (optional) sets the tcp port for this instance. If not set, a value will be automatically assigned by the operator and set as an environment variable on all containers in the workload named according to PortEnvName. The operator will choose a port so that it does not conflict with other ports on the workload. |
| `autoIAMAuthN` _boolean_ | AutoIAMAuthN (optional) Enables IAM Authentication for this instance. Default value is false. |
| `privateIP` _boolean_ | PrivateIP (optional) Enable connection to the Cloud SQL instance's private ip for this instance. Default value is false. |


#### WorkloadSelectorSpec



WorkloadSelectorSpec describes which workloads should be configured with this proxy configuration. To be valid, WorkloadSelectorSpec must specify `kind` and either `name` or `selector`.

_Appears in:_
- [AuthProxyWorkloadSpec](#authproxyworkloadspec)

| Field | Description |
| --- | --- |
| `selector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#labelselector-v1-meta)_ | Selector (optional) selects resources using labels. See "Label selectors" in the kubernetes docs https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors |
| `kind` _string_ | Kind specifies what kind of workload Supported kinds: Deployment, StatefulSet, Pod, ReplicaSet,DaemonSet, Job, CronJob Example: "Deployment" "Deployment.v1" or "Deployment.v1.apps". |
| `name` _string_ | Name specifies the name of the resource to select. |




