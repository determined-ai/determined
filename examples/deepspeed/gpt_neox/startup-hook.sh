export PYTHONPATH=$PYTHONPATH:/gpt-neox

# Copy dataset from docker image to shared filesystem
mkdir -p /run/determined/workdir/shared_fs/data
cp -r -n /gpt-neox/data /run/determined/workdir/shared_fs/

cd /run/determined/workdir
cp gpt_neox_config/determined_cluster.yml /gpt-neox/configs



