FROM debian:10.3-slim
MAINTAINER Determined AI <hello@determined.ai>

RUN apt-get update -yq \
    && apt-get upgrade -s \
        | grep "^Inst" \
        | grep -e "Debian" -e "Debian-Security" \
        | awk -F " " {'print $2'} \
        | DEBIAN_FRONTEND=noninteractive xargs apt-get upgrade -yq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -yq \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY packaging/master.yaml /etc/determined/master.yaml
COPY packaging/LICENSE /usr/share/doc/determined-master/copyright
COPY build /usr/share/determined/master
COPY static /usr/share/determined/master/static
COPY determined-gotmpl /usr/bin/determined-gotmpl
COPY determined-master /usr/bin/determined-master

EXPOSE 8080
ENTRYPOINT ["/usr/bin/determined-master"]
