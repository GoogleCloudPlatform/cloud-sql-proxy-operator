# Cloud SQL Proxy Operator

Cloud SQL Proxy Operator is an open-source Kubernetes operator that automates
most of the intricate steps needed to connect a workload in a kubernetes cluster
to Cloud SQL databases. 

The operator introduces a custom resource AuthProxyWorkload, 
which specifies the Cloud SQL Auth Proxy configuration for a workload. The operator
reads this resource and adds a properly configured Cloud SQL Auth Proxy container
to the matching workload pods. 

