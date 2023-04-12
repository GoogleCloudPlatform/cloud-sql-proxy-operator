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
CERT_MANAGER_VERSION=v1.11.1# renovate datasource=github-tags depName=cert-manager/cert-manager

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
	@echo "TIME: $(shell date) end install tools"

.PHONY: generate
generate:  ctrl_generate ctrl_manifests generate_crd_docs go_lint tf_lint installer reset_image add_copyright_header go_fmt yaml_fmt license_check license_save ## Runs code generation, format, and validation tools
	@echo "TIME: $(shell date) end make generate"

.PHONY: build
build: generate build_push_docker ## Builds and pushes the docker image to tag defined in envvar IMG
	@echo "TIME: $(shell date) end make build"

.PHONY: test
test: generate go_test ## Run tests (but not internal/teste2e)
	@echo "TIME: $(shell date) end make test"

.PHONY: deploy
deploy:  build deploy_with_kubeconfig ## Deploys the operator to the kubernetes cluster using envvar KUBECONFIG. Set $IMG envvar to the image tag.
	@echo "TIME: $(shell date) end make deploy"

.PHONY: e2e_test
e2e_test: e2e_setup e2e_build_deploy e2e_test_run e2e_test_clean ## Run end-to-end tests on Google Cloud GKE
	@echo "TIME: $(shell date) end make e2e_test"

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
	go run github.com/UltiRequiem/yamlfmt@latest -w $(shell find . -iname '*.yaml' -or -iname '*.yml' | grep -v -e '^./bin/' | grep -v -e '^./.github/workflows/')

.PHONY: add_copyright_header
add_copyright_header: # Add the copyright header
	go run github.com/google/addlicense@latest *

.PHONY: update_version_in_docs
update_version_in_docs:  # Fix version numbers that appear in the markdown documentation
	# Update links to the install script
	find . -name '*.md' | xargs sed -i.bak -E 's|storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy-operator/[^/]+/install.sh|storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy-operator/v$(VERSION)/install.sh|g' && \
	find . -name '*.md' | xargs sed -i.bak -E 's|storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy-operator/[^/]+/cloud-sql-proxy-operator.yaml|storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy-operator/v$(VERSION)/cloud-sql-proxy-operator.yaml|g' && \
	find . -name '*.md.bak' | xargs rm -f

.PHONY: generate_crd_docs
generate_crd_docs: crd-ref-docs # Generate the
	 $(CRD_REF_DOCS) --source-path=internal/api/ --config=tools/config-crd-ref-docs.yaml --output-path=bin --renderer=markdown && \
 		cp bin/out.md docs/api.md

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
	$(TERRAFORM) -chdir=infra/permissions fmt
	$(TERRAFORM) -chdir=infra/resources fmt

.PHONY: go_test
go_test: ctrl_manifests envtest # Run tests (but not internal/teste2e)
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
		go test ./internal/.../. -coverprofile cover.out -race

##
# 3rd Party License Checks
.PHONY: license_check
license_check: go-licenses # checks that all deps use allowed licenses
	$(GO_LICENSES) check .


.PHONY: license_save
license_save: go-licenses # Download all 3rd party license for to include in docker image
	( test -d ThirdPartyLicenses && rm -rf ThirdPartyLicenses ) || true
	$(GO_LICENSES) save --save_path ThirdPartyLicenses .


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
install_certmanager: helm # Install the cert-manager operator to manage the certificates for the operator webhooks
	helm repo add jetstack https://charts.jetstack.io
	helm repo update
	helm get all -n cert-manager cert-manager || \
		helm install \
			cert-manager jetstack/cert-manager \
			--namespace cert-manager \
			--version "$(CERT_MANAGER_VERSION)" \
			--create-namespace \
			--set global.leaderElection.namespace=cert-manager \
			--set installCRDs=true

.PHONY: install_crd
install_crd: kustomize kubectl # Install CRDs into the K8s cluster using the kubectl default behavior
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: deploy_operator
deploy_operator: kustomize kubectl # Deploy controller to the K8s cluster using the kubectl default behavior
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -
	$(E2E_KUBECTL) rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s
	$(E2E_PRIVATE_KUBECTL) rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s

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
PRIVATE_KUBECONFIG_E2E ?= $(PWD)/bin/e2e-private-kubeconfig.yaml

# This is the file where Terraform will write the kubeconfig.yaml for the
# GKE cluster.
ENVIRONMENT_NAME ?= $(shell whoami)

# kubectl command with proper environment vars set
E2E_KUBECTL_ENV = USE_GKE_E2E_AUTH_PLUGIN=True KUBECONFIG=$(KUBECONFIG_E2E)
E2E_PRIVATE_KUBECTL_ENV = USE_GKE_E2E_AUTH_PLUGIN=True KUBECONFIG=$(PRIVATE_KUBECONFIG_E2E)
E2E_KUBECTL = $(E2E_KUBECTL_ENV) $(KUBECTL)
E2E_PRIVATE_KUBECTL = $(E2E_PRIVATE_KUBECTL_ENV) $(KUBECTL)

# This is the file where Terraform will write the URL to the e2e container registry
E2E_DOCKER_URL_FILE :=$(PWD)/bin/gcloud-docker-repo.url
E2E_DOCKER_URL=$(shell cat $(E2E_DOCKER_URL_FILE) | tr -d '\n')

# Default value in the Makefile blank. When blank tests will use workload.DefaultProxyImage
E2E_PROXY_URL ?= ""

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

.PHONY: e2e_test_job
e2e_test_job: e2e_setup_job e2e_build_deploy e2e_test_run

.PHONY: e2e_setup_job
e2e_setup_job: e2e_project e2e_cluster_job e2e_cert_manager_deploy

.PHONY: e2e_project
e2e_project: gcloud # Check that the Google Cloud project exists
	@gcloud projects describe $(E2E_PROJECT_ID) 2>/dev/null || \
		( echo "No Google Cloud Project $(E2E_PROJECT_ID) found"; exit 1 )

.PHONY: e2e_cluster_job
e2e_cluster_job: e2e_project terraform # Build infrastructure for e2e tests in the test job
	PROJECT_DIR=$(PWD) \
  		E2E_PROJECT_ID=$(E2E_PROJECT_ID) \
  		KUBECONFIG_E2E=$(KUBECONFIG_E2E) \
  		PRIVATE_KUBECONFIG_E2E=$(PRIVATE_KUBECONFIG_E2E) \
  		E2E_DOCKER_URL_FILE=$(E2E_DOCKER_URL_FILE) \
  		ENVIRONMENT_NAME=$(ENVIRONMENT_NAME) \
  		NODEPOOL_SERVICEACCOUNT_EMAIL=$(NODEPOOL_SERVICEACCOUNT_EMAIL) \
  		WORKLOAD_ID_SERVICEACCOUNT_EMAIL=$(WORKLOAD_ID_SERVICEACCOUNT_EMAIL) \
  		TFSTATE_STORAGE_BUCKET=$(TFSTATE_STORAGE_BUCKET) \
  		TESTINFRA_JSON_FILE=$(LOCALBIN)/testinfra.json \
  		infra/run.sh apply_e2e_job

.PHONY: e2e_cluster
e2e_cluster: e2e_project terraform # Build infrastructure for e2e tests
	PROJECT_DIR=$(PWD) \
  		E2E_PROJECT_ID=$(E2E_PROJECT_ID) \
  		KUBECONFIG_E2E=$(KUBECONFIG_E2E) \
  		PRIVATE_KUBECONFIG_E2E=$(PRIVATE_KUBECONFIG_E2E) \
  		E2E_DOCKER_URL_FILE=$(E2E_DOCKER_URL_FILE) \
  		ENVIRONMENT_NAME=$(ENVIRONMENT_NAME) \
  		TESTINFRA_JSON_FILE=$(LOCALBIN)/testinfra.json \
  		infra/run.sh apply

.PHONY: e2e_cluster_destroy
e2e_cluster_destroy: e2e_project terraform # Destroy the infrastructure for e2e tests
	PROJECT_DIR=$(PWD) \
  		E2E_PROJECT_ID=$(E2E_PROJECT_ID) \
  		KUBECONFIG_E2E=$(KUBECONFIG_E2E) \
  		PRIVATE_KUBECONFIG_E2E=$(PRIVATE_KUBECONFIG_E2E) \
  		E2E_DOCKER_URL_FILE=$(E2E_DOCKER_URL_FILE) \
  		ENVIRONMENT_NAME=$(ENVIRONMENT_NAME) \
  		TESTINFRA_JSON_FILE=$(LOCALBIN)/testinfra.json \
  		infra/run.sh destroy

.PHONY: e2e_cert_manager_deploy
e2e_cert_manager_deploy: e2e_project helm # Deploy the certificate manager
	KUBECONFIG=$(KUBECONFIG_E2E) CERT_MANAGER_VERSION=$(CERT_MANAGER_VERSION) tools/helm-install-certmanager.sh
	KUBECONFIG=$(PRIVATE_KUBECONFIG_E2E) CERT_MANAGER_VERSION=$(CERT_MANAGER_VERSION) tools/helm-install-certmanager.sh

.PHONY: e2e_install_crd
e2e_install_crd: generate e2e_project kustomize kubectl $(E2E_WORK_DIR) # Install CRDs into the GKE cluster
	$(KUSTOMIZE) build config/crd > $(E2E_WORK_DIR)/crd.yaml
	$(E2E_KUBECTL) apply -f $(E2E_WORK_DIR)/crd.yaml
	$(E2E_PRIVATE_KUBECTL) apply -f $(E2E_WORK_DIR)/crd.yaml

.PHONY: e2e_deploy
e2e_deploy: e2e_project kustomize kubectl $(E2E_WORK_DIR) # Deploy the operator to the GKE cluster
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(E2E_OPERATOR_URL)
	$(KUSTOMIZE) build config/default > $(E2E_WORK_DIR)/operator.yaml
	$(E2E_KUBECTL) apply -f $(E2E_WORK_DIR)/operator.yaml
	$(E2E_PRIVATE_KUBECTL) apply -f $(E2E_WORK_DIR)/operator.yaml
	$(E2E_KUBECTL) rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s
	$(E2E_PRIVATE_KUBECTL) rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s


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
	( $(E2E_PRIVATE_KUBECTL) get ns -o=name | \
		grep namespace/test | \
		$(E2E_PRIVATE_KUBECTL_ENV) xargs $(KUBECTL) delete ) || true

.PHONY: e2e_undeploy
e2e_undeploy: e2e_project kustomize kubectl $(E2E_WORK_DIR) # Remove the operator from the GKE cluster
	$(E2E_KUBECTL) delete -f $(E2E_WORK_DIR)/operator.yaml
	$(E2E_PRIVATE_KUBECTL) delete -f $(E2E_WORK_DIR)/operator.yaml

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

###
# Build a version of the cloud-sql-proxy from local sources
E2E_LOCAL_PROXY_PROJECT_DIR?=/not-set
E2E_LOCAL_PROXY_BUILD_URL_FILE=$(PWD)/bin/last-local-proxy-url.txt
E2E_LOCAL_PROXY_BUILD_URL=$(shell cat $(E2E_LOCAL_PROXY_BUILD_URL_FILE) | tr -d "\n")

.PHONY: e2e_image_push
e2e_local_proxy_image_push: # Build and push the proxy image from a local working directory to the e2e artifact repo
	test -d $(E2E_LOCAL_PROXY_PROJECT_DIR) && \
	PROJECT_DIR=$(E2E_LOCAL_PROXY_PROJECT_DIR) \
	IMAGE_NAME=cloud-sql-proxy-dev \
	REPO_URL=$(E2E_DOCKER_URL) \
	IMAGE_URL_OUT=$(E2E_LOCAL_PROXY_BUILD_URL_FILE) \
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
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
GO_LICENSES ?= $(LOCALBIN)/go-licenses
CRD_REF_DOCS ?= $(LOCALBIN)/crd-ref-docs

## Tool Versions
# Important note: avoid adding spaces in the macro declarations as any
# additional whitespace will break the renovate regex rules.

KUBECTL_VERSION=v1.26.3# renovate datasource=github-tags depName=kubernetes/kubernetes
TERRAFORM_VERSION=v1.4.4# renovate datasource=github-tags depName=hashicorp/terraform

CONTROLLER_TOOLS_VERSION=v0.11.3# renovate datasource=go depName=sigs.k8s.io/controller-tools
CRD_REF_DOCS_VERSION=v0.0.8# renovate datasource=go depName=github.com/elastic/crd-ref-docs
ENVTEST_VERSION=v0.0.0-20230301194117-e2d8821b277f# renovate datasource=go depName=sigs.k8s.io/controller-runtime/tools/setup-envtest
GOLANGCI_LINT_VERSION=v1.51.2# renovate datasource=go depName=github.com/golangci/golangci-lint/cmd/golangci-lint
GO_LICENSES_VERSION=v1.6.0# renovate datasource=go depName=github.com/google/go-licenses

KUSTOMIZE_VERSION=v4.5.2# don't manage with renovate, this repo has non-standard tags

GOOS?=$(shell go env GOOS | tr -d '\n')
GOARCH?=$(shell go env GOARCH | tr -d '\n')

remove_tools:
	rm -rf $(KUSTOMIZE) $(CONTROLLER_GEN) $(KUBECTL) $(ENVTEST) $(TERRAFORM) $(GOLANGCI_LINT) $(CRD_REF_DOCS)

all_tools: kustomize controller-gen envtest kubectl terraform golangci-lint crd-ref-docs

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

.PHONY: crd-ref-docs
crd-ref-docs: $(CRD_REF_DOCS) # Download crd-ref-docs locally if necessary.
$(CRD_REF_DOCS): $(LOCALBIN)
	test -s $(LOCALBIN)/crd-ref-docs || GOBIN=$(LOCALBIN) go install github.com/elastic/crd-ref-docs@$(CRD_REF_DOCS_VERSION)

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
		( curl -L -o $@.zip https://releases.hashicorp.com/terraform/$(subst v,,$(TERRAFORM_VERSION))/terraform_$(subst v,,$(TERRAFORM_VERSION))_$(GOOS)_$(GOARCH).zip && \
		cd $(LOCALBIN) && unzip -o $@.zip && \
		rm -f $@.zip && \
		chmod a+x $@ && \
		touch $@ )

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download controller-gen locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	test -s $@ || GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: go-licenses
go-licenses: $(GO_LICENSES) ## Download controller-gen locally if necessary.
$(GO_LICENSES): $(LOCALBIN)
	test -s $@ || GOBIN=$(LOCALBIN) go install github.com/google/go-licenses@$(GO_LICENSES_VERSION)

##
# Tools that need to be installed on the development machine

.PHONY: gcloud
gcloud:
	@which gcloud > /dev/null || \
		(echo "Google Cloud API command line tools are not available in your path" ;\
		 echo "Instructions on how to install https://cloud.google.com/sdk/docs/install " ; \
		 exit 1)

.PHONY: helm
helm:
	@which helm > /dev/null || \
		(echo "Helm command line tools are not available in your path" ; \
		 echo "Instructions on how to install https://helm.sh/docs/helm/helm_install/ " ; \
		 exit 1)

