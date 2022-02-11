export PYTHONPATH=$PYTHONPATH:/gpt-neox

python /gpt-neox/prepare_data.py -d /run/determined/workdir/shared_fs/data

mkdir -p /tmp/checkpoints
mkdir -p /tmp/logs



