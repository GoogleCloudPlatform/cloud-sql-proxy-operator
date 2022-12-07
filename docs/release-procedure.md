# Release Procedure

##Create release trigger commit

Create a commit to trigger Release Please github app to begin a release on the 
main branch

```
git checkout main
git commit --allow-empty -m "chore: release 0.0.3" -m "Release-As: 0.0.3"
```

## Wait for Github Workflows

- Wait for Release Please to create a PR.
- Wait for all PR checks to pass.

## Manually Run E2E tests. 

Until this is automated, we need to manually run the e2e tests. Do the following:

- Check out the pr branch locally
- Run `make e2e_test`
- Add a comment to the PR indicating that the e2e test run succeeded.

## Merge the release PR

Approve the PR and merge to main. release-please will continue running the 

## Wait for release job to complete

[Cloud Build](https://pantheon.corp.google.com/cloud-build/builds?project=cloud-sql-connectors) in the
cloud-sql-connectors project has a trigger named
[csql-operator-release](https://pantheon.corp.google.com/cloud-build/triggers;region=global/edit/a60e1618-4b40-4e32-ab5f-4ab753ce2c6a?project=cloud-sql-connectors) 
that will to build when a new tag is added. 

Go to the cloud build console and see that the release build completes successfully.


