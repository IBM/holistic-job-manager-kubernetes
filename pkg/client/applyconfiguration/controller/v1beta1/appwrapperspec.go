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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1beta1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AppWrapperSpecApplyConfiguration represents an declarative configuration of the AppWrapperSpec type for use
// with apply.
type AppWrapperSpecApplyConfiguration struct {
	Priority      *int32                                    `json:"priority,omitempty"`
	PrioritySlope *float64                                  `json:"priorityslope,omitempty"`
	Service       *AppWrapperServiceApplyConfiguration      `json:"service,omitempty"`
	AggrResources *AppWrapperResourceListApplyConfiguration `json:"resources,omitempty"`
	Selector      *v1.LabelSelector                         `json:"selector,omitempty"`
	SchedSpec     *SchedulingSpecTemplateApplyConfiguration `json:"schedulingSpec,omitempty"`
}

// AppWrapperSpecApplyConfiguration constructs an declarative configuration of the AppWrapperSpec type for use with
// apply.
func AppWrapperSpec() *AppWrapperSpecApplyConfiguration {
	return &AppWrapperSpecApplyConfiguration{}
}

// WithPriority sets the Priority field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Priority field is set to the value of the last call.
func (b *AppWrapperSpecApplyConfiguration) WithPriority(value int32) *AppWrapperSpecApplyConfiguration {
	b.Priority = &value
	return b
}

// WithPrioritySlope sets the PrioritySlope field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PrioritySlope field is set to the value of the last call.
func (b *AppWrapperSpecApplyConfiguration) WithPrioritySlope(value float64) *AppWrapperSpecApplyConfiguration {
	b.PrioritySlope = &value
	return b
}

// WithService sets the Service field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Service field is set to the value of the last call.
func (b *AppWrapperSpecApplyConfiguration) WithService(value *AppWrapperServiceApplyConfiguration) *AppWrapperSpecApplyConfiguration {
	b.Service = value
	return b
}

// WithAggrResources sets the AggrResources field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AggrResources field is set to the value of the last call.
func (b *AppWrapperSpecApplyConfiguration) WithAggrResources(value *AppWrapperResourceListApplyConfiguration) *AppWrapperSpecApplyConfiguration {
	b.AggrResources = value
	return b
}

// WithSelector sets the Selector field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Selector field is set to the value of the last call.
func (b *AppWrapperSpecApplyConfiguration) WithSelector(value v1.LabelSelector) *AppWrapperSpecApplyConfiguration {
	b.Selector = &value
	return b
}

// WithSchedSpec sets the SchedSpec field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SchedSpec field is set to the value of the last call.
func (b *AppWrapperSpecApplyConfiguration) WithSchedSpec(value *SchedulingSpecTemplateApplyConfiguration) *AppWrapperSpecApplyConfiguration {
	b.SchedSpec = value
	return b
}
