# Using Kwok to simulate a large Kubernetes/OpenShift Cluster (with or without GPU)

This is using the OpenSource KWOK tool from https://kwok.sigs.k8s.io/

The Steps below show two ways to simulated a large number of KWOK kubernetes nodes.  
- The first way is running on your Mac laptop
- The second way is running KWOK inside an existing Kubernetes cluster

# First Way: Using KWOK to simulate a large number of nodes and MCAD Appwrappers on a Mac laptop
## Step 0. Pre-Reqs
### 0.1 Make sure you have podman (I don't have Docker to test with), installed on your mac with a podman machine of at least 4 cpu and 8GB memory: 
```
brew update
brew upgrade
brew install podman
podman machine init --cpus 4 --memory 8196
podman machine set --rootful
podman machine start
podman machine list
```

### 0.2 Install helm on your laptop, if you don't already have it:
```
brew install helm
```

### 0.3 Create a kind cluster
```
brew install kind
kind --version
kind create cluster
kubectl get nodes
```
### 0.4 Check that you see your node:
```
kubectl get nodes

NAME                 STATUS   ROLES           AGE   VERSION
kind-control-plane   Ready    control-plane   56m   v1.26.3
```
Note: If you need to get back to your kind cluster context at some point later, the command is:
```
kubectl cluster-info --context kind-kind
```

### 0.5 Install OLM on the cluster:
Note: The latest version changes with time.  You can find the latest releases at: https://github.com/operator-framework/operator-lifecycle-manager/releases/
```
curl -L https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.24.0/install.sh -o install.sh
chmod +x install.sh
./install.sh v0.24.0
```
### 0.6 Check that your OLM pods start:
```
kubectl get pods -A
```
## Step 1. Deploy NCAD on your cluster
### 1.1 Make sure you have room: # You'll at least 2 free cpu and at least 2GB memory free
```
kubectl describe node |grep cpu
kubectl describe node |grep mem
```
### 1.2 Clone the MCAD repo and change directory to it's deployment folder:
```
git clone https://github.com/project-codeflare/multi-cluster-app-dispatcher.git
cd multi-cluster-app-dispatcher/deployment
```
### 1.3 Install via helm using the following command - change the image.tag as necessary if you want something specific...
```
helm install mcad-controller --namespace kube-system --generate-name --set image.repository=quay.io/project-codeflare/mcad-controller --set image.tag=main-v1.29.58
```
### 1.4 Check that mcad is running:
```
kubectl get pods -n kube-system |grep mcad
```

## Step 2. Creating simulated KWOK node(s)
### 2.1 cd to where the MCAD performance scripts are located
```
cd ../test/perf-test
```

### 2.2 Run the script ./nodes.sh
``` 
./nodes.sh
```
### 2.3 Check that the requested number of nodes started:
```
kubectl get nodes --selector type=kwok
```

## Step 3. Create some AppWrapper jobs which create simulated pods on the simulated KWOK nodes
### 3.1 Run the script kwokmcadperf.sh
```
./kwokmcadperf.sh
```
## Step 4. Cleaning up
### 4.1 Clean up all the simulated AppWrapper jobs with the cleanup-mcad-kwok.sh script:
```
./cleanup-mcad-kwok.sh
```
### 4.2 Clean up all the simulated nodes with the following command:
```
kubectl get nodes --selector type=kwok -o name | xargs kubectl delete
```

# Second Way: Using KWOK inside an existing Kubernetes cluster
## Step 0. Pre-Reqs
### 0.1 Requires a Cluster running Kubernetes v1.10 or higher.
```
kubectl version --short=true
```

### 0.2 Access to the `kube-system` namespace.
```
kubectl get pods -n kube-system
```

### 0.3 Requires that the MCAD controller is already installed
```
kubectl get pods -A |grep mcad-controller
```

### 0.4 Install podman, jq, etc...
```
yum install make podman git tree jq go bc -y
```

### 0.5 Install the latest version of Kustomize
```
OS=$(uname)
curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"  | bash
mv kustomize /usr/local/bin
kustomize version
```

## Step 1. Install KWOK in Cluster: 
### 1.1 Variable Prep:
```
export KWOK_WORK_DIR=$(mktemp -d)
export KWOK_REPO=kubernetes-sigs/kwok
export KWOK_LATEST_RELEASE=$(curl "https://api.github.com/repos/${KWOK_REPO}/releases/latest" | jq -r '.tag_name')
```
### 1.2 Render kustomization yaml
```
cat <<EOF > "${KWOK_WORK_DIR}/kustomization.yaml"
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
images:
  - name: registry.k8s.io/kwok/kwok
    newTag: "${KWOK_LATEST_RELEASE}"
resources:
  - "https://github.com/${KWOK_REPO}/kustomize/kwok?ref=${KWOK_LATEST_RELEASE}"
EOF
```
### 1.3 Render it with the prepared variables.
```
kubectl kustomize "${KWOK_WORK_DIR}" > "${KWOK_WORK_DIR}/kwok.yaml"
```
## Step 2. Install the KWOK Controller in kube-system namespace:
### 2.1 Apply your rendered yaml file from step 1.3 above:
```
kubectl apply -f "${KWOK_WORK_DIR}/kwok.yaml"
```
### 2.2 Check to make sure the kwok controller started:
```
kubectl get pods -n kube-system |grep kwok-controller
```

## Step 3. Creating simulated KWOK node(s)
### 3.1 Clone the MCAD repo and change directory to the test/perf-test folder:
``` 
git clone https://github.com/project-codeflare/multi-cluster-app-dispatcher.git
cd multi-cluster-app-dispatcher/test/perf-test
```
 
### 3.2 Run the script ./nodes.sh
```
./nodes.sh
```
### 3.3 Check that the requested number of nodes started:
```
kubectl get nodes --selector type=kwok
```

## Step 4. Create some AppWrapper jobs which create simulated pods on the simulated KWOK nodes
### 4.1 Run the script kwokmcadperf.sh
```
./kwokmcadperf.sh
```
## Step 5. Cleaning up
### 5.1 Clean up all the simulated AppWrapper jobs with the cleanup-mcad-kwok.sh script:
```
./cleanup-mcad-kwok.sh
```
### 5.2 Clean up all the simulated nodes with the following command:
```
kubectl get nodes --selector type=kwok -o name | xargs kubectl delete
```
