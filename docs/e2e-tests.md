# End-to-End Tests

This project uses end-to-end tests to make sure that the operator, proxy and
everything works

## Get Started

After setting up your local development environment, please do the following:

### Configure gcloud cli
Install the gcloud command line application and make sure it is in your system PATH.

Install the GKE auth components by running
`gcloud components install gke-gcloud-auth-plugin` 
and then `gke-gcloud-auth-plugin --version` to ensure it was installed correctly

Log into gcloud.

Set your application default credentials.

### Create an empty project
Create an empty Google Cloud project in your Google Cloud account.

### Check out the Cloud Sql Proxy repo

Check out the [cloud-sql-proxy](https://github.com/GoogleCloudPlatform/cloud-sql-proxy) github
repo into a different directory. Make sure you are on branch 'main'. 


### Update your build.env
Copy `build.sample.env` to `build.env` and edit it to properly set:
- the absolute path to your cloud-sql-proxy working directory
- the empty Google Cloud project name. 

### Run the tests
Run `make gcloud_test` in the base directory of this project. This will run the
following make targets in sequence: 

- `make gcloud_test_infra` will use Terraform to create an artifact registry, 
   GKE cluster and postgres database. 
- `make gcloud_test_run` will  build docker images for the operator and the Cloud SQL Proxy, and
  push those images to the artifact registry.

The first time you run `make gcloud_test` it may take 20-30 minutes to provision
Google Cloud resources for the tests. Subsequent runs will reuse the Google Cloud
resources, and therefore will run much faster. 

# Test Organization

Don't write end-to-end tests when a unit test will do.

## Test Cases

...TODO...

## Helpers
The utility functions in `test/helpers` are intended to be reused between e2e and
integration tests.

The test cases in `helpers/testcases.go` are intended to be reused between e2e and
integration tests. They require a TestCaseParams as input, and then update the *
testing.T as output.

Right now we only have the most basic of tests implemented, so the E2E and
integration testcases are almost identical. The only additional assertions today
are in e2e.TestModifiesNewDeployment, checks the running Deployment pods. In the
integration test, pods are not scheduled or run.

Setup and teardown are very different between integration and e2e. Integration
tests need to start the k8s server on setup, and stop on on teardown. E2e tests
need to know the URL for the docker images and be able to connect to the k8s
cluster. Teardown for e2e is a no-op.

Eventually the testcases will diverge. New E2e tests will try out a lot of
different workload types, network configurations, maybe even k8s cluster
versions. New Integration tests may focus on demonstrating that we are handling
quirky K8S API edge cases correctly. 
