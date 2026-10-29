# Cloud SQL Proxy Operator

Cloud SQL Proxy Operator is an open-source Kubernetes operator that automates
most of the intricate steps needed to connect a workload in a kubernetes cluster
to Cloud SQL databases.

The operator introduces a custom resource AuthProxyWorkload,
which specifies the Cloud SQL Auth Proxy configuration for a workload. The operator
reads this resource and adds a properly configured Cloud SQL Auth Proxy container
to the matching workload pods.

## Installation

Check for the latest version on the [releases page][latest-release] and use the
following instructions.

[latest-release]: https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/releases/latest

Confirm that kubectl can connect to your kubernetes cluster.

```shell
kubectl cluster-info
```

Install cert-manager using helm. Note that you need to use this particular
version with these specific cli arguments to make cert-manager work on
your GKE cluster.

```shell
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm install \
  cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --version "v1.9.1" \
  --create-namespace \
  --set global.leaderElection.namespace=cert-manager \
  --set installCRDs=true
```

Run the following command to install the cloud sql proxy operator into
your kubernetes cluster:

<!-- {x-release-please-start-version} -->

```shell
kubectl apply -f https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy-operator/v1.8.4/cloud-sql-proxy-operator.yaml
```

<!-- {x-release-please-end} -->

Confirm that the operator is installed and running by listing its pods:

```shell
kubectl get pods -n cloud-sql-proxy-operator-system
```

## Usage

See the [Quick Start Guide](docs/quick-start.md) for a description of basic usage.
Additional usage may be found in the [Examples](docs/examples/).

## Post-Quantum Cryptography (PQC)

Both the Cloud SQL Proxy Operator controller and the default Cloud SQL Auth Proxy
sidecar image (`gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.26.3`) negotiate the
hybrid post-quantum key exchange group `X25519MLKEM768` over TLS 1.3 by default:

- **Cloud SQL Auth Proxy sidecar:** Negotiates `X25519MLKEM768` on TLS 1.3 mTLS
  connections to Cloud SQL instances and HTTPS calls to the Cloud SQL Admin API.
- **Operator controller:** Negotiates `X25519MLKEM768` on outbound TLS 1.3 HTTPS
  connections to the Kubernetes API server, and accepts `X25519MLKEM768` on its
  TLS 1.3 admission webhook server from clients (such as `kube-apiserver`) that
  offer it.

No configuration changes are required. If a peer (such as a Cloud SQL instance or
Kubernetes API server) does not yet offer a hybrid post-quantum group, the TLS 1.3
handshake automatically falls back to classical key exchange (`X25519`) without
failing the connection.

### Custom Proxy Image Version Requirement

- If you pin a custom sidecar image using `spec.authProxyContainer.image`, use
  Cloud SQL Auth Proxy **`2.15.2` or later** (for example,
  `gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.15.2` or newer), which is the
  first v2 release built with `go 1.24` in `go.mod` (`tlsmlkem=1` enabled by
  default).
- If building the Operator or Cloud SQL Auth Proxy from source, build with
  **Go 1.24 or later** (with `go 1.24` or higher in `go.mod`).

### Opting Out (`GODEBUG=tlsmlkem=0`)

If you need to disable post-quantum key exchange and use classical TLS 1.3 groups
only, set the Go runtime environment variable `GODEBUG=tlsmlkem=0`:

- **Cloud SQL Auth Proxy sidecar:** `AuthProxyContainerSpec` does not expose a
  first-class `env` field today (configuring the proxy sidecar directly in your
  workload PodSpec or opening a feature request on this repository for sidecar
  `env` support is recommended if you need full operator-managed defaults).

  > **Warning:** `spec.authProxyContainer.container` is a debugging parameter
  > that **completely replaces** the operator-generated proxy container spec.
  > When set, the operator drops all of its generated defaults—including health
  > checks/startup probes, admin server configuration, auth flags,
  > `CSQL_PROXY_*` environment variables (user agent, structured logs, quiet,
  > lazy refresh), telemetry flags, generated instance arguments/ports,
  > security context, and `restartPolicy`—and overwrites `container.name` with
  > an operator-generated name (`csql-<namespace>-<name>`). Only the fields you
  > explicitly write under `container` take effect, so you must supply all
  > required instance arguments, ports, probes, security context, and
  > `restartPolicy` yourself (include `restartPolicy: Always` on Kubernetes
  > 1.29+ clusters where native sidecar `initContainers` are used; omit
  > `restartPolicy` on Kubernetes versions earlier than 1.29):

  ```yaml
  apiVersion: cloudsql.cloud.google.com/v1
  kind: AuthProxyWorkload
  metadata:
    name: authproxyworkload-sample
  spec:
    workloadSelector:
      kind: "Deployment"
      name: "gke-cloud-sql-app"
    instances:
      - connectionString: "<PROJECT>:<REGION>:<INSTANCE>"
        port: 5000
    authProxyContainer:
      container:
        name: placeholder # Required by CRD schema; overwritten with an operator-generated name (csql-<namespace>-<name>)
        image: gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.26.3
        restartPolicy: Always # Required on Kubernetes 1.29+; omit on < 1.29
        args:
          - "--structured-logs"
          - "<PROJECT>:<REGION>:<INSTANCE>?port=5000"
        env:
          - name: GODEBUG
            value: "tlsmlkem=0"
  ```

- **Operator controller:** Set `GODEBUG=tlsmlkem=0` on the `manager` container in
  the `cloud-sql-proxy-operator-controller-manager` Deployment:

  ```shell
  kubectl set env deployment/cloud-sql-proxy-operator-controller-manager \
    -n cloud-sql-proxy-operator-system \
    GODEBUG=tlsmlkem=0
  ```

## Frequently Asked Questions

### Why would I use the Cloud SQL Auth Proxy Operator?

The Cloud SQL Auth Proxy Operator gives you an easy way to add a proxy container
to your kubernetes workloads, configured correctly for production use. The operator
adds the auth proxy sidecar to the workloads you specify in a consistent, stable and secure way.

When you upgrade the operator, it will also upgrade the proxy image and container
configuration on those workloads.

The operator encodes the best practices for configuring
[Cloud SQL Proxy sidecar](https://github.com/GoogleCloudPlatform/cloud-sql-proxy/tree/main/examples/k8s-sidecar)
containers, and allows you to apply those best practices across the workloads in your kubernetes
cluster. This is especially helpful when managing a Kubernetes cluster that runs lots of
applications, each configured by a different team.

However, if you are deploying only a few applications in your kubernetes cluster, it may be
more straightforward to follow the Proxy K8s Sidecar Example and add the sidecar container
directly in your configuration.

The operator does not help set up roles and permissions. This also requires intricate configuration,
especially for workload identity federation. This is now a feature request, see #706

## Reference Documentation

- [Quick Start Guide](docs/quick-start.md)
- [API Documentation](docs/api.md)
- [Cloud SQL Proxy](https://github.com/GoogleCloudPlatform/cloud-sql-proxy)
- [Developer Getting Started](docs/dev.md)
- [Developing End-to-End tests](docs/e2e-tests.md)
- [Contributing](docs/contributing.md)
- [Code of Conduct](docs/code-of-conduct.md)
- [Examples](docs/examples/)

## Support policy

### Major version lifecycle

This project uses [semantic versioning](https://semver.org/), and uses the
following lifecycle regarding support for a major version:

**Active** - Active versions get all new features and security fixes (that
wouldn’t otherwise introduce a breaking change). New major versions are
guaranteed to be "active" for a minimum of 1 year.
**Deprecated** - Deprecated versions continue to receive security and critical
bug fixes, but do not receive new features. Deprecated versions will be publicly
supported for 1 year.
**Unsupported** - Any major version that has been deprecated for >=1 year is
considered publicly unsupported.

## Contributing

Contributions are welcome. Please, see the [Contributing](docs/contributing.md) document
for details.

Please note that this project is released with a Contributor Code of Conduct.
By participating in this project you agree to abide by its terms. See
[Contributor Code of Conduct](docs/code-of-conduct.md) for more information.

