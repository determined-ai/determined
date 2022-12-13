FROM determinedai/model-hub-transformers:0.18.4-dev0 as model_hub

RUN mkdir /wheels && pip wheel -w /wheels model_hub==0.18.4

FROM determinedai/environments:cuda-11.3-pytorch-1.10-tf-2.8-deepspeed-0.7.0-gpu-0.19.4 as runtime

ADD requirements.txt /tmp/requirements.txt
RUN pip install -r /tmp/requirements.txt

COPY --from=model_hub /wheels/model_hub-0.18.4-py3-none-any.whl /tmp/model_hub_wheels/model_hub-0.18.4-py3-none-any.whl
RUN pip install --find-links=/tmp/model_hub_wheel model-hub
