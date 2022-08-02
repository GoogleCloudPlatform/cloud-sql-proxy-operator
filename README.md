# Cloud SQL Proxy Operator

Cloud SQL Proxy Operator is an open-source Kubernetes operator that automates
most of the intricate steps needed to connect a workload in a kubernetes cluster
to Cloud SQL databases. 

The operator introduces a custom resource AuthProxyWorkload, 
which specifies the Cloud SQL Auth Proxy configuration for a workload. The operator
reads this resource and adds a properly configured Cloud SQL Auth Proxy container
to the matching workload pods. 

## Setting up the initial project
These commands will be run to initialize the kubebuilder project 

```
mkdir -p .bin
curl -L -o .bin/kubebuilder https://github.com/kubernetes-sigs/kubebuilder/releases/download/v3.5.0/kubebuilder_darwin_amd64
.bin/kubebuilder init --domain cloud.google.com --repo github.com/hessjcg/cloud-sql-proxy-operator
.bin/kubebuilder create api --group cloudsql --version v99 --kind AuthProxyWorkload --controller --resource --force
.bin/kubebuilder create webhook --group cloudsql --version v1 --kind AuthProxyWorkload --defaulting --programmatic-validation
```


