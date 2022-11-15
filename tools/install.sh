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
IMAGE_URL="__IMAGE_URL__"

echo "Placeholder. Install logic for Cloud SQL Operator v __VERSION__ will go here"

if ! which gcloud ; then
  echo "gcloud, the Google Cloud CLI, was not found in the PATH."
  echo "See https://cloud.google.com/sdk/docs/install for instructions on how to"
  echo "install the Google Cloud CLI."
  exit 1
fi
if ! which kubectl ; then
  echo "kubectl, the kubernetes command line client, was not found in the PATH."
  echo "See https://kubernetes.io/docs/tasks/tools/ for instructions on how to"
  echo "install kubectl."
  exit 1
fi

# Enable GKE auth plugin
if ! which gke-gcloud-auth-plugin ; then
  gcloud components install gke-gcloud-auth-plugin
fi

export USE_GKE_GCLOUD_AUTH_PLUGIN=True

# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.9.1/cert-manager.yaml

# Wait for cert-manager to become available before continuing
kubectl rollout status deployment cert-manager -n cert-manager --timeout=90s

# Install the cloud-sql-proxy-operator
kubectl apply -f https://github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/releases/$VERSION/cloud-sql-proxy-operator.yaml

# Wait for cloud-sql-proxy-operator to become available
kubectl rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s
