#!/bin/bash

#####
# Kokoro release script
# runs on ubuntu-gcp VMs.

set -eo pipefail

echo "Building snapshot release $RELEASE_TEST_BUILD_ID at commit $RELEASE_COMMIT_SHA"

set +x
echo "*"
echo "* Fetching Third Party Licenses"
echo "*"

go run github.com/google/go-licenses@v1.6.0 save --save_path ThirdPartyLicenses .

#echo "*"
#echo "* Publishing installer to GCS"
#echo "*"
# tools/publish-installer.sh

set +x
echo "*"
echo "* Publishing docker container"
echo "*"
set -x
ls -al
which make
make build_push_docker

echo "*"
echo "* Listing artifacts"
echo "*"
ls -al bin/
