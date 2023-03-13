# Update deepspeed
pip install "deepspeed[autotuning_ml]"==0.7.5
# Hack for seeing DEBUG logs from deepspeed
# sed -i 's/level=logging.INFO/level=logging.DEBUG/g' /opt/conda/lib/python3.8/site-packages/deepspeed/utils/logging.py
