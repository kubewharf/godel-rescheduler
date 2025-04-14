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

package policycontroller

import (
	"fmt"
	"hash"
	"hash/fnv"
	"time"

	"github.com/davecgh/go-spew/spew"
	schedulingv1alpha1 "github.com/kubewharf/godel-scheduler-api/pkg/apis/scheduling/v1alpha1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	DefaultRetry = wait.Backoff{
		Steps:    5,
		Duration: time.Second,
		Factor:   1.0,
	}

	// timeout threshold is 15s (1+2+4+8)
	RetryMultipleTimes = wait.Backoff{
		Steps:    5,
		Duration: time.Second,
		Factor:   2.0,
	}

	// timeout threshold is 300s (looks like 299s)
	RetryEvenMoreTimes = wait.Backoff{
		Steps:    300,
		Duration: time.Second,
		Factor:   1.0,
	}
)

type OwnerMovementDetail struct {
	OwnerInfo         *schedulingv1alpha1.OwnerInfo
	NodeSuggestionMap map[string]*schedulingv1alpha1.RecommendedNode
}

func getTaskKey(task *schedulingv1alpha1.TaskInfo) string {
	return fmt.Sprintf("%s/%s/%s", task.Namespace, task.Name, task.UID)
}

func deepHashObject(hasher hash.Hash, objectToWrite interface{}) {
	hasher.Reset()
	printer := spew.ConfigState{
		Indent:         " ",
		SortKeys:       true,
		DisableMethods: true,
		SpewKeys:       true,
	}
	printer.Fprintf(hasher, "%#v", objectToWrite)
}

func computeHash(template *schedulingv1alpha1.Movement) string {
	podTemplateSpecHasher := fnv.New32a()
	deepHashObject(podTemplateSpecHasher, *template)

	return rand.SafeEncodeString(fmt.Sprint(podTemplateSpecHasher.Sum32()))
}

func generateMovementName(movement *schedulingv1alpha1.Movement) string {
	return "movement-" + computeHash(movement)
}
