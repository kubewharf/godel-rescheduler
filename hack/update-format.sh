#!/bin/bash
REPO_ROOT=${REPO_ROOT:-"$(cd "$(dirname "${BASH_SOURCE}")/.." && pwd -P)"}

# source "${REPO_ROOT}/hack/make-rules/lib/init.sh"
local_pkg=${LOCAL_PKG_PREFIX:-"github.com/kubewharf"}

# the first arg is relative path in repo root
# so we can format only for the file or files the directory
path=${1:-${REPO_ROOT}}

find_files() {
    find ${1:-.} -not \( \( \
        -wholename '*/output' \
        -o -wholename '*/.git/*' \
        -o -wholename '*/vendor/*' \
        -o -name 'bindata.go' \
        -o -name 'datafile.go' \
        -o -name '*.go.bak' \
        \) -prune \) \
        -name '*.go'
}

for go_file in $(find_files ${path}); do
    echo "==> goimports format: ${go_file}"
    # delete empty line between import ( and )
    sed -i.bak '/import ($/,/)$/{/^$/d}' ${go_file} && rm -rf ${go_file}.bak
    # sort imports
    goimports -format-only -w -local ${local_pkg} ${go_file}
done
