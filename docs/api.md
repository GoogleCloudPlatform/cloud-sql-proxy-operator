# API Reference

## Packages
- [cloudsql.cloud.google.com/v1alpha1](#cloudsqlcloudgooglecomv1alpha1)


## cloudsql.cloud.google.com/v1alpha1

Package v1alpha1 contains the API Schema definitions for the
the custom resource AuthProxyWorkload version v1alpha1.


### Resource Types
- [AuthProxyWorkload](#authproxyworkload)



#### AdminServerSpec



AdminServerSpec specifies how to start the proxy's admin server: which port and whether to enable debugging or quitquitquit. It controls to the proxy's --admin-port, --debug, and --quitquitquit CLI flags.

_Appears in:_
- [AuthProxyContainerSpec](#authproxycontainerspec)

| Field | Description |
| --- | --- |
| `port` _integer_ | Port the port for the proxy's localhost-only admin server. This sets the proxy container's CLI argument `--admin-port` |
| `enableAPIs` _string array_ | EnableAPIs specifies the list of admin APIs to enable. At least one API must be enabled. Possible values: - "Debug" will enable pprof debugging by setting the `--debug` cli flag. - "QuitQuitQuit" will enable pprof debugging by setting the `--quitquitquit`   cli flag. |


#### AuthProxyContainerSpec



AuthProxyContainerSpec describes how to configure global proxy configuration and kubernetes-specific container configuration.

_Appears in:_
- [AuthProxyWorkloadSpec](#authproxyworkloadspec)

| Field | Description |
| --- | --- |
| `container` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#container-v1-core)_ | Container is debugging parameter that when specified will override the proxy container with a completely custom Container spec. |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#resourcerequirements-v1-core)_ | Resources specifies the resources required for the proxy pod. |
| `telemetry` _[TelemetrySpec](#telemetryspec)_ | Telemetry specifies how the proxy should expose telemetry. Optional, by default |
| `adminServer` _[AdminServerSpec](#adminserverspec)_ | AdminServer specifies the config for the proxy's admin service which is available to other containers in the same pod. |
| `maxConnections` _integer_ | MaxConnections limits the number of connections. Default value is no limit. This sets the proxy container's CLI argument `--max-connections` |
| `maxSigtermDelay` _integer_ | MaxSigtermDelay is the maximum number of seconds to wait for connections to close after receiving a TERM signal. This sets the proxy container's CLI argument `--max-sigterm-delay` and configures `terminationGracePeriodSeconds` on the workload's PodSpec. |
| `sqlAdminAPIEndpoint` _string_ | SQLAdminAPIEndpoint is a debugging parameter that when specified will change the Google Cloud api endpoint used by the proxy. |
| `image` _string_ | Image is the URL to the proxy image. Optional, by default the operator will use the latest known compatible proxy image. |
| `rolloutStrategy` _string_ | RolloutStrategy indicates the strategy to use when rolling out changes to the workloads affected by the results. When this is set to `Workload`, changes to this resource will be automatically applied to a running Deployment, StatefulSet, DaemonSet, or ReplicaSet in accordance with the Strategy set on that workload. When this is set to `None`, the operator will take no action to roll out changes to affected workloads. `Workload` will be used by default if no value is set. See: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#strategy |


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
| `port` _integer_ | Port (optional) sets the tcp port for this instance. If not set, a value will be automatically assigned by the operator and set as an environment variable on all containers in the workload named according to PortEnvName. The operator will choose a port so that it does not conflict with other ports on the workload. |
| `autoIAMAuthN` _boolean_ | AutoIAMAuthN (optional) Enables IAM Authentication for this instance. Default value is false. |
| `privateIP` _boolean_ | PrivateIP (optional) Enable connection to the Cloud SQL instance's private ip for this instance. Default value is false. |
| `portEnvName` _string_ | PortEnvName is name of the environment variable containing this instance's tcp port. Optional, when set this environment variable will be added to all containers in the workload. |
| `hostEnvName` _string_ | HostEnvName The name of the environment variable containing this instances tcp hostname Optional, when set this environment variable will be added to all containers in the workload. |
| `unixSocketPath` _string_ | UnixSocketPath is the path to the unix socket where the proxy will listen for connnections. This will be mounted to all containers in the pod. |
| `unixSocketPathEnvName` _string_ | UnixSocketPathEnvName is the environment variable containing the value of UnixSocketPath. |


#### TelemetrySpec



TelemetrySpec specifies how the proxy container will expose telemetry.

_Appears in:_
- [AuthProxyContainerSpec](#authproxycontainerspec)

| Field | Description |
| --- | --- |
| `httpPort` _integer_ | HTTPPort the port for Prometheus and health check server. This sets the proxy container's CLI argument `--http-port` |


#### WorkloadSelectorSpec



WorkloadSelectorSpec describes which workloads should be configured with this proxy configuration. To be valid, WorkloadSelectorSpec must specify `kind` and either `name` or `selector`.

_Appears in:_
- [AuthProxyWorkloadSpec](#authproxyworkloadspec)

| Field | Description |
| --- | --- |
| `selector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#labelselector-v1-meta)_ | Selector (optional) selects resources using labels. See "Label selectors" in the kubernetes docs https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors |
| `kind` _string_ | Kind specifies what kind of workload Supported kinds: Deployment, StatefulSet, Pod, ReplicaSet,DaemonSet, Job, CronJob Example: "Deployment" "Deployment.v1" or "Deployment.v1.apps". |
| `name` _string_ | Name specifies the name of the resource to select. |




