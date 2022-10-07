#!/usr/bin/env bash
set -euxo

#expects $PROJECT_DIR
if [[ -z "$PROJECT_DIR" ]]; then
  echo "expects PROJECT_DIR to be set to the root directory of the operator project."
  exit 1
fi

#expects $GCLOUD_PROJECT_ID
if [[ -z "$GCLOUD_PROJECT_ID" ]]; then
  echo "expects GCLOUD_PROJECT_ID to be set to the gcloud project id for testing."
  exit 1
fi

#expects $KUBECONFIG_GCLOUD
if [[ -z "$KUBECONFIG_GCLOUD" ]]; then
  echo "expects KUBECONFIG_GCLOUD to be set the location where kubeconfig should be written."
  exit 1
fi

#expects $GCLOUD_DOCKER_URL_FILE
if [[ -z "$GCLOUD_DOCKER_URL_FILE" ]]; then
  echo "expects GCLOUD_DOCKER_URL_FILE to be set the location where docker url should be written."
  exit 1
fi

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

TERRAFORM="$PROJECT_DIR/bin/terraform"
KUBECTL="$PROJECT_DIR/bin/kubectl"

# Begin terraform setup

cd "$SCRIPT_DIR"
DATA_DIR="$SCRIPT_DIR/../../../bin/tf"
mkdir -p "$DATA_DIR"
cp -r $SCRIPT_DIR/* "$DATA_DIR"

"$TERRAFORM" -chdir="$DATA_DIR" init

"$TERRAFORM"  -chdir="$DATA_DIR" apply -parallelism=5 -auto-approve \
  -var "gcloud_bin=$(which gcloud)" \
  -var "gcloud_docker_url_file=$GCLOUD_DOCKER_URL_FILE" \
  -var "project_id=$GCLOUD_PROJECT_ID" \
  -var "kubeconfig_path=$KUBECONFIG_GCLOUD" \
  -var "testinfra_json_path=$PROJECT_DIR/bin/testinfra.json"

gcloud auth configure-docker us-central1-docker.pkg.dev
