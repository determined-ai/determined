FROM debian:10.3-slim
MAINTAINER Determined AI <hello@determined.ai>

RUN apt-get update && apt-get install -y perl-base
# For non-CUDA based images, these variables must be explicitly set for the Nvidia runtime to expose
# the GPUs and `nvidia-smi` binary inside the container. See
# https://github.com/NVIDIA/nvidia-container-runtime#environment-variables-oci-spec.
ENV NVIDIA_VISIBLE_DEVICES=all NVIDIA_DRIVER_CAPABILITIES=utility

# For reasons that are not totally clear, this is sometimes necessary for the agent to be able to
# run `nvidia-smi` successfully (see https://github.com/NVIDIA/nvidia-docker/issues/854 for
# discussion and the origin of this workaround). Even when not necessary, it should be harmless.
RUN ldconfig

COPY packaging/LICENSE /usr/share/doc/determined-agent/copyright
COPY packaging/entrypoint.sh /run/determined/workdir/entrypoint.sh
RUN chmod +x /run/determined/workdir/entrypoint.sh

COPY determined-ee-agent /usr/bin/determined-agent

WORKDIR /run/determined/workdir
ENTRYPOINT [ "/run/determined/workdir/entrypoint.sh" ]
CMD ["run"]
