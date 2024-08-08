source $1

tmp=$(lsof -i :${SOCKS5_PROXY_PORT} | grep ssh)

# Kill any previously estalished ssh connection running on the specified port.
if [[ $tmp =~ "ssh" ]]; then
        echo "killing the current processes running on this port"
        lsof -i :${SOCKS5_PROXY_PORT} -t | xargs -L1 kill
fi

unset HTTPS_PROXY

# Kill previously running ssh processes created by the current user running on the VM.
gcloud compute ssh "$BASTION_INSTANCE_NAME" \
 --project "$GCP_BASTION_PROJECT_ID" \
 --zone="$BASTION_INSTANCE_ZONE" \
 --tunnel-through-iap \
 -- pkill -u '$USER' -x -f '"^sshd: $USER*$"' || true

SOCKS5_PROXY_TUNNEL_OPTS="-D "$SOCKS5_PROXY_PORT" -nNT"

# Establish new ssh connection via dynamic port forwarding to connect to the VPC containing the
# cluster's control plane.
gcloud compute ssh "$BASTION_INSTANCE_NAME" \
 --project "$GCP_BASTION_PROJECT_ID" \
 --zone="$BASTION_INSTANCE_ZONE" \
 --tunnel-through-iap \
 --ssh-flag="-4 -C $SOCKS5_PROXY_TUNNEL_OPTS" &

sleep 5

echo "Established connection to remote host!"

# Get cluster credentials to run kubectl and helm commands.
gcloud container clusters get-credentials "$GKE_CLUSTER_NAME" \
 --location="$GKE_REGION" \
 --project="$CLUSTER_PROJECT_ID"

# We need to set HTTPS_PROXY to forward all traffic through the SOCKS5 proxy.
export SOCKS5_PROXY_URL="socks5://localhost:$SOCKS5_PROXY_PORT"
export HTTPS_PROXY=$SOCKS5_PROXY_URL 

