FROM debian:10.3-slim
MAINTAINER Determined AI <hello@determined.ai>

RUN apt-get update -yq \
    && apt-get install -y ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY packaging/master.yaml /etc/determined/master.yaml
COPY build/share/determined/master /usr/share/determined/master
COPY determined-master /usr/bin/determined-master

EXPOSE 8080
ENTRYPOINT ["/usr/bin/determined-master"]
