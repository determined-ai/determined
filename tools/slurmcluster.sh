#!/bin/bash
# 
# This dev script is a wrapper on the devcluster tool, and provides
# per-user and per cluster configuration of the devcluster-slurm.yaml
# file to enable it to be used for our various clusters.  It dynamically
# fills in the variables within devcluster-slurm.yaml such that the original
# source need not be modified.   By default it also starts/stops SSH
# tunnels inbound to launcher, and outbound to the desktop master.
#
# Pre-requisites:
#   1) Configure your USERPORT_${USER} port below using your login name on
#      the desktop that you are using.
#   2)  Unless you specify both -n -x, you must have password-less ssh configured to the
#   target cluster, to enable the ssh connection without prompts.
#
#      ssh-copy-id {cluster}
#
INTUNNEL=1
TUNNEL=1
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

if [[ $1 == '-h' || $1 == '--help' || -z $1 ]] ; then
    echo "Usage: $0 [-h] [-n] [-x] [-t] {cluster}"
    echo "  -h     This help message.   Options are order sensitive."
    echo "  -n     Disable start of the inbound tunnel (when using Cisco AnyConnect)."
    echo "  -x     Disable start of personal tunnel back to master (if you have done so manually)."
    echo "  -t     Force debug level to trace regardless of cluster configuration value."
    echo "  -p     Use podman as a container host (otherwise singlarity)"
    echo 
    echo "Documentation:"
    head -n 17 $0
    exit 1
fi

CLUSTER=$1

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
if [ -z $USERPORT ]; then
    echo "$0: User $USER does not have a configured port, update the script."
    exit 1
fi

if [ $CLUSTER == "casablanca-login" ]; then
   CLUSTER=casablanca_login
elif [ $CLUSTER != "casablanca" -a $CLUSTER != "horizon"  -a $CLUSTER != "shuco"  ]; then
    echo "$0: Cluster name $CLUSTER does not have a configuration.  Specify one of: casablanca, casablanca-login, horizon, shuco"
    exit 1
fi

# Configuration for casablanca
OPT_name_casablanca=casablanca.us.cray.com
OPT_LAUNCHERHOST_casablanca=localhost
OPT_LAUNCHERPORT_casablanca=8181
OPT_LAUNCHERPROTOCOL_casablanca=http
OPT_CHECKPOINTPATH_casablanca=/lus/scratch/foundation_engineering/determined-cp
OPT_DEBUGLEVEL_casablanca=debug
OPT_MASTERHOST_casablanca=casablanca
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
OPT_CHECKPOINTPATH_casablanca_login=/lus/scratch/foundation_engineering/determined-cp
OPT_DEBUGLEVEL_casablanca_login=debug
OPT_MASTERHOST_casablanca_login=casablanca-login
OPT_MASTERPORT_casablanca_login=$USERPORT
OPT_TRESSUPPORTED_casablanca_login=true

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

export OPT_LAUNCHERHOST=$(lookup "OPT_LAUNCHERHOST_$CLUSTER")
export OPT_LAUNCHERPORT=$(lookup "OPT_LAUNCHERPORT_$CLUSTER")
export OPT_LAUNCHERPROTOCOL=$(lookup "OPT_LAUNCHERPROTOCOL_$CLUSTER")
export OPT_CHECKPOINTPATH=$(lookup "OPT_CHECKPOINTPATH_$CLUSTER")
export OPT_DEBUGLEVEL=$(lookup "OPT_DEBUGLEVEL_$CLUSTER")
export OPT_MASTERHOST=$(lookup "OPT_MASTERHOST_$CLUSTER")
export OPT_MASTERPORT=$(lookup "OPT_MASTERPORT_$CLUSTER")
export OPT_TRESSUPPORTED=$(lookup "OPT_TRESSUPPORTED_$CLUSTER")
export OPT_RENDEVOUSIFACE=$(lookup "OPT_RENDEVOUSIFACE_$CLUSTER")

SLURMCLUSTER=$(lookup "OPT_name_$CLUSTER")
if [[ -z $SLURMCLUSTER ]]; then
    echo "$0: Cluster name $CLUSTER does not have a configuration. Specify one of: $(set -o posix; set | grep OPT_name | cut -f 2 -d =)."
    exit 1
fi

if [[ -z $INTUNNEL ]]; then
    OPT_LAUNCHERHOST=$SLURMCLUSTER
fi

if [[ -n $TRACE ]]; then
    export OPT_DEBUGLEVEL=trace
fi

if [[ -n $PODMAN ]]; then
    export OPT_CONTAINER_RUN_TYPE='podman'
fi


echo
echo "Configuration Used:"
printenv |grep OPT_
echo 

# Terminate our tunnels on exit
trap "kill 0" EXIT
if [[ -n $INTUNNEL ]]; then
   mkintunnel  $OPT_MASTERHOST $OPT_LAUNCHERPORT $SLURMCLUSTER &
fi
if [[ -n $TUNNEL ]]; then
   mktunnel $OPT_MASTERHOST $OPT_MASTERPORT $SLURMCLUSTER &
fi


# Although devcluster supports variables, numeric values fail to load, so
# Manually apply those into a temp file.
TEMPYAML=/tmp/devcluster-$CLUSTER.yaml
rm -f $TEMPYAML
envsubst <  tools/devcluster-slurm.yaml  > $TEMPYAML
devcluster -c $TEMPYAML --oneshot
