#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

helm dep up ./charts/kantaloupe
helm template ./charts/kantaloupe --set global.kantaloupe.imageTag=v$(KANTALOUPE_CHART_VERSION) || exit 1
helm lint ./charts/kantaloupe --debug --set global.kantaloupe.imageTag=v$(KANTALOUPE_CHART_VERSION) || exit 1
