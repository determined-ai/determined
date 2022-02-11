export PYTHONPATH=$PYTHONPATH:/gpt-neox

cd /gpt-neox
python prepare_data.py -d /run/determined/workdir/shared_fs/data

cd /run/determined/workdir
mkdir -p /tmp/checkpoints
mkdir -p /tmp/logs



