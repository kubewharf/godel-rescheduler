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

package movementrecycler

import (
	"context"
	"fmt"

	godelclient "github.com/kubewharf/godel-scheduler-api/pkg/client/clientset/versioned"
	schedulerlister "github.com/kubewharf/godel-scheduler-api/pkg/client/listers/scheduling/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
)

type MovementRecycler struct {
	crdClient      godelclient.Interface
	movementLister schedulerlister.MovementLister
}

func NewMovementRecycler(crdClient godelclient.Interface, movementLister schedulerlister.MovementLister) *MovementRecycler {
	return &MovementRecycler{
		crdClient:      crdClient,
		movementLister: movementLister,
	}
}

func (mr *MovementRecycler) CleanAllMovements() error {
	movements, err := mr.movementLister.List(labels.Everything())
	if err != nil {
		return err
	}
	for _, movement := range movements {
		err := retry.OnError(retry.DefaultRetry,
			func(err error) bool {
				return true
			},
			func() error {
				_, gotErr := mr.crdClient.SchedulingV1alpha1().Movements().Get(context.Background(), movement.Name, metav1.GetOptions{})
				if gotErr != nil {
					if errors.IsNotFound(gotErr) {
						return nil
					}
					return gotErr
				}
				// use evict function in order to not violating pdb
				return mr.crdClient.SchedulingV1alpha1().Movements().Delete(context.Background(), movement.Name, metav1.DeleteOptions{})
			})
		if err != nil {
			return fmt.Errorf("Failed to delete movement %s", movement.Name)
		}
	}

	return retry.OnError(retry.DefaultRetry,
		func(err error) bool {
			return true
		},
		func() error {
			movements, err = mr.movementLister.List(labels.Everything())
			if err != nil {
				return err
			}
			if len(movements) > 0 {
				return fmt.Errorf("Failed to clean all movements, still have %d movements", len(movements))
			}
			return nil
		})
}
