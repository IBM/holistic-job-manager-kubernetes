// ------------------------------------------------------ {COPYRIGHT-TOP} ---
// Copyright 2019, 2021, 2022, 2023 The Multi-Cluster App Dispatcher Authors.
// 
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ------------------------------------------------------ {COPYRIGHT-END} ---
package quota

import (
	arbv1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"
	clusterstateapi "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/controller/clusterstate/api"
)

type QuotaManagerInterface interface {
	Fits(aw *arbv1.AppWrapper, resources *clusterstateapi.Resource, proposedPremptions []*arbv1.AppWrapper) (bool, []*arbv1.AppWrapper, string)
	Release(aw *arbv1.AppWrapper) bool
}
