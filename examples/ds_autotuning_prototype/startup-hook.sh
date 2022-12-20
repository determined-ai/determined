# Update deepspeed
pip install "deepspeed[autotuning_ml]"==0.7.5 timm==0.6.7 torchmetrics==0.9.2 pyarrow
# Hack for seeing DEBUG logs from deepspeed
sed -i 's/level=logging.INFO/level=logging.DEBUG/g' /opt/conda/lib/python3.8/site-packages/deepspeed/utils/logging.py
