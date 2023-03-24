FROM ubuntu:22.04
MAINTAINER Determined AI <hello@determined.ai>

RUN apt-get update -yq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      ca-certificates \
      unattended-upgrades \
    && unattended-upgrade \
    && apt-get remove -y unattended-upgrades \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/*

COPY packaging/master.yaml /etc/determined/master.yaml
COPY packaging/LICENSE /usr/share/doc/determined-master/copyright
COPY build /usr/share/determined/master
COPY static /usr/share/determined/master/static
COPY determined-gotmpl /usr/bin/determined-gotmpl
COPY determined-master /usr/bin/determined-master
RUN chmod 775 /var/cache  # for OpenShift/OKD compatibility

EXPOSE 8080
ENTRYPOINT ["/usr/bin/determined-master"]
