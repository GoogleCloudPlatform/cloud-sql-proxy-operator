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

function remove_pod() {
  ns="$1"
  pod_name="$2"

  # Gracefully delete in 10 seconds, then force delete.
  if ! kubectl -n "$ns" delete pod --grace-period=10 "$pod_name" ; then
    kubectl -n "$ns" delete pod --grace-period=0 --force "$pod_name"
  fi
}

function remove_ns(){
  # Check that the namespace exists, return if not.
  if ! $KUBECTL get namespace "$1" ; then
    return
  fi

  # Tell kubernetes to delete the namespace, If it times out, force delete.
  if ! $KUBECTL delete namespace "$1" --timeout=30s ; then

    # Attempt to delete all the pods in the namespace
    for pod_name in $(kubectl -n "$1" get pods -o json | jq -r .items[].metadata.name) ; do
      remove_pod "$1" "$pod_name"
    done

    # Check if the namespace was deleted. If so, return
    if ! $KUBECTL get namespace "$1" ; then
      return
    fi

    # Get the namespace, remove finalizers from the namespace spec.
    # Force update the namespace resource, removing finalizers.
    # This will allow Kubernetes to continue the deletion of the resource.

    $KUBECTL get namespace "$1" -o json | \
      jq '.spec.finalizers = []' > "bin/ns/$1.json"

    if ! $KUBECTL replace --raw "/api/v1/namespaces/$1/finalize" -f "bin/ns/$1.json" ; then
      echo "Update finalizers failed. Will force delete"
    fi
    sleep 5

    # Check if the namespace was deleted. If so, return
    if ! $KUBECTL get namespace "$1" ; then
      return
    fi

    # Attempt to delete the namespace again
    $KUBECTL delete namespace "$1" --force || true # ignore failure
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