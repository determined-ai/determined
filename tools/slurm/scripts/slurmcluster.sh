#!/usr/bin/env bash

set -e

# This script is invoked by `make slurmcluster`. It should never be necessary to run it directly.

# Valid values
VALID_CONTAINER_TYPES="enroot podman singularity"
VALID_WORKLOAD_MANAGERS="slurm pbs"

# Default values
export OPT_CONTAINER_RUN_TYPE="singularity"
export OPT_WORKLOAD_MANAGER="slurm"
export OPT_LAUNCHER_PORT=8081
DETERMINED_AGENT=
MACHINE_TYPE=
GPUS=

while [[ $# -gt 0 ]]; do
    case $1 in
        -c | --container-run-type)
            export OPT_CONTAINER_RUN_TYPE=$2
            if [[ ! " $VALID_CONTAINER_TYPES " =~ " $OPT_CONTAINER_RUN_TYPE " ]]; then
                echo >&2 "usage $0: Missing or invalid container type specified for the '-c' option.  Valid values are: ${VALID_CONTAINER_TYPES}"
                exit 1
            fi
            shift 2
            ;;
        -w | --workload-manager)
            export OPT_WORKLOAD_MANAGER=$2
            if [[ ! " $VALID_WORKLOAD_MANAGERS " =~ " $OPT_WORKLOAD_MANAGER " ]]; then
                echo >&2 "usage $0: Missing or invalid workload manager specified for the '-r' option.  Valid values are: ${VALID_WORKLOAD_MANAGERS}"
                exit 1
            fi
            shift 2
            ;;
        -A)
            DETERMINED_AGENT=1
            shift
            ;;
        -m)
            # This is processed already by generate-tfvars.sh
            MACHINE_TYPE=$2
            shift 2
            ;;
        -g)
            # This is processed already by generate-tfvars.sh
            GPUS=$2
            shift 2
            ;;
        -d)
            # Fixed, default Launcher port
            export OPT_LAUNCHER_PORT=18080
            shift 1
            ;;
        # The Makefile that calls this script may pass in additional flags used for other scritps
        # which can be ignored.
        -t)
            shift 1
            while [[ -n $1 && $1 != -* ]]; do
                shift 1
            done
            ;;
        -h | --help)
            echo "Usage: $0 [-Acwdmgt]"
            echo ""
            echo "Launches a compute instance with Slurm, Singularity (Apptainer), the HPC"
            echo "Launcher component, and many other dependencies pre-installed. Then, SSH tunnels"
            echo "are opened so that localhost:8081 on your machine points at port 8081 on"
            echo "the compute instance and port 8080 on the compute instance points at"
            echo "localhost:8080 on your machine. Lastly, devcluster is started with the Dispatcher"
            echo "RM pointed at the remote instance, and local development with devcluster works"
            echo "as always."
            echo ""
            echo "FLAGS:"
            echo '  -A '
            echo "           Description: Invokes a slurmcluster that uses agents instead of the launcher."
            echo "           Example: $0 -A"
            echo '  -c {enroot|podman|singularity}'
            echo "           Description: Invokes a slurmcluster using the specified container run type."
            echo "           Options are 'enroot', 'podman', or 'singularity'. Default is 'singularity'."
            echo "           Example: $0 -c podman"
            echo '  -w {slurm|pbs} '
            echo "           Description: Invokes a slurmcluster using the specified workload manager."
            echo "           Options are 'slurm' or 'pbs'. Default is 'slurm'."
            echo "           Example: $0 -w pbs"
            echo '  -d'
            echo "           Description: Connect to a dev launcher manually deployed to the GCP VM using"
            echo "           'loadDevlauncher.sh -g'."
            echo '  -t'
            echo "           Description: Specifies a duration for the GCP VM to run before automatically"
            echo "           terminating. User can input time in d(days)|h(hours)|m(minutes)|s(seconds)"
            echo '  -m {machine_type} '
            echo "           Description: Specify a GCP machine_type value (default is n1-standard-8)"
            echo "           Example: $0 -m g2-standard-4"
            echo '  -g {name}:{count}'
            echo "           Description: Specify a GCP accelerator name and count"
            echo "           Example: $0 -g nvidia-tesla-t4:2"
            echo ""
            echo "You can also combine the flags.  When invoked via 'make slurmcluster' flags are passed"
            echo 'via the FLAGS="options" argument.'
            echo ""
            echo '   Example: FLAGS="-A -c enroot -w pbs -g nvidia-tesla-t4:2"'
            echo ""
            exit 0
            ;;
        -* | --*)
            set +ex
            echo >&2 "$0: Illegal option $1"
            echo >&2 "Usage: $0 [-c {container_type}]"
            set -ex
            exit 1
            ;;
    esac
done

echo "Using ${OPT_CONTAINER_RUN_TYPE} as a container host"
echo "Using ${OPT_WORKLOAD_MANAGER} as a workload manager"

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
until gcloud compute ssh --quiet --zone "$ZONE" "$INSTANCE_NAME" --project "$PROJECT" -- ls; do
    echo "Waiting 5s to try SSH again..."
    sleep 5
    echo "Retrying SSH connection..."
done
echo "SSH is up"

SOCKS5_PROXY_PORT=60000

SOCKS5_PROXY_TUNNEL_OPTS="-D ${SOCKS5_PROXY_PORT} -nNT"

# Launch SSH tunnels.
echo "Launching SSH tunnels"
gcloud compute ssh --quiet --zone "$ZONE" "$INSTANCE_NAME" --project "$PROJECT" -- -NL ${OPT_LAUNCHER_PORT}:localhost:${OPT_LAUNCHER_PORT} -NR 8080:localhost:8080 ${SOCKS5_PROXY_TUNNEL_OPTS} &
TUNNEL_PID=$!
trap 'kill $TUNNEL_PID' EXIT
echo "Started bidirectional tunnels to $INSTANCE_NAME"

# Grab launcher token.
if [[ -z $DETERMINED_AGENT ]]; then
    REMOTE_TOKEN_SOURCE=/opt/launcher/jetty/base/etc/.launcher.token
    LOCAL_TOKEN_DEST=$TEMPDIR/.launcher.token
    gcloud compute scp --quiet --zone "us-west1-b" --project "determined-ai" root@$INSTANCE_NAME:$REMOTE_TOKEN_SOURCE $LOCAL_TOKEN_DEST
    echo "Copied launcher token to $LOCAL_TOKEN_DEST"
    # The launcher service verifies communication with the launcher, so just have to be sure it is started.
    # Also show the status for extra confirmaiton in the logs of the state.
    gcloud compute ssh --zone "$ZONE" "$INSTANCE_NAME" --project "$PROJECT" -- "sudo systemctl start launcher.service ; systemctl status launcher.service --no-pager"
fi

# Build devcluster.yaml.
gcloud_ssh() {
    echo gcloud compute ssh --zone "$ZONE" "$INSTANCE_NAME" --project "$PROJECT" -- $@ 2>/dev/null | tr -d '\r\n'
    gcloud compute ssh --zone "$ZONE" "$INSTANCE_NAME" --project "$PROJECT" -- $@ 2>/dev/null | tr -d '\r\n'
}

export OPT_REMOTE_UID=$(gcloud_ssh id -u)
export OPT_REMOTE_USER=$(gcloud_ssh id -un)
export OPT_REMOTE_GID=$(gcloud_ssh id -g)
export OPT_REMOTE_GROUP=$(gcloud_ssh id -gn)
export OPT_PROJECT_ROOT='../..'
export OPT_CLUSTER_INTERNAL_IP=$(terraform -chdir=terraform output --raw internal_ip)
export OPT_AUTHFILE=$LOCAL_TOKEN_DEST

LOCAL_CPU_IMAGE_STRING=$(grep "CPUImage" ../../master/pkg/schemas/expconf/const.go | awk -F'\"' '{print $2}')
LOCAL_CPU_IMAGE_SQSH=${LOCAL_CPU_IMAGE_STRING//[\/:]/+}.sqsh
LOCAL_CUDA_IMAGE_STRING=$(grep "CUDAImage" ../../master/pkg/schemas/expconf/const.go | awk -F'\"' '{print $2}')
LOCAL_CUDA_IMAGE_SQSH=${LOCAL_CUDA_IMAGE_STRING//[\/:]/+}.sqsh

# Enroot container creation
if [[ $OPT_CONTAINER_RUN_TYPE == "enroot" ]]; then
    # Find the file and assign its name to CPU_IMAGE_SQSH & CUDA_IMAGE_SQSH
    CPU_IMAGE_SQSH=$(gcloud_ssh "ls /srv/enroot/ | grep '^determinedai+environments+py'")
    CUDA_IMAGE_SQSH=$(gcloud_ssh "ls /srv/enroot/ | grep '^determinedai+environments+cuda'")

    if [[ $CPU_IMAGE_SQSH != "$LOCAL_CPU_IMAGE_SQSH" ]]; then
        echo "WARNING: Local CPUImage specified in ../../master/pkg/schemas/expconf/const.go does not match the CPU Image found on existing ${OPT_WORKLOAD_MANAGER} image."
        echo "   Regenerate base image with: make -C tools/slurm/packer build WORKLOAD_MANAGER=${OPT_WORKLOAD_MANAGER}"
    fi
    if [[ $CUDA_IMAGE_SQSH != "$LOCAL_CUDA_IMAGE_SQSH" ]]; then
        echo "WARNING: Local CUDAImage specified in ../../master/pkg/schemas/expconf/const.go does not match the CUDA Image found on existing ${OPT_WORKLOAD_MANAGER} image."
        echo "   Regenerate base image with: make -C tools/slurm/packer build WORKLOAD_MANAGER=${OPT_WORKLOAD_MANAGER}"
    fi

    # If the image has to download during circleci jobs it may cause a timeout waiting for make slurmcluster
    # particilarly if we add GPU tests that need the CUDA image which is larger.
    if [[ -z $GPUS ]]; then
        gcloud_ssh "sudo ENROOT_RUNTIME_PATH=/srv/enroot ENROOT_TEMP_PATH=/srv/enroot manage-enroot-cache -N -s /srv/enroot ${LOCAL_CPU_IMAGE_STRING}"
        gcloud_ssh "ENROOT_RUNTIME_PATH=/srv/enroot ENROOT_TEMP_PATH=/srv/enroot manage-enroot-cache -s /srv/enroot ${LOCAL_CPU_IMAGE_STRING}"
    else
        gcloud_ssh "sudo ENROOT_RUNTIME_PATH=/srv/enroot ENROOT_TEMP_PATH=/srv/enroot manage-enroot-cache -N -s /srv/enroot ${LOCAL_CUDA_IMAGE_STRING}"
        gcloud_ssh "ENROOT_RUNTIME_PATH=/srv/enroot ENROOT_TEMP_PATH=/srv/enroot manage-enroot-cache -s /srv/enroot ${LOCAL_CUDA_IMAGE_STRING}"
    fi
fi

TEMPYAML=$TEMPDIR/slurmcluster.yaml
envsubst <$PARENT_PATH/slurmcluster.yaml >$TEMPYAML
if [[ -n $DETERMINED_AGENT ]]; then
    # When deploying with the determined agent, remove the resource_manager section
    # that would otherwise be used.   This then defaults to the agent rm and
    # the master waits for agents to connect and provide resources.
    sed -i -e '/resource_manager/,/resource_manager_end/d' $TEMPYAML
fi
echo "Generated devcluster file: $TEMPYAML"

# We connect to the Slurm VM using an external IP address, but although it's a
# single node cluster, the Determined master running on the test machine tries
# to connect to the shell container using its private 10.X.X.X address.
# Therefore, we must tell the Determined master to use the SOCKS5 proxy SSH
# tunnel that we configured so it can communicate with the container's private
# IP address.
#
# Note: Do not set ALL_PROXY before calling "gcloud" or it will fail.
export ALL_PROXY=socks5://localhost:${SOCKS5_PROXY_PORT}

# Run devcluster.
echo "Running cluster..."
devcluster -c $TEMPYAML --oneshot
