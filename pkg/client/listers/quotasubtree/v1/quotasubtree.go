/*
Copyright 2019, 2021, 2022, 2023 The Multi-Cluster App Dispatcher Authors.

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

package v1

import (
	v1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/quotaplugins/quotasubtree/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// QuotaSubtreeLister helps list QuotaSubtrees.
// All objects returned here must be treated as read-only.
type QuotaSubtreeLister interface {
	// List lists all QuotaSubtrees in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.QuotaSubtree, err error)
	// QuotaSubtrees returns an object that can list and get QuotaSubtrees.
	QuotaSubtrees(namespace string) QuotaSubtreeNamespaceLister
	QuotaSubtreeListerExpansion
}

// quotaSubtreeLister implements the QuotaSubtreeLister interface.
type quotaSubtreeLister struct {
	indexer cache.Indexer
}

// NewQuotaSubtreeLister returns a new QuotaSubtreeLister.
func NewQuotaSubtreeLister(indexer cache.Indexer) QuotaSubtreeLister {
	return &quotaSubtreeLister{indexer: indexer}
}

// List lists all QuotaSubtrees in the indexer.
func (s *quotaSubtreeLister) List(selector labels.Selector) (ret []*v1.QuotaSubtree, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.QuotaSubtree))
	})
	return ret, err
}

// QuotaSubtrees returns an object that can list and get QuotaSubtrees.
func (s *quotaSubtreeLister) QuotaSubtrees(namespace string) QuotaSubtreeNamespaceLister {
	return quotaSubtreeNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// QuotaSubtreeNamespaceLister helps list and get QuotaSubtrees.
// All objects returned here must be treated as read-only.
type QuotaSubtreeNamespaceLister interface {
	// List lists all QuotaSubtrees in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.QuotaSubtree, err error)
	// Get retrieves the QuotaSubtree from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.QuotaSubtree, error)
	QuotaSubtreeNamespaceListerExpansion
}

// quotaSubtreeNamespaceLister implements the QuotaSubtreeNamespaceLister
// interface.
type quotaSubtreeNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all QuotaSubtrees in the indexer for a given namespace.
func (s quotaSubtreeNamespaceLister) List(selector labels.Selector) (ret []*v1.QuotaSubtree, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.QuotaSubtree))
	})
	return ret, err
}

// Get retrieves the QuotaSubtree from the indexer for a given namespace and name.
func (s quotaSubtreeNamespaceLister) Get(name string) (*v1.QuotaSubtree, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("quotasubtree"), name)
	}
	return obj.(*v1.QuotaSubtree), nil
}
