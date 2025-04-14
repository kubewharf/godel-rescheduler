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

package throttle

import (
	"context"
	"time"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/apis/config"
	reschedulercache "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/policies"
	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/api"
	framework "github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/framework/api"

	godel_framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
)

const (
	Name = "Throttle"
)

type Throttle struct {
	throttleItems []config.ThrottleItem
}

var (
	_ api.DetectorPlugin          = &Throttle{}
	_ api.AlgorithmPlugin         = &Throttle{}
	_ api.PolicyStoreSyncerPlugin = &Throttle{}
)

func New(plArgs apiruntime.Object, rfh framework.ReschedulerFrameworkHandle) (godel_framework.Plugin, error) {
	var throttleItems []config.ThrottleItem
	if args, ok := plArgs.(*config.ThrottleArgs); !ok || args == nil {
		klog.Warningf("Failed to parse plugin args for %s: %v", Name, plArgs)
		throttleItems = []config.ThrottleItem{
			{
				LabelSelectorKey: podutil.PodGroupNameAnnotationKey,
				ThrottleDuration: metav1.Duration{Duration: time.Hour},
				ThrottleValue:    1,
			},
		}
	} else {
		throttleItems = args.ThrottleItems
	}
	return &Throttle{
		throttleItems: throttleItems,
	}, nil
}

func (t *Throttle) Name() string {
	return Name
}

func (t *Throttle) Detect(_ context.Context) (api.DetectorResult, error) {
	return api.DetectorResult{}, nil
}

func (t *Throttle) SyncPolicyStore(ctx context.Context, snapshot *reschedulercache.ReschedulingSnapshot) policies.PolicyStore {
	return newThrottleStore(snapshot, t.throttleItems)
}

func (t *Throttle) Reschedule(ctx context.Context, detectRes api.DetectorResult) (api.RescheduleResult, error) {
	return api.RescheduleResult{}, nil
}
