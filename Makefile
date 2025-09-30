# Global variables
include version.mk
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# BUILD_ARCH ?= linux/$(GOARCH)
# ifeq ($(BUILD_ARM),true)
# ifneq ($(GOARCH),arm64)
# 	  BUILD_ARCH= linux/$(GOARCH),linux/arm64
# endif
# endif
# ifeq ($(BUILD_X86),true)
# ifneq ($(GOARCH),amd64)
# 	  BUILD_ARCH= linux/$(GOARCH),linux/amd64
# endif
# endif

BUILD_ARCH = linux/amd64

# convert to git version to semver version v0.1.1-14-gb943a40 --> v0.1.1+14-gb943a40
KANTALOUPE_VERSION := $(shell echo $(VERSION) | sed 's/-/+/1')
#KANTALOUPE_VERSION := "v0.33.0+dev-75064cd7"

# convert to git version to semver version v0.1.1+14-gb943a40 --> v0.1.1-14-gb943a40
KANTALOUPE_IMAGE_VERSION := $(shell echo $(KANTALOUPE_VERSION) | sed 's/+/-/1')
#KANTALOUPE_IMAGE_VERSION := "v0.33.0-dev-75064cd7"

#v0.1.1 --> 0.1.1 Match the helm chart version specification, remove the preceding prefix `v` character
KANTALOUPE_CHART_VERSION := $(shell echo ${KANTALOUPE_VERSION} |sed  's/^v//g' )
#KANTALOUPE_CHART_VERSION := "0.33.0+dev-75064cd7"

ENVTEST_K8S_VERSION = 1.32.2

REGISTRY_REPO?="ghcr.io/dynamia-ai/kantaloupe"

# Git information
GIT_VERSION ?= $(shell git describe --tags --abbrev=8 --dirty) # attention: gitlab CI: git fetch should not use shallow
GIT_COMMIT_HASH ?= $(shell git rev-parse HEAD)
GIT_TREESTATE = "clean"
GIT_DIFF = $(shell git diff --quiet >/dev/null 2>&1; if [ $$? -eq 1 ]; then echo "1"; fi)
ifeq ($(GIT_DIFF), 1)
    GIT_TREESTATE = "dirty"
endif
BUILDDATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := "-X github.com/dynamia-ai/kantaloupe/pkg/version.gitVersion=$(GIT_VERSION) \
            -X github.com/dynamia-ai/kantaloupe/pkg/version.gitCommit=$(GIT_COMMIT_HASH) \
            -X github.com/dynamia-ai/kantaloupe/pkg/version.gitTreeState=$(GIT_TREESTATE) \
            -X github.com/dynamia-ai/kantaloupe/pkg/version.buildDate=$(BUILDDATE)"

# Build
.PHONY: apiserver
apiserver:
	go build -ldflags $(LDFLAGS) -o bin/kantaloupe-apiserver cmd/apiserver/main.go

.PHONY: controller
controller:
	go build -ldflags $(LDFLAGS) -o bin/kantaloupe-controller-manager cmd/controller-manager/main.go

# Build docker images
.PHONY: kantaloupe-apiserver
kantaloupe-apiserver:
	echo "Building kantaloupe-apiserver for arch = $(BUILD_ARCH)"
	export DOCKER_CLI_EXPERIMENTAL=enabled ;\
	! ( docker buildx ls | grep kantaloupe-apiserver-multi-platform-builder ) && docker buildx create --use --platform=$(BUILD_ARCH) --name kantaloupe-apiserver-multi-platform-builder --driver-opt image=docker.io/moby/buildkit:buildx-stable-1 ;\
	docker buildx build \
			--builder kantaloupe-apiserver-multi-platform-builder \
			--platform $(BUILD_ARCH) \
			--build-arg LDFLAGS=$(LDFLAGS) \
			--tag $(REGISTRY_REPO)/kantaloupe-apiserver:latest  \
			-f ./build/apiserver/Dockerfile \
			--load \
			.

.PHONY: kantaloupe-controller-manager
kantaloupe-controller-manager:
	echo "Building kantaloupe-controller-manager for arch = $(BUILD_ARCH)"
	export DOCKER_CLI_EXPERIMENTAL=enabled ;\
	! ( docker buildx ls | grep kantaloupe-cr-multi-platform-builder ) && docker buildx create --use --platform=$(BUILD_ARCH) --name kantaloupe-cr-multi-platform-builder --driver-opt image=docker.io/moby/buildkit:buildx-stable-1 ;\
	docker buildx build \
			--builder kantaloupe-cr-multi-platform-builder \
			--platform $(BUILD_ARCH) \
			--build-arg LDFLAGS=$(LDFLAGS) \
			--tag $(REGISTRY_REPO)/kantaloupe-controller-manager:latest  \
			-f ./build/controller-manager/Dockerfile \
			--load \
			.

# Lint
.PHONY: test-staticcheck
test-staticcheck:
	hack/verify-staticcheck.sh

.PHONY: verify-code-gen
verify-code-gen:
	hack/verify-codegen.sh
	hack/verify-crdgen.sh

.PHONY: verify-vendor
verify-vendor:
	hack/verify-vendor.sh

.PHONY: verify-proto
verify-proto:
	cd ./api/ && $(MAKE) verify-proto

.PHONY: verify-proto-swagger
verify-proto-swagger:
	cd ./api/ && $(MAKE) verify-swagger

.PHONY: verify-import-aliases
 verify-import-aliases:
	hack/verify-import-aliases.sh

.PHONY: verify_helm_chart
verify_helm_chart:
	hack/verify-helm-chart.sh

.PHONY: verify-all
verify-all: test-staticcheck verify-import-aliases verify-code-gen verify-vendor verify-proto verify-proto-swagger verify-grpc-ts verify_helm_chart

# generate code

.PHONY: genproto
genproto:
	cd ./api/ && $(MAKE) genproto

.PHONY: genswagger
genswagger:
	cd ./api/ && $(MAKE)  genswagger

.PHONY: gen-code-gen
gen-code-gen:
	cd ./api/ && $(MAKE) gen-code-gen

.PHONY: gen-crd-yaml
gen-crd-yaml:
	bash hack/update-crdgen.sh

ENVTEST = $(shell pwd)/bin/setup-envtest
.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef