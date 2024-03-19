FROM ubuntu:22.04
MAINTAINER Determined AI <hello@determined.ai>

RUN curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg \
    && echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list \
    && apt-get update -yq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      ca-certificates \
      unattended-upgrades \
      google-cloud-sdk-gke-gcloud-auth-plugin \
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
