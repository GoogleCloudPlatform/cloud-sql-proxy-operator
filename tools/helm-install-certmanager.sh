#!/usr/bin/env bash

# Configure script to fail on any command error
set -euxo pipefail

# Find project directory, cd to project directory
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
PROJECT_DIR=$( dirname "$SCRIPT_DIR")
cd "$PROJECT_DIR"

# Validate input environment variables
#expects KUBECONFIG to be set by the caller
if [[ -z "${KUBECONFIG:-}" ]]; then
  echo "expects KUBECONFIG to be the path to the kubeconfig file for kubectl."
  exit 1
fi

#expects CERT_MANAGER_VERSION to be set by the caller
if [[ -z "${CERT_MANAGER_VERSION:-}" ]]; then
  echo "expects CERT_MANAGER_VERSION to be set the version of cert manager to install."
  exit 1
fi

helm repo add jetstack https://charts.jetstack.io --kubeconfig "${KUBECONFIG}"
helm repo update --kubeconfig "${KUBECONFIG}"

if helm get all -n cert-manager cert-manager --kubeconfig "${KUBECONFIG}" > /dev/null ; then
  action="upgrade"
else
  action="install"
fi

helm --kubeconfig "${KUBECONFIG}" "$action" \
  cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --version "${CERT_MANAGER_VERSION}" \
  --create-namespace \
  --set global.leaderElection.namespace=cert-manager \
  --set installCRDs=true
