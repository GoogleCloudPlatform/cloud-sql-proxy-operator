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

###
# run.sh is used by make to launch the terraform scripts as part of the
# end-to-end testing process. This is not intended to be a stand-alone shell
# script.
#
# Usage:
# $ run.sh <command>
#
# This script will do these command:
#
# apply - Build an e2e test environment for the local developer to run e2e tests.
#         This will run `terraform apply` on the ./permissions terraform project
#         and then on ./resources project, passing the values between projects.
#
# destroy - Tear down the e2e test environment for the local developer.
#         This runs `terraform destroy` on ./resources project, removing resources
#         from the google cloud project used by the e2e tests.
#
# apply_e2e_job - Build an e2e test environment for the e2e CI jobs to use.
#         CI jobs for e2e testing use pre-configured Google Cloud accounts
#         that require a slightly different configuration than the e2e environment
#         for local development.
#
# This script accepts inputs as environment variables:
#
#   PROJECT_DIR - The directory containing the Makefile.
#   ENVIRONMENT_NAME - The name of the e2e test environment to act upon. There
#     may be many e2e test environments in the same Google Cloud project.
#   E2E_PROJECT_ID - The Google Cloud project ID to act upon.
#   KUBECONFIG_E2E - The output filename for the kubeconfig json file
#     for the kubernetes cluster for the e2e environment.
#   PRIVATE_KUBECONFIG_E2E - The output filename for the kubeconfig json file
#     for the private ip kubernetes cluster for the e2e environment.
#   E2E_DOCKER_URL_FILE - The output filename for a text file containing the
#     URL to the docker container registry for the e2e test environment.
#
#  These additional environment variable are used by apply_e2e_job for E2E CI jobs:
#   NODEPOOL_SERVICEACCOUNT_EMAIL - the name of the service account to assign
#     to the kubernetes cluster node pool.
#   WORKLOAD_ID_SERVICEACCOUNT_EMAIL - the name of the service account to use
#     when testing workload identity.
#   TFSTATE_STORAGE_BUCKET - the name of the Google Cloud Storage bucket to use
#     to store the terraform state.

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

    run_tf resources "$TESTINFRA_JSON_FILE" \
        -var "gcloud_docker_url_file=$E2E_DOCKER_URL_FILE" \
        -var "project_id=$E2E_PROJECT_ID" \
        -var "kubeconfig_path=$KUBECONFIG_E2E" \
        -var "private_kubeconfig_path=$PRIVATE_KUBECONFIG_E2E" \
        -var "environment_name=$ENVIRONMENT_NAME" \
        -var "nodepool_serviceaccount_email=$nodepool_serviceaccount_email"

    gcloud auth configure-docker us-central1-docker.pkg.dev
}

# Destroy the local development terraform resources
function destroy() {
    nodepool_serviceaccount_email=$(jq -r .nodepool_serviceaccount_email < "$DATA_DIR/permissions_out.json")
    run_tf resources TESTINFRA_JSON_FILE -destroy \
        -var "gcloud_docker_url_file=$E2E_DOCKER_URL_FILE" \
        -var "project_id=$E2E_PROJECT_ID" \
        -var "kubeconfig_path=$KUBECONFIG_E2E" \
        -var "private_kubeconfig_path=$PRIVATE_KUBECONFIG_E2E" \
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
      -var "private_kubeconfig_path=$PRIVATE_KUBECONFIG_E2E" \
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

#expects PRIVATE_KUBECONFIG_E2E
if [[ -z "$PRIVATE_KUBECONFIG_E2E" ]]; then
  echo "expects PRIVATE_KUBECONFIG_E2E to be set the location where kubeconfig should be written."
  FAIL=1
fi

#expects $E2E_DOCKER_URL_FILE
if [[ -z "$E2E_DOCKER_URL_FILE" ]]; then
  echo "expects E2E_DOCKER_URL_FILE to be set the location where docker url should be written."
  FAIL=1
fi

#expects TESTINFRA_JSON_FILE
if [[ -z "$TESTINFRA_JSON_FILE" ]]; then
  echo "expects TESTINFRA_JSON_FILE to be set the location where test infrastructure output file be written."
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

echo "TIME: $(date) Start terraform reconcile action $ACTION"

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

echo "TIME: $(date) End terraform reconcile action $ACTION"

if [[ -n "$FAIL" ]] ; then
  exit 1
fi
