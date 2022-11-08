#!/usr/bin/env bash
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


set -euxo
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
PROJECT_DIR=$( dirname "$SCRIPT_DIR")

cd "$PROJECT_DIR"

##
# Release Process
VERSION=$(cat "$PROJECT_DIR/version.txt" | tr -d "\n")
RELEASE_REPO_PATH=cloud-sql-connectors/cloud-sql-operator-dev
RELEASE_IMAGE_NAME=cloud-sql-proxy-operator

RELEASE_TAG_PATH="${RELEASE_REPO_PATH}/${RELEASE_IMAGE_NAME}:${VERSION}"
RELEASE_IMAGE_VERSION_URL="gcr.io/${RELEASE_TAG_PATH}"
RELEASE_IMAGE_TAGS="-t gcr.io/${RELEASE_TAG_PATH} -t us.gcr.io/${RELEASE_TAG_PATH} -t eu.gcr.io/${RELEASE_TAG_PATH} -t asia.gcr.io/${RELEASE_TAG_PATH}"

# Copy tools from the cached image
if [[ -d /tools ]] ; then
  mkdir -p "$PROJECT_DIR/bin"
  cp -r -f /tools/bin/* "$PROJECT_DIR/bin"
fi


# Build the docker image
IMG="$RELEASE_IMAGE_VERSION_URL" \
  EXTRA_IMAGE_TAGS="$RELEASE_IMAGE_TAGS" \
  make -f operator.mk build

# Upload the installer files to the storage bucket
gcloud storage --project cloud-sql-connectors cp bin/install.sh "gs://cloud-sql-connectors/cloud-sql-proxy-operator-dev/${VERSION}/install.sh"
gcloud storage --project cloud-sql-connectors cp bin/cloud-sql-proxy-operator.yaml "gs://cloud-sql-connectors/cloud-sql-proxy-operator-dev/${VERSION}/cloud-sql-proxy-operator.yaml"

