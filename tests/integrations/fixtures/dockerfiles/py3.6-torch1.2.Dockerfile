FROM nvidia/cuda@sha256:ffde9e9ad005eb9e1fa20238371eb5a535863554d331396d7c63e26f7a2d06e0
MAINTAINER Determined AI <hello@determined.ai>

# NVIDIA APT repositories have been flaky in the past. Remove them so that
# Docker builds don't hang on `apt-get update`.
RUN rm /etc/apt/sources.list.d/*

# Ubuntu 16.04 provides Python 3.5 as `python3`, so we use a PPA to
# install Python 3.6 as `python3.6`. Determined code should be careful to
# invoke `python3.6`, not `python3` or `python` (the former is Python
# 3.5, the latter does not exist).
RUN  apt-get update \
  && apt-get -y --no-install-recommends install software-properties-common \
  && add-apt-repository ppa:deadsnakes/ppa \
  && apt-get update \
  && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
     build-essential \
     ca-certificates \
     curl \
     libkrb5-dev \
     libssl-dev \
     git \
     krb5-user \
     python3.6=3.6.10-1+xenial1 \
     python3.6-dev=3.6.10-1+xenial1 \
  && apt-get clean

# Ensure that Python outputs everything from the application rather than
# buffering it.
ENV PYTHONUNBUFFERED 1

# Upgrade pip.
RUN curl -O https://bootstrap.pypa.io/get-pip.py && \
    python3.6 get-pip.py && \
    rm get-pip.py

RUN ln -s /usr/bin/python3.6 /usr/bin/python
ARG TORCH_BUILD
RUN if [ "x$TORCH_BUILD" = "xgpu" ] ; then pip install torch==1.2.0 torchvision==0.4.0; else pip install torch==1.2.0+cpu torchvision==0.4.0+cpu -f https://download.pytorch.org/whl/torch_stable.html; fi
