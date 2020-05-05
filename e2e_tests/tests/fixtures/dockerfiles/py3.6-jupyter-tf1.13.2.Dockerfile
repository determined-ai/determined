# tensorflow/tensorflow:1.13.2-py3-jupyter
FROM tensorflow/tensorflow@sha256:876f0633e72035efc026149826e0de857e98eb4540c726131cdeac8a9b388e02
RUN apt-get update && apt-get install -y \
  git \
  nano \
  vim \
  curl \
  wget

WORKDIR /
ENV DEBIAN_FRONTEND=noninteractive

# configure locale
RUN apt-get update
# make sure that locales package is available
RUN apt-get install --reinstall -y locales
# uncomment chosen locale to enable it's generation
RUN sed -i 's/# pl_PL.UTF-8 UTF-8/pl_PL.UTF-8 UTF-8/' /etc/locale.gen
# generate chosen locale
RUN locale-gen pl_PL.UTF-8
# set system-wide locale settings
ENV LANG pl_PL.UTF-8
ENV LANGUAGE pl_PL
ENV LC_ALL pl_PL.UTF-8
# verify modified configuration
RUN dpkg-reconfigure locales

# install the required libraries
RUN apt-get install -y protobuf-compiler \
  python-pil \
  python-lxml \
  python-tk \
	libkrb5-dev

RUN pip install --user Cython
RUN pip install --user contextlib2
RUN pip install --user jupyter
RUN pip install --user matplotlib

WORKDIR object_detection

EXPOSE 8888
EXPOSE 6006
