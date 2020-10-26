FROM debian:10.3-slim
MAINTAINER Determined AI <hello@determined.ai>

RUN apt-get update -yq \
    && apt-get install -y ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY packaging/master.yaml /etc/determined/master.yaml
COPY packaging/LICENSE /usr/share/doc/determined-master/copyright
COPY build /usr/share/determined/master
COPY static /usr/share/determined/master/static
COPY determined-master /usr/bin/determined-master

EXPOSE 8080
ENTRYPOINT ["/usr/bin/determined-master"]
