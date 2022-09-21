FROM determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-gpt-neox-deepspeed-gpu-096d730

# Install deepspeed & dependencies
RUN apt-get install -y mpich

# Pass in --build-args CACHEBUST=$(date +%s) to docker build command to invalidate cache
# on rebuild when github repo changes.
ARG CACHEBUST=1
RUN git clone -b determined https://github.com/determined-ai/gpt-neox.git

RUN python gpt-neox/megatron/fused_kernels/setup.py install
RUN pip install -r gpt-neox/requirements/requirements.txt
RUN cd gpt-neox; python prepare_data.py -d ./data
