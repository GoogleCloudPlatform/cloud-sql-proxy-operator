# Development



# How this project was created
This project was initially scaffolded by `kubebuilder` 3.6.0. These are the 
commands initially used to set up the project.

```
# Get the kubebuilder binary
mkdir -p .bin
curl -L -o .bin/kubebuilder "https://github.com/kubernetes-sigs/kubebuilder/releases/download/v3.6.0/kubebuilder_$(go env GOOS)_$(go env GOARCH)"
chmod a+x .bin/kubebuilder

# Clean up the root dir for kubebuilder
rm -rf Makefile main.go go.mod go.sum cover.out 

mkdir -p .bin/tmp/
mv docs .bin/tmp/
mv version.txt .bin/tmp/

rm -rf bin
.bin/kubebuilder init --owner "Google LLC" --project-name "cloud-sql-proxy-operator" --domain cloud.google.com --repo github.com/GoogleCloudPlatform/cloud-sql-proxy-operator

mv .bin/tmp/* .

```

Then, to create the CRD for Workload
```
.bin/kubebuilder create api --group cloudsql --version v1alpha1 --kind AuthProxyWorkload --controller --resource --force
.bin/kubebuilder create webhook --group cloudsql --version v1alpha1 --kind AuthProxyWorkload --defaulting --programmatic-validation
```


