# Cloud SQL Proxy Operator

Cloud SQL Proxy Operator is an open-source Kubernetes operator that automates
most of the intricate steps needed to connect a workload in a kubernetes cluster
to Cloud SQL databases. 

The operator introduces a custom resource AuthProxyWorkload, 
which specifies the Cloud SQL Auth Proxy configuration for a workload. The operator
reads this resource and adds a properly configured Cloud SQL Auth Proxy container
to the matching workload pods. 

## Install the Cloud SQL Proxy Operator

Confirm that kubectl can connect to your kubernetes cluster.

```shell
kubectl cluster-info
```

Run the following command to install the cloud sql proxy operator into
your kubernetes cluster:

```shell
curl https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy-operator/v0.0.2-dev/install.sh | bash
```

Confirm that the operator is installed and running by listing its pods:

```shell
kubectl get pods -n cloud-sql-proxy-operator-system
```

# Additional Documentation

- [Quick Start Guide](docs/quick-start.md)
- [Kubernetes Operator for Cloud SQL Proxy](https://docs.google.com/presentation/d/1Zb20y-oyRUBMn6qRjJe0e7_AEPewu1sr-uWX4ac2SpU/edit?resourcekey=0-eVSy_QoAjXkW68hapOOP-Q#slide=id.g4c499b7a9e_0_0) (Google Slides)

## For Developers

- [Developer Getting Started](docs/dev.md)
- [Developing End-to-End tests](docs/e2e-tests.md)
- [Contributing](docs/contributing.md)
- [Code of Conduct](docs/code-of-conduct.md)
- [Examples](docs/examples/)
