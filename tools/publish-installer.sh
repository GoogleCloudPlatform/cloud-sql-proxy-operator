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


RELEASE_PROJECT_ID="cloud-sql-connectors"
if [[ -n "${IS_RELEASE_BUILD:-}" ]] ; then
  BUCKET_PATH="gs://cloud-sql-connectors/cloud-sql-proxy-operator"
else
  BUCKET_PATH="gs://cloud-sql-connectors/cloud-sql-proxy-operator-dev"
fi

##
# Release Process
if [[ -n ${RELEASE_TEST_BUILD_ID:-} ]] ; then
  VERSION="${RELEASE_TEST_BUILD_ID}"
else
  VERSION="v$(cat "$PROJECT_DIR/version.txt" | tr -d "\n")"
fi

# Upload the installer files to the storage bucket
gcloud storage --project "$RELEASE_PROJECT_ID" cp installer/install.sh "${BUCKET_PATH}/${VERSION}/install.sh"
gcloud storage --project "$RELEASE_PROJECT_ID" cp installer/cloud-sql-proxy-operator.yaml "${BUCKET_PATH}/${VERSION}/cloud-sql-proxy-operator.yaml"

