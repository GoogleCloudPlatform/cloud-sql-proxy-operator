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


#expects $PROJECT_DIR
if [[ -z "$PROJECT_DIR" ]]; then
  echo "expects PROJECT_DIR to be set to the root directory of the operator project."
  exit 1
fi

#expects $E2E_PROJECT_ID
if [[ -z "$E2E_PROJECT_ID" ]]; then
  echo "expects E2E_PROJECT_ID to be set to the gcloud project id for testing."
  exit 1
fi

#expects $KUBECONFIG_GCLOUD
if [[ -z "$KUBECONFIG_GCLOUD" ]]; then
  echo "expects KUBECONFIG_GCLOUD to be set the location where kubeconfig should be written."
  exit 1
fi

#expects $E2E_DOCKER_URL_FILE
if [[ -z "$E2E_DOCKER_URL_FILE" ]]; then
  echo "expects E2E_DOCKER_URL_FILE to be set the location where docker url should be written."
  exit 1
fi

if [[ "${1:-}" == "destroy" ]] ; then
  DESTROY="-destroy"
else
  DESTROY=""
fi


SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

TERRAFORM="$PROJECT_DIR/bin/terraform"
KUBECTL="$PROJECT_DIR/bin/kubectl"

set -euxo

# Begin terraform setup

cd "$SCRIPT_DIR"
DATA_DIR="$SCRIPT_DIR/../bin/tf"
mkdir -p "$DATA_DIR"
cp -r $SCRIPT_DIR/* "$DATA_DIR"

"$TERRAFORM" -chdir="$DATA_DIR" init

"$TERRAFORM"  -chdir="$DATA_DIR" apply $DESTROY -parallelism=5 -auto-approve \
  -var "gcloud_bin=$(which gcloud)" \
  -var "gcloud_docker_url_file=$E2E_DOCKER_URL_FILE" \
  -var "project_id=$E2E_PROJECT_ID" \
  -var "kubeconfig_path=$KUBECONFIG_GCLOUD" \
  -var "testinfra_json_path=$PROJECT_DIR/bin/testinfra.json"

gcloud auth configure-docker us-central1-docker.pkg.dev
