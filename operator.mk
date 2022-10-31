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
generate:  ctrl_generate ctrl_manifests reset_image add_copyright_header go_fmt yaml_fmt ## Runs code generation, format, and validation tools

.PHONY: build
build: generate build_push_docker ## Builds and pushes the docker image to tag defined in envvar IMG

.PHONY: test
test: generate go_test ## Run tests (but not internal/teste2e)

.PHONY: deploy
deploy:  build deploy_with_kubeconfig ## Deploys the operator to the kubernetes cluster using envvar KUBECONFIG. Set $IMG envvar to the image tag.

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

.PHONY: build_push_docker
build_push_docker: # Build docker image with the operator. set IMG env var before running: `IMG=example.com/img:1.0 make build`
	@test -n "$(IMG)" || ( echo "IMG environment variable must be set to the public repo where you want to push the image" ; exit 1)
	docker buildx build --platform "linux/amd64" \
	  --build-arg GO_LD_FLAGS="$(VERSION_LDFLAGS)" \
	  -f "Dockerfile-operator" \
	  --push -t "$(IMG)" "$(PWD)"
	echo "$(IMG)" > bin/last-pushed-image-url.txt
##
# Kubernetes configuration targets

.PHONY: go_test
go_test: ctrl_manifests envtest # Run tests (but not internal/teste2e)
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" \
		go test ./internal/.../. -coverprofile cover.out

##@ Kubernetes configuration targets
.PHONY: ctrl_manifests
ctrl_manifests: controller-gen # Use controller-gen to generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: reset_image
reset_image: kustomize # Reset the image used in the kubernetes config to a default image.
	cd config/manager && $(KUSTOMIZE) edit set image controller=cloudsql-proxy-operator:latest

.PHONY: update_image
update_image: kustomize # Update the image used in the kubernetes config to $(IMG)
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)

.PHONY: deploy_with_kubeconfig # Deploy the kubernetes configuration to the current default cluster using kubectl
deploy_with_kubeconfig: install_certmanager install_crd deploy_operator

.PHONY: install_certmanager
install_certmanager: kubectl # Install the cert-manager operator to manage the certificates for the operator webhooks
	$(KUBECTL) apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.9.1/cert-manager.yaml
	$(KUBECTL) rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s

.PHONY: install
install_crd: kustomize kubectl # Install CRDs into the K8s cluster using the kubectl default behavior
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: deploy_operator
deploy_operator: kustomize kubectl # Deploy controller to the K8s cluster using the kubectl default behavior
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -
	$(E2E_KUBECTL) rollout status deployment -n cloud-sql-proxy-operator-system cloud-sql-proxy-operator-controller-manager --timeout=90s


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

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= latest
KUSTOMIZE_VERSION ?= latest
KUBECTL_VERSION ?= v1.24.0
KUSTOMIZE_VERSION ?= v4.5.2

remove_tools:
	rm -rf $(LOCALBIN)/*

all_tools: kustomize controller-gen envtest kubectl

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
