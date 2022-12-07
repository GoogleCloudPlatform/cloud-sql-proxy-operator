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

## RELEASE_TAG is the public image tag for the operator
RELEASE_TAG_PATH=cloud-sql-connectors/cloud-sql-operator/cloud-sql-proxy-operator:$(VERSION)
RELEASE_TAG=gcr.io/$(RELEASE_TAG_PATH)

# When the environment variable IS_RELEASE_BUILD is set, the IMG will be set
# to the RELEASE_TAG, overriding the IMG environment variable. This is intended
# to be used only in a release job to publish artifacts.
ifdef IS_RELEASE_BUILD
IMG=$(RELEASE_TAG)
endif


# IMG is used by build to determine where to push the docker image for the
# operator. You must set the IMG environment variable when you run make build
# or other dependent targets.
IMG ?=

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

## The version to use for the cert-manager operator
CERT_MANAGER_VERSION=v1.9.1

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

# The `help` target prints out special makefile code comments to make it easier
# for developers to use this file.
# Help section headings have ##@ at the beginning of the line
# Target help message is on the same line as the target declaration and begins with ##
# Only add help messages to permanent makefile targets that we want to maintain forever.
# Intermediate or temporary build targets should only be documented with code
# comments, and should not print a help message.

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: install_tools
install_tools: remove_tools all_tools ## Installs all development tools

.PHONY: generate
generate:  ctrl_generate ctrl_manifests go_lint tf_lint installer reset_image add_copyright_header update_version_in_docs go_fmt yaml_fmt ## Runs code generation, format, and validation tools

.PHONY: build
build: generate build_push_docker ## Builds and pushes the docker image to tag defined in envvar IMG

.PHONY: test
test: generate go_test ## Run tests (but not internal/teste2e)

.PHONY: deploy
deploy:  build deploy_with_kubeconfig ## Deploys the operator to the kubernetes cluster using envvar KUBECONFIG. Set $IMG envvar to the image tag.

.PHONY: e2e_test
e2e_test: e2e_setup e2e_build_deploy e2e_test_run e2e_test_clean ## Run end-to-end tests on Google Cloud GKE

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

.PHONY: update_version_in_docs
update_version_in_docs:  # Fix version numbers that appear in the markdown documentation
	# Update links to the install script
	find . -name '*.md' | xargs sed -i.bak -E 's|storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy-operator/[^/]+/install.sh|storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy-operator/v$(VERSION)/install.sh|g' && \
	find . -name '*.md.bak' | xargs rm -f

.PHONY: build_push_docker
build_push_docker: # Build docker image with the operator. set IMG env var before running: `IMG=example.com/img:1.0 make build`
	@test -n "$(IMG)" || ( echo "IMG environment variable must be set to the public repo where you want to push the image" ; exit 1)
	docker buildx build --platform "linux/amd64" \
	  --build-arg GO_LD_FLAGS="$(VERSION_LDFLAGS)" \
	  -f "Dockerfile-operator" \
	  --push -t "$(IMG)" "$(PWD)"
	test -d 'bin' || mkdir -p bin
	echo "$(IMG)" > bin/last-pushed-image-url.txt

.PHONY: go_lint
go_lint: golangci-lint # Run go lint tools, fail if unchecked errors
	# Implements golang CI based on settings described here:
	# See https://betterprogramming.pub/how-to-improve-code-quality-with-an-automatic-check-in-go-d18a5eb85f09
	$(GOLANGCI_LINT) 	run --fix --fast ./...

.PHONY: tf_lint
tf_lint: terraform # Run terraform fmt to ensure terraform code is consistent
	$(TERRAFORM) -chdir=testinfra fmt

.PHONY: go_test
go_test: ctrl_manifests envtest # Run tests (but not internal/teste2e)
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
		go test ./internal/.../. -coverprofile cover.out -race

##
# Kubernetes configuration targets
SOURCE_CODE_IMAGE=cloud-sql-proxy-operator:latest

.PHONY: ctrl_manifests
ctrl_manifests: controller-gen # Use controller-gen to generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: reset_image
reset_image: kustomize # Reset the image used in the kubernetes config to a default image.
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(SOURCE_CODE_IMAGE)

.PHONY: update_image
update_image: kustomize # Update the image used in the kubernetes config to $(IMG)
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)

.PHONY: deploy_with_kubeconfig # Deploy the kubernetes configuration to the current default cluster using kubectl
deploy_with_kubeconfig: install_certmanager install_crd deploy_operator

.PHONY: install_certmanager
install_certmanager: kubectl # Install the cert-manager operator to manage the certificates for the operator webhooks
	$(KUBECTL) apply -f "https://github.com/cert-manager/cert-manager/releases/download/$(CERT_MANAGER_VERSION)/cert-manager.yaml"
	$(KUBECTL) rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s

.PHONY: install_crd
install_crd: kustomize kubectl # Install CRDs into the K8s cluster using the kubectl default behavior
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: deploy_operator
deploy_operator: kustomize kubectl # Deploy controller to the K8s cluster using the kubectl default behavior
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -
	$(E2E_KUBECTL) rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s

##
# Update installer
.PHONY: installer
installer: installer/cloud-sql-proxy-operator.yaml installer/install.sh

.PHONY: installer/cloud-sql-proxy-operator.yaml
installer/cloud-sql-proxy-operator.yaml: kustomize # Build the single yaml file for deploying the operator
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(RELEASE_TAG)
	$(KUSTOMIZE) build config/default > $@
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(SOURCE_CODE_IMAGE)

.PHONY: installer/install.sh
installer/install.sh: ## Build install shell script to deploy the operator
	cat tools/install.sh | \
	sed 's/__VERSION__/v$(VERSION)/g' | \
	sed 's/__CERT_MANAGER_VERSION__/$(CERT_MANAGER_VERSION)/g' > $@


##
##@ Google Cloud End to End Test

# This is the file where Terraform will write the kubeconfig.yaml for the
# GKE cluster.
KUBECONFIG_E2E ?= $(PWD)/bin/e2e-kubeconfig.yaml

# kubectl command with proper environment vars set
E2E_KUBECTL_ENV = USE_GKE_E2E_AUTH_PLUGIN=True KUBECONFIG=$(KUBECONFIG_E2E)
E2E_KUBECTL = $(E2E_KUBECTL_ENV) $(KUBECTL)

# This is the file where Terraform will write the URL to the e2e container registry
E2E_DOCKER_URL_FILE :=$(PWD)/bin/gcloud-docker-repo.url
E2E_DOCKER_URL=$(shell cat $(E2E_DOCKER_URL_FILE) | tr -d '\n')
E2E_PROXY_URL ?= "gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.0.0-preview.2"

E2E_WORK_DIR=$(PWD)/bin/e2e
$(E2E_WORK_DIR):
	mkdir -p "$(E2E_WORK_DIR)"

## Intermediate targets for developers to use when running e2e tests
.PHONY: e2e_setup
e2e_setup: e2e_project e2e_cluster e2e_cert_manager_deploy  ## Provision and reconcile test infrastructure for e2e tests

.PHONY: e2e_build_deploy
e2e_build_deploy: e2e_install_crd e2e_image_push e2e_deploy ## Build and deploy the operator to e2e cluster

.PHONY: e2e_test_run
e2e_test_run: e2e_cleanup_test_namespaces e2e_test_run_gotest  ## Run the golang e2e tests

.PHONY: e2e_test_clean
e2e_test_clean: e2e_cleanup_test_namespaces e2e_undeploy ## Remove all operator and testcase configs from the e2e k8s cluster

.PHONY: e2e_teardown
e2e_teardown: e2e_cluster_destroy ## Remove the test infrastructure for e2e tests from the Google Cloud Project

.PHONY: e2e_project
e2e_project: gcloud # Check that the Google Cloud project exists
	@gcloud projects describe $(E2E_PROJECT_ID) 2>/dev/null || \
		( echo "No Google Cloud Project $(E2E_PROJECT_ID) found"; exit 1 )

.PHONY: e2e_cluster
e2e_cluster: e2e_project terraform # Build infrastructure for e2e tests
	PROJECT_DIR=$(PWD) \
  		E2E_PROJECT_ID=$(E2E_PROJECT_ID) \
  		KUBECONFIG_E2E=$(KUBECONFIG_E2E) \
  		E2E_DOCKER_URL_FILE=$(E2E_DOCKER_URL_FILE) \
  		testinfra/run.sh apply

.PHONY: e2e_cluster_destroy
e2e_cluster_destroy: e2e_project terraform # Destroy the infrastructure for e2e tests
	PROJECT_DIR=$(PWD) \
  		E2E_PROJECT_ID=$(E2E_PROJECT_ID) \
  		KUBECONFIG_E2E=$(KUBECONFIG_E2E) \
  		E2E_DOCKER_URL_FILE=$(E2E_DOCKER_URL_FILE) \
  		testinfra/run.sh destroy

.PHONY: e2e_cert_manager_deploy
e2e_cert_manager_deploy: e2e_project kubectl # Deploy the certificate manager
	$(E2E_KUBECTL) apply -f https://github.com/cert-manager/cert-manager/releases/download/$(CERT_MANAGER_VERSION)/cert-manager.yaml
	# wait for cert manager to become available before continuing
	$(E2E_KUBECTL) rollout status deployment cert-manager -n cert-manager --timeout=90s


.PHONY: e2e_install_crd
e2e_install_crd: generate e2e_project kustomize kubectl $(E2E_WORK_DIR) # Install CRDs into the GKE cluster
	$(KUSTOMIZE) build config/crd > $(E2E_WORK_DIR)/crd.yaml
	$(E2E_KUBECTL) apply -f $(E2E_WORK_DIR)/crd.yaml



.PHONY: e2e_deploy
e2e_deploy: e2e_project kustomize kubectl $(E2E_WORK_DIR) # Deploy the operator to the GKE cluster
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(E2E_OPERATOR_URL)
	$(KUSTOMIZE) build config/default > $(E2E_WORK_DIR)/operator.yaml
	$(E2E_KUBECTL) apply -f $(E2E_WORK_DIR)/operator.yaml
	$(E2E_KUBECTL) rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s


# Note: `go test --count=1` is used to make sure go actually runs the tests every
# time. By default go will skip the tests when go source files are unchanged.
.PHONY: e2e_test_run_gotest
e2e_test_run_gotest:  # Run the golang e2e tests
	USE_GKE_E2E_AUTH_PLUGIN=True \
		TEST_INFRA_JSON=$(LOCALBIN)/testinfra.json \
		PROXY_IMAGE_URL=$(E2E_PROXY_URL) \
		OPERATOR_IMAGE_URL=$(E2E_OPERATOR_URL) \
		go test --count=1 -v -race ./tests/...

.PHONY: e2e_cleanup_test_namespaces
e2e_cleanup_test_namespaces: e2e_project kustomize kubectl # remove e2e test namespaces named "test*"
	( $(E2E_KUBECTL) get ns -o=name | \
		grep namespace/test | \
		$(E2E_KUBECTL_ENV) xargs $(KUBECTL) delete ) || true

.PHONY: e2e_undeploy
e2e_undeploy: e2e_project kustomize kubectl $(E2E_WORK_DIR) # Remove the operator from the GKE cluster
	$(E2E_KUBECTL) delete -f $(E2E_WORK_DIR)/operator.yaml

###
# Build the operator docker image and push it to the
# google cloud project repo.
E2E_OPERATOR_URL_FILE=$(PWD)/bin/last-gcloud-operator-url.txt
E2E_OPERATOR_URL=$(shell cat $(E2E_OPERATOR_URL_FILE) | tr -d "\n")

.PHONY: e2e_image_push
e2e_image_push: generate # Build and push a operator image to the e2e artifact repo
	PROJECT_DIR=$(PWD) \
	IMAGE_NAME=cloud-sql-auth-proxy-operator \
	REPO_URL=$(E2E_DOCKER_URL) \
	IMAGE_URL_OUT=$(E2E_OPERATOR_URL_FILE) \
	PLATFORMS=linux/amd64 \
	DOCKER_FILE_NAME=Dockerfile-operator \
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
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= latest
KUBECTL_VERSION ?= $(shell curl -L -s https://dl.k8s.io/release/stable.txt | tr -d '\n')
TERRAFORM_VERSION ?= 1.2.7
KUSTOMIZE_VERSION ?= v4.5.2
ENVTEST_VERSION ?= latest
GOLANGCI_LINT_VERSION ?= latest

GOOS=$(shell go env GOOS | tr -d '\n')
GOARCH=$(shell go env GOARCH | tr -d '\n')

remove_tools:
	rm -rf $(LOCALBIN)/*

all_tools: kustomize controller-gen envtest kubectl terraform golangci-lint

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
envtest: $(ENVTEST) # Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_VERSION)

.PHONY: kubectl
kubectl: $(KUBECTL) # Download kubectl
$(KUBECTL): $(LOCALBIN)
	test -s $@ || \
		( curl -L -o $@ https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/$(GOOS)/$(GOARCH)/kubectl && \
		chmod a+x $@ && \
		touch $@ )

.PHONY: terraform
terraform: $(TERRAFORM) # Download terraform
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
	test -s $@ || GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

##
# Tools that need to be installed on the development machine

.PHONY: gcloud
gcloud:
	@which gcloud > /dev/null || \
		(echo "Google Cloud API command line tools are not available in your path" ;\
		 echo "Instructions on how to install https://cloud.google.com/sdk/docs/install " ; \
		 exit 1)

