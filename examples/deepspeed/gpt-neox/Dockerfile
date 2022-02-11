FROM determinedai/environments:cuda-11.1-pytorch-1.9-lightning-1.3-tf-2.4-gpt-neox-deepspeed-gpu-0.17.9

# Install deepspeed & dependencies
RUN apt-get install -y mpich

RUN git clone -b determined https://github.com/determined-ai/gpt-neox.git

RUN python gpt-neox/megatron/fused_kernels/setup.py install
RUN pip install -r gpt-neox/requirements/requirements.txt

