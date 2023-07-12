//go:build !private
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
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	arbv1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		aw := createDeploymentAWwith550CPU(context, appendRandomString("aw-deployment-2-550cpu"))
		appwrappers = append(appwrappers, aw)

		err := waitAWPodsReady(context, aw)
		Expect(err).NotTo(HaveOccurred())

		// This should fill up the master node
		aw2 := createDeploymentAWwith350CPU(context, appendRandomString("aw-deployment-2-350cpu"))
		appwrappers = append(appwrappers, aw2)

		// Using quite mode due to creating of pods in earlier step.
		err = waitAWReadyQuiet(context, aw2)
		fmt.Fprintf(os.Stdout, "The error is %v", err)
		Expect(err).NotTo(HaveOccurred())
	})

	It("MCAD CPU Preemption Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD CPU Preemption Test - Started.\n")

		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		// This should fill up the worker node and most of the master node
		aw := createDeploymentAWwith550CPU(context, appendRandomString("aw-deployment-2-550cpu"))
		appwrappers = append(appwrappers, aw)
		time.Sleep(2 * time.Minute)
		err := waitAWPodsReady(context, aw)
		Expect(err).NotTo(HaveOccurred())

		// This should not fit on cluster
		aw2 := createDeploymentAWwith426CPU(context, appendRandomString("aw-deployment-2-426cpu"))
		appwrappers = append(appwrappers, aw2)

		err = waitAWAnyPodsExists(context, aw2)
		Expect(err).To(HaveOccurred())

		// This should fit on cluster, initially queued because of aw2 above but should eventually
		// run after prevention of aw2 above.
		aw3 := createDeploymentAWwith425CPU(context, "aw-deployment-2-425cpu")
		appwrappers = append(appwrappers, aw3)

		// Since preemption takes some time, increasing timeout wait time to 2 minutes
		err = waitAWPodsExists(context, aw3, 120000*time.Millisecond)
		fmt.Fprintf(os.Stdout, "[e2e] The error is %v", err)
		Expect(err).NotTo(HaveOccurred())
	})

	It("MCAD CPU Requeuing - Completion After Enough Requeuing Times Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD CPU Requeuing Test - Started.\n")

		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		// Create a job with init containers that need 200 seconds to be ready before the container starts.
		// The requeuing mechanism is set to start at 1 minute, which is not enough time for the PODs to be completed.
		// The job should be requeued 3 times before it finishes since the wait time is doubled each time the job is requeued (i.e., initially it waits
		// for 1 minutes before requeuing, then 2 minutes, and then 4 minutes). Since the init containers take 3 minutes
		// and 20 seconds to finish, a 4 minute wait should be long enough to finish the job successfully
		aw := createJobAWWithInitContainer(context, "aw-job-3-init-container", 60, "exponential", 0)
		appwrappers = append(appwrappers, aw)

		err := waitAWPodsCompleted(context, aw, 720*time.Second) // This test waits for 10 minutes to make sure all PODs complete
		Expect(err).NotTo(HaveOccurred())
	})

	It("MCAD CPU Requeuing - Deletion After Maximum Requeuing Times Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD CPU Requeuing Test - Started.\n")

		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		// Create a job with init containers that need 200 seconds to be ready before the container starts.
		// The requeuing mechanism is set to fire after 1 second (plus the 60 seconds time interval of the background thread)
		// Within 5 minutes, the AppWrapper will be requeued up to 3 times at which point it will be deleted
		aw := createJobAWWithInitContainer(context, "aw-job-3-init-container", 1, "none", 3)
		appwrappers = append(appwrappers, aw)

		err := waitAWPodsCompleted(context, aw, 300*time.Second)
		Expect(err).To(HaveOccurred())
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

	It("Create AppWrapper  - Check failed pod status", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Create AppWrapper  - Check failed pod status - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createPodCheckFailedStatusAW(context, "aw-checkfailedstatus-1")
		appwrappers = append(appwrappers, aw)

		err := waitAWPodsReady(context, aw)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(2 * time.Minute)
		aw1, err := context.karclient.ArbV1().AppWrappers(aw.Namespace).Get(aw.Name, metav1.GetOptions{})
		if err != nil {
			fmt.Fprint(GinkgoWriter, "Error getting status")
		}
		pass := false
		fmt.Fprintf(GinkgoWriter, "[e2e] status of AW %v.\n", aw1.Status.State)
		if len(aw1.Status.PendingPodConditions) == 0 {
			pass = true
		}
		Expect(pass).To(BeTrue())
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

	It("Create AppWrapper  - Bad Generic Item Only", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Create AppWrapper  - Bad Generic Item Only - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createBadGenericItemAW(context, "aw-bad-generic-item-1")
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

	It("MCAD Custom Pod Resources Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD Custom Pod Resources Test - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		// This should fit on cluster with customPodResources matching deployment resource demands so AW pods are created
		aw := createGenericDeploymentCustomPodResourcesWithCPUAW(
			context, "aw-deployment-2-550-vs-550-cpu", "550m", "550m", 2, 60)

		appwrappers = append(appwrappers, aw)

		err := waitAWAnyPodsExists(context, aw)
		Expect(err).NotTo(HaveOccurred())

		err = waitAWPodsReady(context, aw)
		Expect(err).NotTo(HaveOccurred())
	})

	It("MCAD Scheduling Fail Fast Preemption Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD Scheduling Fail Fast Preemption Test - Started.\n")

		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		// This should fill up the worker node and most of the master node
		aw := createDeploymentAWwith550CPU(context, appendRandomString("aw-deployment-2-550cpu"))
		appwrappers = append(appwrappers, aw)
		time.Sleep(1 * time.Minute)
		err := waitAWPodsReady(context, aw)
		Expect(err).NotTo(HaveOccurred())

		// This should not fit on any node but should dispatch because there is enough aggregated resources.
		aw2 := createGenericDeploymentCustomPodResourcesWithCPUAW(
			context, appendRandomString("aw-ff-deployment-1-850-cpu"), "850m", "850m", 1, 60)

		appwrappers = append(appwrappers, aw2)

		err = waitAWAnyPodsExists(context, aw2)
		fmt.Fprintf(os.Stdout, "The error is %v", err)
		Expect(err).NotTo(HaveOccurred())

		err = waitAWPodsPending(context, aw2)
		Expect(err).NotTo(HaveOccurred())

		// This should fit on cluster after AW aw-deployment-1-700-cpu above is automatically preempted on
		// scheduling failure
		aw3 := createGenericDeploymentCustomPodResourcesWithCPUAW(
			context, appendRandomString("aw-ff-deployment-2-340-cpu"), "340m", "340m", 2, 60)

		appwrappers = append(appwrappers, aw3)

		// Wait for pods to get created, assumes preemption around 10 minutes
		err = waitAWPodsExists(context, aw3, 720000*time.Millisecond)
		Expect(err).NotTo(HaveOccurred())

		// Make sure they are running
		err = waitAWPodsReady(context, aw3)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(2 * time.Minute)
		// Make sure pods from AW aw-deployment-1-700-cpu above do not exist proving preemption
		err = waitAWAnyPodsExists(context, aw2)
		Expect(err).To(HaveOccurred())
	})

	It("MCAD Bad Custom Pod Resources vs. Deployment Pod Resource Not Queuing Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD Bad Custom Pod Resources vs. Deployment Pod Resource Not Queuing Test - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		// This should fill up the worker node and most of the master node
		aw := createDeploymentAWwith550CPU(context, appendRandomString("aw-deployment-2-550cpu"))
		appwrappers = append(appwrappers, aw)

		err := waitAWPodsReady(context, aw)
		Expect(err).NotTo(HaveOccurred())

		// This should not fit on cluster but customPodResources is incorrect so AW pods are created
		aw2 := createGenericDeploymentCustomPodResourcesWithCPUAW(
			context, appendRandomString("aw-deployment-2-425-vs-426-cpu"), "425m", "426m", 2, 60)

		appwrappers = append(appwrappers, aw2)

		err = waitAWAnyPodsExists(context, aw2)
		fmt.Fprintf(os.Stdout, "the error is %v", err)
		Expect(err).NotTo(HaveOccurred())

		err = waitAWPodsReady(context, aw2)
		Expect(err).To(HaveOccurred())
	})

	It("MCAD Bad Custom Pod Resources vs. Deployment Pod Resource Queuing Test 2", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD Bad Custom Pod Resources vs. Deployment Pod Resource Queuing Test 2 - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		// This should fill up the worker node and most of the master node
		aw := createDeploymentAWwith550CPU(context, appendRandomString("aw-deployment-2-550cpu"))
		appwrappers = append(appwrappers, aw)
		time.Sleep(1 * time.Minute)

		err := waitAWPodsReady(context, aw)
		Expect(err).NotTo(HaveOccurred())

		// This should fit on cluster but customPodResources is incorrect so AW pods are not created
		aw2 := createGenericDeploymentCustomPodResourcesWithCPUAW(
			context, appendRandomString("aw-deployment-2-426-vs-425-cpu"), "426m", "425m", 2, 60)

		appwrappers = append(appwrappers, aw2)
		time.Sleep(1 * time.Minute)
		err = waitAWAnyPodsExists(context, aw2)
		Expect(err).To(HaveOccurred())

	})

	It("MCAD appwrapper timeout Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD appwrapper timeout Test - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createGenericAWTimeoutWithStatus(context, "aw-test-jobtimeout-with-comp-1")
		err1 := waitAWPodsReady(context, aw)
		Expect(err1).NotTo(HaveOccurred())
		time.Sleep(60 * time.Second)
		aw1, err := context.karclient.ArbV1().AppWrappers(aw.Namespace).Get(aw.Name, metav1.GetOptions{})
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "Error getting status")
		}
		pass := false
		fmt.Fprintf(GinkgoWriter, "[e2e] status of AW %v.\n", aw1.Status.State)
		if aw1.Status.State == arbv1.AppWrapperStateFailed {
			pass = true
		}
		Expect(pass).To(BeTrue())
		appwrappers = append(appwrappers, aw)
		fmt.Fprintf(os.Stdout, "[e2e] MCAD appwrapper timeout Test - Completed.\n")
	})

	It("MCAD Job Completion Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD Job Completion Test - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createGenericJobAWWithStatus(context, "aw-test-job-with-comp-1")
		err1 := waitAWPodsReady(context, aw)
		Expect(err1).NotTo(HaveOccurred())
		time.Sleep(1 * time.Minute)
		aw1, err := context.karclient.ArbV1().AppWrappers(aw.Namespace).Get(aw.Name, metav1.GetOptions{})
		if err != nil {
			fmt.Fprint(GinkgoWriter, "Error getting status")
		}
		pass := false
		fmt.Fprintf(GinkgoWriter, "[e2e] status of AW %v.\n", aw1.Status.State)
		if aw1.Status.State == arbv1.AppWrapperStateCompleted {
			pass = true
		}
		Expect(pass).To(BeTrue())
		appwrappers = append(appwrappers, aw)
		fmt.Fprintf(os.Stdout, "[e2e] MCAD Job Completion Test - Completed.\n")

	})

	It("MCAD Multi-Item Job Completion Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD Multi-Item Job Completion Test - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createGenericJobAWWithMultipleStatus(context, "aw-test-job-with-comp-ms-21")
		err1 := waitAWPodsReady(context, aw)
		Expect(err1).NotTo(HaveOccurred())
		time.Sleep(1 * time.Minute)
		aw1, err := context.karclient.ArbV1().AppWrappers(aw.Namespace).Get(aw.Name, metav1.GetOptions{})
		if err != nil {
			fmt.Fprint(GinkgoWriter, "Error getting status")
		}
		pass := false
		fmt.Fprintf(GinkgoWriter, "[e2e] status of AW %v.\n", aw1.Status.State)
		if aw1.Status.State == arbv1.AppWrapperStateCompleted {
			pass = true
		}
		Expect(pass).To(BeTrue())
		appwrappers = append(appwrappers, aw)
		fmt.Fprintf(os.Stdout, "[e2e] MCAD Job Completion Test - Completed.\n")

	})

	It("MCAD GenericItem Without Status Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD GenericItem Without Status Test - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createAWGenericItemWithoutStatus(context, "aw-test-job-with-comp-44")
		err1 := waitAWPodsReady(context, aw)
		fmt.Fprintf(os.Stdout, "The error is: %v", err1)
		Expect(err1).To(HaveOccurred())
		fmt.Fprintf(os.Stdout, "[e2e] MCAD GenericItem Without Status Test - Completed.\n")

	})

	It("MCAD Job Completion No-requeue Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD Job Completion No-requeue Test - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createGenericJobAWWithScheduleSpec(context, "aw-test-job-with-scheduling-spec")
		err1 := waitAWPodsReady(context, aw)
		Expect(err1).NotTo(HaveOccurred())
		err2 := waitAWPodsCompleted(context, aw, 90*time.Second)
		Expect(err2).NotTo(HaveOccurred())

		// Once pods are completed, we wait for them to see if they change their status to anything BUT "Completed"
		// which SHOULD NOT happen because the job is done
		err3 := waitAWPodsNotCompleted(context, aw)
		Expect(err3).To(HaveOccurred())

		appwrappers = append(appwrappers, aw)
		fmt.Fprintf(os.Stdout, "[e2e] MCAD Job Completion No-requeue Test - Completed.\n")

	})

	It("MCAD Job Large Compute Requirement Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD Job Large Compute Requirement Test - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createGenericJobAWtWithLargeCompute(context, "aw-test-job-with-large-comp-1")
		err1 := waitAWPodsReady(context, aw)
		Expect(err1).NotTo(HaveOccurred())
		time.Sleep(1 * time.Minute)
		aw1, err := context.karclient.ArbV1().AppWrappers(aw.Namespace).Get(aw.Name, metav1.GetOptions{})
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "Error getting status, %v", err)
		}
		pass := false
		fmt.Fprintf(GinkgoWriter, "[e2e] status of AW %v.\n", aw1.Status.State)
		if aw1.Status.State == arbv1.AppWrapperStateEnqueued {
			pass = true
		}
		Expect(pass).To(BeTrue())
		appwrappers = append(appwrappers, aw)
		fmt.Fprintf(os.Stdout, "[e2e] MCAD Job Large Compute Requirement Test - Completed.\n")

	})

	It("MCAD CPU Accounting Queuing Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD CPU Accounting Queuing Test - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		// This should fill up the worker node and most of the master node
		aw := createDeploymentAWwith550CPU(context, appendRandomString("aw-deployment-2-550cpu"))
		appwrappers = append(appwrappers, aw)
		time.Sleep(1 * time.Minute)
		err := waitAWPodsReady(context, aw)
		Expect(err).NotTo(HaveOccurred())

		// This should not fit on cluster
		aw2 := createDeploymentAWwith426CPU(context, appendRandomString("aw-deployment-2-426cpu"))
		appwrappers = append(appwrappers, aw2)

		err = waitAWAnyPodsExists(context, aw2)
		Expect(err).To(HaveOccurred())
	})

	It("MCAD Deployment RuningHoldCompletion Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD Deployment RuningHoldCompletion Test - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createGenericDeploymentAWWithMultipleItems(context, appendRandomString("aw-deployment-2-status"))
		time.Sleep(1 * time.Minute)
		err1 := waitAWPodsReady(context, aw)
		Expect(err1).NotTo(HaveOccurred())
		aw1, err := context.karclient.ArbV1().AppWrappers(aw.Namespace).Get(aw.Name, metav1.GetOptions{})
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "Error getting status, %v", err)
		}
		pass := false
		fmt.Fprintf(GinkgoWriter, "[e2e] status of AW %v.\n", aw1.Status.State)
		if aw1.Status.State == arbv1.AppWrapperStateRunningHoldCompletion {
			pass = true
		}
		Expect(pass).To(BeTrue())
		appwrappers = append(appwrappers, aw)
		fmt.Fprintf(os.Stdout, "[e2e] MCAD Deployment RuningHoldCompletion Test - Completed.\n")

	})

	It("MCAD Service no RuningHoldCompletion or Complete Test", func() {
		fmt.Fprintf(os.Stdout, "[e2e] MCAD Service no RuningHoldCompletion or Complete Test - Started.\n")
		context := initTestContext()
		var appwrappers []*arbv1.AppWrapper
		appwrappersPtr := &appwrappers
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		aw := createGenericServiceAWWithNoStatus(context, appendRandomString("aw-deployment-2-status"))
		time.Sleep(1 * time.Minute)
		err1 := waitAWPodsReady(context, aw)
		Expect(err1).NotTo(HaveOccurred())
		aw1, err := context.karclient.ArbV1().AppWrappers(aw.Namespace).Get(aw.Name, metav1.GetOptions{})
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "Error getting status, %v", err)
		}
		pass := false
		fmt.Fprintf(GinkgoWriter, "[e2e] status of AW %v.\n", aw1.Status.State)
		if aw1.Status.State == arbv1.AppWrapperStateActive {
			pass = true
		}
		Expect(pass).To(BeTrue())
		appwrappers = append(appwrappers, aw)
		fmt.Fprintf(os.Stdout, "[e2e] MCAD Service no RuningHoldCompletion or Complete Test - Completed.\n")

	})

	It("Create AppWrapper - Generic 50 Deployment Only - 2 pods each", func() {
		fmt.Fprintf(os.Stdout, "[e2e] Generic 50 Deployment Only - 2 pods each - Started.\n")

		context := initTestContext()
		var aws []*arbv1.AppWrapper
		appwrappersPtr := &aws
		defer cleanupTestObjectsPtr(context, appwrappersPtr)

		const (
			awCount           = 50
			reportingInterval = 10
			cpuDemand         = "5m"
		)

		replicas := 2
		modDivisor := int(awCount / reportingInterval)
		for i := 0; i < awCount; i++ {
			name := fmt.Sprintf("aw-generic-deployment-%02d-%03d", replicas, i+1)

			if ((i+1)%modDivisor) == 0 || i == 0 {
				fmt.Fprintf(GinkgoWriter, "[e2e] Creating AW %s with %s cpu and %d replica(s).\n", name, cpuDemand, replicas)
			}
			aw := createGenericDeploymentWithCPUAW(context, name, cpuDemand, replicas)
			aws = append(aws, aw)
		}

		// Give the deployments time to create pods
		// FIXME: do not assume that the pods are in running state in the order of submission.
		time.Sleep(2 * time.Minute)
		for i := 0; i < len(aws); i++ {
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
