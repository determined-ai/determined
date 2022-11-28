#!/bin/bash
#
# This dev script is a wrapper on the devcluster tool, and provides
# per-user and per cluster configuration of the devcluster-slurm.yaml
# file to enable it to be used for our various clusters.  It dynamically
# fills in the variables within devcluster-slurm.yaml such that the original
# source need not be modified.   By default it also starts/stops SSH
# tunnels inbound to launcher, and outbound to the desktop master.
#
# It supports authorized access to the launcher by automatically specifying
# the auth_file if a ~/.{CLUSTER}.token file is present.   Use the -a option
# (one time) to retrieve a remote token file from the cluster.
#
# You can utilize a launcher instance created by the hpc-ard-capsule-core
# loadDevLauncher.sh script with the -d argument.
#
# Pre-requisites:
#   1) Configure your USERPORT_${USER} port below using your login name on
#      the desktop that you are using.
#
#   2)  Unless you specify both -n -x, you must have password-less ssh configured to the
#   target cluster, to enable the ssh connection without prompts.  Configure ~/.ssh/conf
#   to automatically configure your remote user name if your desktop username is
#   different than on the cluster.
#
#      ssh-copy-id {cluster}
#
#   3) To utilize the -a option to retrieve the /opt/launcher/jetty/base/etc/.launcher.token
#   you must have sudo access on the cluster, and authenticate the sudo that will be
#   exectued to retrieve the .launcher.token.

HELPEND=$((LINENO - 1))

INTUNNEL=1
TUNNEL=1
DEVLAUNCHER=
USERNAME=$USER
DEBUGLEVEL=debug
# Variables that can be set before invoking the script (to change the default)
DEFAULTIMAGE=${DEFAULTIMAGE-}

while [[ $# -gt 0 ]]; do
    case $1 in
        -n)
            INTUNNEL=
            shift
            ;;
        -x)
            TUNNEL=
            shift
            ;;
        -t)
            DEBUGLEVEL=trace
            shift
            ;;
        -i)
            DEBUGLEVEL=info
            shift
            ;;
        -p)
            PODMAN=1
            shift
            ;;
        -e)
            ENROOT=1
            shift
            ;;
        -d)
            DEVLAUNCHER=1
            shift
            ;;
        -u)
            USERNAME=$2
            shift 2
            ;;
        -a)
            PULL_AUTH=1
            shift
            ;;
        -c)
            DEFAULTIMAGE=$2
            shift 2
            ;;
        -h | --help)
            echo "Usage: $0 [-anxtpedi] [-c {image}] [-u {username}]  {cluster}"
            echo "  -h     This help message & documentation."
            echo "  -n     Disable start of the inbound tunnel (when using Cisco AnyConnect)."
            echo "  -x     Disable start of personal tunnel back to master (if you have done so manually)."
            echo "  -t     Force debug level to trace regardless of cluster configuration value."
            echo "  -i     Force debug level to INFO regardless of cluster configuration value."
            echo "  -p     Use podman as a container host (otherwise singlarity)."
            echo "  -e     Use enroot as a container host (otherwise singlarity)."
            echo "  -d     Use a developer launcher (port assigned for the user in loadDevLauncher.sh)."
            echo "  -c     Use the specified {image} as the default image.  Useful with -d and for enroot."
            echo "  -u     Use provided {username} to lookup the per-user port number."
            echo "  -a     Attempt to retrieve the .launcher.token - you must have sudo root on the cluster."
            echo
            echo "Documentation:"
            head -n $HELPEND $0 | tail -n $((HELPEND - 1))
            exit 1
            ;;
        -* | --*)
            echo >&2 "$0: Illegal option $1"
            echo >&2 "Usage: $0 [-anxtpde] [-c {image}] [-u {username}]  {cluster}"
            exit 1
            ;;
        *) # Non Option args
            CLUSTER=$1
            shift
            ;;
    esac
done

# Evaluate a dynamically constructed env variable name
function lookup() {
    echo "${!1}"
}

# Setup the reverse tunnel back to the master running locally
function mktunnel() {
    MASTER_HOST=$1
    MASTER_PORT=$2
    SSH_HOST=$3
    ssh -NR ${MASTER_HOST}:${MASTER_PORT}:localhost:8081 ${SSH_HOST}
}

# Setup the inbound tunnel to enable access to the launcher
function mkintunnel() {
    MASTER_HOST=$1
    MASTER_PORT=$2
    SSH_HOST=$3
    ssh -NL ${MASTER_PORT}:${MASTER_HOST}:${MASTER_PORT} ${SSH_HOST}
}

# Attempt to retrieve the auth token from the remote host
# This requires that your account have sudo access to root
# and will likely be prompted for a password.
# Args: {hostname} {cluster}
function pull_auth_token() {
    HOST=$1
    CLUSTER=$2

    echo "Attempting to access /opt/launcher/jetty/base/etc/.launcher.token from $HOST"
    rm -f ~/.token.log
    ssh -t $HOST 'sudo cat /opt/launcher/jetty/base/etc/.launcher.token' | tee ~/.token.log
    # Token is the last line of the output (no newline)
    TOKEN=$(tail -n 1 ~/.token.log)
    if [[ ${TOKEN} != *" "* ]]; then
        echo -n "${TOKEN}" >~/.${CLUSTER}.token
        echo "INFO: Saved token as  ~/.${CLUSTER}.token"
    else
        echo "WARNING: No token retieved: ${TOKEN}" >&2
    fi
}

# Update your username/port pair
USERPORT_madagund=8083
USERPORT_laney=8084
USERPORT_rcorujo=8085
USERPORT_phillipgaisford=8086
USERPORT_pankaj=8087
USERPORT_alyssa=8088
USERPORT_jerryharrow=8090
USERPORT_canmingcobble=8092

USERPORT=$(lookup "USERPORT_$USERNAME")
if [[ -z $USERPORT ]]; then
    echo >&2 "$0: User $USERNAME does not have a configured port, update the script."
    exit 1
fi

# Re-map names that include - as variables with embedded - are treated as math expressions
if [[ $CLUSTER == "casablanca-login" ]]; then
    CLUSTER=casablanca_login
elif [[ $CLUSTER == "casablanca-mgmt1" ]]; then
    CLUSTER=casablanca
elif [[ $CLUSTER == "casablanca-login2" ]]; then
    CLUSTER=casablanca_login2
fi

# Update your JETTY HTTP username/port pair from loadDevLauncher.sh
DEV_LAUNCHER_PORT_madagund=18083
DEV_LAUNCHER_PORT_laney=18084
DEV_LAUNCHER_PORT_rcorujo=18085
DEV_LAUNCHER_PORT_phillipgaisford=18086
DEV_LAUNCHER_PORT_pankaj=18087
DEV_LAUNCHER_PORT_alyssa=18088
DEV_LAUNCHER_PORT_jerryharrow=18090
DEV_LAUNCHER_PORT_canmingcobble=18092
DEV_LAUNCHER_PORT=$(lookup "DEV_LAUNCHER_PORT_$USERNAME")

# Configuration for atlas
OPT_name_atlas=atlas.us.cray.com
OPT_LAUNCHERHOST_atlas=localhost
OPT_LAUNCHERPROTOCOL_atlas=http
OPT_CHECKPOINTPATH_atlas=/lus/scratch/foundation-engineering/determined-cp
OPT_MASTERHOST_atlas=atlas
OPT_MASTERPORT_atlas=$USERPORT
OPT_TRESSUPPORTED_atlas=false
OPT_GRESSUPPORTED_atlas=false
OPT_PROTOCOL_atlas=http

# Configuration for casablanca (really casablanca-mgmt1)
OPT_name_casablanca=casablanca-mgmt1.us.cray.com
OPT_LAUNCHERHOST_casablanca=localhost
OPT_LAUNCHERPORT_casablanca=8181
OPT_LAUNCHERPROTOCOL_casablanca=http
OPT_CHECKPOINTPATH_casablanca=/mnt/lustre/foundation_engineering/determined-cp
OPT_MASTERHOST_casablanca=casablanca-mgmt1.us.cray.com
OPT_MASTERPORT_casablanca=$USERPORT
OPT_TRESSUPPORTED_casablanca=true
OPT_PROTOCOL_casablanca=http

# Configuration for horizon
OPT_name_horizon=horizon.us.cray.com
OPT_LAUNCHERHOST_horizon=localhost
OPT_LAUNCHERPORT_horizon=8181
OPT_LAUNCHERPROTOCOL_horizon=http
OPT_CHECKPOINTPATH_horizon=/lus/scratch/foundation_engineering/determined-cp
OPT_MASTERHOST_horizon=horizon
OPT_MASTERPORT_horizon=$USERPORT
OPT_TRESSUPPORTED_horizon=false
OPT_PROTOCOL_horizon=http

# Configuration for casablanca-login (uses suffix casablanca_login)
OPT_name_casablanca_login=casablanca-login.us.cray.com
OPT_LAUNCHERHOST_casablanca_login=localhost
OPT_LAUNCHERPORT_casablanca_login=8443
OPT_LAUNCHERPROTOCOL_casablanca_login=https
OPT_CHECKPOINTPATH_casablanca_login=/mnt/lustre/foundation_engineering/determined-cp
OPT_MASTERHOST_casablanca_login=casablanca-login
OPT_MASTERPORT_casablanca_login=$USERPORT
OPT_TRESSUPPORTED_casablanca_login=true

# Configuration for casablanca-login2 (uses suffix casablanca_login2)
OPT_name_casablanca_login2=casablanca-login2.us.cray.com
OPT_LAUNCHERHOST_casablanca_login2=localhost
OPT_LAUNCHERPORT_casablanca_login2=8443
OPT_LAUNCHERPROTOCOL_casablanca_login2=http
OPT_CHECKPOINTPATH_casablanca_login2=/mnt/lustre/foundation_engineering/determined-cp
OPT_MASTERHOST_casablanca_login2=casablanca-login2
OPT_MASTERPORT_casablanca_login2=$USERPORT
OPT_TRESSUPPORTED_casablanca_login2=false

# Configuration for sawmill (10.100.97.101)
OPT_name_sawmill=10.100.97.101
OPT_LAUNCHERHOST_sawmill=localhost
OPT_LAUNCHERPROTOCOL_sawmill=http
OPT_CHECKPOINTPATH_sawmill=/scratch2/launcher/determined-cp
OPT_MASTERHOST_sawmill=nid000001
OPT_MASTERPORT_sawmill=$USERPORT
OPT_TRESSUPPORTED_sawmill=false
OPT_GRESSUPPORTED_sawmill=false
OPT_PROTOCOL_sawmill=http
OPT_DEFAULTIMAGE_sawmill=/scratch2/karlon/new/detAI-cuda-11.3-pytorch-1.10-tf-2.8-gpu-nccl-0.19.4.sif
# Indentation of task_container_defaults must match devcluster-slurm.yaml
OPT_TASKCONTAINERDEFAULTS_sawmill=$(
    cat <<EOF
          environment_variables:
            - USE_HOST_LIBFABRIC=y
            - NCCL_DEBUG=INFO
            - OMPI_MCA_orte_tmpdir_base=/dev/shm/
EOF
)
# Indentation of partition_overrides must match devcluster-slurm.yaml
OPT_PARTITIONOVERRIDES_sawmill=$(
    cat <<EOF
             grizzly:
                slot_type: cuda
EOF
)

# Configuration for shuco
OPT_name_shuco=shuco.us.cray.com
OPT_LAUNCHERHOST_shuco=localhost
OPT_LAUNCHERPORT_shuco=8181
OPT_LAUNCHERPROTOCOL_shuco=http
OPT_CHECKPOINTPATH_shuco=/home/launcher/determined-cp
OPT_MASTERHOST_shuco=admin.head.cm.us.cray.com
OPT_MASTERPORT_shuco=$USERPORT
OPT_TRESSUPPORTED_shuco=false
OPT_PROTOCOL_shuco=http
OPT_RENDEVOUSIFACE_shuco=bond0

# Configuration for mosaic
OPT_name_mosaic=10.30.91.220
OPT_LAUNCHERHOST_mosaic=localhost
OPT_LAUNCHERPORT_mosaic=8181
OPT_LAUNCHERPROTOCOL_mosaic=http
OPT_CHECKPOINTPATH_mosaic=/home/launcher/determinedai/checkpoints
OPT_MASTERHOST_mosaic=10.30.91.220
OPT_MASTERPORT_mosaic=$USERPORT
OPT_TRESSUPPORTED_mosaic=false
OPT_PROTOCOL_mosaic=http
OPT_RENDEVOUSIFACE_mosaic=bond0
OPT_REMOTEUSER_mosaic=root@

# Configuration for osprey
OPT_name_osprey=osprey.us.cray.com
OPT_LAUNCHERHOST_osprey=localhost
OPT_LAUNCHERPROTOCOL_osprey=http
OPT_CHECKPOINTPATH_osprey=/lus/scratch/foundation_engineering/determined-cp
OPT_DEBUGLEVEL_osprey=debug
OPT_MASTERHOST_osprey=osprey
OPT_MASTERPORT_osprey=$USERPORT
OPT_TRESSUPPORTED_osprey=false
OPT_PROTOCOL_osprey=http

# Configuration for swan
OPT_name_swan=swan.hpcrb.rdlabs.ext.hpe.com
OPT_LAUNCHERHOST_swan=localhost
OPT_LAUNCHERPROTOCOL_swan=http
OPT_CHECKPOINTPATH_swan=/lus/scratch/foundation_engineering/determined-cp
OPT_MASTERHOST_swan=swan
OPT_MASTERPORT_swan=$USERPORT
OPT_TRESSUPPORTED_swan=false
OPT_PROTOCOL_swan=http

# Configuration for raptor
OPT_name_raptor=raptor.hpcrb.rdlabs.ext.hpe.com
OPT_LAUNCHERHOST_raptor=localhost
OPT_LAUNCHERPROTOCOL_raptor=http
OPT_CHECKPOINTPATH_raptor=/lus/scratch/foundation_engineering/determined-cp
OPT_MASTERHOST_raptor=raptor
OPT_MASTERPORT_raptor=$USERPORT
OPT_TRESSUPPORTED_raptor=false
OPT_PROTOCOL_raptor=http

# Configuration for Genoble system o184i023 aka champollion (see http://o184i124.gre.smktg.hpecorp.net/~bench/)
# Need to request account to access these systems.   Managed GPUs include:
#
#  20 XL675d Gen10+ servers with 2x AMD EPYC 7763 processors (64c/2.45GHz/280W), 1TB/2TB RAM DDR4-3200 2R, 1x 3.2TB NVMe disk, 4x IB HDR ports, 8x NVIDIA A100/80GB SXM4 GPUs (*) -- aka 'Champollion'
#  4 XL270d Gen10 servers with Intel Cascade Lake Gold 6242 processors (16c/2.8GHz/150W), 384-768GB RAM DDR4-2400 2R, 1x basic 6G SFF SATA disk, 4x IB EDR ports, 8x NVIDIA V100/32GB SXM2 32GB GPUs (*)
#  1 XL675d Gen10+ server with 2x AMD EPYC 7543 processors (32c/2.8GHz/225W), 2TB RAM DDR4-3200 2R, 1x 1TB SFF SSD disk, 4x IB HDR ports, 10x NVIDIA A100/40GB PCIe GPUs (*)
#  1 XL675d Gen10+ server with 2x AMD EPYC 7763 processors (64c/2.45GHz/280W), 512GB RAM DDR4-3200 2R, 1x 1TB SFF SSD disk, 4x IB HDR ports, 8x AMD Mi210 PCIe/XGMI GPUs (*)
#  1 XL645d Gen10+ server with 2x AMD EPYC 7702 processors (64c/2.0GHz/200W), 512GB RAM DDR4-2666 2R, 1x 1.6TB NVMe disk, 1x IB EDR port, 4x NVIDIA A100/40GB PCIe GPUs (*)
#
OPT_name_o184i023=16.16.184.23
OPT_LAUNCHERHOST_o184i023=localhost
OPT_LAUNCHERPORT_o184i023=8181
OPT_LAUNCHERPROTOCOL_o184i023=http
OPT_CHECKPOINTPATH_o184i023=/cstor/harrow/determined-cp
OPT_MASTERHOST_o184i023=o184i023
OPT_MASTERPORT_o184i023=$USERPORT
OPT_TRESSUPPORTED_o184i023=false
OPT_GRESSUPPORTED_o184i023=false
OPT_PROTOCOL_o184i023=http
OPT_SLOTTYPE_o184i023=rocm

# This is the list of options that can be injected into devcluster-slurm.yaml
# If a value is not configured for a specific target cluster, it will be
# blank and get the default value.   OPT_TASKCONTAINERDEFAULTS & OPT_PARTITIONOVERRIDES
# are multi-line values and must match the indentation of the associated
# section in devcluster-slurm.yaml.   See OPT_TASKCONTAINERDEFAULTS_sawmill as
# an example of how to provide such multi-line values.
export OPT_LAUNCHERHOST=$(lookup "OPT_LAUNCHERHOST_$CLUSTER")
export OPT_LAUNCHERPORT=$(lookup "OPT_LAUNCHERPORT_$CLUSTER")
export OPT_LAUNCHERPROTOCOL=$(lookup "OPT_LAUNCHERPROTOCOL_$CLUSTER")
export OPT_CHECKPOINTPATH=$(lookup "OPT_CHECKPOINTPATH_$CLUSTER")
export OPT_MASTERHOST=$(lookup "OPT_MASTERHOST_$CLUSTER")
export OPT_MASTERPORT=$(lookup "OPT_MASTERPORT_$CLUSTER")
export OPT_TRESSUPPORTED=$(lookup "OPT_TRESSUPPORTED_$CLUSTER")
export OPT_GRESSUPPORTED=$(lookup "OPT_GRESSUPPORTED_$CLUSTER")
export OPT_RENDEVOUSIFACE=$(lookup "OPT_RENDEVOUSIFACE_$CLUSTER")
export OPT_REMOTEUSER=$(lookup "OPT_REMOTEUSER_$CLUSTER")
export OPT_SLOTTYPE=$(lookup "OPT_SLOTTYPE_$CLUSTER")
export OPT_DEFAULTIMAGE=$(lookup "OPT_DEFAULTIMAGE_$CLUSTER")
export OPT_TASKCONTAINERDEFAULTS=$(lookup "OPT_TASKCONTAINERDEFAULTS_$CLUSTER")
export OPT_PARTITIONOVERRIDES=$(lookup "OPT_PARTITIONOVERRIDES_$CLUSTER")

if [[ -z $OPT_GRESSUPPORTED ]]; then
    export OPT_GRESSUPPORTED="true"
fi

if [[ -n $DEFAULTIMAGE ]]; then
    OPT_DEFAULTIMAGE=$DEFAULTIMAGE
fi

if [[ -n $DEVLAUNCHER ]]; then
    if [ -z $DEV_LAUNCHER_PORT ]; then
        echo >&2 "$0: User $USERNAME does not have a configured DEV_LAUNCHER_PORT, update the script."
        exit 1
    fi
    OPT_LAUNCHERPORT=$DEV_LAUNCHER_PORT
    # Currently devlauncher support config above only has http ports
    OPT_LAUNCHERPROTOCOL=http
fi

SLURMCLUSTER=$(lookup "OPT_name_$CLUSTER")
if [[ -z $SLURMCLUSTER ]]; then
    echo >&2 "$0: Cluster name $CLUSTER does not have a configuration. Specify one of:"
    echo >&2 "$(
        set -o posix
        set | grep OPT_name | cut -f 1 -d = | cut -c 10-
    )"
    exit 1
fi

if [[ -z $OPT_LAUNCHERPORT ]]; then
    echo >&2 "$0: Cluster name $CLUSTER does not have an installed launcher, specify -d to utilize a dev launcher."
    exit 1
fi

if [[ -z $INTUNNEL ]]; then
    OPT_LAUNCHERHOST=$SLURMCLUSTER
fi

if [[ -n $PULL_AUTH ]]; then
    pull_auth_token ${OPT_REMOTEUSER}$SLURMCLUSTER $CLUSTER
fi

export OPT_DEBUGLEVEL=$DEBUGLEVEL

if [[ -n $PODMAN ]]; then
    export OPT_CONTAINER_RUN_TYPE='podman'
fi

if [[ -n $ENROOT ]]; then
    export OPT_CONTAINER_RUN_TYPE='enroot'
fi

if [[ -r ~/.${CLUSTER}.token ]]; then
    export OPT_AUTHFILE=~/.${CLUSTER}.token
fi

echo
echo "Configuration Used:"
printenv | grep OPT_
echo

# Terminate our tunnels on exit
trap "kill 0" EXIT
if [[ -n $INTUNNEL || -n $TUNNEL ]]; then
    # Terminate any tunnels (non-interactive sshd proceses for the user)
    ssh $OPT_MASTERHOST pkill -u '$USER' -x -f '"^sshd: $USER[ ]*$"'
fi
if [[ -n $INTUNNEL ]]; then
    mkintunnel $OPT_MASTERHOST $OPT_LAUNCHERPORT ${OPT_REMOTEUSER}$SLURMCLUSTER &
fi
if [[ -n $TUNNEL ]]; then
    mktunnel $OPT_MASTERHOST $OPT_MASTERPORT ${OPT_REMOTEUSER}$SLURMCLUSTER &
fi
# Give a little time for the tunnels to setup before using
sleep 3

# Although devcluster supports variables, numeric values fail to load, so
# Manually apply those into a temp file.
TEMPYAML=/tmp/devcluster-$CLUSTER.yaml
rm -f $TEMPYAML
envsubst <tools/devcluster-slurm.yaml >$TEMPYAML
echo "INFO: Generated devcluster file: $TEMPYAML"
devcluster -c $TEMPYAML --oneshot
