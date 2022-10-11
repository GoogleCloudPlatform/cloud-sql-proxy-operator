#!/bin/bash

# Copyright 2022 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

##
# release.sh intended to be run by Cloud Build to release the software

set -euxo pipefail
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
PROJECT_DIR=$(dirname "$SCRIPT_DIR")

cd "$PROJECT_DIR"
mkdir -p .bin # for tools related to this script


##
# Update the release build id.
if [[ -n "${TEST_BUILD_ID:-}" ]] ; then
  # This is run from the Cloud Build command line
  export OPERATOR_BUILD_ID="$TEST_BUILD_ID"
elif [[ -n "${TAG_NAME:-}" ]]; then
  # This was run from a trigger in Cloud Build
  export OPERATOR_BUILD_ID="$COMMIT_SHA-$TAG_NAME"
elif [[ -n "${$COMMIT_SHA:-}" ]]; then
  # This was run from a trigger in Cloud Build
  export OPERATOR_BUILD_ID="$COMMIT_SHA"
else
  echo "This script was not run correctly, OPERATOR_BUILD_ID could not be set."
  exit 1
fi

echo "Operator Build ID: $OPERATOR_BUILD_ID"

function all() {
  ##
  # Install debian tools
  apt-get update
  apt-get install -y git build-essential curl

  ##
  # Install Go
  curl -L -o .bin/go.tar.gz  https://go.dev/dl/go1.18.7.linux-amd64.tar.gz
  rm -rf /usr/local/go && tar -C /usr/local -xzf .bin/go.tar.gz
  export PATH=$PATH:/usr/local/go/bin
  which go
  go version

  ##
  # install gcloud cli
  curl -L -o .bin/gcloud.tar.gz https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-405.0.0-linux-x86_64.tar.gz
  rm -rf /workspace/.bin/google-cloud-sdk && tar -C /workspace/.bin -xzf .bin/gcloud.tar.gz
  export PATH=$PATH:/workspace/.bin/google-cloud-sdk/bin
  which gcloud
  gcloud version

  ##
  # Release the container
  make release
}


$@
