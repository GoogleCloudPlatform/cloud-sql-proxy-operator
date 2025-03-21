#!/usr/bin/env bash
# Copyright 2025 Google LLC
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

set -euxo pipefail
KUBECTL=${KUBECTL:-bin/kubectl}
export USE_GKE_E2E_AUTH_PLUGIN=True
export KUBECONFIG=${KUBECONFIG:-bin/e2e-kubeconfig.yaml}

mkdir -p bin/ns
function remove_ns(){
  # Check that the namespace exists, return if not.
  if ! $KUBECTL get namespace "$1" ; then
    return
  fi

  # Tell kubernetes to delete the namespace, If it times out, force delete.
  if ! $KUBECTL delete namespace "$1" --timeout=10s ; then

    # Get the namespace, remove finalizers from the namespace spec.
    $KUBECTL get namespace "$1" -o json | \
      jq '.spec.finalizers = []' > "bin/ns/$1.json"

    # Force update the namespace resource, removing finalizers.
    # This will allow Kubernetes to continue the deletion of the resource.
    $KUBECTL replace --raw "/api/v1/namespaces/$1/finalize" -f "bin/ns/$1.json"
  fi

}


if [[ ${#@} -gt 0 ]] ; then
  remove_ns "$1"
else
  ( $KUBECTL get ns -o=name | grep namespace/test > bin/ns/list.txt ) || true
  namespaces=( $( cat bin/ns/list.txt ) )
  for ns in ${namespaces[*]} ; do
    ns="${ns#*/}" # remove "namespace/" from the beginning of the string
    echo "Deleting $ns"
    remove_ns "$ns"
  done
fi