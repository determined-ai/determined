#!/bin/sh

# First detect any user-provided det-orchestrator on the PATH.
bin="$(which)"
if which det-orchestrator >/dev/null 2>/dev/null ; then
    exec det-orchestrator "$@"
fi

# Otherwise try to use one of the pre-builts based on detectect architecture.
# Based on entries from https://stackoverflow.com/a/45125525.
arch="$(uname -m)"
prebuilt="/run/determined/workdir/orchestrator"
case "$arch" in
    # amd64 variants
    x86_64) exec "$prebuilt.amd64" "$@";;
    amd64)  exec "$prebuilt.amd64" "$@";;

    # aarch64 variants
    aarch64_be) exec "$prebuilt.arm64" "$@";;
    aarch64)    exec "$prebuilt.arm64" "$@";;
    armv8b)     exec "$prebuilt.arm64" "$@";;
    armv8l)     exec "$prebuilt.arm64" "$@";;

    # powerpc64 variants
    ppc64le) exec "$prebuilt.ppc64" "$@";;
    ppc64)   exec "$prebuilt.ppc64" "$@";;
esac

# We didn't recognize the architecture.  Try and tell the user.
msg="
Unable to find a suitable det-orchestrator.  The det-orchestrator is responsible
for high-performance log forwarding from inside a container to the cluster, and
it ensures that logs function the same in determined across many different
cluster backends (determined-agents, k8s, slurm, etc).

Normally we choose a prebuilt det-orchestrator based on the output of uname -m.
In this container, uname -m returns '$arch', which is not a supported value. i

Prebuilt det-orchestrator binaries exist for the following values of uname -m:
  - x86_64
  - amd64
  - aarch64_be
  - aarch64
  - armv8b
  - armv8l
  - ppc64le
  - ppc64

For other architectures, you can compile det-orchestrator and put it the PATH
in the image.
"
python ./fallback-logs.py "$msg"
exit 43
