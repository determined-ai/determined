FROM debian:10.3-slim
MAINTAINER Determined AI <hello@determined.ai>
ARG VERSION
EXPOSE 8080

RUN apt-get update -yq \
    && apt-get install -y ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY dist/determined-master_${VERSION}_linux_amd64.deb /tmp/master.deb
RUN apt-get install -y /tmp/master.deb

WORKDIR /usr/share/determined/master
ENTRYPOINT ["/usr/bin/determined-master"]
