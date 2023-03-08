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
.bin/kubebuilder create api --group cloudsql --version v1 --kind AuthProxyWorkload --controller --resource --force
.bin/kubebuilder create webhook --group cloudsql --version v1 --kind AuthProxyWorkload --defaulting --programmatic-validation
```


## Running E2E tests with a custom proxy image

You may want to write e2e tests for a proxy feature that has
not been released yet. 

Step 1: Check out the cloud-sql-proxy repo.

Step 2: Add `E2E_LOCAL_PROXY_PROJECT_DIR = /home/me/projects/cloud-sql-proxy`
to your `build.env`. This tells your Makefile where your proxy
directory is. Set it to the path of your cloud-sql-proxy working directory.

Step 3: Build a custom image and push it to the e2e environment
repo. Run `make e2e_local_proxy_image_push` This will build and push
a docker image from the proxy repo, and write the file `bin/last-local-proxy-url.txt`

Step 4: Run your e2e tests. The tests will read the contents of
the file `bin/last-local-proxy-url.txt`.

Delete the file `bin/last-local-proxy-url.txt` to go back to using
the public proxy iamge again