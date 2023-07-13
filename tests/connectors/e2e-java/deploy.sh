#!/bin/bash
set -euxo

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
cd "$SCRIPT_DIR"
BIN_DIR="$SCRIPT_DIR/../../../bin"

if [[ "$1" == "deploy" ]] ; then
  mvn package
  export KUBECONFIG="$BIN_DIR/e2e-kubeconfig.yaml"
  for file in k8s/*-pub.yaml ; do
    kubectl apply -f "$file" || echo "failed to deploy $file"
  done

  export KUBECONFIG="$BIN_DIR/e2e-private-kubeconfig.yaml"
  for file in k8s/*-priv.yaml ; do
    kubectl apply -f "$file" || echo "failed to deploy $file"
  done

elif [[ $1 == "delete" ]] ; then
  export KUBECONFIG="$BIN_DIR/e2e-kubeconfig.yaml"
  for file in k8s/*-pub.yaml ; do
    kubectl delete -f "$file" || echo "failed to delete $file"
  done

  export KUBECONFIG="$BIN_DIR/e2e-private-kubeconfig.yaml"
  for file in k8s/*-priv.yaml ; do
    kubectl delete -f "$file" || echo "failed to delete $file"
  done
else
  echo "usage: deploy.sh [deploy|delete]"
  echo "  deploys all java e2e tests"
fi