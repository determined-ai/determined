ARG BASE_IMAGE
# This will be an image from determinedai/environments
FROM ${BASE_IMAGE}

ARG TRANSFORMERS_VERSION
ARG DATASETS_VERSION
RUN pip install transformers==${TRANSFORMERS_VERSION} datasets==${DATASETS_VERSION} attrdict
RUN pip install sentencepiece!=0.1.92 protobuf sklearn conllu seqeval


# Wheel must be built before building the docker image
RUN mkdir -p /tmp/model-hub-wheel
ADD dist /tmp/model-hub-wheel
ARG MODEL_HUB_VERSION
RUN python -m pip install --find-links=/tmp/model-hub-wheel model-hub==${MODEL_HUB_VERSION}
