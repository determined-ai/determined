#!/usr/bin/env bash

ACTION=$1

if [[ $ACTION =~ ^(up|connect|down)$ ]]; then
    if [[ $ACTION =~ ^(up|down)$ ]]; then
        read -p "You are about to modify a team-wide shared cluster. Do you want to proceed? (y/n) " yn
        case $yn in
            [yY]) ;;
            [nN])
                echo "exiting..."
                exit 1
                ;;
            *) echo invalid response ;;
        esac
    fi
    echo ""$ACTION" initiated on shared GKE cluster..."
else
    echo ""$ACTION" is not a valid action (only up, connect, or down)."
    exit 1
fi

# Set a unique name for your cluster.
export GCP_PROJECT_NAME=determined-ai
export GKE_CLUSTER_NAME=backend-gke
export GCS_BUCKET_NAME=backend-gke
export GCS_SUBNET_NAME=backend-gke

# CPU node configurations.
export GKE_REGION=us-west1
export GKE_NODE_LOCATION=us-west1-b
export GKE_MACHINE_TYPE=n1-standard-8
export GKE_NUM_NODES=1

# GPU node pool configurations.
export GKE_GPU_NODE_POOL_NAME=backend-gke
export GKE_GPU_TYPE=nvidia-tesla-t4
export GKE_GPU_PER_NODE=1

# Bastion instance configuration.
export BASTION_INSTANCE_NAME=backend-gke-bastion
export BASTION_INSTANCE_ZONE="$GKE_REGION-b"
export BASTION_INSTANCE_TYPE=e2-micro
export BASTION_INSTANCE_STARTUP_HOOK=bastion-instance-startup.sh

if [ ""$ACTION"" = "up" ]; then
    gcloud container clusters create "$GKE_CLUSTER_NAME" \
        --project "$GCP_PROJECT_NAME" \
        --region "$GKE_REGION" \
        --create-subnetwork=name="$GCS_SUBNET_NAME" \
        --enable-master-authorized-networks \
        --enable-ip-alias \
        --enable-private-nodes \
        --enable-private-endpoint \
        --master-ipv4-cidr 172.16.0.32/28 \
        --cluster-version=latest \
        --node-locations "$GKE_NODE_LOCATION" \
        --num-nodes="$GKE_NUM_NODES" \
        --machine-type="$GKE_MACHINE_TYPE"

    gcloud container node-pools create "$GKE_GPU_NODE_POOL_NAME" \
        --cluster "$GKE_CLUSTER_NAME" \
        --project "$GCP_PROJECT_NAME" \
        \
        --region="$GKE_REGION" \
        --machine-type=n1-standard-8 \
        --scopes=storage-full # --accelerator type="$GKE_GPU_TYPE",count="$GKE_GPU_PER_NODE" \

    gcloud compute instances create "$BASTION_INSTANCE_NAME" \
        --project "$GCP_PROJECT_NAME" \
        --zone="$BASTION_INSTANCE_ZONE" \
        --machine-type="$BASTION_INSTANCE_TYPE" \
        --network-interface=no-address,network-tier=PREMIUM,subnet="$GCS_SUBNET_NAME"

elif [ ""$ACTION"" = "connect" ]; then
    # Kill old tunnels.
    # For some reason setting:
    #   sudo sh -c 'echo "StreamLocalBindUnlink yes" >> /etc/ssh/sshd_config'
    # on the server didn't work.
    gcloud compute ssh "$BASTION_INSTANCE_NAME" \
        --project "$GCP_PROJECT_NAME" \
        --zone="$BASTION_INSTANCE_ZONE" \
        --tunnel-through-iap \
        -- pkill -u '$USER' -x -f '"^sshd: $USER[ ]*$"'

    export MASTER_IP_FROM_INTERNAL=$(
        gcloud compute instances describe "$BASTION_INSTANCE_NAME" \
            --zone "$BASTION_INSTANCE_ZONE" \
            --project "$GCP_PROJECT_NAME" \
            | yq -o=json \
            | jq '.networkInterfaces[0].networkIP'
    )
    export MASTER_PORT_FROM_INTERNAL=$(jot -r 1 2000 65000)

    export SOCKS5_PROXY_PORT=60001 # Exported for envsubst call. Same for anything else exported.
    SOCKS5_PROXY_TUNNEL_OPTS="-D "$SOCKS5_PROXY_PORT" -nNT"
    gcloud compute ssh "$BASTION_INSTANCE_NAME" \
        --project "$GCP_PROJECT_NAME" \
        --zone="$BASTION_INSTANCE_ZONE" \
        --tunnel-through-iap \
        --ssh-flag="-4 -C -NR$MASTER_PORT_FROM_INTERNAL:localhost:$MASTER_PORT_FROM_INTERNAL $SOCKS5_PROXY_TUNNEL_OPTS" &
    TUNNEL_PID=$!
    trap 'kill $TUNNEL_PID' EXIT
    export SOCKS5_PROXY_URL="socks5://localhost:$SOCKS5_PROXY_PORT"

    gcloud container clusters get-credentials "$GKE_CLUSTER_NAME" \
        --region="$GKE_REGION" \
        --project="$GCP_PROJECT_NAME"

    echo "Cluster is setup, please use HTTPS_PROXY when using kubectl, or paste this into your terminal:"
    echo "k() ("
    echo "    HTTPS_PROXY=$SOCKS5_PROXY_URL kubectl $@"
    echo ")"
    # kubectl won't respect ALL_PROXY, only HTTPS_PROXY.
    # https://kubernetes.io/docs/tasks/extend-kubernetes/socks5-proxy-access-api/#client-configuration
    k() (
        HTTPS_PROXY=$SOCKS5_PROXY_URL kubectl $@
    )
    export KUBERNETES_NAMESPACE=$USER-devcluster
    k create namespace $KUBERNETES_NAMESPACE

    # Although devcluster supports variables, numeric values fail to load, so
    # Manually apply those into a temp file.
    TMPYAML=/tmp/devcluster-kubernetes.yaml
    rm -f $TMPYAML
    envsubst <devcluster.tpl.yaml >$TMPYAML

    export ALL_PROXY=$SOCKS5_PROXY_URL
    devcluster --oneshot --config $TMPYAML

    # TODO: Launch HTTPS_PROXY pod to proxy notebook connections, but have it NO_PROXY the CLUSTER_IP.

else
    gcloud compute instances delete "$BASTION_INSTANCE_NAME" \
        --zone="$BASTION_INSTANCE_ZONE" \
        --quiet

    gcloud container node-pools delete "$GKE_GPU_NODE_POOL_NAME" \
        --region="$GKE_REGION" \
        --cluster "$GKE_CLUSTER_NAME" \
        --project "$GCP_PROJECT_NAME" \
        --quiet

    gcloud container clusters delete "$GKE_CLUSTER_NAME" \
        --project "$GCP_PROJECT_NAME" \
        --region "$GKE_REGION" \
        --quiet
fi
