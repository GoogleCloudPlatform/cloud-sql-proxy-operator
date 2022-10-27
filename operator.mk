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
deploy:  build_push_docker deploy_with_kubeconfig ## Deploys the operator to the kubernetes cluster using envvar KUBECONFIG. Set IMG envvar to the image tag.

.PHONY: e2e_test
e2e_test: e2e_test_infra e2e_test_run e2e_test_cleanup ## Run end-to-end tests on Google Cloud GKE

.PHONY: lint
lint: generate go_lint tf_lint  ## Runs generate and then code lint validation tools

.PHONY: release
release: release_container_tools release_image_push release_k8s_publish ## Release all artifacts for the current version

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

.PHONY: go_lint
go_lint: golangci-lint ## Run go lint tools, fail if unchecked errors
	# Implements golang CI based on settings described here:
	# See https://betterprogramming.pub/how-to-improve-code-quality-with-an-automatic-check-in-go-d18a5eb85f09
	$(GOLANGCI_LINT) run --fix --fast ./...

.PHONY: tf_lint
tf_lint: terraform ## Run go lint tools, fail if unchecked errors
	$(TERRAFORM) -chdir=testinfra fmt

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


.PHONY: e2e_test_infra
e2e_test_infra: e2e_project e2e_cluster e2e_cert_manager_deploy e2e_proxy_image_push  ## Build test infrastructure for e2e tests

.PHONY: e2e_test_run
e2e_test_run: e2e_install e2e_operator_image_push e2e_deploy e2e_test_run_gotest ## Build and run the e2e test code

.PHONY: e2e_test_cleanup
e2e_test_cleanup: manifests e2e_cleanup_test_namespaces e2e_undeploy ## Remove all operator and testcase configs from the e2e k8s cluster


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


.PHONY: e2e_cleanup_test_namespaces
e2e_cleanup_test_namespaces: $(KUSTOMIZE) $(KUBECTL) 	## list all namespaces, delete those named "test*"
	$(E2E_KUBECTL) get ns -o=name | \
		grep namespace/test | \
		$(E2E_KUBECTL_ENV) xargs $(KUBECTL) delete

.PHONY: e2e_test_run_gotest
e2e_test_run_gotest: ## Run the golang tests
	USE_GKE_E2E_AUTH_PLUGIN=True \
		TEST_INFRA_JSON=$(LOCALBIN)/testinfra.json \
		PROXY_IMAGE_URL=$(E2E_PROXY_URL) \
		OPERATOR_IMAGE_URL=$(E2E_OPERATOR_URL) \
		go test --count=1 -v github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/teste2e

.PHONY: e2e_k9s
e2e_k9s: ## Connect to the gcloud test cluster using the k9s tool
	USE_GKE_E2E_AUTH_PLUGIN=True KUBECONFIG=$(KUBECONFIG_GCLOUD) k9s

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
# Release Process

RELEASE_REPO_PATH=cloud-sql-connectors/cloud-sql-operator-dev
RELEASE_IMAGE_URL_PATH=$(PWD)/bin/release-image-url.txt
RELEASE_IMAGE_NAME=cloud-sql-proxy-operator

RELEASE_IMAGE_BUILD_ID_URL=gcr.io/$(RELEASE_REPO_PATH)/$(RELEASE_IMAGE_NAME):$(OPERATOR_BUILD_ID)
RELEASE_IMAGE_VERSION_URL=gcr.io/$(RELEASE_REPO_PATH)/$(RELEASE_IMAGE_NAME):$(VERISON)

RELEASE_TAG_PATH=$(RELEASE_REPO_PATH)/$(RELEASE_IMAGE_NAME):$(VERSION)
RELEASE_IMAGE_TAGS=gcr.io/$(RELEASE_TAG_PATH), \
				   us.gcr.io/$(RELEASE_TAG_PATH), \
				   eu.gcr.io/$(RELEASE_TAG_PATH), \
				   asia.gcr.io/$(RELEASE_TAG_PATH)


.PHONY: release_image_push
release_image_push: generate # Build and push a operator image to the release registry
	PROJECT_DIR=$(PWD) \
	IMAGE_NAME=$(RELEASE_IMAGE_NAME) \
	IMAGE_VERSION=$(OPERATOR_BUILD_ID) \
	REPO_URL=gcr.io/$(RELEASE_REPO_PATH) \
	IMAGE_URL_OUT=$(RELEASE_IMAGE_URL_PATH) \
	EXTRA_TAGS="$(RELEASE_IMAGE_TAGS)" \
	PLATFORMS=linux/amd64 \
	DOCKER_FILE_NAME=Dockerfile-operator $(PWD)/tools/docker-build.sh

.PHONY: release_k8s_publish
release_k8s_publish: bin/cloud-sql-proxy-operator.yaml bin/install.sh # Publish install scripts to the release storage bucket
	gcloud storage cp bin/install.sh \
		gs://cloud-sql-connectors/cloud-sql-proxy-operator/$(VERSION)/install.sh
	gcloud storage cp bin/cloud-sql-proxy-operator.yaml \
		gs://cloud-sql-connectors/cloud-sql-proxy-operator/$(VERSION)/cloud-sql-proxy-operator.yaml

.PHONY: bin/cloud-sql-proxy-operator.yaml
bin/cloud-sql-proxy-operator.yaml: kustomize # Build the single yaml file for deploying the operator
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(RELEASE_IMAGE_VERSION_URL)
	mkdir -p bin/k8s/
	$(KUSTOMIZE) build config/crd > $@
	echo "" >> $@
	echo "---" >> $@
	echo "" >> $@
	$(KUSTOMIZE) build config/default >> $@

.PHONY: bin/install.sh
bin/install.sh: # Build install shell script to deploy the operator
	sed 's/__VERSION__/$(VERSION)/g' < tools/install.sh > bin/install.sh.tmp
	sed 's|__IMAGE_URL__|$(RELEASE_IMAGE_VERSION_URL)|g' < bin/install.sh.tmp > bin/install.sh

.PHONY: release_container_tools
release_container_tools: # Copy cached release tools from the /tools/bin directory
	@echo "Release Tools:"
	@echo " Version: $(VERSION)"
	@echo " Release Image: $(RELEASE_IMAGE_VERSION_URL)"
	mkdir -p $(PWD)/bin
	test -d /tools/bin && cp -r /tools/bin/* $(PWD)/bin


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
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

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

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download controller-gen locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	test -s $@ || GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
