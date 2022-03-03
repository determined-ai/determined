export PYTHONPATH=$PYTHONPATH:/gpt-neox

mkdir -p /tmp/checkpoints
mkdir -p /tmp/logs

cd /gpt-neox
python prepare_data.py -d /run/determined/workdir/shared_fs/data

cd /run/determined/workdir
cp gpt_neox_config/determined_cluster.yml /gpt-neox/configs



