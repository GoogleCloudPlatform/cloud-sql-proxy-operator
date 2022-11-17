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

VERSION="__VERSION__"
CERT_MANAGER_VERSION="__CERT_MANAGER_VERSION__"

if ! which kubectl ; then
  echo "kubectl, the kubernetes command line client, was not found in the PATH."
  echo "See https://kubernetes.io/docs/tasks/tools/ for instructions on how to"
  echo "install kubectl."
  exit 1
fi

# Install cert-manager
kubectl apply -f "https://github.com/cert-manager/cert-manager/releases/download/$CERT_MANAGER_VERSION/cert-manager.yaml"

# Wait for cert-manager to become available before continuing
kubectl rollout status deployment cert-manager -n cert-manager --timeout=90s

# Install the cloud-sql-proxy-operator
kubectl apply -f "https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy-operator-dev/$VERSION/cloud-sql-proxy-operator.yaml"

# Wait for cloud-sql-proxy-operator to become available
kubectl rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s
