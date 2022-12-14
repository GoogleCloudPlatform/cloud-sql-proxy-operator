# Cloud SQL Proxy Operator

*Warning*: This project is in Public Preview, and may contain breaking changes
before it becomes Generally Available.

Cloud SQL Proxy Operator is an open-source Kubernetes operator that automates
most of the intricate steps needed to connect a workload in a kubernetes cluster
to Cloud SQL databases. 

The operator introduces a custom resource AuthProxyWorkload, 
which specifies the Cloud SQL Auth Proxy configuration for a workload. The operator
reads this resource and adds a properly configured Cloud SQL Auth Proxy container
to the matching workload pods. 

## Installation

Check for the latest version on the [releases page][releases] and use the
following instructions. 

Confirm that kubectl can connect to your kubernetes cluster.

```shell
kubectl cluster-info
```

Run the following command to install the cloud sql proxy operator into
your kubernetes cluster:

```shell
curl https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy-operator/v0.1.0/install.sh | bash
```

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
to your kubernetes workloads, configured correctly for production use. 

Writing the kubernetes configuration for a proxy to the production level requires
a great deal of deep kubernetes and proxy knowledge. The Cloud SQL Proxy team has
worked to encapsulate that knowledge in this operator. This saves you from having
to know all the details to configure your proxy.

## Reference Documentation
- [Quick Start Guide](docs/quick-start.md)
- [Cloud SQL Proxy](/GoogleCloudPlatform/cloud-sql-proxy)
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
By participating in this project you agree to abide by its terms.  See
[Code of Conduct](docs/code-of-conduct.md) for more information.

