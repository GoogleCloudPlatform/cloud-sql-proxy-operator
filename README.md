# Cloud SQL Proxy Operator

An operator that manages the cloud sql auth proxy on kubernetes workloads

```
kubebuilder init --domain cloud.google.com --repo github.com/hessjcg/cloud-sql-proxy-operator
kubebuilder create api --group cloudsql --version v99 --kind AuthProxyWorkload --controller --resource --force
kubebuilder create webhook --group cloudsql --version v1 --kind AuthProxyWorkload --defaulting --programmatic-validation
```

