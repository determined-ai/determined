FROM determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpt-neox-deepspeed-gpu-ad0591c

# Install deepspeed & dependencies
RUN apt-get install -y mpich

# Pass in --build-args CACHEBUST=$(date +%s) to docker build command to invalidate cache
# on rebuild when github repo changes.
ARG CACHEBUST=1
RUN git clone -b determined https://github.com/determined-ai/gpt-neox.git

RUN python gpt-neox/megatron/fused_kernels/setup.py install
RUN pip install -r gpt-neox/requirements/requirements.txt
RUN gsutil cp -r gs://determined-ai-public-datasets/text_data /gpt-neox && mv /gpt-neox/text_data /gpt-neox/data
RUN chmod -R 777 /gpt-neox
RUN chmod -R 777 /tmp
