#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# This script holds common bash variables and utility functions.

function util::get_api_dirs {
  dirs=( "types" "clusters" "kantaloupeflow" "nodes" "v1" "core" "monitoring" "credentials" "quotas" "storage" "acceleratorcard")
  echo "${dirs[@]}"
  return $?
}

function util::resolve_path {
    local path="$1"
    local dir
    while [ -L "$path" ]; do
        dir=$(dirname "$path")
        path=$(readlink "$path")
        [[ $path != /* ]] && path="$dir/$path"
    done
    echo "$path"
}