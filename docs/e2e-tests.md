# Tests

This project uses end-to-end tests to make sure that the operator, proxy and
everything works.

## Get Started With End-to-End Tests

After setting up your local development environment, please do the following:

### Configure gcloud CLI

[Install the gcloud CLI](https://cloud.google.com/sdk/docs/install) and make sure it is in your system PATH.

Install the GKE auth components by running
`gcloud components install gke-gcloud-auth-plugin`
and then `gke-gcloud-auth-plugin --version` to ensure it was installed correctly

Log into gcloud.

Set your application default credentials.

### Create an empty project

Create an empty Google Cloud project in your Google Cloud account.

### Check out the Cloud Sql Proxy repo

Check out
the [cloud-sql-proxy](https://github.com/GoogleCloudPlatform/cloud-sql-proxy)
github repo into a different directory. Make sure you are on branch 'main'.

### Update your build.env

Copy `build.sample.env` to `build.env` and follow instructions in the comments
to properly up the local environment for E2E tests. You will need to set:

- the absolute path to your cloud-sql-proxy working directory
- the empty Google Cloud project name.

### Run the tests

Run `make gcloud_test` in the base directory of this project. This will run the
following make targets in sequence:

- `make gcloud_test_infra` will use Terraform to create an artifact registry,
  GKE cluster and postgres database.
- `make gcloud_proxy_image_push` will build the Cloud SQL Proxy image from our
  cloud-sql-proxy working directory and push it to the operator.
- `make gcloud_test_run` will build docker images for the operator and the Cloud
  SQL Proxy, and push those images to the artifact registry.
- `make gcloud_test_cleanup` will remove all deployments and configuration from
  the kuberentes created by the end-to-end tests.

The first time you run `make gcloud_test` it may take 20-30 minutes to provision
Google Cloud resources for the tests. Subsequent runs will reuse the Google
Cloud resources, and therefore will run much faster.

When you are developing end-to-end tests, you may sometimes use these build
targets as short-cuts to run the tests faster:

- `make gcloud_test_run` will build and deploy the operator docker images from
  your working directory and run the end-to-end tests.
- `make gcloud_test_run_gotest` will just run the end-to-end tests without
  rebuilding images or checking infrastructure.
- `make k9s` will open the k9s tool pointing at the . K9s is a cool terminal UI
  that simplifies browsing the  
  state of a kubernetes cluster.

Clean up after the end-to-end tests with these targets:

- `make gcloud_test_cleanup` will remove the operator and all test deployments
  from the kubernetes cluster.
- `make gcloud-test-infra-cleanup` will remove all the Google Cloud
  infrastructure used by the test.

# Test Organization

Guidelines for writing test:

Don't write end-to-end tests when a unit test or integration test will do.

Each test case should be performed in its own namespace so that they do not
create conflicts or dependencies between tests.

Each test case should represent a complete scenario. You may not use one test
case to set up the prerequisite state for another test case.

## End-to-end Tests

End-to-end tests go in the `internal/e2e_test` package. These tests run test case
scenarios using real Google Cloud infrastructure.

The Google Cloud infrastructure is always exclusively provisioned using the
Terraform project in `testinfra`. Configuring the Google Cloud infrastructure
by hand after terraform runs is not allowed.

The utility functions in `internal/e2e` are intended to be used exclusively for
end-to-end tests.

## Integration Tests

Integration tests go in the `internal/testintegration_test` package. Integration tests ensure
that our operator interacts correctly with the kubernetes API.

The utility functions in `internal/testintegration` are intended to be used exclusively for
integration tests.

Integration tests are run using the Kubebuilder's `envtest` tool. This tool sets
up a local kubernetes API server so that we can test if our operator interacts
with kubernetes correctly.

Integration tests do not test any live Google Cloud infrastructure. They do not
test any pods created by kubernetes.

## Helpers

The utility functions in `internal/testhelpers` are intended to be reused between e2e
and integration tests.

The test cases in `internal/testhelpers/testcases.go` are intended to be reused between e2e
and integration tests. They require a TestCaseParams as input, and use a
reference to `*testing.T` to report the results.

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

## E2E Test Harness

TODO: Build an e2e test harness application

E2E tests run a test harness application as their primary workload. This
application in `internal/testharness` is a docker container that expects to connect to a
database through the Cloud SQL Proxy. It can be configured to connect via TCP or
Unix sockets. It reports its liveness and readiness through kubernetes.

# E2E Test Cases

## Connectivity
E2e tests need to ensure that the operator works for a variety of possible
connection scenarios. The test cases are designed to cover permutations of these
configuration dimensions:

| Test Dimensions | Values                |                            |            |     |         |     |   
|-----------------|-----------------------|----------------------------|------------|-----|---------|-----|
| K8s workload    | Deployment            | StatefulSet                | DaemonSet  | Job | CronJob | Pod |
| Identity        | GKE Workload Identity | K8s Secret with json creds |            |     |         |     |
| DB Type         | MySQL                 | Postgres                   | SQL Server |     |         |     | 
| DB Endpoint     | Public                | Private IP                 |            |     |         |     | 
| DB User         | Database user         | IAM AuthN                  |            |     |         |     | 
| Socket Type     | TCP                   | Unix                       | FUSE       |     |         |     | 

To cover all these cases, we will implement the following list of end-to-end scenarios

| Network    | Socket | Identity           | DB Type    | DB User   | K8s Workload |
|------------|--------|--------------------|------------|-----------|--------------|
| public ip  | tcp    | workload identity  | mysql      | db-user   | Deployment   |
| public ip  | tcp    | file in k8s secret | postgres   | db-user   | Pod          |
| public ip  | tcp    | workload identity  | sql server | db-user   | StatefulSet  |
| public ip  | tcp    | workload identity  | mysql      | db-user   | Job          |
| public ip  | tcp    | workload identity  | mysql      | db-user   | Cronjob      |
| public ip  | tcp    | workload identity  | mysql      | db-user   | DaemonSet    |
| private ip | tcp    | workload identity  | mysql      | db-user   | Deployment   |
| public ip  | tcp    | workload identity  | mysql      | db-user   | Deployment   |
| public ip  | tcp    | vm identity        | mysql      | db-user   | Deployment   |
| public ip  | tcp    | workload identity  | postgres   | iam-authn | Deployment   |
| public ip  | unix   | file in k8s secret | mysql      | db-user   | Deployment   |
| public ip  | fuse   | file in k8s secret | mysql      | db-user   | Deployment   |

## Correctness of State Changes
Additionally, end-to-end scenarios need to ensure that the operator correctly creates
updates and removes the proxy workloads when operating in a real cluster. These
are scenarios that cannot be tested in integration or unit tests because they require
a live cluster with real workloads.

#### Happy Path Create Workload 
- Create a deployment with 5 replicas, maxUnavailable of 2.
- Create a AuthProxyWorkload matching the deployment
- Check that the workload pods have a proxy
- Check that the workload application can connect to the database through the proxy

#### Happy Path Update Workload
- Do "Happy Path Create Workload"
- Update the deployment's EnvVar settings so that the pods are recreated.
- Check that the updated workload pods have a proxy
- Check that the workload application can connect to the database through the proxy

#### Update AuthProxyWorkload
- Do "Happy Path Create Workload"
- Update the AuthProxyWorkload changing the database connection string
- Check that the workload pods are updated with the new proxy settings
- Check that the workload application can connect to the new database through the proxy
- Check that during the update of the deployment with new proxy configuration, 
  the Deployment's maxUnavailable setting is not violated.

#### Delete AuthProxyWorkload
- Do "Happy Path Create Workload"
- Attempt to delete the AuthProxyWorkload
- Expect an error because the AuthProxyWorkload configuration is in use
- Delete the deployment
- Attempt to delete the AuthProxyWorkload
- Check that the AuthProxyWorkload deletes successfully 

# Load Test Cases
Additionally, end-to-end scenarios need to ensure that the operator correctly creates
updates and removes the proxy workloads when operating in a real cluster. These
are scenarios that cannot be tested in integration or unit tests because they require
a live cluster with real workloads.

Load tests should be run before releases by the dev team to ensure that
the operator works. Load tests should be fully automated like the e2e tests. Load
tests should not be run automatically due to cost and time. 

#### Large Cluster, 1 Big Deployment, 1 AuthProxyWorkload
- Create a k8s cluster with 50 large nodes
- Create a deployment with 100 replicas
- Create an AuthProxyWorkload that matches the 1 deployment
- Check how long it takes to roll out the proxy configuration to the workloads
- Check memory and CPU consumption for the operator

#### Large Cluster, Many Small Deployments, 1 AuthProxyWorkload
- Create a k8s cluster with 50 large nodes
- Create 100 Deployments with 3 replicas
- Create 2 AuthProxyWorkloads, each matches half the deployments
- Check how long it takes to roll out the proxy configuration to the workloads
- Check memory and CPU consumption for the operator

#### Large Cluster, Many Small Deployments, Many AuthProxyWorkload
- Create a k8s cluster with 50 large nodes
- Create 100 Deployments with 3 replicas
- Create 100 AuthProxyWorkloads that matches the 1 deployment
- Check how long it takes to roll out the proxy configuration to the cluster
- Check memory and CPU consumption for the operator

#### Huge cluster
Run the large cluster test cases at 5x to 10x the size, trying to find
the size where the scenario is too big for the operator.  
