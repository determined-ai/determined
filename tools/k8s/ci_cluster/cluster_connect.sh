BASTION_INSTANCE_NAME=gke-circleci-bastion-newnet
BASTION_INSTANCE_ZONE="us-west1-b"
GCP_PROJECT_NAME=determined-ai
    
# Generate random number for remote port forwarding from the bastion instance to local host.
MASTER_PORT_FROM_INTERNAL=$(jot -r 1 2000 65000)

SOCKS5_PROXY_PORT=60001
SOCKS5_PROXY_TUNNEL_OPTS="-D "$SOCKS5_PROXY_PORT" -nNT"

# Port-forward requests to remote private cluster from ssh proxy to bastion host, which will send
# the request to the appropriate endpoint within the private cluster.
#  Send cluster response traffic back to CI with remote port forwarding from cluster control plane
# to bastion host and back to local CI instance.
gcloud compute ssh "$BASTION_INSTANCE_NAME" \
        --project "$GCP_PROJECT_NAME" \
        --zone="$BASTION_INSTANCE_ZONE" \
        --tunnel-through-iap \
        --ssh-flag="-4 -C -NR$MASTER_PORT_FROM_INTERNAL:localhost:$MASTER_PORT_FROM_INTERNAL $SOCKS5_PROXY_TUNNEL_OPTS"

# Needed for kubectl and to connect to K8s API for the private cluster within Determined.
export SOCKS5_PROXY_URL="socks5://localhost:$SOCKS5_PROXY_PORT"
export ALL_PROXY=$SOCKS5_PROXY_URL HTTP_PROXY=$SOCKS5_PROXY_URL HTTPS_PROXY=$SOCKS5_PROXY_URL


# To be used at the end of test execution after all steps in a job have completed.
function cleanup()
{
    # Kill tunnels from previously opened ports on bastion instance.
    gcloud compute ssh "$BASTION_INSTANCE_NAME" \
            --project "$GCP_PROJECT_NAME" \
            --zone="$BASTION_INSTANCE_ZONE" \
            --tunnel-through-iap \
            -- pkill -u '$USER' -x -f '"^sshd: $USER*$"' || true
    
    # Kill process running the master port running on CI instance.
    kill $TUNNEL_PID
}
