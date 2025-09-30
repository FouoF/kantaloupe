#!/usr/bin/env bash

set -o errexit
set -o nounset


REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
LOGCHECK_LINT_PKG="sigs.k8s.io/logtools/logcheck"
LOGCHECK_LINT_VER="v0.9.0"

cd "${REPO_ROOT}"
source "hack/util.sh"

util::install_tools ${LOGCHECK_LINT_PKG} ${LOGCHECK_LINT_VER}


# fetch latest commit
files=$(git diff --name-only HEAD~1 HEAD)  

# for each file in the commit
for file in $files; do
    # exclude e2e test tools
    if [[ "$file" =~ hack/tools/.* ]]; then
            continue
    fi
    if [[ "$file" =~ vendor/.* ]]; then
        continue
    fi
    # exclude generated files
    if [[ "$file" =~ zz_generated.* ]]; then
        continue
    fi
    # exclude api directory
    if [[ "$file" =~ api/.* ]]; then
        continue
    fi
    # exclude test directory
    if [[ "$file" =~ test/.* ]]; then
        continue
    fi
    # check if the file is a go file 
    if [[ "$file" =~ \.go$ ]]; then  
        # extract the directory name  
        dir=$(dirname "$file")

        # Check if the directory exists
        if [[ -d $dir ]]; then
            echo "Checking $dir -- $file"
            logcheck ./$dir
        else
            echo "Skipping $file: Directory $dir not found"
        fi
    fi
done
