#!/bin/bash

set -o pipefail
set -e

prog="$0"

print_help() {
    echo "usage: $prog [OPTIONS] OUTDIR SERVICE_ACCOUNT

where OPTIONS may be any of:
  -h, --help             Show this output.
  -c, --context CONTEXT  Specify a kubectl context to use.
                         Default: use the kubectl default context.
  -s, --static           Fetch static credentials once and exit.
                         This is mostly useful for k8s before v1.21
                         when the TokenRequest API feature was not
                         yet GA.
                         Default: fetch a fresh token every minute.
  -p, --period PERIOD    Configure the wait time between refreshes.
                         PERIOD will be passed to the sleep command.
                         Default: 60
  -v, --verbose          Log more stuff to stdout.

This script uses kubectl to fetch suitable credentials for a
particular service account, which are stored in OUTDIR.

By default, $prog will run in a loop and continuously refresh the
credentials in OUTDIR.

If CONTEXT is not provided, the default k8s context will be used.

The resulting OUTDIR will look like:

    OUTDIR
    ├── ca.crt           # a file expected by InClusterConfig()
    ├── token            # a file expected by InClusterConfig()
    ├── docker-env-file  # for docker run --env-file
    └── server           # for k8s apps without InClusterConfig()

Afterwards, there are two recommended ways to use OUTDIR:

  - Modify a k8s application to read from OUTDIR directly
    (for example, determined-master does this)

  - Run an unmodified k8s application via docker run:

      docker run \\
          -v OUTDIR:/var/run/secrets/kubernetes.io/serviceaccount \\
          --env-file OUTDIR/docker-env-file \\
          my_image_name"
}

context=""
static=n
period="60"
verbose=n
outdir=""
account=""

# Manually parse args since getopt varies across unices.
while test -n "$1"; do
    case "$1" in
        # flags
        --help) print_help && exit 0 ;;
        -h) print_help && exit 0 ;;
        --context)
            context="$2"
            shift
            shift
            ;;
        -c)
            context="$2"
            shift
            shift
            ;;
        --static)
            static=y
            shift
            ;;
        -s)
            static=y
            shift
            ;;
        --period)
            period="$2"
            shift
            shift
            ;;
        -p)
            period="$2"
            shift
            shift
            ;;
        --verbose)
            verbose=y
            shift
            ;;
        -v)
            verbose=y
            shift
            ;;

        -*) echo "unrecognized flag: $1" >&2 && exit 1 ;;

        # positional arguments
        *)
            if [ -z "$outdir" ]; then
                outdir="$1"
            elif [ -z "$account" ]; then
                account="$1"
            else
                echo "too many positional arguments!" >&2
                print_help >&2
                exit 1
            fi
            shift
            ;;
    esac
done

# detect required external tools (after processing --help)
ok=y
need_bin() {
    if ! which "$1" >/dev/null 2>/dev/null; then
        echo "missing required executable: $1"
        ok=n
    fi
}
need_bin kubectl
need_bin jq
test "$ok" = n && exit 1

if [ -z "$account" ]; then
    echo "too few positional arguments!" >&2
    print_help >&2
    exit 1
fi

if [ -z "$context" ]; then
    # by default, pick the current-context from kubectl config
    context="$(kubectl config view -o json | jq -r '."current-context"')"
fi

# first extract server information from the kubectl config
server_info="$(
    kubectl config view --raw -o json \
        | jq -r '.clusters[] | select(.name=="'"$context"'")'
)"
server_url="$(echo "$server_info" | jq -r '.cluster.server')"
if echo "$server_info" | jq -e '.cluster."certificate-authority"' >/dev/null; then
    ca_file="$(echo "$server_info" | jq -r '.cluster."certificate-authority"')"
    ca_crt="$(<$ca_file)"
else
    ca_crt="$(
        echo "$server_info" \
            | jq -r '.cluster."certificate-authority-data"' \
            | base64 -d
    )"
fi

mkdir -p "$outdir"

# Write the server info to OUTDIR.
echo "$server_url" >"$outdir/server"
echo "$ca_crt" >"$outdir/ca.crt"

host="$(echo "$server_url" | sed -e 's|[^/]*//||; s/:[0-9]*$//')"

# port is tricky; handle scheme://host:port and scheme://host
port="$(echo "$server_url" | sed -e 's/^[^:]*:[^:]*://')"
if [ "$port" = "$server_url" ]; then
    # scheme://host case (it didn't have enough colons to match the regex)
    if echo "$port" | grep -q '^https'; then
        port="443"
    else
        port="80"
    fi
fi

# --host=network mode doesn't work on macos
if [ "$(uname)" = "Darwin" ] && [ "$host" = "127.0.0.1" ]; then
    dockerhost="host.docker.internal"
else
    dockerhost="$host"
fi

# Write a suitable --env-file for a docker run command.  These variables are
# what an unmodified k8s client will expect to see in its environment, when it
# calls rest.InClusterConfig().
cat >"$outdir/docker-env-file" <<EOF
KUBERNETES_SERVICE_HOST=$dockerhost
KUBERNETES_SERVICE_PORT=$port
EOF

if [ "$static" = "y" ]; then
    # User specified a static tokens instead of the TokenRequest API.

    secret_name="$(
        kubectl get serviceaccounts "$account" -o json | jq -r .secrets[0].name
    )"

    # Make sure a secret was found.
    if [ "$secret_name" = "null" ]; then
        echo "no secret found for service account \"$account\", is this a" >&2
        echo "k8s installation with static service account keys enabled?" >&2
        echo "(static service account keys are not created by default" >&2
        echo "starting with k8s v1.24)" >&2
        exit 1
    fi

    # Write the token to OUTDIR.
    kubectl --context "$context" get secrets "$secret_name" -o json \
        | jq -r '.data."token"' | base64 -d \
        >"$outdir/token"

    echo "$prog: success!"
    if [ "$verbose" = "y" ]; then
        echo -e "\x1b[31mserver:\x1b[m"
        cat "$outdir/server"
        echo -e "\x1b[31mca.crt:\x1b[m"
        cat "$outdir/ca.crt"
        echo -e "\x1b[31mtoken:\x1b[m"
        cat "$outdir/token"
    fi
    exit 0
fi

dump_token() {
    # token is an rfc 7515 JWS in compressed serialization format.
    # The format is "$b64url_protected.$b64url_payload.$b64url_signature".
    token="$1"
    # First, extract the payload between the two '.'s.
    payload="$(echo "$token" | sed -e 's/^[^.]*\.//; s/\.[^.]*$//;')"
    # Swap out characters from b64url encoding to standard b64 encoding.
    payload="$(echo "$payload" | sed -e 's/\+/-/; s/_/\//')"
    # Add standard b64 padding, which b64url encoding skips.
    padding="$(
        echo "$payload" \
            | sed -e 's/....//g ; s/^.$/===/; s/^..$/==/; s/^[^=]../=/'
    )"
    echo "Got token:"
    echo "$payload$padding" | base64 -d | jq -C
}

refresh_token() {
    token="$(kubectl --context "$context" create token "$account")"
    if [ "$verbose" = "y" ]; then
        dump_token "$token"
    fi
    echo "$token" >"$outdir/token"
}

refresh_token
echo "$prog: successfully fetched initial token; starting refresh loop..."

while true; do
    # HACK: close stdout and stderr to avoid hangs in devcluster after SIGKILL.
    # This will leave an orphaned sleep process, but that's pretty harmless.
    sleep "$period" >/dev/null 2>/dev/null
    if ! refresh_token; then
        echo "$prog: failed to refresh k8s token, will try again later..."
    fi
done
