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
kubectl apply -f https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy-operator/v1.7.0/cloud-sql-proxy-operator.yaml
```

<!-- {x-release-please-end} -->

Confirm that the operator is installed and running by listing its pods:

```shell
kubectl get pods -n cloud-sql-proxy-operator-system
```

## Usage

See the [Quick Start Guide](docs/quick-start.md) for a description of basic usage.
Additional usage may be found in the [Examples](docs/examples/).

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

