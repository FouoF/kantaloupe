#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail
set -ex
source hack/util.sh

SCRIPT_PATH=$(util::resolve_path "${BASH_SOURCE[0]}")
SCRIPT_DIR=$(cd "$(dirname "$SCRIPT_PATH")" && pwd -P)
GO111MODULE=on go install k8s.io/code-generator/cmd/deepcopy-gen
GO111MODULE=on go install k8s.io/code-generator/cmd/register-gen
GO111MODULE=on go install k8s.io/code-generator/cmd/conversion-gen
GO111MODULE=on go install k8s.io/code-generator/cmd/client-gen
GO111MODULE=on go install k8s.io/code-generator/cmd/lister-gen
GO111MODULE=on go install k8s.io/code-generator/cmd/informer-gen
# GO111MODULE=on go install k8s.io/kube-openapi/cmd/openapi-gen
cd $(dirname "$SCRIPT_PATH")/../

echo "Generating with deepcopy-gen"
deepcopy-gen \
  --go-header-file hack/boilerplate/boilerplate.go.txt \
  --output-file=zz_generated.deepcopy.go \
  github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1 github.com/dynamia-ai/kantaloupe/api/crd/apis/kantaloupeflow/v1alpha1

echo "Generating with register-gen"
register-gen \
  --go-header-file hack/boilerplate/boilerplate.go.txt \
  --output-file=zz_generated.register.go \
  github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1 github.com/dynamia-ai/kantaloupe/api/crd/apis/kantaloupeflow/v1alpha1

echo "Generating with conversion-gen"
conversion-gen \
  --go-header-file hack/boilerplate/boilerplate.go.txt \
  --output-file=zz_generated.conversion.go \
  github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1 github.com/dynamia-ai/kantaloupe/api/crd/apis/kantaloupeflow/v1alpha1

echo "Generating with lister-gen"
lister-gen \
  --go-header-file hack/boilerplate/boilerplate.go.txt \
  --output-pkg=github.com/dynamia-ai/kantaloupe/api/crd/generated/listers \
  --output-dir=crd/generated/listers \
  github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1 github.com/dynamia-ai/kantaloupe/api/crd/apis/kantaloupeflow/v1alpha1

echo "Generating with informer-gen"
informer-gen \
  --go-header-file hack/boilerplate/boilerplate.go.txt \
  --versioned-clientset-package=github.com/dynamia-ai/kantaloupe/api/crd/generated/clientset/versioned \
  --listers-package=github.com/dynamia-ai/kantaloupe/api/crd/generated/listers \
  --output-pkg=github.com/dynamia-ai/kantaloupe/api/crd/generated/informers \
  --output-dir=crd/generated/informers \
  github.com/dynamia-ai/kantaloupe/api/crd/apis/cluster/v1alpha1 github.com/dynamia-ai/kantaloupe/api/crd/apis/kantaloupeflow/v1alpha1

echo "Generating with client-gen"
client-gen \
  --go-header-file hack/boilerplate/boilerplate.go.txt \
  --input-base="github.com/dynamia-ai/kantaloupe/api/crd/apis" \
  --input="cluster/v1alpha1,kantaloupeflow/v1alpha1" \
  --output-pkg=github.com/dynamia-ai/kantaloupe/api/crd/generated/clientset \
  --output-dir=crd/generated/clientset \
  --clientset-name=versioned

# cp -r github.com/dynamia-ai/kantaloupe/api/* ./
# rm -rf github.com
go mod tidy
