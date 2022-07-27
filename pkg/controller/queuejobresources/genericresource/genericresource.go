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
package genericresource

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	arbv1 "github.com/IBM/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	clusterstateapi "github.com/IBM/multi-cluster-app-dispatcher/pkg/controller/clusterstate/api"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

var appwrapperJobName = "appwrapper.mcad.ibm.com"
var resourceName = "resourceName"
var appWrapperKind = arbv1.SchemeGroupVersion.WithKind("AppWrapper")

type GenericResources struct {
	clients          *kubernetes.Clientset
	kubeClientConfig *rest.Config
	arbclients       *clientset.Clientset
}

func NewAppWrapperGenericResource(config *rest.Config) *GenericResources {
	return &GenericResources{
		clients:          kubernetes.NewForConfigOrDie(config),
		kubeClientConfig: config,
		arbclients:       clientset.NewForConfigOrDie(config),
	}
}

func join(strs ...string) string {
	var result string
	if strs[0] == "" {
		return strs[len(strs)-1]
	}
	for _, str := range strs {
		result += str
	}
	return result
}

func (gr *GenericResources) SyncQueueJob(aw *arbv1.AppWrapper, awr *arbv1.AppWrapperGenericResource) (podList []*v1.Pod, err error) {
	startTime := time.Now()
	defer func() {
		klog.V(4).Infof("Finished syncing AppWrapper job resource %s (%v)", aw.Name, time.Now().Sub(startTime))
		// klog.V(4).Infof("Finished syncing AppWrapper job resource %q (%v)", awobRes.Template, time.Now().Sub(startTime))
	}()

	namespaced := true
	//todo:DELETEME	dd := common.KubeClient.Discovery()
	dd := gr.clients.Discovery()
	apigroups, err := restmapper.GetAPIGroupResources(dd)
	if err != nil {
		klog.Errorf("Error getting API resources, err=%#v", err)
		return []*v1.Pod{}, err
	}
	ext := awr.GenericTemplate
	restmapper := restmapper.NewDiscoveryRESTMapper(apigroups)
	//versions := &unstructured.Unstructured{}
	//_, gvk, err := unstructured.UnstructuredJSONScheme.Decode(ext.Raw, nil, versions)
	_, gvk, err := unstructured.UnstructuredJSONScheme.Decode(ext.Raw, nil, nil)
	if err != nil {
		klog.Errorf("Decoding error, please check your CR! Aborting handling the resource creation, err:  `%v`", err)
		return []*v1.Pod{}, err
	}
	mapping, err := restmapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		klog.Errorf("mapping error from raw object: `%v`", err)
		return []*v1.Pod{}, err
	}

	//todo:DELETEME		restconfig := common.KubeConfig
	restconfig := gr.kubeClientConfig
	restconfig.GroupVersion = &schema.GroupVersion{
		Group:   mapping.GroupVersionKind.Group,
		Version: mapping.GroupVersionKind.Version,
	}
	dclient, err := dynamic.NewForConfig(restconfig)
	if err != nil {
		klog.Errorf("Error creating new dynamic client, err=%#v", err)
		return []*v1.Pod{}, err
	}

	_, apiresourcelist, err := dd.ServerGroupsAndResources()
	if err != nil {
		klog.Errorf("Error getting supported groups and resources, err=%#v", err)
		return []*v1.Pod{}, err
	}

	rsrc := mapping.Resource
	for _, apiresourcegroup := range apiresourcelist {
		if apiresourcegroup.GroupVersion == join(mapping.GroupVersionKind.Group, "/", mapping.GroupVersionKind.Version) {
			for _, apiresource := range apiresourcegroup.APIResources {
				if apiresource.Name == mapping.Resource.Resource && apiresource.Kind == mapping.GroupVersionKind.Kind {
					rsrc = mapping.Resource
					namespaced = apiresource.Namespaced
				}
			}
		}
	}
	var unstruct unstructured.Unstructured
	unstruct.Object = make(map[string]interface{})
	var blob interface{}
	if err = json.Unmarshal(ext.Raw, &blob); err != nil {
		klog.Errorf("Error unmarshalling, err=%#v", err)
		return []*v1.Pod{}, err
	}
	ownerRef := metav1.NewControllerRef(aw, appWrapperKind)
	unstruct.Object = blob.(map[string]interface{}) //set object to the content of the blob after Unmarshalling
	unstruct.SetOwnerReferences(append(unstruct.GetOwnerReferences(), *ownerRef))
	namespace := "default"
	name := ""
	if md, ok := unstruct.Object["metadata"]; ok {

		metadata := md.(map[string]interface{})
		if objectName, ok := metadata["name"]; ok {
			name = objectName.(string)
		}
		if objectns, ok := metadata["namespace"]; ok {
			namespace = objectns.(string)
		}
	}
	labels := map[string]string{}
	if unstruct.GetLabels() == nil {
		unstruct.SetLabels(labels)
	} else {
		labels = unstruct.GetLabels()
	}
	labels[appwrapperJobName] = aw.Name
	labels[resourceName] = unstruct.GetName()
	unstruct.SetLabels(labels)

	// Add labels to pod templete if one exists.
	podTemplateFound := addLabelsToPodTemplateField(&unstruct, labels)
	if !podTemplateFound {
		klog.V(4).Infof("[SyncQueueJob] No pod template spec exists for resource: %s to add labels.", name)
	}

	// Get the resource  to see if it exists
	labelSelector := fmt.Sprintf("%s=%s, %s=%s", appwrapperJobName, aw.Name, resourceName, unstruct.GetName())
	inEtcd, err := dclient.Resource(rsrc).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return []*v1.Pod{}, err
	}

	// Check to see if object already exists in etcd, if not, create the object.
	if inEtcd == nil || len(inEtcd.Items) < 1 {
		newName := name
		if len(newName) > 63 {
			newName = newName[:63]
		}
		unstruct.SetName(newName)
		err = createObject(namespaced, namespace, newName, rsrc, unstruct, dclient)
		if err != nil {
			if errors.IsAlreadyExists(err) {
				klog.V(4).Infof("%v\n", err.Error())
			} else {
				klog.Errorf("Error creating the object `%v`, the error is `%v`", newName, errors.ReasonForError(err))
				return []*v1.Pod{}, err
			}
		}
	}

	// Get the related resources of created object
	var thisObj *unstructured.Unstructured
	var err1 error
	if namespaced {
		thisObj, err1 = dclient.Resource(rsrc).Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
	} else {
		thisObj, err1 = dclient.Resource(rsrc).Get(context.Background(), name, metav1.GetOptions{})
	}
	if err1 != nil {
		klog.Errorf("Could not get created resource with error %v", err)
	}
	thisOwnerRef := metav1.NewControllerRef(thisObj, thisObj.GroupVersionKind())

	podL, _ := gr.clients.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	pods := []*v1.Pod{}
	for _, pod := range (*podL).Items {
		parent := metav1.GetControllerOf(&pod)
		if reflect.DeepEqual(thisOwnerRef, parent) {
			pods = append(pods, &pod)
		}
		klog.V(10).Infof("[SyncQueueJob] pod %s created from a Generic Item\n", pod.Name)
	}
	return pods, nil
}

//checks if object has pod template spec and add new labels
func addLabelsToPodTemplateField(unstruct *unstructured.Unstructured, labels map[string]string) (hasFields bool) {
	spec, isFound, _ := unstructured.NestedMap(unstruct.UnstructuredContent(), "spec")
	if !isFound {
		klog.V(10).Infof("[addLabelsToPodTemplateField] 'spec' field not found.")
		return false
	}
	template, isFound, _ := unstructured.NestedMap(spec, "template")
	if !isFound {
		klog.V(10).Infof("[addLabelsToPodTemplateField] 'spec.template' field not found.")
		return false
	}

	marshal, _ := json.Marshal(template)
	unmarshal := v1.PodTemplateSpec{}
	if err := json.Unmarshal(marshal, &unmarshal); err != nil {
		klog.Warning(err)
		return false
	}
	existingLabels, isFound, _ := unstructured.NestedStringMap(template, "metadata", "labels")
	if !isFound {
		klog.V(10).Infof("[addLabelsToPodTemplateField] 'spec.template.metadata.labels' field not found.")
		return false
	}
	newLength := len(existingLabels) + len(labels)
	m := make(map[string]string, newLength) // convert map[string]string into map[string]interface{}
	for k, v := range existingLabels {
		m[k] = v
	}

	for k, v := range labels {
		m[k] = v
	}

	if err := unstructured.SetNestedStringMap(unstruct.Object, m, "spec", "template", "metadata", "labels"); err != nil {
		klog.Warning(err)
		return false
	}

	return isFound
}

//checks if object has replicas and containers field
func hasFields(obj runtime.RawExtension) (hasFields bool, replica float64, containers []v1.Container) {
	var unstruct unstructured.Unstructured
	unstruct.Object = make(map[string]interface{})
	var blob interface{}
	if err := json.Unmarshal(obj.Raw, &blob); err != nil {
		klog.Errorf("Error unmarshalling, err=%#v", err)
		return false, 0, nil
	}
	unstruct.Object = blob.(map[string]interface{})
	spec, isFound, _ := unstructured.NestedMap(unstruct.UnstructuredContent(), "spec")
	if !isFound {
		klog.Warningf("[hasFields] No spec field found in raw object: %#v", unstruct.UnstructuredContent())
	}

	replicas, isFound, _ := unstructured.NestedFloat64(spec, "replicas")
	// Set default to 1 if no replicas field is found (handles the case of a single pod creation without replicaset.
	if !isFound {
		replicas = 1
	}

	template, isFound, _ := unstructured.NestedMap(spec, "template")
	// If spec does not contain a podtemplate, check for pod singletons
	var subspec map[string]interface{}
	if !isFound {
		subspec = spec
		klog.V(6).Infof("[hasFields] No template field found in raw object: %#v", spec)
	} else {
		subspec, isFound, _ = unstructured.NestedMap(template, "spec")
	}

	containerList, isFound, _ := unstructured.NestedSlice(subspec, "containers")
	if !isFound {
		klog.Warningf("[hasFields] No containers field found in raw object: %#v", subspec)
		return false, 0, nil
	}
	objContainers := make([]v1.Container, len(containerList))
	for _, container := range containerList {
		marshal, _ := json.Marshal(container)
		unmarshal := v1.Container{}
		_ = json.Unmarshal(marshal, &unmarshal)
		objContainers = append(objContainers, unmarshal)
	}
	return isFound, replicas, objContainers
}

func createObject(namespaced bool, namespace string, name string, rsrc schema.GroupVersionResource, unstruct unstructured.Unstructured, dclient dynamic.Interface) (erro error) {
	var err error
	if !namespaced {
		res := dclient.Resource(rsrc)
		_, err = res.Create(context.Background(), &unstruct, metav1.CreateOptions{})
		if err != nil {
			if errors.IsAlreadyExists(err) {
				klog.Errorf("%v\n", err.Error())
				return nil
			} else {
				klog.Errorf("Error creating the object `%v`, the error is `%v`", name, errors.ReasonForError(err))
				return err
			}
		} else {
			klog.V(4).Infof("Resource `%v` created\n", name)
			return nil
		}
	} else {
		res := dclient.Resource(rsrc).Namespace(namespace)
		_, err = res.Create(context.Background(), &unstruct, metav1.CreateOptions{})
		if err != nil {
			if errors.IsAlreadyExists(err) {
				klog.Errorf("%v\n", err.Error())
				return nil
			} else {
				klog.Errorf("Error creating the object `%v`, the error is `%v`", name, errors.ReasonForError(err))
				return err
			}
		} else {
			klog.V(4).Infof("Resource `%v` created\n", name)
			return nil

		}
	}
}

func GetListOfPodResourcesFromOneGenericItem(awr *arbv1.AppWrapperGenericResource) (resource []*clusterstateapi.Resource, er error) {
	var podResourcesList []*clusterstateapi.Resource

	podTotalresource := clusterstateapi.EmptyResource()
	var err error
	err = nil
	if awr.GenericTemplate.Raw != nil {
		hasContainer, replicas, containers := hasFields(awr.GenericTemplate)
		if hasContainer {
			// Add up all the containers in a pod
			for _, container := range containers {
				res := getContainerResources(container, 1)
				podTotalresource = podTotalresource.Add(res)
			}
			klog.V(8).Infof("[GetListOfPodResourcesFromOneGenericItem] Requested total pod allocation resource from containers `%v`.\n", podTotalresource)
		} else {
			podresources := awr.CustomPodResources
			for _, item := range podresources {
				res := getPodResources(item)
				podTotalresource = podTotalresource.Add(res)
			}
			klog.V(8).Infof("[GetListOfPodResourcesFromOneGenericItem] Requested total allocation resource from 1 pod `%v`.\n", podTotalresource)
		}

		// Addd individual pods to results
		var replicaCount int = int(replicas)
		for i := 0; i < replicaCount; i++ {
			podResourcesList = append(podResourcesList, podTotalresource)
		}
	}

	return podResourcesList, err
}

func GetResources(awr *arbv1.AppWrapperGenericResource) (resource *clusterstateapi.Resource, er error) {

	totalresource := clusterstateapi.EmptyResource()
	var err error
	err = nil
	if awr.GenericTemplate.Raw != nil {
		hasContainer, replicas, containers := hasFields(awr.GenericTemplate)
		if hasContainer {
			for _, item := range containers {
				res := getContainerResources(item, replicas)
				totalresource = totalresource.Add(res)
			}
			klog.V(8).Infof("[GetResources] Requested total allocation resource from containers `%v`.\n", totalresource)
		} else {
			podresources := awr.CustomPodResources
			for _, item := range podresources {
				res := getPodResources(item)
				totalresource = totalresource.Add(res)
			}
			klog.V(8).Infof("[GetResources] Requested total allocation resource from pods `%v`.\n", totalresource)
		}
	} else {
		err = fmt.Errorf("generic template raw object is not defined (nil)")
	}

	return totalresource, err
}

func getPodResources(pod arbv1.CustomPodResourceTemplate) (resource *clusterstateapi.Resource) {
	replicas := pod.Replicas
	req := clusterstateapi.NewResource(pod.Requests)
	limit := clusterstateapi.NewResource(pod.Limits)
	tolerance := 0.001

	// Use limit if request is 0
	if diff := math.Abs(req.MilliCPU - float64(0.0)); diff < tolerance {
		req.MilliCPU = limit.MilliCPU
	}

	if diff := math.Abs(req.Memory - float64(0.0)); diff < tolerance {
		req.Memory = limit.Memory
	}

	if req.GPU <= 0 {
		req.GPU = limit.GPU
	}
	req.MilliCPU = req.MilliCPU * float64(replicas)
	req.Memory = req.Memory * float64(replicas)
	req.GPU = req.GPU * int64(replicas)
	return req
}

func getContainerResources(container v1.Container, replicas float64) *clusterstateapi.Resource {
	req := clusterstateapi.NewResource(container.Resources.Requests)
	limit := clusterstateapi.NewResource(container.Resources.Limits)

	tolerance := 0.001

	// Use limit if request is 0
	if diff := math.Abs(req.MilliCPU - float64(0.0)); diff < tolerance {
		req.MilliCPU = limit.MilliCPU
	}

	if diff := math.Abs(req.Memory - float64(0.0)); diff < tolerance {
		req.Memory = limit.Memory
	}

	if req.GPU <= 0 {
		req.GPU = limit.GPU
	}

	req.MilliCPU = req.MilliCPU * float64(replicas)
	req.Memory = req.Memory * float64(replicas)
	req.GPU = req.GPU * int64(replicas)
	return req
}

func GetGenericItemKind(aw *arbv1.AppWrapperGenericResource) (kindstring string) {
	ext := aw.GenericTemplate
	_, gvk, err := unstructured.UnstructuredJSONScheme.Decode(ext.Raw, nil, nil)
	if err != nil {
		klog.Errorf("Decoding error, please check your CR! Aborting handling the resource creation, err:  `%v`", err)
	}
	return gvk.GroupKind().Kind
}

//aw *arbv1.AppWrapperGenericResource, kindstring string, namespace string
func (gr *GenericResources) GetGenericItemKindStatus(aw *arbv1.AppWrapperGenericResource, namespace string) (completed string) {
	dd := gr.clients.Discovery()
	apigroups, err := restmapper.GetAPIGroupResources(dd)
	if err != nil {
		klog.Errorf("Error getting API resources, err=%#v", err)
	}
	restmapper := restmapper.NewDiscoveryRESTMapper(apigroups)
	//versions := &unstructured.Unstructured{}
	//_, gvk, err := unstructured.UnstructuredJSONScheme.Decode(ext.Raw, nil, versions)
	_, gvk, err := unstructured.UnstructuredJSONScheme.Decode(aw.GenericTemplate.Raw, nil, nil)
	if err != nil {
		klog.Errorf("Decoding error, please check your CR! Aborting handling the resource creation, err:  `%v`", err)
	}

	mapping, err := restmapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		klog.Errorf("mapping error from raw object: `%v`", err)
	}
	restconfig := gr.kubeClientConfig
	restconfig.GroupVersion = &schema.GroupVersion{
		Group:   mapping.GroupVersionKind.Group,
		Version: mapping.GroupVersionKind.Version,
	}
	rsrc := mapping.Resource
	dclient, err := dynamic.NewForConfig(restconfig)
	if err != nil {
		klog.Errorf("Error creating new dynamic client, err=%#v", err)
	}

	inEtcd, err := dclient.Resource(rsrc).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		klog.Errorf("Error listing object: ", err)
	}
	//klog.Infof("The Items in etcd are: %v", inEtcd.Items)

	// status, isStatusPresent, errStatus := unstructured.NestedMap(inEtcd.UnstructuredContent(), "status")
	// klog.Infof("isStatusPresent, %v", isStatusPresent)
	// if errStatus != nil {
	// 	klog.Infof("the error to condition map is %v", errStatus)
	// }
	// conditions, isPresent, error := unstructured.NestedMap(status, "conditions")
	// klog.Infof("converted, %v", isPresent)
	// if error != nil {
	// 	klog.Infof("the error to condition map is %v", error)
	// }
	// for k, v := range conditions {
	// 	klog.Infof("The key is %v", k)
	// 	if k == "type" {
	// 		klog.Infof("The completion status is %v", v)
	// 	}
	// }
	for _, job := range inEtcd.Items {
		completionRequiredBlock := aw.CompletionStatus
		// if completionRequiredBlock != nil {
		// 	completionRequiredFlag :=completionRequiredBlock[0].CompletionRequired
		// }
		// var completionRequiredFlag bool
		// for item := range completionRequired {
		// 	if string(item) == "completionrequired" {
		// 		completionRequiredFlag = true
		// 	}
		// 	if string(item) == "statuspath" {
		// 		klog.Infof("The status path supplied by user is %v", item)
		// 	}
		// }
		//klog.Infof("The completion required flag is %v", completionRequiredFlag)
		//completionRequiredFlag && kindstring == "Job"
		if completionRequiredBlock != nil {
			if completionRequiredBlock[0].CompletionRequired {
				//completions := job.Object["spec"].(map[string]interface{})["completions"]
				//succeeded := job.Object["status"].(map[string]interface{})["succeeded"]
				// status := job.Object["status"].(map[string]interface{})
				// klog.Infof("The status is: %v", status)
				statusPath := strings.Split(aw.CompletionStatus[0].StatusPath, ".")
				//klog.Infof("the splitted string path is %v", statusPath)
				conditions := job.Object[statusPath[0]].(map[string]interface{})[statusPath[1]].([]interface{})
				//TODO: check all conditions where multiple conditions exists
				klog.V(3).Infof("The conditions array len is : %v", len(conditions))
				for _, item := range conditions {
					//klog.V(3).Infof("The item  is : %v", item)
					//completionType = fmt.Sprint(item.(map[string]interface{})[statusPath[2]])
					completionType := fmt.Sprint(item.(map[string]interface{})[statusPath[2]])
					//Move this to utils package?
					userSpecfiedCompletionConditions := aw.CompletionStatus[0].CompletionConditions
					//klog.V(3).Infof("The user specified conditions are : %v", userSpecfiedCompletionConditions)
					for _, condition := range userSpecfiedCompletionConditions {
						//klog.V(3).Infof("Checking for condition : %v", condition)
						if strings.Contains(strings.ToLower(completionType), strings.ToLower(condition)) {
							//klog.V(3).Infof("The completionType is: %v", completionType)
							return "true"
						}
					}
				}

			}
		} else { //user did not supply path look status in default location
			// conditions := job.Object["status"].(map[string]interface{})["conditions"].([]interface{})
			// for _, item := range conditions {
			// 	//completionType = fmt.Sprint(item.(map[string]interface{})[statusPath[2]])
			// 	completionType := fmt.Sprint(item.(map[string]interface{})["type"])
			// 	if completionType == "Complete" {
			// 		klog.Infof("The completionType in else is: %v", completionType)
			// 		return "completed"
			// 	}
			// }
		}
		// klog.Infof(
		// 	"completions: %v\n",
		// 	job.Object["spec"].(map[string]interface{})["completions"],
		// )
		// klog.Infof(
		// 	"succeeded: %v\n",
		// 	job.Object["status"].(map[string]interface{})["succeeded"],
		// )
	}
	return ""
}
