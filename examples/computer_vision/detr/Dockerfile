FROM determinedai/environments:cuda-10.0-pytorch-1.4-tf-1.15-gpu-0.5.0
RUN git clone https://github.com/facebookresearch/detr.git
RUN cd detr && git reset --hard 4e1a9281bc5621dcd65f3438631de25e255c4269 && cd ..
RUN pip install attrdict
RUN pip install pycocotools

RUN pip install determined
