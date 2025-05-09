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
	v1alpha1 "github.com/kubewharf/katalyst-api/pkg/apis/workload/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ServiceProfileDescriptorLister helps list ServiceProfileDescriptors.
type ServiceProfileDescriptorLister interface {
	// List lists all ServiceProfileDescriptors in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.ServiceProfileDescriptor, err error)
	// ServiceProfileDescriptors returns an object that can list and get ServiceProfileDescriptors.
	ServiceProfileDescriptors(namespace string) ServiceProfileDescriptorNamespaceLister
	ServiceProfileDescriptorListerExpansion
}

// serviceProfileDescriptorLister implements the ServiceProfileDescriptorLister interface.
type serviceProfileDescriptorLister struct {
	indexer cache.Indexer
}

// NewServiceProfileDescriptorLister returns a new ServiceProfileDescriptorLister.
func NewServiceProfileDescriptorLister(indexer cache.Indexer) ServiceProfileDescriptorLister {
	return &serviceProfileDescriptorLister{indexer: indexer}
}

// List lists all ServiceProfileDescriptors in the indexer.
func (s *serviceProfileDescriptorLister) List(selector labels.Selector) (ret []*v1alpha1.ServiceProfileDescriptor, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ServiceProfileDescriptor))
	})
	return ret, err
}

// ServiceProfileDescriptors returns an object that can list and get ServiceProfileDescriptors.
func (s *serviceProfileDescriptorLister) ServiceProfileDescriptors(namespace string) ServiceProfileDescriptorNamespaceLister {
	return serviceProfileDescriptorNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ServiceProfileDescriptorNamespaceLister helps list and get ServiceProfileDescriptors.
type ServiceProfileDescriptorNamespaceLister interface {
	// List lists all ServiceProfileDescriptors in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.ServiceProfileDescriptor, err error)
	// Get retrieves the ServiceProfileDescriptor from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.ServiceProfileDescriptor, error)
	ServiceProfileDescriptorNamespaceListerExpansion
}

// serviceProfileDescriptorNamespaceLister implements the ServiceProfileDescriptorNamespaceLister
// interface.
type serviceProfileDescriptorNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ServiceProfileDescriptors in the indexer for a given namespace.
func (s serviceProfileDescriptorNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.ServiceProfileDescriptor, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ServiceProfileDescriptor))
	})
	return ret, err
}

// Get retrieves the ServiceProfileDescriptor from the indexer for a given namespace and name.
func (s serviceProfileDescriptorNamespaceLister) Get(name string) (*v1alpha1.ServiceProfileDescriptor, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("serviceprofiledescriptor"), name)
	}
	return obj.(*v1alpha1.ServiceProfileDescriptor), nil
}
