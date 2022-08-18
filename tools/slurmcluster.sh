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
# Pre-requisites:
#   1) Configure your USERPORT_${USER} port below using your login name on
#      the desktop that you are using.
#
#   2)  Unless you specify both -n -x, you must have password-less ssh configured to the
#   target cluster, to enable the ssh connection without prompts.
#
#      ssh-copy-id {cluster}
#
#   3) To utilize the -a option to retrieve the /opt/launcher/jetty/base/etc/.launcher.token
#   you must have sudo access on the cluster, and authenticate the sudo that will be
#   exectued to retrieve the .launcher.token.

HELPEND=$(($LINENO - 1))


INTUNNEL=1
TUNNEL=1
DEVLAUNCHER=

if [[ $1 == '-n' ]]; then
    INTUNNEL=
    shift
fi
if [[ $1 == '-x' ]]; then
    TUNNEL=
    shift
fi
if [[ $1 == '-t' ]]; then
    TRACE=1
    shift
fi
if [[ $1 == '-p' ]]; then
    PODMAN=1
    shift
fi
if [[ $1 == '-d' ]]; then
    DEVLAUNCHER=1
    shift
fi
if [[ $1 == '-a' ]]; then
    PULL_AUTH=1
    shift
fi

if [[ $1 == '-h' || $1 == '--help' || -z $1 ]] ; then
    echo "Usage: $0 [-h] [-n] [-x] [-t] [-p] [-d] [-a] {cluster}"
    echo "  -h     This help message.   Options are order sensitive."
    echo "  -n     Disable start of the inbound tunnel (when using Cisco AnyConnect)."
    echo "  -x     Disable start of personal tunnel back to master (if you have done so manually)."
    echo "  -t     Force debug level to trace regardless of cluster configuration value."
    echo "  -p     Use podman as a container host (otherwise singlarity)."
    echo "  -d     Use a developer launcher (port assigned for the user in loadDevLauncher.sh)."
    echo "  -a     Attempt to retrieve the .launcher.token - you must have sudo root on the cluster."
    echo
    echo "Documentation:"
    head -n $HELPEND $0 | tail -n $(($HELPEND  - 1))
    exit 1
fi

CLUSTER=$1
CLUSTERS=('casablanca'  'mosaic' 'osprey'  'shuco' 'horizon' 'swan' 'casablanca-login' 'casablanca-mgmt1', 'raptor', 'casablanca-login2')

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

    echo  "Attempting to access /opt/launcher/jetty/base/etc/.launcher.token from $HOST"
    rm -f ~/.token.log
    ssh -t $HOST 'sudo cat /opt/launcher/jetty/base/etc/.launcher.token' | tee ~/.token.log
    # Token is the last line of the output (no newline)
    TOKEN=$(tail -n 1 ~/.token.log)
    echo -n "${TOKEN}" > ~/.${CLUSTER}.token
    cat ~/.${CLUSTER}.token
}

# Update your username/port pair
USERPORT_stokc=8084
USERPORT_rcorujo=8085
USERPORT_phillipgaisford=8086
USERPORT_pankaj=8087
USERPORT_alyssa=8088
USERPORT_charlestran=8089
USERPORT_jerryharrow=8090
USERPORT_cameronquilici=8093
USERPORT_cobble=8092

USERPORT=$(lookup "USERPORT_$USER")
if [[ -z $USERPORT ]]; then
    echo "$0: User $USER does not have a configured port, update the script."
    exit 1
fi

if [[ $CLUSTER == "casablanca-login" ]]; then
   CLUSTER=casablanca_login
elif [[ $CLUSTER == "casablanca-mgmt1" ]]; then
   CLUSTER=casablanca
elif [[ $CLUSTER == "casablanca-login2" ]]; then
   CLUSTER=casablanca_login2
elif [[ ! " ${CLUSTERS[*]} " =~ " $CLUSTER "  ]]; then
    echo "$0: Cluster name $CLUSTER does not have a configuration.  Specify one of: ${CLUSTERS[*]}"
    exit 1
fi

# Update your JETTY HTTP/SSL username/port pair from loadDevLauncher.sh
DEV_LAUNCHER_PORT_stokc=18084
DEV_LAUNCHER_PORT_rcorujo=18085
DEV_LAUNCHER_PORT_phillipgaisford=18086
DEV_LAUNCHER_PORT_pankaj=18087
DEV_LAUNCHER_PORT_alyssa=18088
DEV_LAUNCHER_PORT_jerryharrow=18090
DEV_LAUNCHER_PORT_cobble=18092
DEV_LAUNCHER_PORT=$(lookup "DEV_LAUNCHER_PORT_$USER")

# Configuration for casablanca (really casablanca-mgmt1)
OPT_name_casablanca=casablanca-mgmt1.us.cray.com
OPT_LAUNCHERHOST_casablanca=localhost
OPT_LAUNCHERPORT_casablanca=8181
OPT_LAUNCHERPROTOCOL_casablanca=http
OPT_CHECKPOINTPATH_casablanca=/mnt/lustre/foundation_engineering/determined-cp
OPT_DEBUGLEVEL_casablanca=debug
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
OPT_DEBUGLEVEL_horizon=debug
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
OPT_DEBUGLEVEL_casablanca_login=debug
OPT_MASTERHOST_casablanca_login=casablanca-login
OPT_MASTERPORT_casablanca_login=$USERPORT
OPT_TRESSUPPORTED_casablanca_login=true

# Configuration for casablanca-login2 (uses suffix casablanca_login2)
OPT_name_casablanca_login2=casablanca-login2.us.cray.com
OPT_LAUNCHERHOST_casablanca_login2=localhost
OPT_LAUNCHERPORT_casablanca_login2=8443
OPT_LAUNCHERPROTOCOL_casablanca_login2=http
OPT_CHECKPOINTPATH_casablanca_login2=/mnt/lustre/foundation_engineering/determined-cp
OPT_DEBUGLEVEL_casablanca_login2=debug
OPT_MASTERHOST_casablanca_login2=casablanca-login2
OPT_MASTERPORT_casablanca_login2=$USERPORT
OPT_TRESSUPPORTED_casablanca_login2=true

# Configuration for shuco
OPT_name_shuco=shuco.us.cray.com
OPT_LAUNCHERHOST_shuco=localhost
OPT_LAUNCHERPORT_shuco=8181
OPT_LAUNCHERPROTOCOL_shuco=http
OPT_CHECKPOINTPATH_shuco=/home/launcher/determined-cp
OPT_DEBUGLEVEL_shuco=debug
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
OPT_DEBUGLEVEL_mosaic=debug
OPT_MASTERHOST_mosaic=10.30.91.220
OPT_MASTERPORT_mosaic=$USERPORT
OPT_TRESSUPPORTED_mosaic=false
OPT_PROTOCOL_mosaic=http
OPT_RENDEVOUSIFACE_mosaic=bond0
OPT_REMOTEUSER_mosaic=root@

# Configuration for osprey
OPT_name_osprey=osprey.us.cray.com
OPT_LAUNCHERHOST_osprey=localhost
OPT_LAUNCHERPORT_osprey=8181
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
OPT_LAUNCHERPORT_swan=8181
OPT_LAUNCHERPROTOCOL_swan=http
OPT_CHECKPOINTPATH_swan=/lus/scratch/foundation_engineering/determined-cp
OPT_DEBUGLEVEL_swan=debug
OPT_MASTERHOST_swan=swan
OPT_MASTERPORT_swan=$USERPORT
OPT_TRESSUPPORTED_swan=false
OPT_PROTOCOL_swan=http

# Configuration for raptor
OPT_name_raptor=raptor.hpcrb.rdlabs.ext.hpe.com
OPT_LAUNCHERHOST_raptor=localhost
OPT_LAUNCHERPORT_raptor=8181
OPT_LAUNCHERPROTOCOL_raptor=http
OPT_CHECKPOINTPATH_raptor=/lus/scratch/foundation_engineering/determined-cp
OPT_DEBUGLEVEL_raptor=debug
OPT_MASTERHOST_raptor=raptor
OPT_MASTERPORT_raptor=$USERPORT
OPT_TRESSUPPORTED_raptor=false
OPT_PROTOCOL_raptor=http

export OPT_LAUNCHERHOST=$(lookup "OPT_LAUNCHERHOST_$CLUSTER")
export OPT_LAUNCHERPORT=$(lookup "OPT_LAUNCHERPORT_$CLUSTER")
export OPT_LAUNCHERPROTOCOL=$(lookup "OPT_LAUNCHERPROTOCOL_$CLUSTER")
export OPT_CHECKPOINTPATH=$(lookup "OPT_CHECKPOINTPATH_$CLUSTER")
export OPT_DEBUGLEVEL=$(lookup "OPT_DEBUGLEVEL_$CLUSTER")
export OPT_MASTERHOST=$(lookup "OPT_MASTERHOST_$CLUSTER")
export OPT_MASTERPORT=$(lookup "OPT_MASTERPORT_$CLUSTER")
export OPT_TRESSUPPORTED=$(lookup "OPT_TRESSUPPORTED_$CLUSTER")
export OPT_RENDEVOUSIFACE=$(lookup "OPT_RENDEVOUSIFACE_$CLUSTER")
export OPT_REMOTEUSER=$(lookup "OPT_REMOTEUSER_$CLUSTER")

if [[ -n $DEVLAUNCHER ]]; then
    if [ -z $DEV_LAUNCHER_PORT ]; then
        echo "$0: User $USER does not have a configured DEV_LAUNCHER_PORT, update the script."
        exit 1
    fi
    OPT_LAUNCHERPORT=$DEV_LAUNCHER_PORT
fi


SLURMCLUSTER=$(lookup "OPT_name_$CLUSTER")
if [[ -z $SLURMCLUSTER ]]; then
    echo "$0: Cluster name $CLUSTER does not have a configuration. Specify one of: $(set -o posix; set | grep OPT_name | cut -f 2 -d =)."
    exit 1
fi

if [[ -z $INTUNNEL ]]; then
    OPT_LAUNCHERHOST=$SLURMCLUSTER
fi

if [[ -n $PULL_AUTH ]]; then
    pull_auth_token ${OPT_REMOTEUSER}$SLURMCLUSTER $CLUSTER
fi


if [[ -n $TRACE ]]; then
    export OPT_DEBUGLEVEL=trace
fi

if [[ -n $PODMAN ]]; then
    export OPT_CONTAINER_RUN_TYPE='podman'
fi

if [[ -r ~/.${CLUSTER}.token ]]; then
    export OPT_AUTHFILE=~/.${CLUSTER}.token
fi

echo
echo "Configuration Used:"
printenv |grep OPT_
echo

# Terminate our tunnels on exit
trap "kill 0" EXIT
if [[ -n $INTUNNEL ]]; then
   mkintunnel  $OPT_MASTERHOST $OPT_LAUNCHERPORT ${OPT_REMOTEUSER}$SLURMCLUSTER &
fi
if [[ -n $TUNNEL ]]; then
   mktunnel $OPT_MASTERHOST $OPT_MASTERPORT ${OPT_REMOTEUSER}$SLURMCLUSTER &
fi


# Although devcluster supports variables, numeric values fail to load, so
# Manually apply those into a temp file.
TEMPYAML=/tmp/devcluster-$CLUSTER.yaml
rm -f $TEMPYAML
envsubst <  tools/devcluster-slurm.yaml  > $TEMPYAML
devcluster -c $TEMPYAML --oneshot
