#!/bin/bash

# The intent of this script is to clean up any left over vpc networks and firewall rules
# that are not deleted. This script should be run nightly in order to ensure
# the maximum quota of VPC networks is not reached.

# Function to check if a compute instance is in use
is_instance_in_use() {
    INSTANCE_NAME="$1"
    STATUS=$(gcloud compute instances describe "$INSTANCE_NAME" --zone="us-west1-b" --format="value(status)")
    if [[ $STATUS == "RUNNING" || $STATUS == "PROVISIONING" || $STATUS == "STAGING" ]]; then
        # If the instance is RUNNING, PROVISIONING, or STAGING, we consider it to be "in use".
        # This will avoid the paradoxical situation of an instance trying to delete itself.
        return 0
    fi
    return 1
}

# Function to delete firewall rules and networks
delete_resources() {
    NETWORK="$1"
    # Checks the devboxes for the vpc network. If there is no match
    # the vpc network and firewall rules will get deleted as they are unnecessary.
    if [[ -n $(echo $DEVBOXES | grep $NETWORK) ]]; then
        # Check if corresponding instance is in use
        if is_instance_in_use "$NETWORK"; then
            echo "Skipping $NETWORK as the instance is in use"
            return 0
        fi
    fi
    echo "Deleting firewall rule and network: $NETWORK"
    gcloud compute firewall-rules delete "$NETWORK" --quiet
    gcloud compute networks delete "$NETWORK" --quiet
}

export -f delete_resources
export -f is_instance_in_use

# Get a list of all resources. Note that with 'make slurmcluster' and
# our CircleCI tests, the name of compute instances, VPC networks,
# and firewall rules are the same.
NETWORKS=$(gcloud compute networks list --format="value(name)")
DEVBOXES="$(gcloud compute instances list --format="value(name)")"
for NETWORK in $NETWORKS; do
    if [[ $NETWORK =~ ^circleci-job-.*-dev-box$ ]]; then
        delete_resources $NETWORK
    fi
done
