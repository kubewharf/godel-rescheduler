# Copyright 2024 The Godel Rescheduler Authors
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

# Go installs the above commands to get installed in $GOBIN if defined, and $GOPATH/bin otherwise:
GOBIN="$(go env GOBIN)"
gobin="${GOBIN:-$(go env GOPATH)/bin}"

echo "Generating deepcopy funcs"
"${gobin}/deepcopy-gen" --input-dirs github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config -O zz_generated.deepcopy --bounding-dirs . -h "$PWD"/hack/boilerplate.go.txt

echo "Generating defaulters"
"${GOPATH}/bin/defaulter-gen"  --input-dirs github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config -O zz_generated.defaults -h "$PWD"/hack/boilerplate.go.txt