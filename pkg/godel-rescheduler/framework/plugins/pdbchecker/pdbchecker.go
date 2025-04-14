// Copyright 2024 The Godel Rescheduler Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pdbchecker

import (
	"context"

	reschedulercache "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/api"
	framework "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/api"

	godel_framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
)

const (
	Name = "PDBChecker"
)

type Operator string

type PDBChecker struct {
}

var (
	_ api.DetectorPlugin          = &PDBChecker{}
	_ api.AlgorithmPlugin         = &PDBChecker{}
	_ api.PolicyStoreSyncerPlugin = &PDBChecker{}
)

func New(plArgs apiruntime.Object, rfh framework.ReschedulerFrameworkHandle) (godel_framework.Plugin, error) {
	return &PDBChecker{}, nil
}

func (p *PDBChecker) Name() string {
	return Name
}

func (p *PDBChecker) Detect(_ context.Context) (api.DetectorResult, error) {
	return api.DetectorResult{}, nil
}

func (p *PDBChecker) SyncPolicyStore(ctx context.Context, snapshot *reschedulercache.ReschedulingSnapshot) policies.PolicyStore {
	return newPDBCheckerStore(snapshot)
}

func (p *PDBChecker) Reschedule(ctx context.Context, detectRes api.DetectorResult) (api.RescheduleResult, error) {
	return api.RescheduleResult{}, nil
}
