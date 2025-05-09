/*
Copyright 2022 The Katalyst Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/kubewharf/katalyst-api/pkg/apis/overcommit/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// NodeOvercommitConfigLister helps list NodeOvercommitConfigs.
type NodeOvercommitConfigLister interface {
	// List lists all NodeOvercommitConfigs in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.NodeOvercommitConfig, err error)
	// Get retrieves the NodeOvercommitConfig from the index for a given name.
	Get(name string) (*v1alpha1.NodeOvercommitConfig, error)
	NodeOvercommitConfigListerExpansion
}

// nodeOvercommitConfigLister implements the NodeOvercommitConfigLister interface.
type nodeOvercommitConfigLister struct {
	indexer cache.Indexer
}

// NewNodeOvercommitConfigLister returns a new NodeOvercommitConfigLister.
func NewNodeOvercommitConfigLister(indexer cache.Indexer) NodeOvercommitConfigLister {
	return &nodeOvercommitConfigLister{indexer: indexer}
}

// List lists all NodeOvercommitConfigs in the indexer.
func (s *nodeOvercommitConfigLister) List(selector labels.Selector) (ret []*v1alpha1.NodeOvercommitConfig, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.NodeOvercommitConfig))
	})
	return ret, err
}

// Get retrieves the NodeOvercommitConfig from the index for a given name.
func (s *nodeOvercommitConfigLister) Get(name string) (*v1alpha1.NodeOvercommitConfig, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("nodeovercommitconfig"), name)
	}
	return obj.(*v1alpha1.NodeOvercommitConfig), nil
}
