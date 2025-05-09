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

// KatalystCustomConfigLister helps list KatalystCustomConfigs.
type KatalystCustomConfigLister interface {
	// List lists all KatalystCustomConfigs in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.KatalystCustomConfig, err error)
	// KatalystCustomConfigs returns an object that can list and get KatalystCustomConfigs.
	KatalystCustomConfigs(namespace string) KatalystCustomConfigNamespaceLister
	KatalystCustomConfigListerExpansion
}

// katalystCustomConfigLister implements the KatalystCustomConfigLister interface.
type katalystCustomConfigLister struct {
	indexer cache.Indexer
}

// NewKatalystCustomConfigLister returns a new KatalystCustomConfigLister.
func NewKatalystCustomConfigLister(indexer cache.Indexer) KatalystCustomConfigLister {
	return &katalystCustomConfigLister{indexer: indexer}
}

// List lists all KatalystCustomConfigs in the indexer.
func (s *katalystCustomConfigLister) List(selector labels.Selector) (ret []*v1alpha1.KatalystCustomConfig, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.KatalystCustomConfig))
	})
	return ret, err
}

// KatalystCustomConfigs returns an object that can list and get KatalystCustomConfigs.
func (s *katalystCustomConfigLister) KatalystCustomConfigs(namespace string) KatalystCustomConfigNamespaceLister {
	return katalystCustomConfigNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// KatalystCustomConfigNamespaceLister helps list and get KatalystCustomConfigs.
type KatalystCustomConfigNamespaceLister interface {
	// List lists all KatalystCustomConfigs in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.KatalystCustomConfig, err error)
	// Get retrieves the KatalystCustomConfig from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.KatalystCustomConfig, error)
	KatalystCustomConfigNamespaceListerExpansion
}

// katalystCustomConfigNamespaceLister implements the KatalystCustomConfigNamespaceLister
// interface.
type katalystCustomConfigNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all KatalystCustomConfigs in the indexer for a given namespace.
func (s katalystCustomConfigNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.KatalystCustomConfig, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.KatalystCustomConfig))
	})
	return ret, err
}

// Get retrieves the KatalystCustomConfig from the indexer for a given namespace and name.
func (s katalystCustomConfigNamespaceLister) Get(name string) (*v1alpha1.KatalystCustomConfig, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("katalystcustomconfig"), name)
	}
	return obj.(*v1alpha1.KatalystCustomConfig), nil
}
