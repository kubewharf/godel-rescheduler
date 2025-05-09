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
	v1alpha1 "github.com/kubewharf/katalyst-api/pkg/apis/config/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// CustomNodeConfigLister helps list CustomNodeConfigs.
type CustomNodeConfigLister interface {
	// List lists all CustomNodeConfigs in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.CustomNodeConfig, err error)
	// Get retrieves the CustomNodeConfig from the index for a given name.
	Get(name string) (*v1alpha1.CustomNodeConfig, error)
	CustomNodeConfigListerExpansion
}

// customNodeConfigLister implements the CustomNodeConfigLister interface.
type customNodeConfigLister struct {
	indexer cache.Indexer
}

// NewCustomNodeConfigLister returns a new CustomNodeConfigLister.
func NewCustomNodeConfigLister(indexer cache.Indexer) CustomNodeConfigLister {
	return &customNodeConfigLister{indexer: indexer}
}

// List lists all CustomNodeConfigs in the indexer.
func (s *customNodeConfigLister) List(selector labels.Selector) (ret []*v1alpha1.CustomNodeConfig, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.CustomNodeConfig))
	})
	return ret, err
}

// Get retrieves the CustomNodeConfig from the index for a given name.
func (s *customNodeConfigLister) Get(name string) (*v1alpha1.CustomNodeConfig, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("customnodeconfig"), name)
	}
	return obj.(*v1alpha1.CustomNodeConfig), nil
}
