FROM determinedai/environments:cuda-10.2-base-gpu-0.20.1

RUN pip install tensorboard cmake onnx   # cmake from apt-get is too old
RUN pip install torch==1.10 torchvision==0.11.1 -f https://download.pytorch.org/whl/cu101/torch_stable.html

RUN pip install 'git+https://github.com/facebookresearch/fvcore'
# install detectron2
RUN git clone https://github.com/facebookresearch/detectron2 detectron2_repo
# set FORCE_CUDA because during `docker build` cuda is not accessible
ENV FORCE_CUDA="1"
# This will by default build detectron2 for all common cuda architectures and take a lot more time,
# because inside `docker build`, there is no way to tell which architecture will be used.
#ARG TORCH_CUDA_ARCH_LIST="Kepler;Kepler+Tesla;Maxwell;Maxwell+Tegra;Pascal;Volta;Turing"
ARG TORCH_CUDA_ARCH_LIST="Kepler;Kepler+Tesla"
ENV TORCH_CUDA_ARCH_LIST="${TORCH_CUDA_ARCH_LIST}"

RUN pip install -e detectron2_repo

RUN pip install horovod==0.24.2

