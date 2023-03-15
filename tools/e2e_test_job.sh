#!/usr/bin/env bash
# Copyright 2023 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

###
# This script is used by the E2E test job defined in .build/e2e_test.yaml
# to prepare the Cloud Build environment and run the end-to-end tests.
#
echo "TIME: $(date) Begin Script"
set -euxo

if ! which zip ; then
  echo "TIME: $(date) Install GCC"
  # Install GCC and other essential build tools
  apt-get update
  apt-get install -y zip unzip build-essential
fi

# Install and configure GCloud CLI
mkdir -p bin
if ! which gcloud ; then
  echo "TIME: $(date) Install GCloud CLI"
  curl -L -o bin/gcloud-cli.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-413.0.0-linux-x86_64.tar.gz
  ( cd bin && tar -zxf gcloud-cli.tar.gz )
  ./bin/google-cloud-sdk/bin/gcloud config set project "$E2E_PROJECT_ID"
  ./bin/google-cloud-sdk/bin/gcloud config set compute/zone "us-central1"
  export PATH=$PATH:$PWD/bin/google-cloud-sdk/bin
  which gcloud
else
  echo "Using installed gcloud"
  gcloud version
fi

gcloud components install --quiet gke-gcloud-auth-plugin

# Install helm
if ! which helm ; then
  echo "TIME: $(date) Install Helm"
  curl -L -o bin/helm.tar.gz https://get.helm.sh/helm-v3.10.3-linux-amd64.tar.gz
  ( cd bin && tar -zxf helm.tar.gz && ls -al)
  export PATH=$PATH:$PWD/bin/linux-amd64
  which helm
else
  echo "Using installed helm"
  helm version
fi

# Install go1.20
if ! ( which go && go version | grep 'go1.20' )  ; then
  echo "TIME: $(date) Install Go"
  curl -L -o bin/go.tar.gz https://go.dev/dl/go1.20.linux-amd64.tar.gz
  rm -rf /usr/local/go && tar -C /usr/local -xzf bin/go.tar.gz
  export PATH=$PATH:/usr/local/go/bin
  go version
else
  echo "Using installed go"
  go version
fi

# Set the e2e test project id and other params from
# the Cloud Build environment
echo "TIME: $(date) Configure Make Env"

# Don't log the command, it contains secrets
set +x
cat > build.env <<EOF
E2E_PROJECT_ID=$E2E_PROJECT_ID
NODEPOOL_SERVICEACCOUNT_EMAIL=$NODEPOOL_SERVICEACCOUNT_EMAIL
WORKLOAD_ID_SERVICEACCOUNT_EMAIL=$WORKLOAD_ID_SERVICEACCOUNT_EMAIL
TFSTATE_STORAGE_BUCKET=$TFSTATE_STORAGE_BUCKET
EOF
set -x

echo "TIME: $(date) Run Tests"
echo "Running tests on environment ${ENVIRONMENT_NAME:-undefined}"

# Run e2e test, filtering the stdout so that it only logs go test results.
if make e2e_test_job > bin/e2e_test.log 2>&1 ; then
  echo "STATUS: E2E Test Passed"
  test_exit_code=0
else
  echo "STATUS: E2E Test Failed"
  test_exit_code=1
fi

# Upload full e2e log to the storage bucket
gcloud storage cp bin/e2e_test.log "gs://$TFSTATE_STORAGE_BUCKET/github-action-log/run-$GITHUB_RUN_ID.log"
echo "Uploaded full e2e log to <storage-bucket>/github-action-logs/run-$GITHUB_RUN_ID.log"

# Print go test results to stdout, ignore error
egrep '(---)|(github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/tests)' bin/e2e_test.log || true

echo "TIME: $(date) Done"
exit $test_exit_code
