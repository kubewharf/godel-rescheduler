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

package commonstores

import (
	"sync"

	"github.com/kubewharf/godel-rescheduler/pkg/godel-rescheduler/cache/api"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type StoreType int

const (
	Cache StoreType = iota
	Snapshot
)

type BaseStore interface {
	AssumePod(*api.CachePodInfo) error
	ForgetPod(*api.CachePodInfo) error

	PeriodWorker(mu *sync.RWMutex)

	AddDeployItems(deploy *appsv1.Deployment) error
	UpdateDeployItems(oldDeploy, newDeploy *appsv1.Deployment) error
	DeleteDeployItems(deploy *appsv1.Deployment) error

	AddPod(pod *corev1.Pod) error
	UpdatePod(oldPod, newPod *corev1.Pod) error
	RemovePod(pod *corev1.Pod) error
}

type CommonStore interface {
	BaseStore

	Name() StoreName
	UpdateSnapshot(CommonStore) error
}
