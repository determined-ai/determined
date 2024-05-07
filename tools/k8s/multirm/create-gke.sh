#!/bin/bash

# Set common configuration variables
REGION="us-west1"
ZONE="us-west1-b"
STANDARD_MACHINE_TYPE="n1-standard-16"
GPU_MACHINE_TYPE="n1-standard-32"
MIN_NODES=0
MAX_NODES=2
NUM_NODES=1

# Function to create GKE cluster
create_gke_cluster() {
    local cluster_name=$1
    gcloud container clusters create ${cluster_name} \
        --region ${REGION} \
        --node-locations ${ZONE} \
        --num-nodes=${NUM_NODES} \
        --machine-type=${STANDARD_MACHINE_TYPE}
}

# Function to create GPU node pool
create_gpu_node_pool() {
    local cluster_name=$1
    local node_pool_name=$2
    local gpu_type=$3
    local gpus_per_node=$4
    gcloud container node-pools create ${node_pool_name} \
        --cluster ${cluster_name} \
        --accelerator type=${gpu_type},count=${gpus_per_node} \
        --zone ${ZONE} \
        --num-nodes=${MIN_NODES} \
        --enable-autoscaling \
        --min-nodes=${MIN_NODES} \
        --max-nodes=${MAX_NODES} \
        --machine-type=${GPU_MACHINE_TYPE} \
        --scopes=storage-full,cloud-platform
}

# Main script logic
GKE_CLUSTER_NAME="h-mrm-ingress-cluster"
GKE_GPU_NODE_POOL_NAME="determined-node-pool"
GCS_BUCKET_NAME="determined-checkpoint-bucket"
GPU_TYPE="nvidia-tesla-t4"
GPUS_PER_NODE=4

create_gke_cluster ${GKE_CLUSTER_NAME}
create_gpu_node_pool ${GKE_CLUSTER_NAME} ${GKE_GPU_NODE_POOL_NAME} ${GPU_TYPE} ${GPUS_PER_NODE}

# Deploy GPU DaemonSet
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/container-engine-accelerators/master/nvidia-driver-installer/cos/daemonset-preloaded.yaml

# Create GCS bucket for checkpoints
gsutil mb gs://${GCS_BUCKET_NAME}

echo "Setup complete."
