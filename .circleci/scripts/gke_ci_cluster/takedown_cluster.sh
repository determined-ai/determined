#!/bin/bash

# Exports.
export GCP_PROJECT_NAME=dai-dev-55
export GKE_REGION=us-west1
export GKE_CLUSTER_NAME=gke-circleci
export GKE_NODE_POOL_NAME=gke-circleci-compute
export BASTION_INSTANCE_NAME=gke-circleci-bastion


# Delete bastion instance. 
gcloud compute instances delete "$BASTION_INSTANCE_NAME" \
        --project "$GCP_PROJECT_NAME" \
        --zone="$GKE_REGION-b" \
        --project="$GCP_PROJECT_NAME" \
        --quiet

# Delete cluster.
gcloud container clusters delete "$GKE_CLUSTER_NAME" \
    --project "$GCP_PROJECT_NAME" \
    --region "$GKE_REGION" \
    --quiet
