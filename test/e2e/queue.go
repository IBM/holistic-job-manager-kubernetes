// +build !private

/*
Copyright 2018 The Kubernetes Authors.

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
/*
Copyright 2019, 2021 The Multi-Cluster App Dispatcher Authors.

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
package e2e

import (
	"fmt"
	arbv1 "github.com/IBM/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"
	"time"
)

var _ = Describe("AppWrapper E2E Test", func() {

	/* 	It("Create AppWrapper - Generic 100 Deployment Only - 2 pods each", func() {
		context := initTestContext()
		defer cleanupTestContextExtendedTime(context, (240 * time.Second))

		const (
			awCount = 100
		)
		modDivisor := int(awCount / 10)
		replicas := 2
		var aws [awCount]*arbv1.AppWrapper
		for i := 0; i < awCount; i++ {
			name := fmt.Sprintf("%s%d-", "aw-generic-deployment-", replicas)
			if i < 99 {
				name = fmt.Sprintf("%s%s", name, "0")
			}
			if i < 9 {
				name = fmt.Sprintf("%s%s", name, "0")
			}
			name = fmt.Sprintf("%s%d", name, i+1)
			cpuDemand := "5m"
			if ((i+1)%modDivisor) == 0 || i == 0 {
				fmt.Fprintf(os.Stdout, "[e2e] Creating AW %s with %s cpu and %d replica(s).\n", name, cpuDemand, replicas)
			}
			aws[i] = createGenericDeploymentWithCPUAW(context, name, cpuDemand, replicas)
		}

		// Give the deployments time to create pods
		time.Sleep(2 * time.Minute)
		for i := 0; i < awCount; i++ {
			if ((i+1)%modDivisor) == 0 || i == 0 {
				fmt.Fprintf(os.Stdout, "[e2e] Checking for %d replicas running for AW %s.\n", replicas, aws[i].Name)
			}
			err := waitAWReadyQuiet(context, aws[i])
			Expect(err).NotTo(HaveOccurred())
		}
	}) */

	It("MCAD CPU Accounting Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD CPU Accounting Test - Started.\n")

		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		// This should fill up the worker node and most of the master node
		aw := createDeploymentAWwith550CPU(context, "aw-deployment-2-550cpu")
		appwrappers = append(appwrappers, aw)

		err := waitAWPodsReady(context, aw)
		Expect(err).NotTo(HaveOccurred())

		// This should fill up the master node
		aw2 := createDeploymentAWwith350CPU(context, "aw-deployment-2-350cpu")
		appwrappers = append(appwrappers, aw2)

		// Using quite mode due to creating of pods in earlier step.
		err = waitAWReadyQuiet(context, aw2)
		Expect(err).NotTo(HaveOccurred())

	})

	It("Create AppWrapper - StatefulSet Only - 2 Pods", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Create AppWrapper - StatefulSet Only - 2 Pods - Started.\n")

		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createStatefulSetAW(context, "aw-statefulset-2")
		appwrappers = append(appwrappers, aw)

		err := waitAWPodsReady(context, aw)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Create AppWrapper - Generic StatefulSet Only - 2 Pods", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Create AppWrapper - Generic StatefulSet Only - 2 Pods - Started.\n")

		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createGenericStatefulSetAW(context, "aw-generic-statefulset-2")
		appwrappers = append(appwrappers, aw)

		err := waitAWPodsReady(context, aw)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Create AppWrapper - Deployment Only - 3 Pods", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Create AppWrapper - Deployment Only 3 Pods - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createDeploymentAW(context, "aw-deployment-3")
		appwrappers = append(appwrappers, aw)

		fmt.Fprintf(os.Stdout, "[e2e] Awaiting %d pods running for AW %s.\n", aw.Spec.SchedSpec.MinAvailable, aw.Name)
		err := waitAWPodsReady(context, aw)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Create AppWrapper - Job - 1 Parallelism==Completions", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Create AppWrapper - Job - 1 Parallelism==Completions.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createDeploymentAW(context, "test-job")
		appwrappers = append(appwrappers, aw)

		fmt.Fprintf(os.Stdout, "[e2e] Awaiting %d pods running for AW %s.\n", aw.Spec.SchedSpec.MinAvailable, aw.Name)
		err := waitAWPodsReady(context, aw)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Create AppWrapper - Generic Deployment Only - 3 pods", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Create AppWrapper - Generic Deployment Only - 3 pods - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createGenericDeploymentAW(context, "aw-generic-deployment-3")
		appwrappers = append(appwrappers, aw)

		err := waitAWPodsReady(context, aw)
		Expect(err).NotTo(HaveOccurred())
	})

	//NOTE: Recommend this test not to be the last test in the test suite it may pass
	//      the local test but may cause controller to fail which is not
	//      part of this test's validation.

	It("Create AppWrapper- Bad PodTemplate", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Create AppWrapper- Bad PodTemplate - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createBadPodTemplateAW(context, "aw-bad-podtemplate-2")
		appwrappers = append(appwrappers, aw)

		err := waitAWPodsReady(context, aw)
		Expect(err).To(HaveOccurred())
	})

	It("Create AppWrapper  - Bad Generic PodTemplate Only", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Create AppWrapper  - Bad Generic PodTemplate Only - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw, err := createBadGenericPodTemplateAW(context, "aw-generic-podtemplate-2")
		if err == nil {
			appwrappers = append(appwrappers, aw)
		}
		Expect(err).To(HaveOccurred())
	})

	It("Create AppWrapper  - PodTemplate Only - 2 Pods", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Create AppWrapper  - PodTemplate Only - 2 Pods - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createPodTemplateAW(context, "aw-podtemplate-2")
		appwrappers = append(appwrappers, aw)

		err := waitAWPodsReady(context, aw)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Create AppWrapper  - Generic Pod Only - 1 Pod", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Create AppWrapper  - Generic Pod Only - 1 Pod - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createGenericPodAW(context, "aw-generic-pod-1")
		appwrappers = append(appwrappers, aw)

		err := waitAWPodsReady(context, aw)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Create AppWrapper  - Generic Pod Too Big - 1 Pod", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Create AppWrapper  - Generic Pod Too Big - 1 Pod - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createGenericPodTooBigAW(context, "aw-generic-big-pod-1")
		appwrappers = append(appwrappers, aw)

		err := waitAWAnyPodsExists(context, aw)
		Expect(err).To(HaveOccurred())
	})

	It("Create AppWrapper  - Bad Generic Pod Only", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Create AppWrapper  - Bad Generic Pod Only - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createBadGenericPodAW(context, "aw-bad-generic-pod-1")
		appwrappers = append(appwrappers, aw)

		err := waitAWPodsReady(context, aw)
		Expect(err).To(HaveOccurred())

	})

	It("Create AppWrapper - Namespace Only - 0 Pods", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Create AppWrapper - Namespace Only - 0 Pods - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createNamespaceAW(context, "aw-namespace-0")
		appwrappers = append(appwrappers, aw)

		err := waitAWNonComputeResourceActive(context, aw)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Create AppWrapper - Generic Namespace Only - 0 Pods", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Create AppWrapper - Generic Namespace Only - 0 Pods - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createGenericNamespaceAW(context, "aw-generic-namespace-0")
		appwrappers = append(appwrappers, aw)

		err := waitAWNonComputeResourceActive(context, aw)
		Expect(err).NotTo(HaveOccurred())

	})

	It("MCAD CPU Accounting Queuing Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD CPU Accounting Queuing Test - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		// This should fill up the worker node and most of the master node
		aw := createDeploymentAWwith550CPU(context, "aw-deployment-2-550cpu")
		appwrappers = append(appwrappers, aw)

		err := waitAWPodsReadyDebug(context, aw)
		Expect(err).NotTo(HaveOccurred())

		// This should not fit on cluster
		aw2 := createDeploymentAWwith426CPU(context, "aw-deployment-2-426cpu")
		appwrappers = append(appwrappers, aw2)

		err = waitAWAnyPodsExists(context, aw2)
		Expect(err).To(HaveOccurred())
	})

	It("Create AppWrapper - Generic 100 Deployment Only - 2 pods each", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Generic 100 Deployment Only - 2 pods each - Started.\n")

		context := initTestContext()
		var aws []*arbv1.AppWrapper
		//appwrappersPtr := &aws
		//defer cleanupTestObjectsPtr(context, appwrappersPtr)

		const (
			awCount = 100
			reportingInterval = 10
		)

		replicas := 2
		modDivisor := int(awCount / reportingInterval)
		for i := 0; i < awCount; i++ {
			name := fmt.Sprintf("%s%d-", "aw-generic-deployment-", replicas)

			// Pad name with '0' when i < 100
			if i < (awCount - 1) {
				name = fmt.Sprintf("%s%s", name, "0")
			}

			// Pad name with '0' when i < 10
			if i < (reportingInterval - 1) {
				name = fmt.Sprintf("%s%s", name, "0")
			}

			name = fmt.Sprintf("%s%d", name, i+1)
			cpuDemand := "5m"
			if ((i+1)%modDivisor) == 0 || i == 0 {
				fmt.Fprintf(os.Stdout, "[e2e] Creating AW %s with %s cpu and %d replica(s).\n", name, cpuDemand, replicas)
			}
			aw := createGenericDeploymentWithCPUAW(context, name, cpuDemand, replicas)
			aws = append(aws, aw)
		}

		// Give the deployments time to create pods
		time.Sleep(2 * time.Minute)
		for i := 0; i < len(aws); i++ {
			if ((i+1)%modDivisor) == 0 || i == 0 {
				fmt.Fprintf(os.Stdout, "[e2e] Checking for %d replicas running for AW %s.\n", replicas, aws[i].Name)
			}
			err := waitAWReadyQuiet(context, aws[i])
			Expect(err).NotTo(HaveOccurred())
		}
	})

	/*
		It("Gang scheduling", func() {
			context := initTestContext()
			defer cleanupTestContext(context)
			rep := clusterSize(context, oneCPU)/2 + 1

			replicaset := createReplicaSet(context, "rs-1", rep, "nginx", oneCPU)
			err := waitReplicaSetReady(context, replicaset.Name)
			Expect(err).NotTo(HaveOccurred())

			job := &jobSpec{
				name:      "gang-qj",
				namespace: "test",
				tasks: []taskSpec{
					{
						img: "busybox",
						req: oneCPU,
						min: rep,
						rep: rep,
					},
				},
			}

			_, pg := createJobEx(context, job)
			err = waitPodGroupPending(context, pg)
			Expect(err).NotTo(HaveOccurred())

			waitPodGroupUnschedulable(context, pg)
			Expect(err).NotTo(HaveOccurred())

			err = deleteReplicaSet(context, replicaset.Name)
			Expect(err).NotTo(HaveOccurred())

			err = waitPodGroupReady(context, pg)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Reclaim", func() {
			context := initTestContext()
			defer cleanupTestContext(context)

			slot := oneCPU
			rep := clusterSize(context, slot)

			job := &jobSpec{
				tasks: []taskSpec{
					{
						img: "nginx",
						req: slot,
						min: 1,
						rep: rep,
					},
				},
			}

			job.name = "q1-qj-1"
			job.queue = "q1"
			_, aw1 := createJobEx(context, job)
			err := waitAWPodsReady(context, aw1)
			Expect(err).NotTo(HaveOccurred())

			expected := int(rep) / 2
			// Reduce one pod to tolerate decimal fraction.
			if expected > 1 {
				expected--
			} else {
				err := fmt.Errorf("expected replica <%d> is too small", expected)
				Expect(err).NotTo(HaveOccurred())
			}

			job.name = "q2-qj-2"
			job.queue = "q2"
			_, aw2 := createJobEx(context, job)
			err = waitAWPodsReadyEx(context, aw2, expected)
			Expect(err).NotTo(HaveOccurred())

			err = waitAWPodsReadyEx(context, aw1, expected)
			Expect(err).NotTo(HaveOccurred())
		})
	*/
})
