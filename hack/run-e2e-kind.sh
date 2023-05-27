#!/bin/bash
# Copyright 2014 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# Copyright 2019, 2021, 2022, 2023 The Multi-Cluster App Dispatcher Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

export ROOT_DIR="$(dirname "$(dirname "$(readlink -fn "$0")")")"
export LOG_LEVEL=${TEST_LOG_LEVEL:-2}
export CLEANUP_CLUSTER=${CLEANUP_CLUSTER:-"true"}
export CLUSTER_CONTEXT="--name test"
# Using older image due to older version of kubernetes cluster"
export IMAGE_ECHOSERVER="kicbase/echo-server:1.0"
export IMAGE_UBUNTU_LATEST="ubuntu:latest"
export KIND_OPT=${KIND_OPT:=" --config ${ROOT_DIR}/hack/e2e-kind-config.yaml"}
export KA_BIN=_output/bin
export WAIT_TIME="20s"
export IMAGE_REPOSITORY_MCAD="${1}"
export IMAGE_TAG_MCAD="${2}"
export MCAD_IMAGE_PULL_POLICY="${3-Always}"
export IMAGE_MCAD="${IMAGE_REPOSITORY_MCAD}:${IMAGE_TAG_MCAD}"
CLUSTER_STARTED="false"
export KUTTL_VERSION=0.15.0
export KUTTL_TEST_OPT="--config ${ROOT_DIR}/kuttl-test.yaml"
# FOR DEBUGGING
#export KUTTL_TEST_OPT="--config ${ROOT_DIR}/kuttl-test.yaml --skip-delete"

function update_test_host {
  
  local arch="$(go env GOARCH)"
  if [ -z $arch ]
  then
    echo "Unable to determine downloads architecture"
    exit 1
  fi
  echo "CPU architecture for downloads is: ${arch}"

  #Only run this function if we are running on the travis build machinbe,
  if [ "$(lsb_release -c -s 2>&1 | grep xenial)" == "xenial" ]; then 
    sudo apt-get update && sudo apt-get install -y apt-transport-https curl 
    curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
    echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee -a /etc/apt/sources.list.d/kubernetes.list
    sudo apt-get update
  fi
  
  which kubectl >/dev/null 2>&1
  if [ $? -ne 0 ]
  then 
      sudo apt-get install -y --allow-unauthenticated kubectl
      [ $? -ne 0 ] && echo "Failed to install kubectl" && exit 1  
      echo "kubectl was sucessfully installed."    
  fi    
  
  which kind >/dev/null 2>&1
  if [ $? -ne 0 ] 
  then
    # Download kind binary (0.18.0)
    echo "Downloading and installing kind...."
    sudo curl -o /usr/local/bin/kind -L https://github.com/kubernetes-sigs/kind/releases/download/v0.18.0/kind-linux-${arch} && \
    sudo chmod +x /usr/local/bin/kind  
    [ $? -ne 0 ] && echo "Failed to download kind" && exit 1  
    echo "Kind was sucessfully installed."    
  fi

  which helm >/dev/null 2>&1
  if [ $? -ne 0 ]
  then 
    # Installing helm3
    echo "Downloading and installing helm..."
    curl -fsSL -o ${ROOT_DIR}/get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 && 
      chmod 700 ${ROOT_DIR}/get_helm.sh && ${ROOT_DIR}/get_helm.sh
    [ $? -ne 0 ] && echo "Failed to download and install helm" && exit 1
    echo "Helm was sucessfully installed."
    rm -rf ${ROOT_DIR}/get_helm.sh
  fi
  
  kubectl kuttl version >/dev/null 2>&1
  if [ $? -ne 0 ]
  then
    if [[ "$arch" == "amd64" ]]
    then 
      local kuttl_arch="x86_64"
    else
      local kuttl_arch=$arch  
    fi
    # Download kuttl plugin
    echo "Downloading and installing kuttl...."
    sudo curl -sSLf --output /tmp/kubectl-kuttl https://github.com/kudobuilder/kuttl/releases/download/v${KUTTL_VERSION}/kubectl-kuttl_${KUTTL_VERSION}_linux_${kuttl_arch} && \
    sudo mv /tmp/kubectl-kuttl /usr/local/bin && \
    sudo chmod a+x /usr/local/bin/kubectl-kuttl
    [ $? -ne 0 ] && echo "Failed to download and install helm" && exit 1
    echo "Kuttl was sucessfully installed."
  fi
}

# check if pre-requizites are installed.
function check-prerequisites {
  echo "checking prerequisites"
  which kind >/dev/null 2>&1
  if [ $? -ne 0 ] 
  then
    echo "kind not installed, exiting."
    exit 1
  else
    echo -n "found kind, version: " && kind version
  fi

  which kubectl >/dev/null 2>&1
  if [ $? -ne 0 ]
  then
    echo "kubectl not installed, exiting."
    exit 1
  else
    echo -n "found kubectl, " && kubectl version 
  fi
  kubectl kuttl version >/dev/null 2>&1
  if [ $? -ne 0 ]
  then
    echo "kuttl plugin for kubectl not installed, exiting."
    exit 1
  else
    echo -n "found kuttl plugin for kubectl, " && kubectl kuttl version
  fi
  
  if [[ $IMAGE_REPOSITORY_MCAD == "" ]]
  then
    echo "No MCAD image was provided."
    exit 1
  elif [[ $IMAGE_TAG_MCAD == "" ]]
  then
    echo "No MCAD image tag was provided for: ${IMAGE_REPOSITORY_MCAD}."
    exit 1
  else
    echo "end to end test with ${IMAGE_MCAD}."
  fi
  
  which helm >/dev/null 2>&1
  if [ $? -ne 0 ]
  then
    echo "helm not installed, exiting."
    exit 1
  else
    echo -n "found helm, " && helm version --short
  fi  

  
  which helm >/dev/null 2>&1
  if [ $? -ne 0 ]
  then
    echo "helm not installed, exiting."
    exit 1
  else
    echo -n "found helm, " && helm version --short
  fi  

}

function kind-up-cluster {
  echo "Running kind: [kind create cluster ${CLUSTER_CONTEXT} ${KIND_OPT}]"
  kind create cluster ${CLUSTER_CONTEXT} ${KIND_OPT} --wait ${WAIT_TIME}
  if [ $? -ne 0 ]
  then
    echo "Failed to start kind cluster"
    exit 1
  fi
  CLUSTER_STARTED="true"
  if [ $? -ne 0 ]
  then
    echo "Failed to start kind cluster"
    exit 1
  fi
  CLUSTER_STARTED="true"

  docker pull ${IMAGE_ECHOSERVER} 
  if [ $? -ne 0 ]
  then
    echo "Failed to pull ${IMAGE_ECHOSERVER}"
    exit 1
  fi

  docker pull ${IMAGE_UBUNTU_LATEST}
  if [ $? -ne 0 ]
  then
    echo "Failed to pull ${IMAGE_UBUNTU_LATEST}"
    exit 1
  fi

  docker pull ${IMAGE_ECHOSERVER} 
  if [ $? -ne 0 ]
  then
    echo "Failed to pull ${IMAGE_ECHOSERVER}"
    exit 1
  fi

  docker pull ${IMAGE_UBUNTU_LATEST}
  if [ $? -ne 0 ]
  then
    echo "Failed to pull ${IMAGE_UBUNTU_LATEST}"
    exit 1
  fi

  if [[ "$MCAD_IMAGE_PULL_POLICY" = "Always" ]]
  then
    docker pull ${IMAGE_MCAD}
    if [ $? -ne 0 ]
    then
      echo "Failed to pull ${IMAGE_MCAD}"
      exit 1
    fi
    if [ $? -ne 0 ]
    then
      echo "Failed to pull ${IMAGE_MCAD}"
      exit 1
    fi
  fi
  docker images

  for image in ${IMAGE_ECHOSERVER} ${IMAGE_UBUNTU_LATEST} ${IMAGE_MCAD}
  do
    kind load docker-image ${image} ${CLUSTER_CONTEXT}
    if [ $? -ne 0 ]
    then
      echo "Failed to load image ${image} in cluster"
      exit 1
    fi
  done 
}

# clean up
function cleanup {
    echo "==========================>>>>> Cleaning up... <<<<<=========================="
    echo " "
    if [[ ${CLUSTER_STARTED} == "false" ]]
    then
      echo "Cluster was not started, nothing more to do."
      return
    fi  

    echo "Custom Resource Definitions..."
    echo "kubectl get crds"
    kubectl get crds

    echo "---"
    echo "Get All AppWrappers..."
    kubectl get appwrappers --all-namespaces -o yaml

    echo "---"
    echo "Describe all AppWrappers..."
    kubectl describe appwrappers --all-namespaces

    echo "---"
    echo "'test' Pod list..."
    kubectl get pods -n test

    echo "---"
    echo "'test' Pod yaml..."
    kubectl get pods -n test -o yaml

    echo "---"
    echo "'test' Pod descriptions..."
    kubectl describe pods -n test

    echo "---"
    echo "'all' Namespaces  list..."
    kubectl get namespaces

    echo "---"
    echo "'aw-namespace-1' Namespace  list..."
    kubectl get namespace aw-namespace-1 -o yaml

    echo "===================================================================================="
    echo "==========================>>>>> MCAD Controller Logs <<<<<=========================="
    echo "===================================================================================="
    local mcad_pod=$(kubectl get pods -n kube-system | grep mcad-controller | awk '{print $1}')
    if [[ "$mcad_pod" != "" ]]
    then
      echo "kubectl logs ${mcad_pod} -n kube-system"
      kubectl logs ${mcad_pod} -n kube-system
    fi
    rm -f kubeconfig
  
    if [[ $CLEANUP_CLUSTER == "true" ]]
    then
      kind delete cluster ${CLUSTER_CONTEXT}     
    else 
      echo "Cluster requested to stay up, not deleting cluster"     
    fi 
}

function mcad-quota-management-down {

    # Helm chart install name
    local helm_chart_name=$(helm list -n kube-system --short | grep mcad-controller)

    # start mcad controller
    echo "Stopping MCAD Controller for Quota Management Testing..."
    echo "helm delete ${helm_chart_name}"
    helm delete -n kube-system ${helm_chart_name}
    if [ $? -ne 0 ]
    then
      echo "Failed to undeploy controller"
      exit 1
    fi
    echo "Waiting for the test namespace to be cleaned up.."
    sleep 60
}

function mcad-up {
    echo "helm install mcad-controller namespace kube-system wait set loglevel=2 set resources.requests.cpu=1000m set resources.requests.memory=1024Mi set resources.limits.cpu=4000m set resources.limits.memory=4096Mi set image.repository=$IMAGE_REPOSITORY_MCAD set image.tag=$IMAGE_TAG_MCAD set image.pullPolicy=$MCAD_IMAGE_PULL_POLICY"
    helm upgrade  --install mcad-controller ${ROOT_DIR}/deployment/mcad-controller  --namespace kube-system --wait \
                  --set loglevel=${LOG_LEVEL} --set resources.requests.cpu=1000m --set resources.requests.memory=1024Mi \
                  --set resources.limits.cpu=4000m --set resources.limits.memory=4096Mi \
                  --set configMap.name=mcad-controller-configmap --set configMap.podCreationTimeout='"120000"' \
                  --set configMap.quotaEnabled='"false"' --set coscheduler.rbac.apiGroup=scheduling.sigs.k8s.io \
                  --set coscheduler.rbac.resource=podgroups --set image.repository=$IMAGE_REPOSITORY_MCAD \
                  --set image.tag=$IMAGE_TAG_MCAD --set image.pullPolicy=$MCAD_IMAGE_PULL_POLICY
    if [ $? -ne 0 ]
    then
      echo "Failed to deploy MCAD controller"
      exit 1
    fi
}

function setup-mcad-env {
  echo "Installing Podgroup CRD"
  kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/scheduler-plugins/277b6bdec18f8a9e9ccd1bfeaf4b66495bfc6f92/config/crd/bases/scheduling.sigs.k8s.io_podgroups.yaml
 
  # Turn off master taints
  kubectl taint nodes --all node-role.kubernetes.io/master-
 
  # This is meant to orchestrate initial cluster configuration such that accounting tests can be consistent
  echo "Orchestrate cluster..."
  echo "kubectl cordon test-worker"
  kubectl cordon test-worker
  a=$(kubectl -n kube-system get pods | grep coredns | cut -d' ' -f1)
  for b in $a
  do
    echo "kubectl -n kube-system delete pod $b"
    kubectl -n kube-system delete pod $b
  done
  echo "kubectl uncordon test-worker"
  kubectl uncordon test-worker

  # sleep to allow the pods to restart
  echo "Waiting for pod in the kube-system namespace to be ready to become ready"
  while [[ $(kubectl get pods -n kube-system -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}' | tr ' ' '\n' | sort -u) != "True" ]]
  do
    echo -n "." && sleep 1; 
  done

}

function kuttl-tests {
  echo "kubectl kuttl test ${KUTTL_TEST_OPT}"
  kubectl kuttl test ${KUTTL_TEST_OPT}
  if [ $? -ne 0 ]
  then
    echo "quota management kuttl e2e tests failure, exiting."
    exit 1
  else
    # Takes a bit of time for namespace created in kuttl testing to completely delete.
    sleep 40
  fi
  rm -f kubeconfig
}

trap cleanup EXIT
update_test_host
check-prerequisites 
kind-up-cluster
setup-mcad-env
# MCAD with quotamanagement options is started by kuttl-tests
kuttl-tests
mcad-quota-management-down
mcad-up
go test ./test/e2e -v -timeout 120m -count=1