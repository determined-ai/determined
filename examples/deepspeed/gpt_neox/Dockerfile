FROM determinedai/environments:cuda-11.3-pytorch-1.10-lightning-1.5-tf-2.8-gpt-neox-deepspeed-gpu-83dbcaa

# Install deepspeed & dependencies
RUN apt-get install -y mpich

RUN git clone -b determined https://github.com/determined-ai/gpt-neox.git

RUN python gpt-neox/megatron/fused_kernels/setup.py install
RUN pip install -r gpt-neox/requirements/requirements.txt
RUN cd gpt-neox; python prepare_data.py -d ./data
