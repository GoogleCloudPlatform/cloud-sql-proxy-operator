# API Reference

## Packages
- [cloudsql.cloud.google.com/v1](#cloudsqlcloudgooglecomv1)


## cloudsql.cloud.google.com/v1

Package v1 contains the API Schema definitions for the
the custom resource AuthProxyWorkload version v1.


### Resource Types
- [AuthProxyWorkload](#authproxyworkload)



#### AdminServerSpec



AdminServerSpec specifies how to start the proxy's admin server:
which port and whether to enable debugging or quitquitquit. It controls
to the proxy's --admin-port, --debug, and --quitquitquit CLI flags.



_Appears in:_
- [AuthProxyContainerSpec](#authproxycontainerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `port` _integer_ | Port the port for the proxy's localhost-only admin server.<br />This sets the proxy container's CLI argument `--admin-port` |  | Minimum: 1 <br /> |
| `enableAPIs` _string array_ | EnableAPIs specifies the list of admin APIs to enable. At least one<br />API must be enabled. Possible values:<br />- "Debug" will enable pprof debugging by setting the `--debug` cli flag.<br />- "QuitQuitQuit" will enable pprof debugging by setting the `--quitquitquit`<br />  cli flag. |  | MinItems: 1 <br /> |


#### AuthProxyContainerSpec



AuthProxyContainerSpec describes how to configure global proxy configuration and
kubernetes-specific container configuration.



_Appears in:_
- [AuthProxyWorkloadSpec](#authproxyworkloadspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `container` _[Container](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#container-v1-core)_ | Container is debugging parameter that when specified will override the<br />proxy container with a completely custom Container spec. |  | Optional: \{\} <br /> |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#resourcerequirements-v1-core)_ | Resources specifies the resources required for the proxy pod. |  | Optional: \{\} <br /> |
| `securityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#securitycontext-v1-core)_ | SecurityContext specifies the security context for the proxy container. |  | Optional: \{\} <br /> |
| `telemetry` _[TelemetrySpec](#telemetryspec)_ | Telemetry specifies how the proxy should expose telemetry.<br />Optional, by default |  | Optional: \{\} <br /> |
| `adminServer` _[AdminServerSpec](#adminserverspec)_ | AdminServer specifies the config for the proxy's admin service which is<br />available to other containers in the same pod. |  |  |
| `authentication` _[AuthenticationSpec](#authenticationspec)_ | Authentication specifies the config for how the proxy authenticates itself<br />to the Google Cloud API. |  |  |
| `maxConnections` _integer_ | MaxConnections limits the number of connections. Default value is no limit.<br />This sets the proxy container's CLI argument `--max-connections` |  | Minimum: 0 <br />Optional: \{\} <br /> |
| `maxSigtermDelay` _integer_ | MaxSigtermDelay is the maximum number of seconds to wait for connections to<br />close after receiving a TERM signal. This sets the proxy container's<br />CLI argument `--max-sigterm-delay` and<br />configures `terminationGracePeriodSeconds` on the workload's PodSpec. |  | Minimum: 0 <br />Optional: \{\} <br /> |
| `minSigtermDelay` _integer_ | MinSigtermDelay is the minimum number of seconds to wait for connections to<br />close after receiving a TERM signal. This sets the proxy container's<br />CLI argument `--min-sigterm-delay` |  | Minimum: 0 <br />Optional: \{\} <br /> |
| `sqlAdminAPIEndpoint` _string_ | SQLAdminAPIEndpoint is a debugging parameter that when specified will<br />change the Google Cloud api endpoint used by the proxy. |  | Optional: \{\} <br /> |
| `image` _string_ | Image is the URL to the proxy image. Optional, by default the operator<br />will use the latest Cloud SQL Auth Proxy version as of the release of the<br />operator.<br /><br />The operator ensures that all workloads configured with the default proxy<br />image are upgraded automatically to use to the latest released proxy image.<br /><br />When the customer upgrades the operator, the operator upgrades all<br />workloads using the default proxy image to the latest proxy image. The<br />change to the proxy container image is applied in accordance with<br />the RolloutStrategy. |  | Optional: \{\} <br /> |
| `rolloutStrategy` _string_ | RolloutStrategy indicates the strategy to use when rolling out changes to<br />the workloads affected by the results. When this is set to<br />`Workload`, changes to this resource will be automatically applied<br />to a running Deployment, StatefulSet, DaemonSet, or ReplicaSet in<br />accordance with the Strategy set on that workload. When this is set to<br />`None`, the operator will take no action to roll out changes to affected<br />workloads. `Workload` will be used by default if no value is set.<br />See: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#strategy | Workload | Enum: [Workload None] <br />Optional: \{\} <br /> |
| `refreshStrategy` _string_ | RefreshStrategy indicates which refresh strategy the proxy should use.<br />When this is set to `lazy`, the proxy will use a lazy refresh strategy,<br />and will be configured to run with the --lazy-refresh flag. When this<br />omitted or set to `background`, the proxy will use the default background<br />refresh strategy.<br />See: https://github.com/GoogleCloudPlatform/cloud-sql-proxy/?tab=readme-ov-file#configuring-a-lazy-refresh | background | Enum: [lazy background] <br />Optional: \{\} <br /> |
| `quiet` _boolean_ | Quiet configures the proxy's --quiet flag to limit the amount of<br />logging generated by the proxy container. |  |  |


#### AuthProxyWorkload



AuthProxyWorkload declares how a Cloud SQL Proxy container should be applied
to a matching set of workloads, and shows the status of those proxy containers.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `cloudsql.cloud.google.com/v1` | | |
| `kind` _string_ | `AuthProxyWorkload` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[AuthProxyWorkloadSpec](#authproxyworkloadspec)_ |  |  |  |




#### AuthProxyWorkloadSpec



AuthProxyWorkloadSpec describes where and how to configure the proxy.



_Appears in:_
- [AuthProxyWorkload](#authproxyworkload)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `workloadSelector` _[WorkloadSelectorSpec](#workloadselectorspec)_ | Workload selects the workload where the proxy container will be added. |  | Required: \{\} <br /> |
| `instances` _[InstanceSpec](#instancespec) array_ | Instances describes the Cloud SQL instances to configure on the proxy container. |  | MinItems: 1 <br />Required: \{\} <br /> |
| `authProxyContainer` _[AuthProxyContainerSpec](#authproxycontainerspec)_ | AuthProxyContainer describes the resources and config for the Auth Proxy container. |  | Optional: \{\} <br /> |




#### AuthenticationSpec



AuthenticationSpec specifies how the proxy is authenticated with the
Google Cloud SQL Admin API. This configures proxy's
--impersonate-service-account flag.



_Appears in:_
- [AuthProxyContainerSpec](#authproxycontainerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `impersonationChain` _string array_ | ImpersonationChain is a list of one or more service<br />accounts. The first entry in the chain is the impersonation target. Any<br />additional service accounts after the target are delegates. The<br />roles/iam.serviceAccountTokenCreator must be configured for each account<br />that will be impersonated. This sets the --impersonate-service-account<br />flag on the proxy. |  |  |


#### InstanceSpec



InstanceSpec describes the configuration for how the proxy should expose
a Cloud SQL database instance to a workload.


In the minimum recommended configuration, the operator will choose
a non-conflicting TCP port and set environment
variables MY_DB_SERVER_PORT MY_DB_SERVER_HOST with the value of the TCP port
and hostname. The application can read these values to connect to the database
through the proxy. For example:


	`{
			   "connectionString":"my-project:us-central1:my-db-server",
			   "portEnvName":"MY_DB_SERVER_PORT"
			   "hostEnvName":"MY_DB_SERVER_HOST"
	}`


If you want to assign a specific port number for a database, set the `port`
field. For example:


	`{ "connectionString":"my-project:us-central1:my-db-server", "port":5000 }`



_Appears in:_
- [AuthProxyWorkloadSpec](#authproxyworkloadspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionString` _string_ | ConnectionString is the connection string for the Cloud SQL Instance<br />in the format `project_id:region:instance_name` |  | Pattern: `^([^:]+(:[^:]+)?):([^:]+):([^:]+)$` <br />Required: \{\} <br /> |
| `port` _integer_ | Port (optional) sets the tcp port for this instance. If not set, a value will<br />be automatically assigned by the operator and set as an environment variable<br />on all containers in the workload named according to PortEnvName. The operator will choose<br />a port so that it does not conflict with other ports on the workload. |  | Minimum: 1 <br />Optional: \{\} <br /> |
| `autoIAMAuthN` _boolean_ | AutoIAMAuthN (optional) Enables IAM Authentication for this instance.<br />Default value is false. |  | Optional: \{\} <br /> |
| `privateIP` _boolean_ | PrivateIP (optional) Enable connection to the Cloud SQL instance's private ip for this instance.<br />Default value is false. |  | Optional: \{\} <br /> |
| `psc` _boolean_ | PSC (optional) Enable connection to the Cloud SQL instance's private<br />service connect endpoint. May not be used with PrivateIP.<br />Default value is false. |  | Optional: \{\} <br /> |
| `portEnvName` _string_ | PortEnvName is name of the environment variable containing this instance's tcp port.<br />Optional, when set this environment variable will be added to all containers in the workload. |  | Optional: \{\} <br /> |
| `hostEnvName` _string_ | HostEnvName The name of the environment variable containing this instances tcp hostname<br />Optional, when set this environment variable will be added to all containers in the workload. |  | Optional: \{\} <br /> |
| `unixSocketPath` _string_ | UnixSocketPath is the path to the unix socket where the proxy will listen<br />for connnections. This will be mounted to all containers in the pod. |  | Optional: \{\} <br /> |
| `unixSocketPathEnvName` _string_ | UnixSocketPathEnvName is the environment variable containing the value of<br />UnixSocketPath. |  | Optional: \{\} <br /> |


#### TelemetrySpec



TelemetrySpec specifies how the proxy container will expose telemetry.



_Appears in:_
- [AuthProxyContainerSpec](#authproxycontainerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `quotaProject` _string_ | QuotaProject Specifies the project to use for Cloud SQL Admin API quota tracking.<br />The IAM principal must have the "serviceusage.services.use" permission<br />for the given project. See https://cloud.google.com/service-usage/docs/overview and<br />https://cloud.google.com/storage/docs/requester-pays<br />This sets the proxy container's CLI argument `--quota-project` |  | Optional: \{\} <br /> |
| `prometheus` _boolean_ | Prometheus Enables Prometheus HTTP endpoint /metrics on localhost<br />This sets the proxy container's CLI argument `--prometheus` |  | Optional: \{\} <br /> |
| `prometheusNamespace` _string_ | PrometheusNamespace is used the provided Prometheus namespace for metrics<br />This sets the proxy container's CLI argument `--prometheus-namespace` |  | Optional: \{\} <br /> |
| `telemetryProject` _string_ | TelemetryProject enables Cloud Monitoring and Cloud Trace with the provided project ID.<br />This sets the proxy container's CLI argument `--telemetry-project` |  | Optional: \{\} <br /> |
| `telemetryPrefix` _string_ | TelemetryPrefix is the prefix for Cloud Monitoring metrics.<br />This sets the proxy container's CLI argument `--telemetry-prefix` |  | Optional: \{\} <br /> |
| `telemetrySampleRate` _integer_ | TelemetrySampleRate is the Cloud Trace sample rate. A smaller number means more traces.<br />This sets the proxy container's CLI argument `--telemetry-sample-rate` |  | Optional: \{\} <br /> |
| `httpPort` _integer_ | HTTPPort the port for Prometheus and health check server.<br />This sets the proxy container's CLI argument `--http-port` |  | Optional: \{\} <br /> |
| `disableTraces` _boolean_ | DisableTraces disables Cloud Trace testintegration (used with telemetryProject)<br />This sets the proxy container's CLI argument `--disable-traces` |  | Optional: \{\} <br /> |
| `disableMetrics` _boolean_ | DisableMetrics disables Cloud Monitoring testintegration (used with telemetryProject)<br />This sets the proxy container's CLI argument `--disable-metrics` |  | Optional: \{\} <br /> |


#### WorkloadSelectorSpec



WorkloadSelectorSpec describes which workloads should be configured with this
proxy configuration. To be valid, WorkloadSelectorSpec must specify `kind`
and either `name` or `selector`.



_Appears in:_
- [AuthProxyWorkloadSpec](#authproxyworkloadspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `selector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.24/#labelselector-v1-meta)_ | Selector (optional) selects resources using labels. See "Label selectors" in the kubernetes docs<br />https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors |  | Optional: \{\} <br /> |
| `kind` _string_ | Kind specifies what kind of workload<br />Supported kinds: Deployment, StatefulSet, Pod, ReplicaSet,DaemonSet, Job, CronJob<br />Example: "Deployment" "Deployment.v1" or "Deployment.v1.apps". |  | Pattern: `\w+(\.\w+)*` <br />Required: \{\} <br /> |
| `name` _string_ | Name specifies the name of the resource to select. |  | Optional: \{\} <br /> |




