# Import the local build environment file. This holds configuration specific
# to the local environment. build.sample.env describes the required configuration
# environment variables.
include build.env

# if build.env is missing, copy build.sample.env to build.env
build.env:
	test -f $@ || cp build.sample.env build.env


# Should be set by build.env
PROXY_PROJECT_DIR ?= $(PWD)/tmp/cloud-sql-proxy
GCLOUD_PROJECT_ID ?= error-no-project-id-set

# Enable CRD Generation
CRD_OPTIONS ?= "crd"


# The local dev architecture
GOOS:=$(shell go env GOOS)
GOARCH:=$(shell go env GOARCH)


# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.24.2

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ Github Workflow Targets
.PHONY: github_lint
github_lint: pre_commit ## run the the github workflow lint check locally

##@ Local Development Targets

.PHONY: pre_commit ## run checks to make sure boilerplate workflows will pass
pre_commit: git_workdir_clean  ## Run all the formatting and checks before committing
	make lint
	@git diff --exit-code --stat || (echo ; echo ; echo "ERROR: Lint tools caused changes to the working dir. "; echo "       Please review the changes before you commit."; echo ; exit 1)
	@echo "Pre commit checks OK"

.PHONY: lint
lint: $(KUSTOMIZE) ## runs code format and validation tools
	cd config/manager && $(KUSTOMIZE) edit set image controller=cloudsql-proxy-operator:latest
	make build manifests
	make add_copyright_header
	make go_fmt yaml_fmt
	make go_lint
	make tf_lint

.PHONY: git_workdir_clean
git_workdir_clean: # Checks if the git working directory is clean. Fails if there are unstaged changes.
	@git diff --exit-code --stat || (echo ; echo; echo "ERROR: git working directory has unstaged changes. "; echo "       Add or stash all changes before you commit."; echo ; exit 1)


.PHONY: add_pre_commit_hook ## run checks to make sure boilerplate workflows will pass
add_pre_commit_hook: ## Add the pre_commit hook to the local repo
	mkdir -p $(shell  git rev-parse --git-path hooks)
	echo "#!/bin/bash" > $(shell  git rev-parse --git-path hooks)/pre-commit
	echo "cd `git rev-parse --show-toplevel` && make pre_commit" >> $(shell  git rev-parse --git-path hooks)/pre-commit
	chmod a+x $(shell  git rev-parse --git-path hooks)/pre-commit

.PHONY: go_fmt
go_fmt: ## Automatically formats go files
	go mod tidy
	go run golang.org/x/tools/cmd/goimports@latest -w .

yaml_fmt: ## Automatically formats all yaml files
	go run github.com/UltiRequiem/yamlfmt@latest -w $(shell find . -iname '*.yaml' -or -iname '*.yml')

YAML_FILES_MISSING_HEADER = $(shell find . -name '*.yaml' -or -iname '*.yml' | \
		xargs egrep -L 'Copyright .... Google LLC' | \
		git check-ignore --stdin --non-matching --verbose | \
		egrep '^::' | cut -c 4-)
GO_FILES_MISSING_HEADER := $(shell find . -iname '*.go' | \
		xargs egrep -L 'Copyright .... Google LLC' | \
		git check-ignore --stdin --non-matching --verbose | \
		egrep '^::' | cut -c 4-)

.PHONY: add_copyright_header ## Adds the copyright header to any go or yaml file that is missing the header
add_copyright_header: $(GO_FILES_MISSING_HEADER) $(YAML_FILES_MISSING_HEADER) ## Add the copyright header

.PHONY: $(YAML_FILES_MISSING_HEADER)
$(YAML_FILES_MISSING_HEADER):
	cat hack/boilerplate.yaml.txt $@ > $@.tmp && mv $@.tmp $@

.PHONY: $(GO_FILES_MISSING_HEADER)
$(GO_FILES_MISSING_HEADER):
	cat hack/boilerplate.go.txt $@ > $@.tmp && mv $@.tmp $@
	go fmt $@

.PHONY: go_lint
go_lint: golangci-lint ## Run go lint tools, fail if unchecked errors
	# Implements golang CI based on settings described here:
	# See https://betterprogramming.pub/how-to-improve-code-quality-with-an-automatic-check-in-go-d18a5eb85f09
	$(GOLANGCI_LINT) run --fix --fast ./...
.PHONY: tf_lint
tf_lint: $(TERRAFORM) ## Run go lint tools, fail if unchecked errors
	# Implements golang CI based on settings described here:
	# See https://betterprogramming.pub/how-to-improve-code-quality-with-an-automatic-check-in-go-d18a5eb85f09
	$(TERRAFORM) -chdir=test/e2e/tf fmt

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

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

test: manifests generate fmt vet envtest ## Run tests (but not test/e2e)
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
		go test $(shell go list ./... | grep -v 'test/e2e') -coverprofile cover.out

##@ Build
# Load active version from version.txt
VERSION=$(shell cat $(PWD)/version.txt | tr -d '\n')
BUILD_ID:=$(shell $(PWD)/tools/build-identifier.sh | tr -d '\n')
GO_BUILD_FLAGS = -ldflags "-X main.version=$(VERSION) -X main.buildID=$(BUILD_ID)"

.PHONY: build
build: generate fmt vet manifests ## Build manager binary.
	go build -o bin/manager main.go
	go build $(GO_BUILD_FLAGS) -o bin/manager  main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GO_BUILD_FLAGS) -o bin/manager_linux_arm64 main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o bin/manager_linux_amd64 main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -


config/certmanager-deployment/certmanager-deployment.yaml: ## Download the cert-manager deployment
	test -s $@ || curl -L -o $@ \
  		https://github.com/cert-manager/cert-manager/releases/download/v1.9.1/cert-manager.yaml

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	test -d $@ || mkdir -p $@

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
ENVTEST ?= $(LOCALBIN)/setup-envtest
TERRAFORM ?= $(LOCALBIN)/terraform
KUBECTL ?= $(LOCALBIN)/kubectl

## Tool Versions
KUSTOMIZE_VERSION ?= v4.5.2
CONTROLLER_TOOLS_VERSION ?= v0.9.2
KUBEBUILDER_VERSION ?= v3.6.0
TERRAFORM_VERSION ?= 1.2.7
KUBECTL_VERSION ?= v1.24.0

.PHONY: download_tools
download_tools: kustomize controller-gen envtest kubebuilder kubectl terraform  ## Download all the tools

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download controller-gen locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	test -s $@ || GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: kubebuilder
kubebuilder: $(KUBEBUILDER) ## Download kubebuilder
$(KUBEBUILDER): $(LOCALBIN)
	test -s $@ || \
		(curl -L -o $@ https://github.com/kubernetes-sigs/kubebuilder/releases/download/$(KUBEBUILDER_VERSION)/kubebuilder_$(GOOS)_$(GOARCH) && \
		chmod a+x $@ && \
		touch $@ )

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

.PHONY: k9s
k9s: ## Check that k9s is installed
	which k9s || \
		(echo "Please install k9s, https://k9scli.io/topics/install/" ; exit 1)

##@ Google Cloud End to End Test
.PHONY: gcloud_test
gcloud_test: gcloud_test_infra gcloud_test_run gcloud_test_cleanup ## Run end-to-end tests on Google Cloud

.PHONY: gcloud_test_infra
gcloud_test_infra: gcloud_project gcloud_cluster gcloud_cert_manager_deploy ## Build test infrastructure for e2e tests

.PHONY: gcloud_test_run
gcloud_test_run: gcloud_install gcloud_proxy_image_push gcloud_operator_image_push gcloud_deploy gcloud_test_run_gotest ## Build and run the e2e test code

.PHONY: gcloud_test_cleanup
gcloud_test_cleanup: manifests gcloud_cleanup_test_namespaces gcloud_undeploy ## Remove all operator and testcase configs from the e2e k8s cluster

.PHONY: gcloud_test_cleanup
gcloud_test_infra_cleanup: gcloud_project ## Remove all operator and testcase configs from the e2e k8s cluster


# The URL to the container image repo provisioned for e2e tests
GCLOUD_DOCKER_URL_FILE :=$(PWD)/bin/gcloud-docker-repo.url
GCLOUD_DOCKER_URL=$(shell cat $(GCLOUD_DOCKER_URL_FILE))

## This is the default location from terraform
KUBECONFIG_GCLOUD ?= $(PWD)/bin/gcloud-kubeconfig.yaml

# kubectl command with proper environment vars set
GCLOUD_KUBECTL_ARGS = USE_GKE_GCLOUD_AUTH_PLUGIN=True KUBECONFIG=$(KUBECONFIG_GCLOUD)
GCLOUD_KUBECTL = $(GCLOUD_KUBECTL_ARGS) $(KUBECTL)

.PHONY: gcloud_project
gcloud_project: ## Check that the Google Cloud project exists
	gcloud projects describe $(GCLOUD_PROJECT_ID) 2>/dev/null || \
		( echo "No Google Cloud Project $(GCLOUD_PROJECT_ID) found"; exit 1 )

gcloud_cluster: gcloud_project  $(TERRAFORM) ## Build infrastructure for e2e tests
	PROJECT_DIR=$(PWD) \
  		GCLOUD_PROJECT_ID=$(GCLOUD_PROJECT_ID) \
  		KUBECONFIG_GCLOUD=$(KUBECONFIG_GCLOUD) \
  		GCLOUD_DOCKER_URL_FILE=$(GCLOUD_DOCKER_URL_FILE) \
  		test/e2e/tf/run.sh apply

gcloud_cluster_cleanup: gcloud_project  $(TERRAFORM) ## Build infrastructure for e2e tests
	PROJECT_DIR=$(PWD) \
  		GCLOUD_PROJECT_ID=$(GCLOUD_PROJECT_ID) \
  		KUBECONFIG_GCLOUD=$(KUBECONFIG_GCLOUD) \
  		GCLOUD_DOCKER_URL_FILE=$(GCLOUD_DOCKER_URL_FILE) \
  		test/e2e/tf/run.sh destroy

.PHONY: gcloud_cert_manager_deploy
gcloud_cert_manager_deploy: $(KUBECTL) ## Deploy the certificate manager
	$(GCLOUD_KUBECTL) apply -f config/certmanager-deployment/certmanager-deployment.yaml
	# wait for cert manager to become available before continuing
	$(GCLOUD_KUBECTL) rollout status deployment cert-manager -n cert-manager --timeout=90s


.PHONY: gcloud_install
gcloud_install: manifests $(KUSTOMIZE) $(KUBECTL) ## Install CRDs into the GKE cluster
	$(KUSTOMIZE) build config/crd | $(GCLOUD_KUBECTL) apply -f -

.PHONY: gcloud_deploy
gcloud_deploy: manifests $(KUSTOMIZE) $(KUBECTL) ## Deploy controller to the GKE cluster
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(GCLOUD_OPERATOR_URL)
	$(KUSTOMIZE) build config/default | USE_GKE_GCLOUD_AUTH_PLUGIN=True  KUBECONFIG=$(KUBECONFIG_GCLOUD) $(KUBECTL) apply -f -
	$(GCLOUD_KUBECTL) rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s

.PHONY: gcloud_undeploy
gcloud_undeploy: manifests $(KUSTOMIZE) $(KUBECTL) ## Deploy controller to the GKE cluster
	$(KUSTOMIZE) build config/default | $(GCLOUD_KUBECTL) delete -f -

###
# Build the cloudsql-proxy v2 docker image and push it to the
# google cloud project repo.
GCLOUD_PROXY_URL_FILE=$(PWD)/bin/last-proxy-image-url.txt
GCLOUD_PROXY_URL=$(shell cat $(GCLOUD_PROXY_URL_FILE) | tr -d "\n")

.PHONY: gcloud_proxy_image_push
gcloud_proxy_image_push: $(GCLOUD_PROXY_URL_FILE) ## Build and push a proxy image

.PHONY: $(GCLOUD_PROXY_URL_FILE)
$(GCLOUD_PROXY_URL_FILE):
	PROJECT_DIR=$(PROXY_PROJECT_DIR) \
	IMAGE_NAME=proxy-v2 \
	REPO_URL=${GCLOUD_DOCKER_URL} \
	IMAGE_URL_OUT=$@ \
	PLATFORMS=linux/arm64/v8,linux/amd64 \
	DOCKER_FILE_NAME=Dockerfile \
	$(PWD)/tools/docker-build.sh

###
# Build the operator docker image and push it to the
# google cloud project repo.
GCLOUD_OPERATOR_URL_FILE=$(PWD)/bin/last-gcloud-operator-url.txt
GCLOUD_OPERATOR_URL=$(shell cat $(GCLOUD_OPERATOR_URL_FILE) | tr -d "\n")

.PHONY: gcloud_operator_image_push
gcloud_operator_image_push: $(GCLOUD_OPERATOR_URL_FILE) ## Build and push a operator image

.PHONY: $(GCLOUD_OPERATOR_URL_FILE)
$(GCLOUD_OPERATOR_URL_FILE): build
	PROJECT_DIR=$(PWD) \
	IMAGE_NAME=cloud-sql-auth-proxy-operator \
	REPO_URL=${GCLOUD_DOCKER_URL} \
	IMAGE_URL_OUT=$@ \
	PLATFORMS=linux/arm64/v8,linux/amd64 \
	DOCKER_FILE_NAME=Dockerfile \
	$(PWD)/tools/docker-build.sh


.PHONY: gcloud_cleanup_test_namespaces
gcloud_cleanup_test_namespaces: $(KUSTOMIZE) $(KUBECTL) 	## list all namespaces, delete those named "test*"
	$(GCLOUD_KUBECTL) get ns -o=name | \
		grep namespace/test | \
		$(GCLOUD_KUBECTL_ENV) xargs $(KUBECTL) delete


.PHONY: gcloud_test_run_gotest
gcloud_test_run_gotest: ## Run the golang tests
	USE_GKE_GCLOUD_AUTH_PLUGIN=True \
		TEST_INFRA_JSON=$(LOCALBIN)/testinfra.json \
		PROXY_IMAGE_URL=$(GCLOUD_PROXY_URL) \
		OPERATOR_IMAGE_URL=$(GCLOUD_OPERATOR_URL) \
		go test --count=1 -v github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/test/e2e

.PHONY: gcloud_k9s
gcloud_k9s: ## Connect to the gcloud test cluster using the k9s tool
	USE_GKE_GCLOUD_AUTH_PLUGIN=True KUBECONFIG=$(KUBECONFIG_GCLOUD) k9s
