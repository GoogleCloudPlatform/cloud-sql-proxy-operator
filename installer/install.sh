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

set -euxo # exit 1 from the script when command fails

# If CSQL_OPERATOR_VERSION is not set, use the release version: v1.1.1-dev.
CSQL_OPERATOR_VERSION="${CSQL_OPERATOR_VERSION:-v1.1.1-dev}"

# If CSQL_CERT_MANAGER_VERSION is not set, use the default: v1.12.1.
CSQL_CERT_MANAGER_VERSION="${CSQL_CERT_MANAGER_VERSION:-v1.12.1}"

# If CSQL_OPERATOR_URL is not set, use the default value from the CSQL_OPERATOR_VERSION
CSQL_OPERATOR_URL="${CSQL_OPERATOR_URL:-https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy-operator/$CSQL_OPERATOR_VERSION/cloud-sql-proxy-operator.yaml}"

# Ensure kubectl exists
if ! which kubectl ; then
  echo "kubectl, the kubernetes command line client, was not found in the PATH."
  echo "See https://kubernetes.io/docs/tasks/tools/ for instructions on how to"
  echo "install kubectl."
  exit 1
fi

# Ensure helm exists
if ! which helm ; then
  echo "helm, the installer for kubernetes applications, was not found in the PATH."
  echo "See https://helm.sh/docs/intro/install/ for instructions on how to"
  echo "install helm."
  exit 1
fi

# Install cert-manager using helm
if ! helm get all -n cert-manager cert-manager > /dev/null ; then
  helm repo add jetstack https://charts.jetstack.io
  helm repo update
  helm install \
    cert-manager jetstack/cert-manager \
    --namespace cert-manager \
    --version "$CSQL_CERT_MANAGER_VERSION" \
    --create-namespace \
    --set global.leaderElection.namespace=cert-manager \
    --set installCRDs=true
fi

# Install the cloud-sql-proxy-operator
kubectl apply -f "$CSQL_OPERATOR_URL"

# Wait for cloud-sql-proxy-operator to become available
kubectl rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s
