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

function install() {
  helm repo update --kubeconfig "${KUBECONFIG}"

  helm --kubeconfig "${KUBECONFIG}" uninstall cloud-sql-proxy-operator || true

  helm --kubeconfig "${KUBECONFIG}" uninstall cloud-sql-proxy-operator-crds || true

  kubectl delete ns helm-cloud-sql-operator || true

  helm --kubeconfig "${KUBECONFIG}" "install" --replace \
    cloud-sql-proxy-operator-crds "$PROJECT_DIR/helm/cloud-sql-operator-crds" \
    --set "operatorNamespace=helm-cloud-sql-operator" \
    --set "operatorName=cloud-sql-proxy-operator"

  helm --kubeconfig "${KUBECONFIG}" "install" --replace \
    cloud-sql-proxy-operator "$PROJECT_DIR/helm/cloud-sql-operator" \
    --create-namespace \
    --namespace helm-cloud-sql-operator \
    --set "image.repository=$E2E_OPERATOR_URL"
}


# Configure script to fail on any command error
set -euxo pipefail

# Find project directory, cd to project directory
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
PROJECT_DIR=$( dirname "$SCRIPT_DIR")
cd "$PROJECT_DIR"

# Validate input environment variables
#expects KUBECONFIG to be set by the caller
if [[ -z "${KUBECONFIG_E2E:-}" ]]; then
  echo "expects KUBECONFIG_E2E to be the path to the kubeconfig file for kubectl."
  exit 1
fi
if [[ -z "${PRIVATE_KUBECONFIG_E2E:-}" ]]; then
  echo "expects PRIVATE_KUBECONFIG_E2E to be the path to the kubeconfig file for kubectl."
  exit 1
fi

#expects E2E_OPERATOR_URL to be set by the caller
if [[ -z "${E2E_OPERATOR_URL:-}" ]]; then
  echo "expects E2E_OPERATOR_URL to be the URL to the operator image."
  exit 1
fi

export KUBECONFIG=$KUBECONFIG_E2E
install

