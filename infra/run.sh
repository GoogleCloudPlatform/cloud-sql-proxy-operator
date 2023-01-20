#!/usr/bin/env bash
# Copyright 2023 Google LLC
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


# Run terraform with appropriate settings
function run_tf() {
  subproject=$1
  shift
  output_json=$1
  shift
  arr=("$@")

  tf_dir="$DATA_DIR/$subproject"

  "$TERRAFORM" -chdir="$tf_dir" init

  "$TERRAFORM"  -chdir="$tf_dir" apply -parallelism=5 -auto-approve \
    -var "gcloud_bin=$(which gcloud)" -var "output_json_path=$output_json" "${arr[@]}"
}

# Apply the terraform for local development e2e tests
function apply() {
    run_tf permissions "$DATA_DIR/permissions_out.json" \
        -var "project_id=$E2E_PROJECT_ID" \
        -var "environment_name=$ENVIRONMENT_NAME"

    # Read nodepool_service_acount from the output of the permissions project
    nodepool_serviceaccount_email=$(jq -r .nodepool_serviceaccount_email < "$DATA_DIR/permissions_out.json")

    run_tf resources "$PROJECT_DIR/bin/testinfra.json" \
        -var "gcloud_docker_url_file=$E2E_DOCKER_URL_FILE" \
        -var "project_id=$E2E_PROJECT_ID" \
        -var "kubeconfig_path=$KUBECONFIG_E2E" \
        -var "environment_name=$ENVIRONMENT_NAME" \
        -var "nodepool_serviceaccount_email=$nodepool_serviceaccount_email"

    gcloud auth configure-docker us-central1-docker.pkg.dev
}

# Destroy the local development terraform resources
function destroy() {
    nodepool_serviceaccount_email=$(jq -r .nodepool_serviceaccount_email < "$DATA_DIR/permissions_out.json")
    run_tf resources "$PROJECT_DIR/bin/testinfra.json" -destroy \
        -var "gcloud_docker_url_file=$E2E_DOCKER_URL_FILE" \
        -var "project_id=$E2E_PROJECT_ID" \
        -var "kubeconfig_path=$KUBECONFIG_E2E" \
        -var "environment_name=$ENVIRONMENT_NAME" \
        -var "nodepool_serviceaccount_email=$nodepool_serviceaccount_email"
}

# Apply the terraform resources for the e2e test job
function apply_e2e_job() {

  #expects NODEPOOL_SERVICEACCOUNT_EMAIL to be set by the caller
  if [[ -z "${NODEPOOL_SERVICEACCOUNT_EMAIL:-}" ]]; then
    echo "expects NODEPOOL_SERVICEACCOUNT_EMAIL to be set the email address for the nodepool service account."
    exit 1
  fi

  #expects WORKLOAD_ID_SERVICEACCOUNT_EMAIL to be set by the caller
  if [[ -z "${WORKLOAD_ID_SERVICEACCOUNT_EMAIL:-}" ]]; then
    echo "expects WORKLOAD_ID_SERVICEACCOUNT_EMAIL to be set the email address for the workload id service account."
    exit 1
  fi

  #expects TFSTATE_STORAGE_BUCKET to be set by the caller
  if [[ -z "${TFSTATE_STORAGE_BUCKET:-}" ]]; then
    echo "expects TFSTATE_STORAGE_BUCKET to be set the name of the cloud storage bucket where state is maintained."
    exit 1
  fi

  # Use a remote backend for the state defined in the storage bucket, so that the
  # state can be reused between runs
  cat > $DATA_DIR/resources/backend.tf <<EOF
terraform {
 backend "gcs" {
   bucket  = "$TFSTATE_STORAGE_BUCKET"
   prefix  = "terraform/$ENVIRONMENT_NAME"
 }
}
EOF

  run_tf resources "$PROJECT_DIR/bin/testinfra.json" \
      -var "gcloud_docker_url_file=$E2E_DOCKER_URL_FILE" \
      -var "project_id=$E2E_PROJECT_ID" \
      -var "kubeconfig_path=$KUBECONFIG_E2E" \
      -var "environment_name=$ENVIRONMENT_NAME" \
      -var "nodepool_serviceaccount_email=$NODEPOOL_SERVICEACCOUNT_EMAIL"

  gcloud auth configure-docker us-central1-docker.pkg.dev

}


SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
TERRAFORM="$PROJECT_DIR/bin/terraform"
KUBECTL="$PROJECT_DIR/bin/kubectl"

FAIL=""

##
# Validate Script Inputs

#expects $PROJECT_DIR
if [[ -z "$PROJECT_DIR" ]]; then
  echo "expects PROJECT_DIR to be set to the root directory of the operator project."
  FAIL=1
fi

#expects $ENVIRONMENT_NAME
if [[ -z "$ENVIRONMENT_NAME" ]]; then
  echo "expects ENVIRONMENT_NAME to be set to the root directory of the operator project."
  FAIL=1
fi

#expects $E2E_PROJECT_ID
if [[ -z "$E2E_PROJECT_ID" ]]; then
  echo "expects E2E_PROJECT_ID to be set to the gcloud project id for testing."
  FAIL=1
fi

#expects KUBECONFIG_E2E
if [[ -z "$KUBECONFIG_E2E" ]]; then
  echo "expects KUBECONFIG_E2E to be set the location where kubeconfig should be written."
  FAIL=1
fi

#expects $E2E_DOCKER_URL_FILE
if [[ -z "$E2E_DOCKER_URL_FILE" ]]; then
  echo "expects E2E_DOCKER_URL_FILE to be set the location where docker url should be written."
  FAIL=1
fi

ACTION="${1:-}"
shift

case $ACTION in
"apply")
  ;;
"destroy")
  ;;
"apply_e2e_job")
  ;;
 *)
   echo "Unknown action: $ACTION"
   FAIL=1
  ;;
esac

if [[ -n "$FAIL" ]] ; then
  exit 1
fi


set -euxo

##
# Run the script

cd "$SCRIPT_DIR"
DATA_DIR="$PROJECT_DIR/bin/tf-new"
mkdir -p "$DATA_DIR"
cp -r $SCRIPT_DIR/* "$DATA_DIR"

case $ACTION in
"apply")
    apply
  ;;
"destroy")
    destroy
  ;;
"apply_e2e_job")
    apply_e2e_job
  ;;
 *)
   echo "Unknown action: $ACTION"
   FAIL=1
  ;;
esac
