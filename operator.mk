# Copyright 2022 Google LLC.
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

###
# Global settings

# Image URL to use all building/pushing image targets
IMG ?= example.com/cloud-sql-proxy-operator:latest

# Import the local build environment file. This holds configuration specific
# to the local environment. build.sample.env describes the required configuration
# environment variables.
include build.env

# if build.env is missing, copy build.sample.env to build.env
build.env:
	test -f $@ || cp build.sample.env build.env


#
###

# Set the build parameter PWD if it was not already set in the shell calling make.
PWD ?= $(shell pwd)

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: install_tools
install_tools: remove_tools all_tools ## Installs all development tools

.PHONY: generate
generate:  ctrl_generate ctrl_manifests reset_image add_copyright_header go_fmt yaml_fmt ## Runs code generation, format, and validation tools

.PHONY: build
build: build_push_docker ## Builds and pushes the docker image to tag defined in envvar IMG

.PHONY: test
test: generate go_test ## Run tests (but not internal/teste2e)

.PHONY: deploy
deploy:  build_push_docker deploy_with_kubeconfig ## Deploys the operator to the kubernetes cluster using envvar KUBECONFIG. Set $IMG envvar to the image tag.

##
# Development targets

# Load active version from version.txt
VERSION=$(shell cat $(PWD)/version.txt | tr -d '\n')
BUILD_ID:=$(shell $(PWD)/tools/build-identifier.sh | tr -d '\n')
VERSION_LDFLAGS=-X main.version=$(VERSION) -X main.buildID=$(BUILD_ID)


.PHONY: ctrl_generate
ctrl_generate: controller-gen # Use controller-gen to generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: go_fmt
go_fmt: # Automatically formats go files
	go vet ./...
	go mod tidy
	go run golang.org/x/tools/cmd/goimports@latest -w .

yaml_fmt: # Automatically formats all yaml files
	go run github.com/UltiRequiem/yamlfmt@latest -w $(shell find . -iname '*.yaml' -or -iname '*.yml')

.PHONY: add_copyright_header
add_copyright_header: # Add the copyright header
	go run github.com/google/addlicense@latest *

.PHONY: build
build_push_docker: generate # Build docker image with the operator. set IMG env var before running: `IMG=example.com/img:1.0 make build`
	@test -n "$(IMG)" || ( echo "IMG environment variable must be set to the public repo where you want to push the image" ; exit 1)
	docker buildx build --platform "linux/amd64" \
	  --build-arg GO_LD_FLAGS="$(VERSION_LDFLAGS)" \
	  -f "Dockerfile-operator" \
	  --push -t "$(IMG)" "$(PWD)"
	echo "$(IMG)" > bin/last-pushed-image-url.txt
##
# Kubernetes configuration targets

.PHONY: go_test
go_test: envtest # Run tests (but not internal/teste2e)
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
		go test $(shell go list ./... | grep -v 'internal/e2e') -coverprofile cover.out

##@ Kubernetes configuration targets
.PHONY: ctrl_manifests
ctrl_manifests: controller-gen # Use controller-gen to generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: reset_image
reset_image: kustomize # Resets the image used in the kubernetes config to a default image.
	cd config/manager && $(KUSTOMIZE) edit set image controller=cloudsql-proxy-operator:latest

.PHONY: update_image
update_image: kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)

.PHONY: deploy_with_kubeconfig
deploy_with_kubeconfig: install_certmanager install_crd deploy_operator

.PHONY: install_certmanager
install_certmanager:  # Install the certmanager operator
	$(KUBECTL) apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.9.1/cert-manager.yaml
	$(KUBECTL) rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s

.PHONY: install
install_crd: ctrl_manifests kustomize # Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: deploy_operator
deploy_operator: ctrl_manifests kustomize # Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -
	$(E2E_KUBECTL) rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s


##
# Google Cloud End to End Test

## This is the default location from terraform
KUBECONFIG_GCLOUD ?= $(PWD)/bin/gcloud-kubeconfig.yaml

# kubectl command with proper environment vars set
E2E_KUBECTL_ARGS = USE_GKE_E2E_AUTH_PLUGIN=True KUBECONFIG=$(KUBECONFIG_GCLOUD)
E2E_KUBECTL = $(E2E_KUBECTL_ARGS) $(KUBECTL)

# This file contains the URL to the e2e container registry created by terraform
E2E_DOCKER_URL_FILE :=$(PWD)/bin/gcloud-docker-repo.url
E2E_DOCKER_URL=$(shell cat $(E2E_DOCKER_URL_FILE))


.PHONY: e2e_project
e2e_project: ## Check that the Google Cloud project exists
	@gcloud projects describe $(E2E_PROJECT_ID) 2>/dev/null || \
		( echo "No Google Cloud Project $(E2E_PROJECT_ID) found"; exit 1 )

e2e_cluster: e2e_project terraform ## Build infrastructure for e2e tests
	PROJECT_DIR=$(PWD) \
  		E2E_PROJECT_ID=$(E2E_PROJECT_ID) \
  		KUBECONFIG_GCLOUD=$(KUBECONFIG_GCLOUD) \
  		E2E_DOCKER_URL_FILE=$(E2E_DOCKER_URL_FILE) \
  		testinfra/run.sh apply

.PHONY: e2e_cert_manager_deploy
e2e_cert_manager_deploy: kubectl ## Deploy the certificate manager
	$(E2E_KUBECTL) apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.9.1/cert-manager.yaml
	# wait for cert manager to become available before continuing
	$(E2E_KUBECTL) rollout status deployment cert-manager -n cert-manager --timeout=90s


.PHONY: e2e_install
e2e_install: ctrl_manifests kustomize kubectl ## Install CRDs into the GKE cluster
	$(KUSTOMIZE) build config/crd | $(E2E_KUBECTL) apply -f -

.PHONY: e2e_deploy
e2e_deploy: ctrl_manifests kustomize kubectl ## Deploy controller to the GKE cluster
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(E2E_OPERATOR_URL)
	$(KUSTOMIZE) build config/default | USE_GKE_E2E_AUTH_PLUGIN=True  KUBECONFIG=$(KUBECONFIG_GCLOUD) $(KUBECTL) apply -f -
	$(E2E_KUBECTL) rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s

.PHONY: e2e_undeploy
e2e_undeploy: manifests  kustomize kubectl ## Deploy controller to the GKE cluster
	$(KUSTOMIZE) build config/default | $(E2E_KUBECTL) delete -f -

###
# Build the cloudsql-proxy v2 docker image and push it to the
# google cloud project repo.
E2E_PROXY_URL_FILE=$(PWD)/bin/last-proxy-image-url.txt
E2E_PROXY_URL=$(shell cat $(E2E_PROXY_URL_FILE) | tr -d "\n")

.PHONY: e2e_proxy_image_push
e2e_proxy_image_push: $(E2E_PROXY_URL_FILE) ## Build and push a proxy image

.PHONY: $(E2E_PROXY_URL_FILE)
$(E2E_PROXY_URL_FILE):
	PROJECT_DIR=$(PROXY_PROJECT_DIR) \
	IMAGE_NAME=proxy-v2 \
	REPO_URL=$(E2E_DOCKER_URL) \
	IMAGE_URL_OUT=$@ \
	PLATFORMS=linux/amd64 \
	DOCKER_FILE_NAME=Dockerfile \
	$(PWD)/tools/docker-build.sh

###
# Build the operator docker image and push it to the
# google cloud project repo.
E2E_OPERATOR_URL_FILE=$(PWD)/bin/last-gcloud-operator-url.txt
E2E_OPERATOR_URL=$(shell cat $(E2E_OPERATOR_URL_FILE) | tr -d "\n")

.PHONY: e2e_operator_image_push
e2e_operator_image_push: $(E2E_OPERATOR_URL_FILE) ## Build and push a operator image

.PHONY: $(E2E_OPERATOR_URL_FILE)
$(E2E_OPERATOR_URL_FILE): build
	PROJECT_DIR=$(PWD) \
	IMAGE_NAME=cloud-sql-auth-proxy-operator \
	REPO_URL=$(E2E_DOCKER_URL) \
	IMAGE_URL_OUT=$@ \
	PLATFORMS=linux/amd64 \
	DOCKER_FILE_NAME=Dockerfile \
	$(PWD)/tools/docker-build.sh



##
# Build tool dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	test -d $@ || mkdir -p $@

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
KUBECTL ?= $(LOCALBIN)/kubectl
ENVTEST ?= $(LOCALBIN)/setup-envtest
TERRAFORM ?= $(LOCALBIN)/terraform

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= latest
KUSTOMIZE_VERSION ?= latest
KUBECTL_VERSION ?= v1.24.0
TERRAFORM_VERSION ?= 1.2.7
KUSTOMIZE_VERSION ?= v4.5.2

remove_tools:
	rm -rf $(KUSTOMIZE) $(CONTROLLER_GEN) $(ENVTEST) $(KUBECTL) $(TERRAFORM)

all_tools: kustomize controller-gen envtest kubectl terraform

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) # Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) # Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
       test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: kubectl
kubectl: $(KUBECTL) ## Download kubectl
$(KUBECTL): $(LOCALBIN)
	test -s $@ || \
		( curl -L -o $@ https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/$(GOOS)/$(GOARCH)/kubectl && \
		chmod a+x $@ && \
		touch $@ )

.PHONY: terraform
terraform: $(TERRAFORM) ## Download terraform
$(TERRAFORM): $(LOCALBIN)
	test -s $@ || \
		( curl -L -o $@.zip https://releases.hashicorp.com/terraform/$(TERRAFORM_VERSION)/terraform_$(TERRAFORM_VERSION)_$(GOOS)_$(GOARCH).zip && \
		cd $(LOCALBIN) && unzip -o $@.zip && \
		rm -f $@.zip && \
		chmod a+x $@ && \
		touch $@ )
