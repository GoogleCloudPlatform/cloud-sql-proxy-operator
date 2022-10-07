# End-to-End Tests

This project uses end-to-end tests to make sure that the operator, proxy and
everything works

## Get Started

After setting up your local development environment, please do the following:

- Create a new project in your

Run `gcloud components install gke-gcloud-auth-plugin`
and `gke-gcloud-auth-plugin --version`
to make sure you have the component installed.

# Test Organization

## Helpers
The utility functions in `helpers/testcases.go` are intended to be reused between e2e and

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
