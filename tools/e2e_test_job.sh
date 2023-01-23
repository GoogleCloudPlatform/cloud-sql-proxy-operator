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

set -euxo

E2E_PROJECT_ID=cloud-sql-operator-testing

# Install GCC and other essential build tools
apt-get update
apt-get install -y zip unzip build-essential

# Install and configure GCloud CLI
mkdir -p bin
curl -L -o bin/gcloud-cli.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-413.0.0-linux-x86_64.tar.gz
( cd bin && tar -zxf gcloud-cli.tar.gz )
./bin/google-cloud-sdk/bin/gcloud config set project "$E2E_PROJECT_ID"
./bin/google-cloud-sdk/bin/gcloud config set compute/zone "us-central1"
export PATH=$PATH:$PWD/bin/google-cloud-sdk/bin
which gcloud
gcloud components install --quiet gke-gcloud-auth-plugin

# Install helm
curl -L -o bin/helm.tar.gz https://get.helm.sh/helm-v3.10.3-linux-amd64.tar.gz
( cd bin && tar -zxf helm.tar.gz && ls -al)
export PATH=$PATH:$PWD/bin/linux-amd64
which helm

# Install go
curl -L -o bin/go.tar.gz https://go.dev/dl/go1.18.10.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf bin/go.tar.gz
export PATH=$PATH:/usr/local/go/bin
go version

# Set the e2e test project id and other params from
# the Cloud Build environment
cat > build.env <<EOF
E2E_PROJECT_ID=$E2E_PROJECT_ID
NODEPOOL_SERVICEACCOUNT_EMAIL=$NODEPOOL_SERVICEACCOUNT_EMAIL
WORKLOAD_ID_SERVICEACCOUNT_EMAIL=$WORKLOAD_ID_SERVICEACCOUNT_EMAIL
TFSTATE_STORAGE_BUCKET=$TFSTATE_STORAGE_BUCKET
EOF

echo "Running tests on landscape ${ENVIRONMENT_NAME:-undefined}"

# Run e2e test
if make e2e_test_job ; then
  echo "E2E Test Passed"
  test_exit_code=0
else
  echo "E2E Test Failed"
  test_exit_code=1
fi

exit $test_exit_code
