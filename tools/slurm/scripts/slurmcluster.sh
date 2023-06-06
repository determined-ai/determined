#!/usr/bin/env bash
set -ex

ZONE=$(terraform -chdir=terraform output --raw zone)
INSTANCE_NAME=$(terraform -chdir=terraform output --raw instance_name)
PROJECT=$(terraform -chdir=terraform output --raw project)
PARENT_PATH=$(
    cd "$(dirname "${BASH_SOURCE[0]}")"
    pwd -P
)
TEMPDIR=$(mktemp -d)
echo "Using tempdir $TEMPDIR"

# Wait for SSH (terraform returns before ssh is up).
echo "Trying SSH connection..."
until gcloud compute ssh --zone "$ZONE" "$INSTANCE_NAME" --project "$PROJECT" -- ls; do
    echo "Waiting 5s to try SSH again..."
    sleep 5
    echo "Retrying SSH connection..."
done
echo "SSH is up"

# Launch SSH tunnels.
gcloud compute ssh --zone "$ZONE" "$INSTANCE_NAME" --project "$PROJECT" -- -NL 8081:localhost:8081 -NR 8080:localhost:8080 &
TUNNEL_PID=$!
trap 'kill $TUNNEL_PID' EXIT
echo "Started bidirectional tunnels to $INSTANCE_NAME"

# Grab launcher token.
REMOTE_TOKEN_SOURCE=/opt/launcher/jetty/base/etc/.launcher.token
LOCAL_TOKEN_DEST=$TEMPDIR/.launcher.token
gcloud compute scp --zone "us-west1-b" --project "determined-ai" root@$INSTANCE_NAME:$REMOTE_TOKEN_SOURCE $LOCAL_TOKEN_DEST
echo "Copied launcher token to $LOCAL_TOKEN_DEST"

# Build devcluster.yaml.
gcloud_ssh() {
    gcloud compute ssh --zone "$ZONE" "$INSTANCE_NAME" --project "$PROJECT" -- $@ 2>/dev/null | tr -d '\r\n'
}

export OPT_REMOTE_UID=$(gcloud_ssh id -u)
export OPT_REMOTE_USER=$(gcloud_ssh id -un)
export OPT_REMOTE_GID=$(gcloud_ssh id -g)
export OPT_REMOTE_GROUP=$(gcloud_ssh id -gn)
export OPT_PROJECT_ROOT='../..'
export OPT_CLUSTER_INTERNAL_IP=$(terraform -chdir=terraform output --raw internal_ip)
export OPT_AUTHFILE=$LOCAL_TOKEN_DEST

TEMPYAML=$TEMPDIR/slurmcluster.yaml
envsubst <$PARENT_PATH/slurmcluster.yaml >$TEMPYAML
echo "Generated devcluster file: $TEMPYAML"

# Run devcluster.
echo "Running cluster..."
devcluster -c $TEMPYAML
