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

package api

import "github.com/kubewharf/godel-scheduler/pkg/framework/api"

type CachePodInfo struct {
	PodInfo            *api.CachePodInfo
	MovementGeneration uint
}

type CachePodInfoWrapper struct{ obj *CachePodInfo }

func MakeCachePodInfoWrapper() *CachePodInfoWrapper {
	return &CachePodInfoWrapper{&CachePodInfo{}}
}

func (w *CachePodInfoWrapper) Obj() *CachePodInfo {
	return w.obj
}

func (w *CachePodInfoWrapper) PodInfo(pInfo *api.CachePodInfo) *CachePodInfoWrapper {
	w.obj.PodInfo = pInfo
	return w
}

func (w *CachePodInfoWrapper) MovementGeneration(generation uint) *CachePodInfoWrapper {
	w.obj.MovementGeneration = generation
	return w
}
