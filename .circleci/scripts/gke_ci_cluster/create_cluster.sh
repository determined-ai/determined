#!/bin/bash

#Exports.
export GCP_PROJECT_NAME=dai-dev-55
export GKE_CLUSTER_NAME=gke-circleci
export GKE_REGION=us-west1
export GKE_MACHINE_TYPE=n1-standard-8
export GKE_NUM_NODES=1
export GCS_NETWORK_NAME=gke-circleci-vpc

export BASTION_INSTANCE_NAME=gke-circleci-bastion
export BASTION_INSTANCE_ZONE="$GKE_REGION-b"
export BASTION_INSTANCE_TYPE=e2-micro

# Create dedicated network for the cluster.
gcloud compute networks create "$GCS_NETWORK_NAME"
gcloud compute firewall-rules create allow-ingress-from-circleci --network gke-circleci-vpc --allow tcp:22,tcp:3389,tcp:443
gcloud compute firewall-rules create allow-ingress-from-iap-ci-vpc --network gke-circleci-vpc --allow tcp:22,tcp:3389,tcp:443



# Create cluster with public IP addresses.
gcloud container clusters create "$GKE_CLUSTER_NAME" \
        --project "$GCP_PROJECT_NAME" \
        --region "$GKE_REGION" \
        --network="$GCS_NETWORK_NAME" \
        --enable-autoscaling \
        --enable-autoprovisioning \
        --enable-autorepair \
        --enable-autoupgrade \
        --max-cpu=7 \
        --max-memory=25 \
        --num-nodes="$GKE_NUM_NODES" \
        --enable-ip-alias \
        --enable-master-authorized-networks \
        --master-authorized-networks="35.235.240.0/20,3.228.39.90/32,18.213.67.41/32,34.194.94.201/32,\
34.194.144.202/32,34.197.6.234/32,35.169.17.173/32,35.174.253.146/32,52.3.128.216/32,52.4.195.249/32,\
52.5.58.121/32,52.21.153.129/32,52.72.72.233/32,54.92.235.88/32,54.161.182.76/32,54.164.161.41/32,\
54.166.105.113/32,54.167.72.230/32,54.172.26.132/32,54.205.138.102/32,54.208.72.234/32,54.209.115.53/32" \
        --cluster-version=latest \
        --machine-type="$GKE_MACHINE_TYPE" \
        --maintenance-window="8:00"

# Create bastion instance.
gcloud compute instances create "$BASTION_INSTANCE_NAME" \
        --project "$GCP_PROJECT_NAME" \
        --zone="$BASTION_INSTANCE_ZONE" \
        --machine-type="$BASTION_INSTANCE_TYPE" \
        --network-interface=no-address,subnet="$GCS_NETWORK_NAME"
